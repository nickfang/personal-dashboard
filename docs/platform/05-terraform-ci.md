# Phase 5 — Terraform in CI

The last phase. Terraform stops being something you run and becomes something the repository runs:
plan on every pull request, apply to staging on merge, apply to prod behind an approval.

Everything needed is already in place — remote state with locking (Phase 2), decomposed stacks
(Phase 3), and per-tier runner identities that only a job declaring the matching `environment:` can
impersonate (Phase 1). This phase is mostly YAML.

## 1. Separate identities for plan and apply

Phase 1 created three runners with write access. Pull request plans shouldn't use them.

A plan reads state and calls read APIs, and that's all it needs. Giving PR jobs an identity that can
also write means any change to a workflow file — reviewable by whoever opened the PR — runs with
credentials that can modify infrastructure. Splitting them costs a few lines and removes the whole
category.

Add to `infra/platform/runner_iam.tf`:

```hcl
resource "google_service_account" "tf_planner" {
  for_each = toset(["dev", "staging", "prod"])

  account_id   = "tf-planner-${each.key}"
  display_name = "Terraform planner, read-only (${each.key})"
}

resource "google_service_account_iam_member" "tf_planner_wif" {
  for_each = toset(["dev", "staging", "prod"])

  service_account_id = google_service_account.tf_planner[each.key].name
  role               = "roles/iam.workloadIdentityUser"
  member             = "principalSet://iam.googleapis.com/${google_iam_workload_identity_pool.github.name}/attribute.environment/${each.key}-plan"
}

locals {
  planner_roles = [
    "roles/viewer",
    "roles/iam.securityReviewer",   # plan reads IAM policies; roles/viewer alone does not cover them
  ]
}

resource "google_project_iam_member" "planner_staging" {
  for_each = toset(local.planner_roles)

  project = var.staging_project_id
  role    = each.value
  member  = "serviceAccount:${google_service_account.tf_planner["staging"].email}"
}

resource "google_project_iam_member" "planner_prod" {
  for_each = toset(local.planner_roles)

  project = var.prod_project_id
  role    = each.value
  member  = "serviceAccount:${google_service_account.tf_planner["prod"].email}"
}

# Read state, never write it.
resource "google_storage_bucket_iam_member" "planner_state" {
  for_each = toset(["dev", "staging", "prod"])

  bucket = "${var.state_bucket_prefix}-tfstate-${each.key}"
  role   = "roles/storage.objectViewer"
  member = "serviceAccount:${google_service_account.tf_planner[each.key].email}"
}
```

Note the principalSet binds to `attribute.environment/staging-plan`, a *different* GitHub
environment from `staging`. That's what keeps the two apart: a PR job declares
`environment: staging-plan` and can only ever reach the read-only identity.

> **`terraform plan` normally takes a state lock.** With `objectViewer` it can't, and the run fails
> before planning anything. Pass `-lock=false` on plan-only jobs. It's safe precisely because the
> job cannot write state — the lock exists to protect writers.

Add the outputs:

```hcl
# infra/platform/outputs.tf
output "planner_service_accounts" {
  description = "Map of tier to read-only planner service account email"
  value       = { for k, sa in google_service_account.tf_planner : k => sa.email }
}
```

## 2. GitHub environments

Create six under **Settings → Environments**:

| Environment | Purpose | Protection |
|---|---|---|
| `staging-plan` | PR plans against staging | none |
| `staging` | applies to staging | none |
| `prod-plan` | PR plans against prod | none |
| `prod` | **the approval gate only** — one no-op job | **required reviewers: you**; branch rule: `main` and tags |
| `prod-apply` | the jobs that actually apply to prod | none — reachable only downstream of `prod` |
| `dev-plan` | plans for dev stacks | none |
| `dev` | ephemeral env create/destroy | none |

The `prod` / `prod-apply` split is what keeps a release to one approval instead of five; §5 explains
the mechanism. If you'd rather start simple, point everything at `prod` and accept the prompts —
just don't let that become the reason the gate gets removed later.

The protection rules on `prod` are the real approval gate. GitHub evaluates them *before* the job
starts, which means before an OIDC token is issued — so an unapproved prod apply never gets
credentials at all. The gate is enforced by IAM, not by a step in a workflow that someone could
reorder.

Per-environment variables:

| Variable | `staging` / `staging-plan` | `prod-apply` / `prod-plan` |
|---|---|---|
| `PLATFORM_PROJECT_ID` | `fang-dash-platform` | `fang-dash-platform` |
| `WIF_PROVIDER` | from `terraform output wif_provider` | same |
| `TF_RUNNER` | `tf-runner-staging@…` | `tf-runner-prod@…` |
| `TF_PLANNER` | `tf-planner-staging@…` | `tf-planner-prod@…` |
| `DEPLOYER_SA` | `deployer-staging@…` | `deployer-prod@…` |
| `STATE_BUCKET` | `fang-dash-tfstate-staging` | `fang-dash-tfstate-prod` |
| `GCP_REGION` | `us-central1` | `us-central1` |

All `vars`, no `secrets` — consistent with `_deploy-service.yml`, which already uses Workload
Identity Federation and holds no service account keys.

> **Your existing deploy workflows use `environment: production`, not `prod`.** Six caller workflows
> declare it. The WIF principalSets in Phase 1 use `prod`, and the two strings have to match exactly
> or the deploy identity can't be impersonated. Either rename the GitHub environment to `prod` and
> update the six callers, or change the principalSet to `production`. Renaming the environment is
> cleaner, but note it drops any protection rules attached to the old one — re-add them afterwards.

## 3. The reusable workflow

```yaml
# .github/workflows/_terraform.yml
name: Terraform

on:
  workflow_call:
    inputs:
      env_name:     { required: true,  type: string }   # staging | prod
      stack:        { required: true,  type: string }   # foundation | collector | ...
      prefix:       { required: true,  type: string }   # staging/collector-weather
      var_files:    { required: true,  type: string }   # space-separated, relative to the stack dir
      action:       { required: true,  type: string }   # plan | apply
      environment:  { required: true,  type: string }   # staging-plan | staging | prod-plan | prod

permissions:
  contents: read
  id-token: write
  pull-requests: write

jobs:
  terraform:
    runs-on: ubuntu-latest
    environment: ${{ inputs.environment }}

    # Never let two applies touch one state prefix at once. The GCS lock would
    # catch it, but failing here is clearer than failing mid-apply.
    concurrency:
      group: tf-${{ inputs.prefix }}
      cancel-in-progress: false

    env:
      TF_IN_AUTOMATION: "true"

    steps:
      - uses: actions/checkout@v4

      - uses: google-github-actions/auth@v2
        with:
          project_id: ${{ vars.PLATFORM_PROJECT_ID }}
          workload_identity_provider: ${{ vars.WIF_PROVIDER }}
          service_account: ${{ inputs.action == 'apply' && vars.TF_RUNNER || vars.TF_PLANNER }}

      - uses: hashicorp/setup-terraform@v3
        with:
          terraform_version: "1.15.8"

      - name: Init
        run: |
          terraform -chdir=infra/stacks/${{ inputs.stack }} init -input=false -reconfigure \
            -backend-config="bucket=${{ vars.STATE_BUCKET }}" \
            -backend-config="prefix=${{ inputs.prefix }}"

      - name: Format check
        run: terraform -chdir=infra/stacks/${{ inputs.stack }} fmt -check -recursive

      - name: Validate
        run: terraform -chdir=infra/stacks/${{ inputs.stack }} validate

      - name: Plan
        id: plan
        run: |
          ARGS=""
          for f in ${{ inputs.var_files }}; do ARGS="$ARGS -var-file=$f"; done

          # -lock=false only when planning: the planner identity is read-only and
          # cannot take a lock. Applies always lock.
          LOCK="-lock=true"
          [ "${{ inputs.action }}" = "plan" ] && LOCK="-lock=false"

          terraform -chdir=infra/stacks/${{ inputs.stack }} plan \
            -input=false -no-color $LOCK $ARGS \
            -out=tfplan | tee /tmp/plan.txt

      - name: Comment plan on PR
        if: inputs.action == 'plan' && github.event_name == 'pull_request'
        env:
          GH_TOKEN: ${{ github.token }}
        run: |
          {
            echo "### \`${{ inputs.prefix }}\`"
            echo
            echo '```terraform'
            # Plans can be enormous; the tail holds the summary and the last changes.
            tail -c 60000 /tmp/plan.txt
            echo '```'
          } > /tmp/comment.md
          gh pr comment ${{ github.event.pull_request.number }} --body-file /tmp/comment.md

      - name: Apply
        if: inputs.action == 'apply'
        run: terraform -chdir=infra/stacks/${{ inputs.stack }} apply -input=false -no-color tfplan
```

Two details that matter:

- **Apply consumes the saved `tfplan`**, not a fresh plan. Terraform applies exactly what was
  computed and refuses if the state moved underneath it. Re-planning at apply time means what gets
  applied is not what anyone reviewed.
- **The identity is chosen by `action`.** A plan job physically cannot get the write identity, so a
  workflow edit that flips `plan` to `apply` still has to pass through the `prod` environment's
  protection rules to get anywhere.

## 4. Staging: plan on PR, apply on merge

```yaml
# .github/workflows/terraform-staging.yml
name: Terraform — staging

on:
  pull_request:
    branches: [main]
    paths: ["infra/**"]
  push:
    branches: [main]
    paths: ["infra/**"]

jobs:
  foundation:
    uses: ./.github/workflows/_terraform.yml
    with:
      env_name:    staging
      stack:       foundation
      prefix:      staging/foundation
      var_files:   "../../envs/staging.tfvars"
      action:      ${{ github.event_name == 'push' && 'apply' || 'plan' }}
      environment: ${{ github.event_name == 'push' && 'staging' || 'staging-plan' }}

  collectors:
    needs: foundation
    strategy:
      fail-fast: false
      matrix:
        component: [weather-collector, pollen-collector, forecast-collector]
    uses: ./.github/workflows/_terraform.yml
    with:
      env_name:    staging
      stack:       collector
      prefix:      staging/${{ matrix.component }}
      var_files:   "../../envs/staging.tfvars ../../envs/staging/${{ matrix.component }}.tfvars"
      action:      ${{ github.event_name == 'push' && 'apply' || 'plan' }}
      environment: ${{ github.event_name == 'push' && 'staging' || 'staging-plan' }}

  providers:
    needs: foundation
    strategy:
      fail-fast: false
      matrix:
        component: [weather-provider, pollen-provider]
    uses: ./.github/workflows/_terraform.yml
    with:
      env_name:    staging
      stack:       provider
      prefix:      staging/${{ matrix.component }}
      var_files:   "../../envs/staging.tfvars ../../envs/staging/${{ matrix.component }}.tfvars"
      action:      ${{ github.event_name == 'push' && 'apply' || 'plan' }}
      environment: ${{ github.event_name == 'push' && 'staging' || 'staging-plan' }}

  aggregator:
    needs: providers
    uses: ./.github/workflows/_terraform.yml
    with:
      env_name:    staging
      stack:       aggregator
      prefix:      staging/dashboard-api
      var_files:   "../../envs/staging.tfvars ../../envs/staging/aggregator.tfvars"
      action:      ${{ github.event_name == 'push' && 'apply' || 'plan' }}
      environment: ${{ github.event_name == 'push' && 'staging' || 'staging-plan' }}
```

`needs:` reproduces the dependency order that `terraform_remote_state` hid from Terraform — the same
ordering `components.json` encodes for the scripts. The collectors run in parallel with the
providers because nothing connects them, which is the decomposition paying off.

> **Fork PRs get no credentials.** GitHub withholds `id-token: write` and environment variables from
> pull requests opened from forks, so those jobs fail at the auth step rather than running with
> access. That's correct behaviour — don't reach for `pull_request_target` to "fix" it. That trigger
> runs the base repo's workflow with full credentials against the fork's code, which is how CI
> systems get compromised.

## 5. Prod: apply on release, behind approval

```yaml
# .github/workflows/terraform-prod.yml
name: Terraform — prod

on:
  release:
    types: [created]
  workflow_dispatch:

jobs:
  foundation:
    uses: ./.github/workflows/_terraform.yml
    with:
      env_name:    prod
      stack:       foundation
      prefix:      prod/foundation
      var_files:   "../../envs/prod.tfvars"
      action:      apply
      environment: prod

  collectors:
    needs: foundation
    strategy:
      fail-fast: false
      matrix:
        component: [weather-collector, pollen-collector, forecast-collector]
    uses: ./.github/workflows/_terraform.yml
    with:
      env_name:    prod
      stack:       collector
      prefix:      prod/${{ matrix.component }}
      var_files:   "../../envs/prod.tfvars ../../envs/prod/${{ matrix.component }}.tfvars"
      action:      apply
      environment: prod

  providers:
    needs: foundation
    strategy:
      fail-fast: false
      matrix:
        component: [weather-provider, pollen-provider]
    uses: ./.github/workflows/_terraform.yml
    with:
      env_name:    prod
      stack:       provider
      prefix:      prod/${{ matrix.component }}
      var_files:   "../../envs/prod.tfvars ../../envs/prod/${{ matrix.component }}.tfvars"
      action:      apply
      environment: prod

  aggregator:
    needs: providers
    uses: ./.github/workflows/_terraform.yml
    with:
      env_name:    prod
      stack:       aggregator
      prefix:      prod/dashboard-api
      var_files:   "../../envs/prod.tfvars ../../envs/prod/aggregator.tfvars"
      action:      apply
      environment: prod
```

As written, every job targets `environment: prod`, so **every one of them waits separately** — five
approval prompts for one release. That's not a security property, it's an annoyance, and annoyances
of that shape get resolved by deleting the gate. Better to have one gate that people respect than
five they route around.

Restructure it as a single approval that releases the rest:

```yaml
jobs:
  # The only job bound to the protected environment. Approving it is approving
  # the release. It does no work -- its entire purpose is to be the gate.
  approve:
    runs-on: ubuntu-latest
    environment: prod
    steps:
      - run: echo "Release approved for ${{ github.ref_name }}"

  foundation:
    needs: approve
    uses: ./.github/workflows/_terraform.yml
    with:
      env_name:    prod
      stack:       foundation
      prefix:      prod/foundation
      var_files:   "../../envs/prod.tfvars"
      action:      apply
      environment: prod-apply      # unprotected; reachable only after `approve`
```

with every downstream job carrying `needs:` back to `approve` (directly or transitively) and using
`prod-apply`.

That means a second GitHub environment, `prod-apply`, with no protection rules but the same
variables — and the WIF binding for `tf-runner-prod` moves to `attribute.environment/prod-apply`:

```hcl
resource "google_service_account_iam_member" "tf_runner_wif" {
  for_each = {
    dev     = "dev"
    staging = "staging"
    prod    = "prod-apply"     # the gate is `prod`; the work runs as `prod-apply`
  }

  service_account_id = google_service_account.tf_runner[each.key].name
  role               = "roles/iam.workloadIdentityUser"
  member             = "principalSet://iam.googleapis.com/${google_iam_workload_identity_pool.github.name}/attribute.environment/${each.value}"
}
```

The security property survives intact: `prod-apply` is only reachable from a job whose `needs:`
chain runs through `approve`, and `approve` cannot start until a human approves it. GitHub evaluates
protection rules before issuing any token, so an unapproved release still never gets credentials.

> The obvious shortcut — one job that applies all seven stacks in sequence — is worse. You lose the
> parallelism between collectors and providers, and a failure halfway leaves you re-running
> everything to retry one stack.

You'll also want prod plans on PRs, so changes are reviewable before a release. Add a
`pull_request` trigger with `action: plan` / `environment: prod-plan`, mirroring the staging file.

## 6. How this fits the existing deploy workflows

`_deploy-service.yml` **does** change, in two ways — the identity it assumes and the registry it
pushes to. The twelve caller workflows don't change at all.

The split of responsibility stays exactly as `docs/ARCHITECTURE_INFRASTRUCTURE.md` describes it:

- **Terraform owns infrastructure** — services, jobs, IAM, schedulers, databases
- **Deploy workflows own image tags** — `gcloud run services update --image`

The `lifecycle { ignore_changes = [...containers[0].image] }` blocks are the seam, and they were
already there. Phase 3 only changed where Terraform gets the *initial* tag from: an input instead of
a `gcloud builds submit` it ran itself.

### The two edits to `_deploy-service.yml`

**Identity.** The workflow currently assumes `github-actions-sa`, created by `module.github_oidc` in
the old roots. That module is deleted in Phase 3. It now assumes the tier's `deployer-<tier>` SA
through the platform WIF pool from
[01-bootstrap-and-platform.md §3.4](01-bootstrap-and-platform.md).

```diff
       - uses: google-github-actions/auth@v2
         with:
-          project_id: ${{ vars.GCP_PROJECT_ID }}
+          project_id: ${{ vars.PLATFORM_PROJECT_ID }}
           workload_identity_provider: ${{ vars.WIF_PROVIDER }}
-          service_account: ${{ vars.GCP_SA_EMAIL }}
+          service_account: ${{ vars.DEPLOYER_SA }}
```

The caller workflows already bind to a GitHub environment (`staging` / `production`), which is what
mints the `environment` claim the new WIF binding requires. One caveat: the prod callers use
`environment: production` while the runner bindings use `prod`. Either rename the GitHub environment
to `prod` or change the principalSet to match — they have to be the same string.

**Registry.** Images move to the shared repository in the platform project:

```diff
-  IMAGE: ${{ vars.GCP_REGION }}-docker.pkg.dev/${{ vars.GCP_PROJECT_ID }}/personal-dashboard/...
+  IMAGE: ${{ vars.GCP_REGION }}-docker.pkg.dev/${{ vars.PLATFORM_PROJECT_ID }}/personal-dashboard/...
```

`roles/artifactregistry.writer` for the deploy SAs is already granted in
[01-bootstrap-and-platform.md §3.4](01-bootstrap-and-platform.md), alongside the runners'
`artifactregistry.admin` from [03-stacks.md §2.3](03-stacks.md).

**Do both of these, and confirm one green deploy, before Phase 3 removes the old roots.** The old
WIF pool soft-deletes for 30 days holding its ID, so there is no quick rollback once it's gone.

## 7. What you have at the end

- Every infra change gets a plan posted to its PR, generated by an identity that cannot write
- Merging to `main` applies to staging automatically
- Prod applies require your approval, enforced before credentials exist
- A compromised staging workflow cannot read prod state, let alone change prod
- An engineer gets a scoped environment from a dropdown, and it deletes itself
- No service account keys anywhere; org policy makes creating one impossible

---

## Verification gate

```bash
# 1. A PR gets a plan, and no credentials leak into logs
git checkout -b test-tf-ci
# make a visible no-op change, e.g. bump max_instances in envs/staging.tfvars
gh pr create --fill
gh run watch
```
A comment appears on the PR showing the diff. Check the run log for the auth step — the token must
be masked.

```bash
# 2. The plan identity really is read-only
```
In the run log, confirm the plan job authenticated as `tf-planner-staging@…`, not `tf-runner-…`.
Then, as a deliberate test, temporarily set `action: apply` on the PR job and re-run: it must fail
with a permissions error on the state bucket. Revert afterwards.

```bash
# 3. Merging applies to staging
gh pr merge --squash
gh run watch
```
The apply job runs against `environment: staging` and the change lands.

```bash
# 4. Prod waits for a human
gh workflow run terraform-prod.yml
gh run list --workflow=terraform-prod.yml
```
Status `waiting`. The GitHub UI shows "Review deployments". Nothing runs until you approve.

```bash
# 5. Cross-tier access is denied
```
Temporarily point a staging job's `STATE_BUCKET` at `fang-dash-tfstate-prod` and run it. It must
fail with a 403 at init. This is the single most important check in the tutorial — it's the
difference between having designed isolation and having it. Revert immediately after.

```bash
# 6. Keys really cannot be created
gcloud iam service-accounts keys create /tmp/k.json \
  --iam-account="tf-runner-prod@fang-dash-platform.iam.gserviceaccount.com"
```
Fails on the org policy from Phase 1.

---

## Where to go next

- **Drift detection.** A nightly scheduled plan across every stack that opens an issue when
  something changed outside Terraform. Cheap, and it catches console edits.
- **Policy as code.** `terraform plan -out` piped into Conftest or OPA, checking things like "no
  `allUsers` binding outside the aggregator." A natural extension of the plan job.
- **Cost visibility.** Infracost on PR plans. Now that sizing is a per-tier profile, the cost of
  `min_instances = 1` becomes a reviewable number.
- **Per-PR environments.** [04-ephemeral-envs.md §9](04-ephemeral-envs.md) — the automatic version,
  once the `workflow_dispatch` form has proven itself.

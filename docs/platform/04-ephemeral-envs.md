# Phase 4 — Ephemeral environments

The payoff phase. An engineer working on the forecast collector gets a real environment containing
foundation and that one collector — not a full deploy, not a shared sandbox they have to coordinate
over.

```bash
infra/bin/env-create pr-123 --component forecast-collector --ttl 3
```

Everything here rests on two things already built: **Phase 1's dev runner**, which can create and
destroy projects but only inside `folders/dev`, and **Phase 3's decomposition**, which makes "just
this component" an expressible thing.

## 1. What an environment is

Three layers, and it's worth being precise because the scripts encode this:

1. **A GCP project** in `folders/dev`, labelled with an owner and an expiry
2. **A foundation stack** — APIs, Firestore, secrets, registry access
3. **Zero or more component stacks** — whichever the engineer asked for

State for all of it lives under one prefix tree:

```
gs://fang-dash-tfstate-dev/
  dev/pr-123/project
  dev/pr-123/foundation
  dev/pr-123/collector-forecast
```

Which makes teardown pleasantly blunt: delete the project, delete the prefix tree, done.

## 2. The project factory

```hcl
# infra/stacks/project/main.tf
terraform {
  required_version = ">= 1.13"
  backend "gcs" {}
  required_providers {
    google = { source = "hashicorp/google", version = "~> 5.0" }
  }
}

provider "google" {
  region = var.region
}

resource "google_project" "env" {
  name       = var.env_name
  project_id = "${var.project_prefix}-${var.env_name}"
  folder_id  = var.folder_dev

  billing_account     = var.billing_account
  auto_create_network = false

  labels = {
    environment = "dev"
    owner       = var.owner
    expires-at  = var.expires_at    # YYYY-MM-DD; the reaper reads this
    managed-by  = "terraform"
  }
}

# The engineer who owns this environment gets to work in it. Nobody else needs to.
resource "google_project_iam_member" "owner_access" {
  for_each = toset([
    "roles/run.developer",
    "roles/logging.viewer",
    "roles/datastore.user",
    "roles/cloudscheduler.admin",
  ])

  project = google_project.env.project_id
  role    = each.value
  member  = "user:${var.owner_email}"
}
```

```hcl
# infra/stacks/project/variables.tf
variable "env_name" {
  description = "Environment name, e.g. pr-123 or nick-sandbox. Lowercase, dashes."
  type        = string

  validation {
    condition     = can(regex("^[a-z][a-z0-9-]{1,20}$", var.env_name))
    error_message = "env_name must be lowercase alphanumeric with dashes, 2-21 chars."
  }
}

variable "project_prefix"  { type = string }
variable "folder_dev"      { type = string }
variable "billing_account" { type = string }
variable "owner"           { type = string }
variable "owner_email"     { type = string }

variable "region" {
  type    = string
  default = "us-central1"
}

variable "expires_at" {
  description = "Date after which the reaper may delete this, as YYYY-MM-DD"
  type        = string
}
```

```hcl
# infra/stacks/project/outputs.tf
output "project_id" {
  value = google_project.env.project_id
}

output "project_number" {
  value = google_project.env.number
}
```

> **Project IDs are held for 30 days after deletion.** Delete `fang-dash-pr-123` today and you
> cannot recreate that ID until next month. Names tied to something naturally unique — a PR number,
> a date — avoid this. Reusing `nick-sandbox` repeatedly will bite you.

## 3. The component manifest

The scripts need to know what components exist, which stack builds them, and what they depend on.
Put that in data rather than in a case statement, so CI can read the same file.

```json
// infra/components.json
{
  "weather-collector": {
    "stack": "collector",
    "vars": "collector-weather.tfvars",
    "requires": []
  },
  "pollen-collector": {
    "stack": "collector",
    "vars": "collector-pollen.tfvars",
    "requires": []
  },
  "forecast-collector": {
    "stack": "collector",
    "vars": "collector-forecast.tfvars",
    "requires": []
  },
  "weather-provider": {
    "stack": "provider",
    "vars": "provider-weather.tfvars",
    "requires": []
  },
  "pollen-provider": {
    "stack": "provider",
    "vars": "provider-pollen.tfvars",
    "requires": []
  },
  "dashboard-api": {
    "stack": "aggregator",
    "vars": "aggregator.tfvars",
    "requires": ["weather-provider", "pollen-provider"]
  }
}
```

`requires` is what Phase 3 gave up when the aggregator started reading provider URLs through
`terraform_remote_state`. Terraform can no longer see that ordering, so it lives here.

## 4. `infra/bin/env-create`

```bash
#!/usr/bin/env bash
# Create an ephemeral dev environment with a subset of components.
#
#   env-create pr-123 --component forecast-collector --ttl 3
#   env-create nick-sandbox --component dashboard-api --image-tag abc1234
#
set -euo pipefail

INFRA_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
STATE_BUCKET="fang-dash-tfstate-dev"
TIER="dev"

ENV_NAME="${1:?usage: env-create <env-name> [--component NAME]... [--ttl DAYS] [--image-tag TAG]}"
shift

COMPONENTS=()
TTL_DAYS=3

# Empty means the stacks fall back to the placeholder image. An environment
# then comes up even when nothing has been built for the branch yet, and the
# first deploy replaces it.
IMAGE_TAG=""

while [[ $# -gt 0 ]]; do
  case "$1" in
    --component) COMPONENTS+=("$2"); shift 2 ;;
    --ttl)       TTL_DAYS="$2";      shift 2 ;;
    --image-tag) IMAGE_TAG="$2";     shift 2 ;;
    *) echo "unknown argument: $1" >&2; exit 1 ;;
  esac
done

MANIFEST="$INFRA_DIR/components.json"
EXPIRES_AT="$(date -u -d "+${TTL_DAYS} days" +%Y-%m-%d)"
OWNER="$(gcloud config get-value account 2>/dev/null)"
OWNER_LABEL="$(echo "$OWNER" | cut -d@ -f1 | tr '.' '-' | tr '[:upper:]' '[:lower:]')"

# --- expand dependencies -----------------------------------------------------
# Asking for dashboard-api implies the two providers it reads state from.
# Order matters: requirements are emitted before the component that needs them.
expand() {
  local out=() dep
  for c in "${COMPONENTS[@]}"; do
    while read -r dep; do
      [[ -n "$dep" ]] || continue
      out+=("$dep")
    done < <(jq -r --arg c "$c" '.[$c].requires[]?' "$MANIFEST")
    out+=("$c")
  done
  printf '%s\n' "${out[@]}" | awk '!seen[$0]++'
}

# --- validate before creating anything ---------------------------------------
# An environment with no components is almost always a forgotten flag, and
# finding out after a project has been created and billed is a poor way to
# learn it. Note this guard also protects the loops below: under `set -u`,
# expanding an empty array is an error on bash before 4.4 -- which includes
# the bash that ships with macOS.
if [[ ${#COMPONENTS[@]} -eq 0 ]]; then
  echo "no components requested; pass at least one --component" >&2
  echo "known: $(jq -r 'keys | join(", ")' "$MANIFEST")" >&2
  exit 1
fi

for c in "${COMPONENTS[@]}"; do
  if ! jq -e --arg c "$c" 'has($c)' "$MANIFEST" >/dev/null; then
    echo "unknown component: $c" >&2
    echo "known: $(jq -r 'keys | join(", ")' "$MANIFEST")" >&2
    exit 1
  fi
done

# --- refuse to run past the environment cap ----------------------------------
# The dev runner holds projectCreator, projectDeleter and billing.user, and
# this script is reachable from a workflow_dispatch form. Nothing else stops a
# retry loop -- or an impatient human -- from creating projects until quota is
# gone and the billing account has absorbed it. The cap is crude and effective.
MAX_ENVIRONMENTS="${MAX_ENVIRONMENTS:-6}"
LIVE="$(gcloud projects list \
  --filter="parent.id=${FOLDER_DEV:?set FOLDER_DEV} AND lifecycleState=ACTIVE" \
  --format="value(projectId)" | wc -l)"

if [[ "$LIVE" -ge "$MAX_ENVIRONMENTS" ]]; then
  echo "refusing: $LIVE live dev environments, cap is $MAX_ENVIRONMENTS" >&2
  echo "destroy one with env-destroy, or raise MAX_ENVIRONMENTS deliberately" >&2
  exit 1
fi

apply_stack() {
  local stack="$1" prefix="$2"; shift 2
  echo "==> ${stack} (${prefix})"
  terraform -chdir="$INFRA_DIR/stacks/$stack" init -reconfigure -input=false \
    -backend-config="bucket=$STATE_BUCKET" \
    -backend-config="prefix=$prefix"
  terraform -chdir="$INFRA_DIR/stacks/$stack" apply -auto-approve -input=false "$@"
}

# --- 1. the project ----------------------------------------------------------
apply_stack project "${TIER}/${ENV_NAME}/project" \
  -var="env_name=$ENV_NAME" \
  -var="owner=$OWNER_LABEL" \
  -var="owner_email=$OWNER" \
  -var="expires_at=$EXPIRES_AT" \
  -var-file="$INFRA_DIR/envs/dev.tfvars"

PROJECT_ID="$(terraform -chdir="$INFRA_DIR/stacks/project" output -raw project_id)"

# --- 2. foundation -----------------------------------------------------------
apply_stack foundation "${TIER}/${ENV_NAME}/foundation" \
  -var="project_id=$PROJECT_ID" \
  -var-file="$INFRA_DIR/envs/dev.tfvars"

# --- 3. components, in dependency order --------------------------------------
for component in $(expand); do
  stack="$(jq -r --arg c "$component" '.[$c].stack' "$MANIFEST")"
  varfile="$(jq -r --arg c "$component" '.[$c].vars' "$MANIFEST")"

  apply_stack "$stack" "${TIER}/${ENV_NAME}/${component}" \
    -var="project_id=$PROJECT_ID" \
    -var="env_name=$ENV_NAME" \
    -var="state_bucket=$STATE_BUCKET" \
    -var="image_tag=$IMAGE_TAG" \
    -var-file="$INFRA_DIR/envs/dev.tfvars" \
    -var-file="$INFRA_DIR/envs/dev/$varfile"
done

cat <<EOF

Environment ready.

  name        $ENV_NAME
  project     $PROJECT_ID
  components  ${COMPONENTS[*]}
  image tag   $IMAGE_TAG
  expires     $EXPIRES_AT

  console  https://console.cloud.google.com/home/dashboard?project=$PROJECT_ID
  destroy  infra/bin/env-destroy $ENV_NAME
EOF
```

A few decisions in there worth noticing:

- **`-chdir` rather than `cd`.** Stack sources stay where they are, and the script can be run from
  anywhere without relative-path surprises — the exact failure mode that made `services_path` in the
  old `null_resource` so fragile.
- **`-reconfigure` on every init.** The same stack directory gets initialized against many different
  prefixes. Without it, Terraform tries to migrate state between them, which is not what you want.
- **Validation before creation.** A typo'd component name should fail in a second, not after a
  project has been created and billed.
- **The image tag is an input, never a build.** The image must already be in the shared registry.
  For a PR environment, the build workflow runs first and passes its SHA.

## 5. `infra/bin/env-destroy`

```bash
#!/usr/bin/env bash
# Destroy an ephemeral dev environment and its state.
set -euo pipefail

STATE_BUCKET="fang-dash-tfstate-dev"
PROJECT_PREFIX="fang-dash"
FOLDER_DEV="${FOLDER_DEV:?set FOLDER_DEV to the numeric dev folder ID}"
ENV_NAME="${1:?usage: env-destroy <env-name>}"
PROJECT_ID="${PROJECT_PREFIX}-${ENV_NAME}"

# A name prefix is not a safety boundary. Before deleting anything, confirm the
# project actually lives in folders/dev -- otherwise a mistyped or malicious
# argument reaches whatever project happens to match the prefix, and --yes
# skips the only other thing standing in the way.
PARENT="$(gcloud projects describe "$PROJECT_ID" --format="value(parent.id)" 2>/dev/null || true)"

if [[ -z "$PARENT" ]]; then
  echo "no such project: $PROJECT_ID" >&2
  exit 1
fi

if [[ "$PARENT" != "$FOLDER_DEV" ]]; then
  echo "refusing to delete $PROJECT_ID: parent is $PARENT, not the dev folder ($FOLDER_DEV)" >&2
  exit 1
fi

if [[ "${2:-}" != "--yes" ]]; then
  read -rp "Delete project $PROJECT_ID and all its state? [y/N] " ans
  [[ "$ans" == "y" ]] || exit 1
fi

# Deleting the project removes every resource in it, which is faster and far
# more reliable than destroying seven stacks in reverse dependency order --
# a single failed destroy would otherwise leave the environment half-alive.
gcloud projects delete "$PROJECT_ID" --quiet

# The state now describes nothing. Remove it so the prefix is reusable and
# `env-list` does not report a ghost.
gcloud storage rm -r "gs://${STATE_BUCKET}/dev/${ENV_NAME}/" 2>/dev/null || true

echo "Deleted $PROJECT_ID. Note: the project ID is reserved for 30 days."
```

Deleting the project rather than running `terraform destroy` per stack is deliberate. Ordered
destroys fail partway — a Firestore database with data, a soft-deleted resource, a dependency
Terraform can no longer see across states — and a half-destroyed environment is worse than either
outcome. Project deletion is atomic from your side.

## 6. `infra/bin/env-list`

```bash
#!/usr/bin/env bash
# List ephemeral environments and their expiry.
set -euo pipefail

FOLDER_DEV="${FOLDER_DEV:?set FOLDER_DEV to the numeric dev folder ID}"

gcloud projects list \
  --filter="parent.id=${FOLDER_DEV} AND lifecycleState=ACTIVE" \
  --format="table(
    projectId,
    labels.owner:label=OWNER,
    labels.expires-at:label=EXPIRES,
    createTime.date('%Y-%m-%d'):label=CREATED
  )"
```

## 7. A budget on the dev folder

A count cap in `env-create` only constrains the path that goes through `env-create`. Put a real
budget behind it, in `infra/platform`:

```hcl
# infra/platform/budgets.tf
resource "google_billing_budget" "dev" {
  billing_account = var.billing_account
  display_name    = "Dev environments"

  budget_filter {
    projects = []                                   # empty = the whole billing account
    resource_ancestors = ["folders/${var.folder_dev}"]
  }

  amount {
    specified_amount {
      currency_code = "USD"
      units         = "50"
    }
  }

  threshold_rules {
    threshold_percent = 0.5
  }
  threshold_rules {
    threshold_percent = 0.9
  }
  threshold_rules {
    threshold_percent = 1.0
  }
}
```

Budgets alert; they don't cap. Nothing here stops spend — it tells you it's happening. Combined with
`max_instances = 1` and paused schedulers in `envs/dev.tfvars`, the realistic failure mode is a slow
leak of forgotten environments rather than a sudden bill, and that's what the reaper is for.

Worth doing the same for staging and prod with different amounts, once you know what normal looks
like.

## 8. The reaper

Ephemeral environments are only ephemeral if something deletes them. Without this you'll exhaust
project quota in a fortnight and the factory becomes useless exactly when you've started relying
on it.

```yaml
# .github/workflows/reap-dev-environments.yml
name: Reap expired dev environments

on:
  schedule:
    - cron: "0 7 * * *"
  workflow_dispatch:

permissions:
  contents: read
  id-token: write

jobs:
  reap:
    runs-on: ubuntu-latest
    environment: dev          # required — this is what mints a token for tf-runner-dev
    steps:
      - uses: actions/checkout@v4

      - uses: google-github-actions/auth@v2
        with:
          project_id: ${{ vars.PLATFORM_PROJECT_ID }}
          workload_identity_provider: ${{ vars.WIF_PROVIDER }}
          service_account: ${{ vars.TF_RUNNER_DEV }}

      - uses: google-github-actions/setup-gcloud@v2

      - name: Delete expired environments
        env:
          FOLDER_DEV: ${{ vars.FOLDER_DEV }}
          STATE_BUCKET: fang-dash-tfstate-dev
        run: |
          set -euo pipefail

          # The folder filter below is the only thing keeping this loop away
          # from staging and prod. An unset repository variable would silently
          # empty it, so fail loudly instead.
          : "${FOLDER_DEV:?FOLDER_DEV is unset -- refusing to enumerate projects}"

          TODAY="$(date -u +%Y-%m-%d)"

          gcloud projects list \
            --filter="parent.id=${FOLDER_DEV} AND lifecycleState=ACTIVE" \
            --format="value(projectId, labels.expires-at)" \
          | while read -r project expires; do
              if [[ -z "$expires" ]]; then
                echo "SKIP  $project (no expires-at label)"
                continue
              fi
              if [[ "$expires" < "$TODAY" ]]; then
                echo "REAP  $project (expired $expires)"
                gcloud projects delete "$project" --quiet
                env_name="${project#fang-dash-}"
                gcloud storage rm -r "gs://${STATE_BUCKET}/dev/${env_name}/" 2>/dev/null || true
              else
                echo "KEEP  $project (expires $expires)"
              fi
            done
```

Note the unlabelled case is skipped rather than deleted. A project with no `expires-at` is either
hand-created or created by a version of the factory that predates the label — deleting on missing
metadata is how automation eats something it shouldn't. Skipping is the safe default; `env-list`
surfaces the strays.

> **This TTL is self-attested, and that's a real limitation.** Expiry lives in a label on the
> project itself, and the environment's owner has edit rights on their own project — so anyone can
> set `expires-at` to 2099 and the reaper will honour it. Nothing here is enforcing a lifetime; it
> is reminding cooperative people to clean up.
>
> That's fine for a team that trusts each other, and it's why the `MAX_ENVIRONMENTS` cap and the
> billing budget matter — those aren't self-attested. If you ever need the TTL to be a control
> rather than a convention, derive it from `createTime` (which the owner can't change) and enforce a
> maximum age regardless of the label:
>
> ```bash
> created=$(gcloud projects describe "$project" --format="value(createTime.date('%Y-%m-%d'))")
> # reap if createTime is older than MAX_AGE_DAYS, whatever the label says
> ```

## 9. The self-service frontend

You mentioned wanting "scripts or frontends that deploy what's needed." The scripts above are the
API. The cheapest usable UI on top is a `workflow_dispatch` form — no portal to host, and it works
for anyone with repo access rather than only people with `gcloud` configured.

```yaml
# .github/workflows/create-dev-environment.yml
name: Create dev environment

on:
  workflow_dispatch:
    inputs:
      env_name:
        description: "Environment name (e.g. pr-123, nick-sandbox)"
        required: true
      component:
        description: "Component to deploy"
        required: true
        type: choice
        options:
          - forecast-collector
          - weather-collector
          - pollen-collector
          - weather-provider
          - pollen-provider
          - dashboard-api
      ttl_days:
        description: "Days before automatic deletion"
        default: "3"
      image_tag:
        description: "Image tag to deploy (must exist in the shared registry)"
        default: "latest"

permissions:
  contents: read
  id-token: write

jobs:
  create:
    runs-on: ubuntu-latest
    environment: dev
    steps:
      - uses: actions/checkout@v4

      - uses: google-github-actions/auth@v2
        with:
          project_id: ${{ vars.PLATFORM_PROJECT_ID }}
          workload_identity_provider: ${{ vars.WIF_PROVIDER }}
          service_account: ${{ vars.TF_RUNNER_DEV }}

      - uses: google-github-actions/setup-gcloud@v2
      - uses: hashicorp/setup-terraform@v3
        with:
          terraform_version: "1.15.8"

      - run: |
          infra/bin/env-create "${{ inputs.env_name }}" \
            --component "${{ inputs.component }}" \
            --ttl "${{ inputs.ttl_days }}" \
            --image-tag "${{ inputs.image_tag }}"
```

That's the whole platform interface: a form, a dropdown fed by `components.json`, and a scheduled
job that cleans up. If you later want a Backstage portal or a CLI, it calls the same scripts — the
interface is already separated from the mechanism.

## 10. Wiring it to pull requests

The natural next step, once the manual version works: a workflow on `pull_request` that reads which
paths changed, maps them to components, builds the images, and creates an environment tagged with
the PR SHA. Then a `pull_request: [closed]` trigger calls `env-destroy`.

Get the `workflow_dispatch` version working first. Automatic per-PR environments amplify every rough
edge in the scripts, and you want the rough edges gone before they're firing on every push.

---

## Verification gate

The end-to-end test — this is the one that proves the whole tutorial worked.

```bash
# 1. Create an environment with exactly one collector
infra/bin/env-create test-forecast --component forecast-collector --ttl 1
```

```bash
# 2. Only the requested component exists.  Nothing else got dragged in.
gcloud run jobs list --project=fang-dash-test-forecast
gcloud run services list --project=fang-dash-test-forecast
```
One job (`forecast-collector-job`), **zero services**. Any provider or aggregator here means the
dependency expansion or the stack decomposition is wrong.

```bash
# 3. It actually runs
gcloud secrets versions add google-maps-api-key --project=fang-dash-test-forecast --data-file=- <<< "$REAL_KEY"
gcloud run jobs execute forecast-collector-job --region=us-central1 --project=fang-dash-test-forecast --wait
```
Completes successfully. If it fails pulling the image, revisit the Cloud Run Service Agent binding
in [03-stacks.md §2.3](03-stacks.md).

```bash
# 4. State is where it should be, and only what is needed exists
gcloud storage ls -r gs://fang-dash-tfstate-dev/dev/test-forecast/
```
Shows `project`, `foundation`, and `forecast-collector` — no provider or aggregator prefixes.

```bash
# 5. Isolation holds
gcloud projects get-iam-policy fang-dash-test-forecast --format=json | grep -c "tf-runner-prod"
```
Expect `0`.

```bash
# 6. Schedulers exist but are dormant — the thing that keeps dev cheap
gcloud scheduler jobs describe trigger-forecast-collector \
  --location=us-central1 --project=fang-dash-test-forecast \
  --format="value(state)"
```
Expect `PAUSED`. If it says `ENABLED`, every dev environment is calling the Google Maps API on a
cron and that is your largest bill.

```bash
# 7. The destroy guard refuses anything outside folders/dev
FOLDER_DEV=<your-dev-folder-id> infra/bin/env-destroy prod --yes
```
Must exit non-zero with *"refusing to delete ... not the dev folder"*, and must not call
`gcloud projects delete`. This is the check worth running before you trust the script or the reaper
with anything.

```bash
# 8. Teardown is clean
infra/bin/env-destroy test-forecast --yes
gcloud projects describe fang-dash-test-forecast --format="value(lifecycleState)"
```
`DELETE_REQUESTED`.

```bash
# 9. The reaper is honest about what it would do
gh workflow run reap-dev-environments.yml
gh run watch
```
Check the log lists `KEEP` and `SKIP` lines with reasons, and reaps only genuinely expired projects.

---

**Next:** [05-terraform-ci.md](05-terraform-ci.md) — the last phase: Terraform runs itself.

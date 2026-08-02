# Phase 2 — Move staging and prod state into GCS

Short phase, high stakes. You're moving the only authoritative record of your live infrastructure
off your laptop. Do staging first, verify it completely, then do prod.

## 1. Before you touch anything

Confirm both state files exist and are current:

```bash
cd infra/staging && terraform plan
cd ../prod && terraform plan
```

**Both must report no changes.** If either has pending drift, resolve it now. Migrating state while
a plan is dirty means you can't tell afterward whether a diff came from the migration or was already
there — and that ambiguity is exactly what you don't want when the answer determines whether you
trust the new backend.

Take a copy you can fall back to:

```bash
cd /home/nfang/workspace/personal-dashboard/infra
cp staging/terraform.tfstate /tmp/staging.tfstate.pre-migration
cp prod/terraform.tfstate    /tmp/prod.tfstate.pre-migration
```

## 2. Backend configuration files

You could hardcode the bucket in each root's `backend` block. Don't — use **partial configuration**
instead, where the backend block is empty and the values come from a file at init time:

```hcl
terraform {
  backend "gcs" {}
}
```

The reason is Phase 4. An ephemeral environment is, mechanically, the same code initialized against
a different state prefix:

```bash
terraform init -backend-config="prefix=dev/pr-123-forecast/collector-forecast"
```

That only works if the prefix isn't baked into the source. Learning the pattern here, on two
environments you can check by hand, is better than meeting it for the first time inside a script.

Create the two config files:

```hcl
# infra/envs/staging.gcs.tfbackend
bucket = "fang-dash-tfstate-staging"
prefix = "staging/monolith"
```

```hcl
# infra/envs/prod.gcs.tfbackend
bucket = "fang-dash-tfstate-prod"
prefix = "prod/monolith"
```

> `monolith` because that's what these roots currently are — one state holding the entire graph.
> Phase 3 splits them into `staging/foundation`, `staging/collector-weather`, and so on. Naming it
> honestly now makes the Phase 3 diff easier to read.

## 3. Add the backend block

Same file in both roots:

```hcl
# infra/staging/backend.tf   (and infra/prod/backend.tf)
terraform {
  backend "gcs" {}
}
```

While you're here, pin the Terraform version. Neither root has `required_version`, so today any CLI
version will happily open your state — including a newer one that upgrades the state format and
locks out everyone still on the old version.

```hcl
# add to the existing terraform { } block in infra/staging/main.tf and infra/prod/main.tf
terraform {
  required_version = "~> 1.13"   # bounded, matching the Phase 1 roots
  # ... existing required_providers ...
}
```

Both roots already commit `.terraform.lock.hcl`; keep that, and keep the exact CLI release in the
repo-root `.terraform-version` file from Phase 1 §2.6 so laptops and CI stay in lockstep.

## 4. Migrate staging

```bash
cd infra/staging
terraform init -migrate-state -backend-config=../envs/staging.gcs.tfbackend
```

Terraform detects the backend change and asks:

```
Do you want to copy existing state to the new backend?
  Pre-existing state was found while migrating the previous "local" backend to the
  newly configured "gcs" backend. ...
  Enter "yes" to copy and "no" to start with an empty state.
```

Answer **`yes`**. Answering `no` gives you an empty remote state, and the next apply will try to
create a second copy of everything you already have.

### The gate

```bash
terraform plan
```

**No changes. Nothing else is acceptable.**

If Terraform wants to *create* resources, the state didn't come across — you're looking at an empty
remote state while your real state sits in the local file. Recover:

```bash
rm -rf .terraform
cp /tmp/staging.tfstate.pre-migration terraform.tfstate
terraform init      # back to local, confirms you are whole
terraform plan      # should be clean again
```

Then retry, watching for the copy prompt.

If Terraform wants to *change* a handful of things, that's different — it means drift that existed
before the migration and got surfaced by the fresh read. Compare against the plan you ran in §1.

Confirm the object landed where you expect:

```bash
gcloud storage ls -r gs://fang-dash-tfstate-staging/staging/
# gs://fang-dash-tfstate-staging/staging/monolith.tfstate
```

## 5. Verify locking works

This is the thing local state never gave you, and it's worth seeing once rather than assuming.

```bash
cd infra/staging
terraform plan -lock-timeout=0 &
sleep 1
terraform plan -lock-timeout=0
wait
```

The second should fail with:

```
Error: Error acquiring the state lock
Lock Info:
  ID:        ...
  Operation: OperationTypePlan
```

The GCS backend implements this natively using object generation preconditions — no separate lock
table, nothing extra to provision. It's timing-dependent, so if both succeed just try again with a
larger config; the point is to know the mechanism exists.

> If you ever hit a stale lock after an interrupted run, `terraform force-unlock <LOCK_ID>` clears
> it. Read the lock info first and be certain nothing is actually running — force-unlocking a live
> apply is how state gets corrupted.

## 6. Migrate prod

Only once staging is fully verified.

```bash
cd infra/prod
terraform init -migrate-state -backend-config=../envs/prod.gcs.tfbackend
terraform plan     # must be: No changes
gcloud storage ls -r gs://fang-dash-tfstate-prod/prod/
```

## 7. Commit the tfvars

Your `infra/.gitignore` currently says:

```gitignore
# Local variable files (contains private Project IDs/secrets)
*.tfvars
*.tfvars.json
```

The comment is the reason to change it. `project_id`, `region`, `github_repository`, and
`api_domain` are identifiers, not secrets — they're already in `docs/DISASTER_RECOVERY.md` and in
your GitHub environment variables. Meanwhile the file being uncommitted has real costs: CI can't
plan without it, nobody can reproduce your plan locally, and a config change lands with no diff for
anyone to review.

Before changing anything, check that assumption against your actual files:

```bash
cat infra/staging/terraform.tfvars infra/prod/terraform.tfvars
```

If there's genuinely a credential in there, stop and move it to Secret Manager first. There
shouldn't be — the four declared variables are all identifiers.

Then:

```diff
 # Local state files
 *.tfstate
 *.tfstate.backup
 *.tfstate.*.backup
 
-# Local variable files (contains private Project IDs/secrets)
-*.tfvars
-*.tfvars.json
+# Environment inputs are committed — see docs/platform/README.md §4.1.
+# They hold identifiers, not credentials. Real secrets live in Secret Manager
+# and are added out-of-band; Terraform only ever creates empty containers.
+#
+# Still ignored: anything explicitly marked local-only.
+*.local.tfvars
```

Note `*.tfstate` stays ignored. State genuinely is sensitive — every value Terraform reads gets
written to it in plaintext — and it now lives in the bucket anyway.

Move the variable files somewhere the whole repo can see them:

```bash
cd /home/nfang/workspace/personal-dashboard/infra
mkdir -p envs
git mv staging/terraform.tfvars envs/staging.tfvars 2>/dev/null || mv staging/terraform.tfvars envs/staging.tfvars
git mv prod/terraform.tfvars    envs/prod.tfvars    2>/dev/null || mv prod/terraform.tfvars    envs/prod.tfvars
```

They're no longer auto-loaded from the root directory, so pass them explicitly:

```bash
cd infra/staging && terraform plan -var-file=../envs/staging.tfvars
cd infra/prod    && terraform plan -var-file=../envs/prod.tfvars
```

### Fix the gap in the staging example

`infra/staging/variables.tf` declares `api_domain` with no default, but
`infra/staging/terraform.tfvars.example` never lists it:

```hcl
project_id         = "your-staging-project-id-here"
region             = "us-central1"
github_repository  = "nickfang/personal-dashboard"
```

Anyone following `docs/DISASTER_RECOVERY.md` from that example gets an interactive prompt — or in
CI, where `-input=false` is set, a hard failure. Now that the real files are committed the examples
are largely redundant, so delete them:

```bash
git rm infra/staging/terraform.tfvars.example infra/prod/terraform.tfvars.example
```

## 8. Update the Makefile

The existing targets in the root `Makefile` don't pass a var file and no longer work:

```make
tf-plan-staging:  cd infra/staging && terraform plan
```

Replace the terraform section with something that takes the environment as a parameter:

```make
# --- Terraform ---------------------------------------------------------------
# Usage: make tf-plan ENV=staging

ENV ?= staging
TF_DIR := infra/$(ENV)
TF_BACKEND := ../envs/$(ENV).gcs.tfbackend
TF_VARS := ../envs/$(ENV).tfvars

.PHONY: tf-init tf-plan tf-apply

tf-init:
	cd $(TF_DIR) && terraform init -backend-config=$(TF_BACKEND)

tf-plan: tf-init
	cd $(TF_DIR) && terraform plan -var-file=$(TF_VARS)

tf-apply: tf-init
	cd $(TF_DIR) && terraform apply -var-file=$(TF_VARS)
```

`ENV=prod` now requires typing it, which is a small and useful piece of friction.

## 9. Retire the local state files

Only after **both** environments show a clean plan against the bucket:

```bash
cd /home/nfang/workspace/personal-dashboard/infra
mv staging/terraform.tfstate staging/terraform.tfstate.retired
mv prod/terraform.tfstate    prod/terraform.tfstate.retired
```

Leave them for a week or so. Once you've done a few applies against the remote backend without
surprises, delete them along with `/tmp/*.pre-migration`.

## 10. `docs/DISASTER_RECOVERY.md` is now wrong

It says:

> State is local (not remote). Each environment has its own `terraform.tfstate` in its directory.
> ... The "rebuild from scratch" DR principle means we don't need remote state.

That decision is reversed. Rewrite that section to cover:

- state lives in `gs://fang-dash-tfstate-<tier>/<env>/<stack>`, versioned
- recovering a corrupted state from a previous object generation:
  ```bash
  gcloud storage ls -a gs://fang-dash-tfstate-staging/staging/monolith.tfstate
  gcloud storage cp gs://fang-dash-tfstate-staging/staging/monolith.tfstate#<generation> \
                    gs://fang-dash-tfstate-staging/staging/monolith.tfstate
  ```
- what happens if the platform project is lost — the buckets have `prevent_destroy`, but a project
  deletion takes them with it, so this is the genuine worst case and deserves a written answer
- while you're in there: the old rebuild-from-scratch story was never as clean as it read, because
  `google_iam_workload_identity_pool` soft-deletes for 30 days and holds its ID. Rebuilding into the
  same project inside that window fails.

---

## Verification gate

```bash
# 1. Both roots plan clean against the remote backend
cd infra/staging && terraform plan -var-file=../envs/staging.tfvars
cd ../prod       && terraform plan -var-file=../envs/prod.tfvars
```
`No changes.` twice.

```bash
# 2. State objects are where they should be
gcloud storage ls -r gs://fang-dash-tfstate-staging/staging/
gcloud storage ls -r gs://fang-dash-tfstate-prod/prod/
```

```bash
# 3. Locking is live
cd infra/staging && terraform plan -lock-timeout=0 & sleep 1; terraform plan -lock-timeout=0; wait
```
Second invocation fails on the lock.

```bash
# 4. No state files are tracked by git
git -C /home/nfang/workspace/personal-dashboard ls-files | grep -c '\.tfstate'
```
Expect `0`.

```bash
# 5. The tfvars are tracked
git -C /home/nfang/workspace/personal-dashboard ls-files infra/envs/
```
Shows `staging.tfvars`, `prod.tfvars`, and the two `.gcs.tfbackend` files.

---

**You could stop here.** State is shared, versioned, and locked; CI is now *able* to run Terraform.
Phases 3–5 are what turn that into a platform.

**Next:** [03-stacks.md](03-stacks.md) — decompose the monolith so a single component can be
deployed on its own.

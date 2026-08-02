# Phase 1 — Bootstrap and the platform project

This is the first Terraform you write. It creates the platform project, the state buckets that
everything else depends on, the shared image registry, and one Terraform runner identity per tier.

Two roots, because they have very different lifecycles:

| Root | Contains | Changes | State lives |
|---|---|---|---|
| `infra/bootstrap/` | platform project, state buckets | almost never | in the bucket it creates |
| `infra/platform/` | org policies, registry, WIF, runner SAs | occasionally | in that bucket |

## 1. The chicken-and-egg, and how to settle it

Terraform stores state somewhere. You want that somewhere to be a GCS bucket. But the bucket has to
be created by something, and if that something is Terraform, where does *its* state go?

Three answers, and it's worth knowing why the tutorial picks the third:

1. **Create the bucket with `gcloud`, by hand.** Works, but the bucket's configuration — versioning,
   access prevention, lifecycle rules — then isn't code, and it's exactly the configuration you
   least want to get wrong.
2. **Keep bootstrap's state local forever.** Works until you lose the laptop, or a second person
   needs to change it.
3. **Self-hosting: apply with local state, then migrate that state into the bucket it just
   created.** One awkward moment, then it's normal Terraform forever.

Option 3 is standard practice. The awkward moment is about ten seconds long and you'll do it once.

## 2. `infra/bootstrap/`

### 2.1 `infra/bootstrap/main.tf`

```hcl
terraform {
  # Bounded to the 1.x series — an open ">= 1.13" accepts a future 2.0 silently.
  # The exact release lives in .terraform-version (§2.6) so laptops and CI agree.
  required_version = "~> 1.13"

  required_providers {
    google = {
      source  = "hashicorp/google"
      version = "~> 5.0"
    }
  }
}

provider "google" {
  region = var.region
}

# --- The platform project -----------------------------------------------------

resource "google_project" "platform" {
  name       = "platform"
  project_id = var.platform_project_id
  folder_id  = var.folder_platform

  billing_account = var.billing_account

  # Skip the legacy default VPC. Nothing here needs it, and it is easier to
  # never create it than to delete it later.
  auto_create_network = false

  lifecycle {
    prevent_destroy = true
  }
}

resource "google_project_service" "bootstrap" {
  for_each = toset([
    "cloudresourcemanager.googleapis.com",
    "serviceusage.googleapis.com",
    "storage.googleapis.com",
    "iam.googleapis.com",
    "orgpolicy.googleapis.com",
  ])

  project = google_project.platform.project_id
  service = each.value

  disable_on_destroy = false
}

# --- State buckets, one per tier ----------------------------------------------

resource "google_storage_bucket" "tfstate" {
  for_each = toset(["platform", "dev", "staging", "prod"])

  name    = "${var.state_bucket_prefix}-tfstate-${each.key}"
  project = google_project.platform.project_id

  # Multi-region. Bucket location is immutable — changing it later is a full
  # state migration — and at state-file sizes the durability upgrade over a
  # single region costs cents per month. Decide it once, here.
  location = var.state_bucket_location

  # Every object is owned by bucket-level IAM. No per-object ACLs, which means
  # one place to reason about who can read state.
  uniform_bucket_level_access = true
  public_access_prevention    = "enforced"

  # State files are the one thing you cannot rebuild from the repo.
  versioning {
    enabled = true
  }

  # Versioning protects against a bad apply. Soft delete protects against the
  # principal that holds objectAdmin on the bucket: a compromised runner can
  # delete state AND its noncurrent versions, and this window is what lets an
  # admin still recover them afterwards.
  soft_delete_policy {
    retention_duration_seconds = 7776000 # 90 days
  }

  # Keep enough history to recover from a bad apply, not enough to pay for it
  # forever. Under apply-on-merge CI (Phase 5), 20 versions is days of history
  # for an active repo — whichever rule fires first deletes, so size the count
  # rule for the cadence you'll actually have.
  lifecycle_rule {
    condition {
      num_newer_versions = 100
    }
    action {
      type = "Delete"
    }
  }

  lifecycle_rule {
    condition {
      days_since_noncurrent_time = 90
    }
    action {
      type = "Delete"
    }
  }

  lifecycle {
    prevent_destroy = true
  }

  depends_on = [google_project_service.bootstrap]
}

# --- Guards on the platform project itself -----------------------------------

# prevent_destroy stops Terraform; it does nothing against the console or
# gcloud. A lien blocks project deletion at the API, whoever asks.
resource "google_resource_manager_lien" "platform" {
  parent       = "projects/${google_project.platform.number}"
  restrictions = ["resourcemanager.projects.delete"]
  origin       = "tf-bootstrap"
  reason       = "Holds all Terraform state, the shared registry, and CI identities."
}

# GCS data-access logs are off by default, so nothing records who READ a state
# file — and state contains secrets and a full resource inventory. At state-file
# volumes this stays inside the logging free tier.
resource "google_project_iam_audit_config" "gcs_data_access" {
  project = google_project.platform.project_id
  service = "storage.googleapis.com"

  audit_log_config {
    log_type = "DATA_READ"
  }
  audit_log_config {
    log_type = "DATA_WRITE"
  }
}
```

Four things in there are doing real work:

- **`prevent_destroy` plus the lien.** They guard different doors. `prevent_destroy` is Terraform
  core — it makes `terraform destroy` fail rather than proceed, but a console click or a PR that
  deletes the block sails past it. The lien is enforced by GCP itself: project deletion fails at
  the API regardless of who asks or with what tool. Deleting a state bucket loses the record of
  every environment it tracks, and there is no undo; this project deserves both locks.
- **Versioning, soft delete, and lifecycle rules — three different protections.** Versioning saves
  you from a bad apply (restore the previous generation). Soft delete saves you from the deleting
  *principal* — a runner with `objectAdmin` can delete versions too, and soft delete is the window
  where an admin can still recover them. Lifecycle rules stop old versions accumulating forever.
- **One bucket per tier**, not one shared bucket with four prefixes. You could do prefix-scoped IAM
  conditions on a single bucket, and it would work. Separate buckets are easier to audit: "can the
  staging runner read prod state?" is answerable by looking at one bucket's IAM policy instead of
  by reasoning about a condition expression.
- **Data-access audit logs on GCS.** Without them, reading prod's state — secrets, full inventory —
  leaves no trace. With them, every read is attributable. For a team this is the difference between
  "someone looked at prod state" being answerable and being a shrug.

### 2.2 `infra/bootstrap/org_policy.tf`

Org policies live in *this* root, not `infra/platform`, and the reason is scoping: resources whose
parent is the organization force whoever applies the root to hold org-level roles. Bootstrap is
already the human-applied, almost-never-changes root — concentrating the org-scope resources here
means `infra/platform` needs only project- and folder-scope permissions, which is what makes giving
it a real CI runner possible later. Same lifecycle, same blast radius, same root.

```hcl
# infra/bootstrap/org_policy.tf

# Secure-by-default orgs (created on or after May 3, 2024) already have both of
# these policies — Google created them at org birth, and Phase 0 §5 briefly
# dropped and restored the domain restriction. Terraform can't create what
# exists, so declare the imports in config: on the first apply Terraform adopts
# the existing policies instead of failing with "already exists", and a second
# engineer cloning fresh never hits the trap.
import {
  to = google_org_policy_policy.disable_sa_keys
  id = "organizations/${var.org_id}/policies/iam.disableServiceAccountKeyCreation"
}

import {
  to = google_org_policy_policy.domain_restricted_sharing
  id = "organizations/${var.org_id}/policies/iam.allowedPolicyMemberDomains"
}

# You authenticate CI with Workload Identity Federation, so no workflow needs a
# service account key. Keeping key creation off org-wide means a leaked key
# cannot exist, rather than merely should not.
resource "google_org_policy_policy" "disable_sa_keys" {
  name   = "organizations/${var.org_id}/policies/iam.disableServiceAccountKeyCreation"
  parent = "organizations/${var.org_id}"

  spec {
    rules {
      enforce = "TRUE"
    }
  }
}

# Domain-restricted sharing: enforced at the org since creation. Your job here
# is exemptions, not adoption — it blocks new `allUsers` bindings, and
# infra/modules/cloud-run-aggregator/main.tf:55-60 binds roles/run.invoker to
# allUsers to make the dashboard API public. The existing binding survives
# (enforcement is not retroactive), but Terraform can't recreate it until the
# folder exemptions below exist.
resource "google_org_policy_policy" "domain_restricted_sharing" {
  name   = "organizations/${var.org_id}/policies/iam.allowedPolicyMemberDomains"
  parent = "organizations/${var.org_id}"

  spec {
    rules {
      values {
        allowed_values = ["is:${var.customer_id}"] # C0xxxxxxx from Phase 0
      }
    }
  }
}

resource "google_org_policy_policy" "allow_public_services" {
  for_each = toset([var.folder_staging, var.folder_prod])

  name   = "folders/${each.value}/policies/iam.allowedPolicyMemberDomains"
  parent = "folders/${each.value}"

  spec {
    rules {
      allow_all = "TRUE"
    }
  }
}
```

> **Check the first plan before applying.** If Google's original policy expresses the restriction
> differently than the HCL above (for example an organization principal set instead of
> `is:C0xxxxxxx`), align the HCL to what's actually there rather than letting Terraform rewrite the
> policy. Then, after applying, immediately `curl` your staging API and re-run `terraform plan` in
> `infra/staging`. If that plan wants to recreate `public_invoker`, the folder exemption isn't
> taking effect — roll back with `terraform destroy -target` on the exemption resources and
> revisit. Don't leave this half-applied over a weekend.

### 2.3 `infra/bootstrap/variables.tf`

```hcl
variable "platform_project_id" {
  description = "Project ID for the platform project (globally unique)"
  type        = string
}

variable "folder_platform" {
  description = "Numeric ID of the platform folder created in Phase 0"
  type        = string
}

variable "billing_account" {
  description = "Billing account ID to link, e.g. 01ABCD-234567-89EFGH"
  type        = string
}

variable "state_bucket_prefix" {
  description = "Prefix for state bucket names; must make them globally unique"
  type        = string
}

variable "state_bucket_location" {
  description = "Location for state buckets. Immutable after creation — multi-region by default"
  type        = string
  default     = "US"
}

variable "org_id" {
  description = "Numeric organization ID (org policies live in this root)"
  type        = string
}

variable "customer_id" {
  description = "Cloud Identity customer ID (C0xxxxxxx), for domain-restricted sharing"
  type        = string
}

variable "folder_staging" {
  description = "Numeric ID of the staging folder (domain-restriction exemption)"
  type        = string
}

variable "folder_prod" {
  description = "Numeric ID of the prod folder (domain-restriction exemption)"
  type        = string
}

variable "region" {
  description = "Default provider region"
  type        = string
  default     = "us-central1"
}
```

### 2.4 `infra/bootstrap/outputs.tf`

```hcl
output "platform_project_id" {
  description = "Project ID of the platform project"
  value       = google_project.platform.project_id
}

output "state_buckets" {
  description = "Map of tier to state bucket name; infra/platform consumes this via terraform_remote_state"
  value       = { for k, b in google_storage_bucket.tfstate : k => b.name }
}
```

### 2.5 `infra/bootstrap/terraform.tfvars`

Commit this. See §4 of the [README](README.md) for why these aren't secrets.

```hcl
platform_project_id = "fang-dash-platform"
folder_platform     = "111111111111"
folder_staging      = "333333333333"
folder_prod         = "444444444444"
org_id              = "123456789012"
customer_id         = "C01abcdef"
billing_account     = "01ABCD-234567-89EFGH"
state_bucket_prefix = "fang-dash"
region              = "us-central1"
```

> Bucket names are globally unique across all of GCS. If `terraform apply` fails with
> `409 Conflict`, someone else has your prefix — change `state_bucket_prefix` and retry.

### 2.6 Apply it

First, pin the toolchain so every laptop and CI runner resolves the same versions:

```bash
cd infra/bootstrap
terraform version   # note the exact release, e.g. 1.13.5
echo "1.13.5" > ../../.terraform-version   # committed; CI reads the same file
terraform init

# The lock file records provider checksums. Add the Linux platform so the same
# lock file verifies on CI runners, then commit it alongside the config.
terraform providers lock -platform=darwin_arm64 -platform=linux_amd64
```

Without the lock file, two machines resolving `~> 5.0` on different days get different provider
minors and start producing different plans for identical config — the classic "works on my laptop"
of infrastructure. Commit `.terraform.lock.hcl` for every root, always.

Then:

```bash
gcloud auth application-default login   # as nick@yourdomain.com
terraform plan    # check the org-policy imports adopt cleanly (§2.2)
terraform apply
```

Confirm the buckets came out right:

```bash
gcloud storage buckets describe gs://fang-dash-tfstate-prod \
  --format="yaml(name, versioning, iamConfiguration)"
```

You want `versioning.enabled: true`, `uniformBucketLevelAccess.enabled: true`, and
`publicAccessPrevention: enforced`.

### 2.7 Self-host bootstrap's state

Now the awkward ten seconds. Add a backend block:

```hcl
# infra/bootstrap/backend.tf
terraform {
  backend "gcs" {
    bucket = "fang-dash-tfstate-platform"
    prefix = "bootstrap"
  }
}
```

> This one is hardcoded rather than passed as partial config. Bootstrap only ever has one instance,
> so there's nothing to parameterize — and a hardcoded bucket is one less thing to get wrong when
> you come back to this file in a year.

```bash
terraform init -migrate-state
```

Terraform notices the backend changed and offers to copy your local state up. Answer `yes`. Then
the check that matters:

```bash
terraform plan
```

**It must report no changes.** If it wants to create the project and buckets again, the state didn't
migrate — you're looking at an empty remote state. Stop and fix it before continuing; the local
`terraform.tfstate` is still on disk and still correct.

Once the plan is clean:

```bash
gcloud storage ls gs://fang-dash-tfstate-platform/bootstrap/
# gs://fang-dash-tfstate-platform/bootstrap/default.tfstate

mv terraform.tfstate terraform.tfstate.local-backup
```

Keep the backup until Phase 2 is done, then delete it.

## 3. `infra/platform/`

The platform project exists and has buckets. Now fill it: org policy, the shared registry, the WIF
pool, and the runner identities.

### 3.1 `infra/platform/main.tf`

```hcl
terraform {
  required_version = "~> 1.13"

  backend "gcs" {
    bucket = "fang-dash-tfstate-platform"
    prefix = "platform"
  }

  required_providers {
    google = {
      source  = "hashicorp/google"
      version = "~> 5.0"
    }
  }
}

provider "google" {
  project = var.platform_project_id
  region  = var.region
}

resource "google_project_service" "platform" {
  for_each = toset([
    "artifactregistry.googleapis.com",
    "iamcredentials.googleapis.com",
    "sts.googleapis.com",
    "cloudbilling.googleapis.com",
    "billingbudgets.googleapis.com",
  ])

  service            = each.value
  disable_on_destroy = false
}
```

### 3.2 Org policy — lives in bootstrap, deliberately

There is deliberately no `org_policy.tf` in this root — the org policies moved to
`infra/bootstrap` (§2.2). The reason is worth internalizing, because it's a general rule for
structuring roots: **a root's required permissions are set by its most privileged resource.** One
org-scope resource in `infra/platform` and whoever applies it needs org-level roles forever, which
forecloses ever giving this root a sanely scoped CI runner. With org policies in bootstrap, this
root touches only projects and folders — which is what makes §3.6's plan CI (and one day a real
apply runner) possible without handing CI the keys to the organization.

### 3.3 Shared image registry

```hcl
# infra/platform/registry.tf

# One registry for every environment. An ephemeral env then references an image
# tag that already exists instead of building its own, which is the difference
# between a 30-second spin-up and a 5-minute one.
resource "google_artifact_registry_repository" "shared" {
  location      = var.region
  repository_id = "personal-dashboard"
  description   = "Shared container images for all environments"
  format        = "DOCKER"

  # A shared registry with mutable tags quietly bypasses every tier boundary
  # this document builds: all three deployers hold writer here, so a compromised
  # dev workflow could re-push the tag prod runs. Immutable tags close that —
  # a tag, once pushed, can never point at different bytes. The deploy workflows
  # already push :${git sha}, which is unique per commit, so nothing breaks.
  docker_config {
    immutable_tags = true
  }

  depends_on = [google_project_service.platform]
}

output "registry_url" {
  description = "Registry URL that environment stacks pull images from"
  value       = "${var.region}-docker.pkg.dev/${var.platform_project_id}/${google_artifact_registry_repository.shared.repository_id}"
}
```

This replaces the per-project registry in `infra/modules/foundation/main.tf`. Phase 3 rewires the
environments to pull from here and grants their runtime service accounts cross-project read access.

### 3.4 Workload Identity Federation, scoped per tier

This is the part that makes environment isolation real, so it's worth reading slowly.

Your current setup in `infra/modules/github-oidc/main.tf` conditions on the repository alone:

```hcl
attribute_condition = "assertion.repository == \"${var.github_repository}\""
```

Any workflow, on any branch, in any pull request, can assume that identity. With one repo driving
three tiers, that isn't a boundary at all — it's an open door with a sign on it.

The fix is to map GitHub's `environment` claim and bind each runner to it:

```hcl
# infra/platform/wif.tf

resource "google_iam_workload_identity_pool" "github" {
  workload_identity_pool_id = "github"
  display_name              = "GitHub Actions"

  depends_on = [google_project_service.platform]
}

resource "google_iam_workload_identity_pool_provider" "github" {
  workload_identity_pool_id          = google_iam_workload_identity_pool.github.workload_identity_pool_id
  workload_identity_pool_provider_id = "github"
  display_name                       = "GitHub Actions OIDC"

  attribute_mapping = {
    "google.subject"        = "assertion.sub"
    "attribute.repository"  = "assertion.repository"
    "attribute.environment" = "assertion.environment"
    "attribute.ref"         = "assertion.ref"
  }

  # First gate: the token must come from your repo at all.
  attribute_condition = "assertion.repository == \"${var.github_repository}\""

  oidc {
    issuer_uri = "https://token.actions.githubusercontent.com"
  }
}

resource "google_service_account" "tf_runner" {
  for_each = toset(["dev", "staging", "prod"])

  account_id   = "tf-runner-${each.key}"
  display_name = "Terraform runner (${each.key})"
}

# Second gate, and the important one: only a job that declares
# `environment: prod` gets a token carrying that claim, and only that token can
# impersonate tf-runner-prod.
resource "google_service_account_iam_member" "tf_runner_wif" {
  for_each = toset(["dev", "staging", "prod"])

  service_account_id = google_service_account.tf_runner[each.key].name
  role               = "roles/iam.workloadIdentityUser"
  member             = "principalSet://iam.googleapis.com/${google_iam_workload_identity_pool.github.name}/attribute.environment/${each.key}"
}
```

Why this matters: GitHub only puts the `environment` claim in the OIDC token when the job declares
`environment:`, and environment protection rules — required reviewers, branch restrictions — run
*before* the job starts, which means before a token is ever issued. A job with no `environment:`
gets a token with no such claim, matches no principalSet, and can impersonate nothing.

But be precise about what enforces what, because the chain is longer than it looks. GCP verifies
exactly one thing: the token carries the claim string. Whether *obtaining* that token required
approval depends entirely on GitHub environment configuration — which lives outside version
control, in the repo settings UI, editable by any repo admin. The real trust chain is:

> IAM binding → claim string → GitHub environment protection rules → **the set of repo admins**

Every link matters. If the `prod` environment exists with no protection rules — the default when a
workflow first references one — then any workflow on any branch that declares `environment: prod`
mints a prod token, and the IAM binding will honor it. So:

**Create the environments now, with their rules, before anything binds to the claims.** In the repo:
**Settings → Environments**, create `dev`, `staging`, `prod`, and `platform-plan` (§3.6). On `prod`:
add yourself as a required reviewer and restrict deployment branches to `main`. On `staging`:
restrict branches to `main`. The branch restriction is what backs the claim with "this ran from
reviewed history" — it's the free, GitHub-layer equivalent of pinning the principalSet to a ref.

Two standing rules to write into your contributing docs while you're here: repo admin membership
*is* prod access — treat granting it accordingly — and `pull_request_target` is banned in this
repo, because it runs workflow code from a fork's PR with access to secrets and the repo's OIDC
identity.

#### The deploy identity has to move here too

The runners above are *Terraform* identities. There is a second, entirely separate identity in play:
the one your existing deploy workflows use to push images and update Cloud Run.

Today it lives in `infra/modules/github-oidc`, instantiated from `infra/staging/main.tf:223` and
`infra/prod/main.tf:211`. It creates the pool `github-actions-pool` and the service account
`github-actions-sa`, holding `artifactregistry.writer`, `run.developer`, and `act_as` on all six
runtime service accounts. `.github/workflows/_deploy-service.yml` authenticates as it via
`vars.WIF_PROVIDER` and `vars.GCP_SA_EMAIL`.

**Phase 3 deletes that module along with the old roots.** If you get there without a replacement,
every deploy workflow breaks — and because `google_iam_workload_identity_pool` soft-deletes for 30
days while holding its ID, you cannot simply recreate `github-actions-pool` to recover. Build the
replacement now, while the old one is still running.

```hcl
# infra/platform/deploy_identity.tf

# One deploy identity per tier, same environment-scoped pattern as the runners.
resource "google_service_account" "deployer" {
  for_each = toset(["dev", "staging", "prod"])

  account_id   = "deployer-${each.key}"
  display_name = "Image deploy identity (${each.key})"
}

resource "google_service_account_iam_member" "deployer_wif" {
  for_each = toset(["dev", "staging", "prod"])

  service_account_id = google_service_account.deployer[each.key].name
  role               = "roles/iam.workloadIdentityUser"
  member             = "principalSet://iam.googleapis.com/${google_iam_workload_identity_pool.github.name}/attribute.environment/${each.key}"
}

# Push images to the shared registry.
resource "google_artifact_registry_repository_iam_member" "deployer_push" {
  for_each = toset(["dev", "staging", "prod"])

  location   = google_artifact_registry_repository.shared.location
  repository = google_artifact_registry_repository.shared.name
  role       = "roles/artifactregistry.writer"
  member     = "serviceAccount:${google_service_account.deployer[each.key].email}"
}

# Update Cloud Run services and jobs in its own tier, and nowhere else.
resource "google_project_iam_member" "deployer_staging" {
  project = var.staging_project_id
  role    = "roles/run.developer"
  member  = "serviceAccount:${google_service_account.deployer["staging"].email}"
}

resource "google_project_iam_member" "deployer_prod" {
  project = var.prod_project_id
  role    = "roles/run.developer"
  member  = "serviceAccount:${google_service_account.deployer["prod"].email}"
}

resource "google_folder_iam_member" "deployer_dev" {
  folder = "folders/${var.folder_dev}"
  role   = "roles/run.developer"
  member = "serviceAccount:${google_service_account.deployer["dev"].email}"
}
```

The one thing that can't live here is `act_as`. Deploying a revision means setting the runtime
service account on it, which requires `roles/iam.serviceAccountUser` on that SA — and those SAs are
created per environment by the component stacks in Phase 3. So the binding belongs there, next to
the SA it refers to. Phase 3 adds it; this is a forward reference, noted so it isn't a surprise.

> **Repoint the GitHub variables before Phase 3, not during it.** Once these exist, update
> `WIF_PROVIDER` and `GCP_SA_EMAIL` in each GitHub environment to the new provider and the matching
> `deployer-<tier>` address, then run one deploy workflow per tier to confirm it still works. Only
> after a green deploy should you go near `git rm -r infra/staging`.
>
> **The prod workflows will fail until you fix a name mismatch.** The six
> `.github/workflows/deploy-*-prod.yml` files declare `environment: production`, but the
> principalSet above binds `attribute.environment/prod`. The token says `production`, the binding
> expects `prod`, and STS returns `unauthenticated` — at the worst possible moment to debug claim
> mapping. Rename the workflows to `environment: prod` (matching the tier name every folder, SA,
> and bucket already uses) and recreate the protection rules on the `prod` environment before the
> cutover.

### 3.5 Runner permissions — scope honestly, then constrain

First, the honest version of what's being built here, because the obvious framing — "enumerate
roles instead of granting owner" — overstates what enumeration buys you. Any role containing
`setIamPolicy` at a scope is **owner-equivalent at that scope**: one API call grants anything to
anyone. A Terraform runner that manages IAM bindings needs `resourcemanager.projectIamAdmin`,
which contains exactly that call. So the security boundary in this design is not the role list —
it's that each runner is scoped to *one tier's project or folder* and can reach nothing else. The
role list documents intent and trims incidental permissions; the tier scoping is the wall.

That said, GCP has a mechanism that makes the role list real: an IAM Condition on
`modifiedGrantsByRole`, which caps *which roles* an IAM-admin grant can hand out. With it, the
runner can bind the runtime roles the stacks actually use and nothing else — the difference
between "IAM admin" and "can wire up its own services."

```hcl
# infra/platform/runner_iam.tf

# Read bootstrap's outputs rather than reconstructing names by string
# convention — one source of truth for what the state buckets are called.
data "terraform_remote_state" "bootstrap" {
  backend = "gcs"

  config = {
    bucket = "fang-dash-tfstate-platform"
    prefix = "bootstrap"
  }
}

locals {
  # Everything the stacks in Phase 3 actually create. If a later apply fails
  # with a permission error, add the role here rather than reaching for owner.
  # Deliberately absent: resourcemanager.projectIamAdmin — it gets its own
  # conditioned binding below.
  runner_project_roles = [
    "roles/run.admin",
    "roles/cloudscheduler.admin",
    "roles/datastore.owner",
    "roles/secretmanager.admin",
    "roles/iam.serviceAccountAdmin",
    "roles/serviceusage.serviceUsageAdmin",
  ]

  # The only roles a runner may grant or revoke: the runtime roles the stacks
  # bind to service accounts. Extend this list when a stack needs a new one —
  # the apply error will name the missing role.
  runner_grantable_roles = [
    "roles/run.invoker",
    "roles/secretmanager.secretAccessor",
    "roles/iam.serviceAccountUser",
    "roles/datastore.user",
  ]
}

# --- staging and prod: scoped to their one project ---------------------------

resource "google_project_iam_member" "runner_staging" {
  for_each = toset(local.runner_project_roles)

  project = var.staging_project_id
  role    = each.value
  member  = "serviceAccount:${google_service_account.tf_runner["staging"].email}"
}

resource "google_project_iam_member" "runner_prod" {
  for_each = toset(local.runner_project_roles)

  project = var.prod_project_id
  role    = each.value
  member  = "serviceAccount:${google_service_account.tf_runner["prod"].email}"
}

# projectIamAdmin, conditioned: the runner manages IAM bindings, but only for
# the runtime roles in the list. Without the condition this role is
# owner-equivalent (setIamPolicy grants anything to anyone in one call).
resource "google_project_iam_member" "runner_iam_admin" {
  for_each = {
    staging = var.staging_project_id
    prod    = var.prod_project_id
  }

  project = each.value
  role    = "roles/resourcemanager.projectIamAdmin"
  member  = "serviceAccount:${google_service_account.tf_runner[each.key].email}"

  condition {
    title       = "grant-runtime-roles-only"
    description = "Runner may only grant/revoke the runtime roles the stacks bind"
    expression  = "api.getAttribute('iam.googleapis.com/modifiedGrantsByRole', []).hasOnly(${jsonencode(local.runner_grantable_roles)})"
  }
}

# --- dev: creates and destroys whole projects, but only inside folders/dev ---

resource "google_folder_iam_member" "runner_dev" {
  for_each = toset(concat(local.runner_project_roles, [
    "roles/resourcemanager.projectCreator",
    "roles/resourcemanager.projectDeleter",
  ]))

  folder = "folders/${var.folder_dev}"
  role   = each.value
  member = "serviceAccount:${google_service_account.tf_runner["dev"].email}"
}

# Same conditioned IAM-admin grant for dev, at folder scope: new projects need
# their runtime bindings wired up, and nothing more.
resource "google_folder_iam_member" "runner_dev_iam_admin" {
  folder = "folders/${var.folder_dev}"
  role   = "roles/resourcemanager.projectIamAdmin"
  member = "serviceAccount:${google_service_account.tf_runner["dev"].email}"

  condition {
    title       = "grant-runtime-roles-only"
    description = "Runner may only grant/revoke the runtime roles the stacks bind"
    expression  = "api.getAttribute('iam.googleapis.com/modifiedGrantsByRole', []).hasOnly(${jsonencode(local.runner_grantable_roles)})"
  }
}

# Creating a project means linking billing to it.
resource "google_billing_account_iam_member" "runner_dev_billing" {
  billing_account_id = var.billing_account
  role               = "roles/billing.user"
  member             = "serviceAccount:${google_service_account.tf_runner["dev"].email}"
}

# --- state access: each runner reaches exactly one bucket --------------------

# objectAdmin, not objectViewer, because the GCS backend's locking works by
# creating and deleting a lock object — writing state requires admin. That is
# also why soft delete on the buckets matters (§2.1): admin includes deleting
# versions, and soft delete is the recovery window behind that.
resource "google_storage_bucket_iam_member" "runner_state" {
  for_each = toset(["dev", "staging", "prod"])

  bucket = data.terraform_remote_state.bootstrap.outputs.state_buckets[each.key]
  role   = "roles/storage.objectAdmin"
  member = "serviceAccount:${google_service_account.tf_runner[each.key].email}"
}

# --- everyone can pull shared images -----------------------------------------

resource "google_artifact_registry_repository_iam_member" "runner_pull" {
  for_each = toset(["dev", "staging", "prod"])

  location   = google_artifact_registry_repository.shared.location
  repository = google_artifact_registry_repository.shared.name
  role       = "roles/artifactregistry.reader"
  member     = "serviceAccount:${google_service_account.tf_runner[each.key].email}"
}
```

Note the shape of the dev grant: **folder-level**, so the dev runner can create and destroy projects
freely — but only inside `folders/dev`. It has no path to staging or prod. That single distinction
is most of what makes a self-service environment factory safe to hand to other people.

The `runner_state` block is the other half. The staging runner holds `objectAdmin` on
`fang-dash-tfstate-staging` and nothing on the prod bucket, so it cannot read prod's state — which
would otherwise hand it a full inventory of prod's resources and any value Terraform has read.

### 3.6 Plan CI for the roots that have no runner

Stop and notice an asymmetry this document has been quietly building: Phase 5 gives `dev`,
`staging`, and `prod` full plan-on-PR / apply-on-merge CI — but `infra/bootstrap` and
`infra/platform`, the roots that control org policy, every IAM grant, and all four state buckets,
would be applied from a laptop forever, with no reviewed plan and no drift detection. That's the
pattern inverted: the most privileged layer with the least scrutiny.

The full fix would be an apply runner for these roots too. The honest, cheap fix — and a
recognized pattern even in Google's own `terraform-example-foundation`, whose bootstrap stage is
human-run — is: **applies stay manual, everything else becomes automatic.** Say the quiet part in
prose: whoever can apply `infra/bootstrap` is org-admin-equivalent; on this project that's you,
via ADC. The mitigation is that no change reaches `main` without a machine-posted plan on the PR,
and a weekly scheduled plan catches drift between applies.

The read-only identity (this is the last new service account, promise):

```hcl
# infra/platform/planner.tf

resource "google_service_account" "tf_planner_platform" {
  account_id   = "tf-planner-platform"
  display_name = "Terraform planner for bootstrap + platform roots (read-only)"
}

resource "google_service_account_iam_member" "tf_planner_platform_wif" {
  service_account_id = google_service_account.tf_planner_platform.name
  role               = "roles/iam.workloadIdentityUser"
  member             = "principalSet://iam.googleapis.com/${google_iam_workload_identity_pool.github.name}/attribute.environment/platform-plan"
}

# Viewer covers resources; securityReviewer covers reading IAM policies
# (getIamPolicy is not part of Viewer). Read-only by construction.
locals {
  planner_read_projects = [var.platform_project_id, var.staging_project_id, var.prod_project_id]
  planner_read_roles    = ["roles/viewer", "roles/iam.securityReviewer"]
}

resource "google_project_iam_member" "planner_read" {
  for_each = {
    for pair in setproduct(local.planner_read_projects, local.planner_read_roles) :
    "${pair[0]}-${pair[1]}" => pair
  }

  project = each.value[0]
  role    = each.value[1]
  member  = "serviceAccount:${google_service_account.tf_planner_platform.email}"
}

# Planning the bootstrap root also means reading org policies, folders, and
# org-level IAM. These two org-scope READ grants are the only org-touching
# resources left in this root; they change approximately never.
resource "google_organization_iam_member" "planner_org_read" {
  for_each = toset([
    "roles/orgpolicy.policyViewer",
    "roles/browser",
    "roles/iam.securityReviewer",
  ])

  org_id = var.org_id
  role   = each.value
  member = "serviceAccount:${google_service_account.tf_planner_platform.email}"
}

# Read state, don't write it. Plans run with -lock=false for the same reason.
resource "google_storage_bucket_iam_member" "planner_state_read" {
  bucket = data.terraform_remote_state.bootstrap.outputs.state_buckets["platform"]
  role   = "roles/storage.objectViewer"
  member = "serviceAccount:${google_service_account.tf_planner_platform.email}"
}
```

And the workflow — this one you can add as soon as this phase is applied, without waiting for
Phase 5:

```yaml
# .github/workflows/terraform-plan-platform.yml
name: Terraform Plan (privileged roots)

on:
  pull_request:
    paths: ["infra/bootstrap/**", "infra/platform/**"]
  schedule:
    - cron: "0 12 * * 1" # weekly drift check

permissions:
  contents: read
  id-token: write
  issues: write

jobs:
  plan:
    runs-on: ubuntu-latest
    environment: platform-plan
    strategy:
      matrix:
        root: [bootstrap, platform]
    steps:
      - uses: actions/checkout@v4
      - uses: hashicorp/setup-terraform@v3
        with:
          terraform_version: "1.13.5" # keep in lockstep with .terraform-version
      - uses: google-github-actions/auth@v2
        with:
          workload_identity_provider: ${{ vars.WIF_PROVIDER }}
          service_account: ${{ vars.TF_PLANNER_PLATFORM }}
      - run: terraform -chdir=infra/${{ matrix.root }} init -lockfile=readonly
      - id: plan
        run: terraform -chdir=infra/${{ matrix.root }} plan -detailed-exitcode -lock=false -no-color
        continue-on-error: true
      # Exit code 2 means "plan has changes". On a PR that's the point; on the
      # weekly run it's drift — someone changed reality without changing code.
      - if: github.event_name == 'schedule' && steps.plan.outputs.exitcode == '2'
        run: >
          gh issue create
          --title "Terraform drift in infra/${{ matrix.root }}"
          --body "The weekly drift plan found pending changes. Run \`terraform plan\` locally to see them."
        env:
          GH_TOKEN: ${{ github.token }}
```

The `platform-plan` GitHub environment (created in §3.4) needs no protection rules — planning is
read-only, and gating it would only stop PRs from getting plans. On a public repo the Actions
minutes are free; the whole arrangement costs nothing.

### 3.7 Budget guard on the dev folder

Phase 4's factory will create billed projects from PR-driven workflows — the least-protected
trigger in the system, by design. Budgets are free, and the alert is the alarm that fires before
quota does. Create it now, while the folder is empty, not after the first surprise.

```hcl
# infra/platform/budgets.tf

resource "google_billing_budget" "dev_folder" {
  billing_account = var.billing_account
  display_name    = "dev folder — ephemeral environments"

  budget_filter {
    resource_ancestors = ["folders/${var.folder_dev}"]
  }

  amount {
    specified_amount {
      currency_code = "USD"
      units         = "25" # pick a number that would make you investigate
    }
  }

  threshold_rules {
    threshold_percent = 0.5
  }
  threshold_rules {
    threshold_percent = 1.0
  }
}
```

Alerts go to billing account admins by email by default — that's you. Wire a Pub/Sub notification
later if you want automation to react.

### 3.8 `infra/platform/variables.tf`

Note what's *not* here anymore: `customer_id`, `folder_staging`, `folder_prod` — they moved to
bootstrap with the org policies. `org_id` remains only for the planner's org-scope *read* grants
(§3.6).

```hcl
variable "org_id" {
  description = "Numeric organization ID (planner read grants only)"
  type        = string
}

variable "platform_project_id" {
  description = "Project ID of the platform project"
  type        = string
}

variable "staging_project_id" {
  description = "Project ID of the staging environment"
  type        = string
}

variable "prod_project_id" {
  description = "Project ID of the production environment"
  type        = string
}

variable "folder_dev" {
  description = "Numeric ID of the dev folder"
  type        = string
}

variable "billing_account" {
  description = "Billing account ID"
  type        = string
}

variable "github_repository" {
  description = "GitHub repository allowed to federate, as owner/repo"
  type        = string
  default     = "nickfang/personal-dashboard"
}

variable "region" {
  description = "Default region"
  type        = string
  default     = "us-central1"
}
```

### 3.9 `infra/platform/terraform.tfvars`

```hcl
org_id              = "123456789012"
platform_project_id = "fang-dash-platform"
staging_project_id  = "fang-gcp-staging"
prod_project_id     = "fang-gcp"
folder_dev          = "222222222222"
billing_account     = "01ABCD-234567-89EFGH"
github_repository   = "nickfang/personal-dashboard"
region              = "us-central1"
```

### 3.10 `infra/platform/outputs.tf`

Phase 5 needs these for the GitHub workflow configuration.

```hcl
output "wif_provider" {
  description = "Full WIF provider resource name for google-github-actions/auth"
  value       = google_iam_workload_identity_pool_provider.github.name
}

output "runner_service_accounts" {
  description = "Map of tier to Terraform runner service account email"
  value       = { for k, sa in google_service_account.tf_runner : k => sa.email }
}

output "deployer_service_accounts" {
  description = "Map of tier to image-deploy service account email; goes into GCP_SA_EMAIL"
  value       = { for k, sa in google_service_account.deployer : k => sa.email }
}

output "registry_url" {
  description = "Shared Artifact Registry URL"
  value       = "${var.region}-docker.pkg.dev/${var.platform_project_id}/${google_artifact_registry_repository.shared.repository_id}"
}

output "tf_planner_platform" {
  description = "Planner SA email; goes into the TF_PLANNER_PLATFORM repo variable for §3.6's workflow"
  value       = google_service_account.tf_planner_platform.email
}
```

### 3.11 Apply

```bash
cd infra/platform
terraform init
terraform providers lock -platform=darwin_arm64 -platform=linux_amd64
terraform plan     # read it properly — this grants IAM across every tier
terraform apply
```

Commit `.terraform.lock.hcl` with the config, same as bootstrap's (§2.6).

Then record the outputs; Phase 3 and Phase 5 both consume them.

```bash
terraform output -json > /tmp/platform-outputs.json
terraform output wif_provider
terraform output runner_service_accounts
```

## 4. What you have now

- A platform project holding four state buckets — versioned, soft-deleting, multi-region, with
  data-access logging — and a lien plus `prevent_destroy` guarding the project itself
- Bootstrap's own state living in one of those buckets
- Org policies (key creation off, domain-restricted sharing with folder exemptions) adopted into
  Terraform via import blocks, managed from the bootstrap root
- A shared image registry with immutable tags — no tier can rewrite what another tier deploys
- Three Terraform runner identities, each reaching exactly one tier, each impersonable only by a
  GitHub job that declares the matching `environment:`, with IAM-admin rights conditioned to the
  runtime roles the stacks actually bind
- Three deploy identities on the same pattern, replacing the per-project `github-actions-sa`
- A read-only planner for the two privileged roots, plan-on-PR and weekly drift detection for
  them, and GitHub environments created *with* their protection rules
- A budget alarm on the dev folder, armed before the project factory exists

Nothing in `infra/staging` or `infra/prod` has changed yet. Those still run on local state — that's
Phase 2. The old `module.github_oidc` is also still in place and still working; it gets removed in
Phase 3, once the GitHub variables point at the identities you just created.

---

## Verification gate

```bash
# 1. Four buckets, correctly configured
for t in platform dev staging prod; do
  gcloud storage buckets describe "gs://fang-dash-tfstate-$t" \
    --format="value(name, versioning.enabled, iamConfiguration.publicAccessPrevention)"
done
```
Four rows, each `True` and `enforced`.

```bash
# 2. Bootstrap is self-hosted and stable
cd infra/bootstrap && terraform plan
```
`No changes.`

```bash
# 3. Platform is stable
cd infra/platform && terraform plan
```
`No changes.`

```bash
# 4. The isolation actually holds — this is the one worth running
gcloud storage buckets get-iam-policy gs://fang-dash-tfstate-prod \
  --format=json | grep -c "tf-runner-staging"
```
Expect `0`. The staging runner must not appear anywhere in the prod bucket's policy. If it does,
your `runner_state` `for_each` is wired wrong and the boundary you just built isn't there.

```bash
# 5. Keys are off
gcloud org-policies describe iam.disableServiceAccountKeyCreation \
  --organization="$ORG_ID"
```
Shows `enforce: true`.

```bash
# 6. Tags really are immutable — the same tag pushed twice must fail
gcloud artifacts repositories describe personal-dashboard \
  --project=fang-dash-platform --location=us-central1 \
  --format="value(dockerConfig)"
```
Shows `IMMUTABLE_TAGS`.

```bash
# 7. The lien holds — this deletion attempt must be REFUSED
gcloud projects delete fang-dash-platform
```
Fails with a lien error. (If it instead asks you to confirm, answer no and go fix the lien.)

---

**Next:** [02-state-migration.md](02-state-migration.md) — move staging and prod off your laptop.

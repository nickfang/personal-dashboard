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
  required_version = ">= 1.13"

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
  ])

  project = google_project.platform.project_id
  service = each.value

  disable_on_destroy = false
}

# --- State buckets, one per tier ----------------------------------------------

resource "google_storage_bucket" "tfstate" {
  for_each = toset(["platform", "dev", "staging", "prod"])

  name     = "${var.state_bucket_prefix}-tfstate-${each.key}"
  project  = google_project.platform.project_id
  location = var.region

  # Every object is owned by bucket-level IAM. No per-object ACLs, which means
  # one place to reason about who can read state.
  uniform_bucket_level_access = true
  public_access_prevention    = "enforced"

  # State files are the one thing you cannot rebuild from the repo.
  versioning {
    enabled = true
  }

  # Keep enough history to recover from a bad apply, not enough to pay for it
  # forever.
  lifecycle_rule {
    condition {
      num_newer_versions = 20
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
```

Three things in there are doing real work:

- **`prevent_destroy`** on both the project and the buckets. This is Terraform core, not a provider
  feature — it makes `terraform destroy` fail rather than proceed. Deleting a state bucket loses the
  record of every environment it tracks, and there is no undo. Blocking that at the config level is
  worth the mild annoyance when you genuinely want to tear something down.
- **Versioning plus lifecycle rules.** Versioning is what saves you when an apply corrupts state —
  you restore the previous generation of the object. The lifecycle rules stop old versions
  accumulating forever.
- **One bucket per tier**, not one shared bucket with four prefixes. You could do prefix-scoped IAM
  conditions on a single bucket, and it would work. Separate buckets are easier to audit: "can the
  staging runner read prod state?" is answerable by looking at one bucket's IAM policy instead of
  by reasoning about a condition expression.

### 2.2 `infra/bootstrap/variables.tf`

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

variable "region" {
  description = "Region for buckets and default provider region"
  type        = string
  default     = "us-central1"
}
```

### 2.3 `infra/bootstrap/outputs.tf`

```hcl
output "platform_project_id" {
  description = "Project ID of the platform project"
  value       = google_project.platform.project_id
}

output "state_buckets" {
  description = "Map of tier to state bucket name"
  value       = { for k, b in google_storage_bucket.tfstate : k => b.name }
}
```

### 2.4 `infra/bootstrap/terraform.tfvars`

Commit this. See §4 of the [README](README.md) for why these aren't secrets.

```hcl
platform_project_id = "fang-dash-platform"
folder_platform     = "111111111111"
billing_account     = "01ABCD-234567-89EFGH"
state_bucket_prefix = "fang-dash"
region              = "us-central1"
```

> Bucket names are globally unique across all of GCS. If `terraform apply` fails with
> `409 Conflict`, someone else has your prefix — change `state_bucket_prefix` and retry.

### 2.5 Apply it

```bash
cd infra/bootstrap
gcloud auth application-default login   # as nick@yourdomain.com
terraform init
terraform apply
```

Confirm the buckets came out right:

```bash
gcloud storage buckets describe gs://fang-dash-tfstate-prod \
  --format="yaml(name, versioning, iamConfiguration)"
```

You want `versioning.enabled: true`, `uniformBucketLevelAccess.enabled: true`, and
`publicAccessPrevention: enforced`.

### 2.6 Self-host bootstrap's state

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
  required_version = ">= 1.13"

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
    "orgpolicy.googleapis.com",
    "cloudbilling.googleapis.com",
  ])

  service            = each.value
  disable_on_destroy = false
}
```

### 3.2 Org policy

```hcl
# infra/platform/org_policy.tf

# You authenticate CI with Workload Identity Federation, so no workflow needs a
# service account key. Turning key creation off org-wide means a leaked key
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
```

> **Optional, and verify before you commit to it.** `constraints/iam.allowedPolicyMemberDomains`
> restricts IAM members to your Cloud Identity customer ID. It's a genuinely good control — it
> stops anyone binding a role to an arbitrary Gmail address. But it also blocks `allUsers`, and
> `infra/modules/cloud-run-aggregator/main.tf:55-60` binds `roles/run.invoker` to `allUsers` to make
> the dashboard API public.
>
> If you want it, you enforce at the org and relax it on the folders that host public services:
>
> ```hcl
> resource "google_org_policy_policy" "domain_restricted_sharing" {
>   name   = "organizations/${var.org_id}/policies/iam.allowedPolicyMemberDomains"
>   parent = "organizations/${var.org_id}"
>   spec {
>     rules {
>       values {
>         allowed_values = ["is:${var.customer_id}"]   # C0xxxxxxx from Phase 0
>       }
>     }
>   }
> }
>
> resource "google_org_policy_policy" "allow_public_services" {
>   for_each = toset([var.folder_staging, var.folder_prod])
>   name     = "folders/${each.value}/policies/iam.allowedPolicyMemberDomains"
>   parent   = "folders/${each.value}"
>   spec {
>     rules {
>       allow_all = "TRUE"
>     }
>   }
> }
> ```
>
> Apply it, then immediately `curl` your staging API and re-run `terraform plan` in `infra/staging`.
> If the plan wants to recreate `public_invoker`, the exemption isn't taking effect. Roll the policy
> back — `terraform destroy -target` on those two resources — and revisit. Don't leave this
> half-applied over a weekend.

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
`environment:`. And GitHub environment protection rules — required reviewers, branch restrictions —
are enforced *before* the job starts, which means before a token is ever issued. So "prod applies
need my approval" stops being a convention in a workflow file and becomes something GCP enforces at
the IAM layer.

A job with no `environment:` gets a token with no such claim, matches no principalSet, and can
impersonate nothing.

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
> `deployer-<tier>` address, then run one deploy workflow to confirm it still works. Only after a
> green deploy should you go near `git rm -r infra/staging`.

### 3.5 Runner permissions — enumerate, don't grant owner

It's tempting to give each runner `roles/owner` on its target and move on. Don't. The runner is the
most privileged automated thing you have; `owner` includes the ability to rewrite its own IAM, which
makes the blast radius of a compromised workflow unbounded.

```hcl
# infra/platform/runner_iam.tf

locals {
  # Everything the stacks in Phase 3 actually create. If a later apply fails
  # with a permission error, add the role here rather than reaching for owner.
  runner_project_roles = [
    "roles/run.admin",
    "roles/cloudscheduler.admin",
    "roles/datastore.owner",
    "roles/secretmanager.admin",
    "roles/iam.serviceAccountAdmin",
    "roles/resourcemanager.projectIamAdmin",
    "roles/serviceusage.serviceUsageAdmin",
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

# Creating a project means linking billing to it.
resource "google_billing_account_iam_member" "runner_dev_billing" {
  billing_account_id = var.billing_account
  role               = "roles/billing.user"
  member             = "serviceAccount:${google_service_account.tf_runner["dev"].email}"
}

# --- state access: each runner reaches exactly one bucket --------------------

resource "google_storage_bucket_iam_member" "runner_state" {
  for_each = toset(["dev", "staging", "prod"])

  bucket = "${var.state_bucket_prefix}-tfstate-${each.key}"
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

### 3.6 `infra/platform/variables.tf`

```hcl
variable "org_id" {
  description = "Numeric organization ID"
  type        = string
}

variable "customer_id" {
  description = "Cloud Identity customer ID (C0xxxxxxx), for domain-restricted sharing"
  type        = string
  default     = ""
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

variable "folder_staging" {
  description = "Numeric ID of the staging folder"
  type        = string
}

variable "folder_prod" {
  description = "Numeric ID of the prod folder"
  type        = string
}

variable "billing_account" {
  description = "Billing account ID"
  type        = string
}

variable "state_bucket_prefix" {
  description = "Prefix used for state bucket names"
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

### 3.7 `infra/platform/terraform.tfvars`

```hcl
org_id              = "123456789012"
customer_id         = "C01abcdef"
platform_project_id = "fang-dash-platform"
staging_project_id  = "fang-gcp-staging"
prod_project_id     = "fang-gcp"
folder_dev          = "222222222222"
folder_staging      = "333333333333"
folder_prod         = "444444444444"
billing_account     = "01ABCD-234567-89EFGH"
state_bucket_prefix = "fang-dash"
github_repository   = "nickfang/personal-dashboard"
region              = "us-central1"
```

### 3.8 `infra/platform/outputs.tf`

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
```

### 3.9 Apply

```bash
cd infra/platform
terraform init
terraform plan     # read it properly — this grants org-level permissions
terraform apply
```

Then record the outputs; Phase 3 and Phase 5 both consume them.

```bash
terraform output -json > /tmp/platform-outputs.json
terraform output wif_provider
terraform output runner_service_accounts
```

## 4. What you have now

- A platform project holding four state buckets, versioned and locked down
- Bootstrap's own state living in one of those buckets
- A shared image registry every environment will pull from
- Three Terraform runner identities, each reaching exactly one tier, each impersonable only by a
  GitHub job that declares the matching `environment:`
- Three deploy identities on the same pattern, replacing the per-project `github-actions-sa`
- Service account key creation disabled org-wide

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

---

**Next:** [02-state-migration.md](02-state-migration.md) — move staging and prod off your laptop.

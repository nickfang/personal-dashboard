# Phase 3 — Decompose into stacks

This is the phase that makes "deploy only the component I'm working on" possible. It's also the
largest refactor, so it's split into three independent pieces you can do on separate days:

1. **Revise the modules** — add sizing, drop the build, take an image tag
2. **Write the stacks** — foundation plus one stack per component
3. **Move the state** — the mechanical part, and the only risky one

## 1. Why the monolith can't do partial deploys

`infra/staging/main.tf` instantiates ten modules into one state file, with every module carrying
`depends_on = [module.foundation]`. There's no supported way to apply a piece of it.

`terraform apply -target=module.forecast_collector` is the obvious move and it's a trap. Terraform's
documentation is explicit that `-target` exists for recovering from mistakes, not for routine work:
it bypasses normal dependency resolution, so it produces plans that don't reflect the real graph.
Used regularly, it drifts state away from config in ways that surface much later.

What you want instead:

```
staging/foundation              <- APIs, Firestore, secrets, human IAM.  Applied once.
staging/collector-weather       <- own state.  Apply without touching anything else.
staging/collector-pollen
staging/collector-forecast
staging/provider-weather
staging/provider-pollen
staging/aggregator              <- reads provider outputs across state
```

Each is a separate state object under the tier's bucket. Applying `collector-forecast` plans seven
resources, not seventy, and takes only its own lock — so two engineers working on two collectors
don't queue behind each other.

The cost is real and worth naming: the aggregator needs the providers' URLs, and they're now in a
different state file. That's what `terraform_remote_state` is for, and §4.4 covers it.

## 2. Module revisions

### 2.1 Take an image tag instead of building one

Every compute module currently does this:

```hcl
# infra/modules/cloud-run-job/main.tf:25-34  — delete this
resource "null_resource" "bootstrap" {
  provisioner "local-exec" {
    command = <<EOT
      gcloud builds submit ${var.services_path} \
        --config ${var.services_path}/${var.name}/cloudbuild.yaml \
        --substitutions=_IMAGE_TAG=${var.artifact_registry_url}/${var.name}:latest \
        --project ${var.project_id}
    EOT
  }
}
```

Four problems, and they compound:

- No `triggers`, so it runs once per state and never again. It's a seeding step wearing a build
  step's clothes.
- Needs `gcloud` on the runner and the `services/` tree at `../../services` — a relative path that
  only resolves when the working directory is `infra/staging` or `infra/prod`.
- Creating infrastructure requires building containers, so an ephemeral environment pays a full
  Cloud Build before it can exist.
- `terraform plan` can't see any of it, so breakage only appears at apply time.

Replace it with a variable. In all three of `cloud-run-job`, `cloud-run-provider`, and
`cloud-run-aggregator`:

```hcl
# variables.tf — remove services_path and artifact_registry_url, add:

variable "image" {
  description = "Fully-qualified container image, including tag"
  type        = string

  # A brand-new component has no image in the registry yet, and
  # google_cloud_run_v2_service must pass a readiness check to be created --
  # so a missing image means the apply hangs and then fails. Google's public
  # hello container is a valid placeholder that starts and serves. Because
  # `ignore_changes` covers the image field, the first real deploy replaces
  # this and Terraform never reverts it.
  default = "us-docker.pkg.dev/cloudrun/container/hello"
}
```

```hcl
# main.tf — delete null_resource.bootstrap, then:
containers {
  image = var.image      # was "${var.artifact_registry_url}/${var.name}:latest"
}
```

and drop `depends_on = [null_resource.bootstrap]` from each Cloud Run resource.

Nothing is lost. The `lifecycle` block already tells Terraform not to manage the image after
creation:

```hcl
lifecycle {
  ignore_changes = [
    template[0].containers[0].image,
    ...
  ]
}
```

So Terraform sets the image once at create time and GitHub Actions owns it from then on — exactly
as it does today. The only change is that Terraform now takes the tag as input rather than causing
it to exist.

### 2.2 What actually differs between tiers

Sizing is the obvious difference and the least interesting one. Before writing any code, it's worth
enumerating what genuinely varies, because several of these are hardcoded in your modules today and
one of them is the dominant cost in the whole system.

| Concern | dev | staging | prod |
|---|---|---|---|
| **Scheduler cadence** | cron set, **paused** | daily | natural (hourly / 2× / 6h) |
| `max_instances` (spend cap) | 1 | 3 | 10 |
| `min_instances` | 0 | 0 | 0 |
| Firestore PITR | off | off | on |
| Firestore `deletion_policy` | `DELETE` | `DELETE` | `ABANDON` |
| Firestore delete protection | off | off | on |
| Firestore location | regional | regional | decide explicitly |
| Public invoker | owner only | `allUsers` | `allUsers` |
| Image tag | branch SHA or seed | `latest` | pinned SHA |

Two things to notice.

**`min_instances` is 0 everywhere, including prod.** Always-warm instances are the single largest
Cloud Run cost, and for this workload cold starts are tolerable. The knob exists and is set
deliberately per tier — that's the point — but "prod is different" doesn't have to mean "prod is
expensive." If you later decide the dashboard needs a warm aggregator, it's a one-line change in
`envs/prod.tfvars` and you'll know exactly what it costs.

**Scheduler cadence is the real cost lever, not Cloud Run.** Three collectors on crons, each run
calling the Google Maps API. Staging currently runs the *same* cadence as prod, burning identical
API quota for data nobody reads. And an ephemeral environment with unpaused schedulers starts
billing API calls the moment it's created — multiply that by a few concurrent dev environments and
it dwarfs everything else in this design.

#### Sizing

Neither `resources` nor `scaling` appears anywhere in the modules today, so every service runs on
GCP defaults. Add a profile object rather than a scatter of separate variables — it keeps the tier
distinction legible in the tfvars.

```hcl
# infra/modules/cloud-run-provider/variables.tf  (same for cloud-run-aggregator)

variable "profile" {
  description = "Per-tier sizing and scaling"
  type = object({
    cpu           = string
    memory        = string
    min_instances = number
    max_instances = number
  })
}
```

```hcl
# infra/modules/cloud-run-provider/main.tf
resource "google_cloud_run_v2_service" "service" {
  name     = "${var.name}-service"
  location = var.region

  template {
    service_account = google_service_account.service_account.email

    scaling {
      min_instance_count = var.profile.min_instances
      max_instance_count = var.profile.max_instances
    }

    containers {
      image = var.image

      resources {
        limits = {
          cpu    = var.profile.cpu
          memory = var.profile.memory
        }

        # With a warm instance you want CPU always allocated, otherwise the
        # instance is resident but throttled and you have paid for warmth
        # without getting it.
        cpu_idle          = var.profile.min_instances == 0
        startup_cpu_boost = true
      }

      ports {
        container_port = var.port
        name           = "h2c"
      }

      dynamic "env" {
        for_each = var.env_vars
        content {
          name  = env.key
          value = env.value
        }
      }
    }
  }

  lifecycle {
    ignore_changes = [
      template[0].containers[0].image,
      client,
      client_version,
      template[0].labels,
      template[0].annotations,
    ]
  }
}
```

`cpu_idle` is derived rather than configured, and it's worth understanding even though every tier
currently sets `min_instances = 0`. With `min_instance_count = 1` and `cpu_idle = true`, an instance
stays resident but its CPU is throttled between requests — you've paid for a warm container and
still get a slow first response. Tying it to `min_instances > 0` means that if you ever do turn
warmth on, you get the thing you're paying for, without having to remember a second setting.

Think of `max_instances` as a **spend cap** rather than a scaling target. It's what stops a retry
loop or a traffic spike from costing you a hundred dollars overnight, which is why dev gets `1`.

Jobs only need `resources`; there's nothing to scale:

```hcl
# infra/modules/cloud-run-job/variables.tf
variable "profile" {
  type = object({
    cpu    = string
    memory = string
  })
}
```

```hcl
# infra/modules/cloud-run-job/main.tf, inside containers { }
resources {
  limits = {
    cpu    = var.profile.cpu
    memory = var.profile.memory
  }
}
```

#### Scheduling — the expensive one

`modules/cloud-run-job/main.tf:87-88` hardcodes `time_zone` and `attempt_deadline`, and the module
has no way to express "create this schedule but don't run it." `google_cloud_scheduler_job` has a
`paused` attribute, which is exactly what an ephemeral environment needs.

```hcl
# infra/modules/cloud-run-job/variables.tf
variable "schedule_policy" {
  description = "Per-tier scheduling behaviour"
  type = object({
    paused           = bool
    time_zone        = string
    attempt_deadline = string
  })
  default = {
    paused           = false
    time_zone        = "America/Chicago"
    attempt_deadline = "320s"
  }
}
```

```hcl
# infra/modules/cloud-run-job/main.tf
resource "google_cloud_scheduler_job" "trigger" {
  name             = "trigger-${var.name}"
  description      = var.scheduler_description
  schedule         = var.schedule
  paused           = var.schedule_policy.paused
  time_zone        = var.schedule_policy.time_zone
  attempt_deadline = var.schedule_policy.attempt_deadline

  # ... http_target unchanged ...
}
```

A dev environment still gets the scheduler resource — so you can see it, and trigger the job
manually with `gcloud run jobs execute` — it just never fires on its own.

The cadence itself is already a module variable (`var.schedule`), passed per component. What changes
is that staging stops copying prod's values:

```hcl
# infra/envs/staging/collector-weather.tfvars
schedule = "0 6 * * *"        # daily, not hourly — staging data only needs to be plausible
```

#### Exposure

`modules/cloud-run-aggregator/main.tf:20` pins `ingress = "INGRESS_TRAFFIC_ALL"` and `:59` pins
`member = "allUsers"`. Both are correct for prod and wrong for a dev environment, which shouldn't
put an unauthenticated API on the public internet for three days.

```hcl
# infra/modules/cloud-run-aggregator/variables.tf
variable "ingress" {
  type    = string
  default = "INGRESS_TRAFFIC_ALL"
}

variable "public" {
  description = "Bind roles/run.invoker to allUsers"
  type        = bool
  default     = false
}

variable "invoker_members" {
  description = "Additional principals allowed to invoke, used when public is false"
  type        = list(string)
  default     = []
}
```

```hcl
# infra/modules/cloud-run-aggregator/main.tf
resource "google_cloud_run_v2_service" "service" {
  ingress = var.ingress
  # ...
}

resource "google_cloud_run_v2_service_iam_member" "public_invoker" {
  count = var.public ? 1 : 0

  name     = google_cloud_run_v2_service.service.name
  location = google_cloud_run_v2_service.service.location
  role     = "roles/run.invoker"
  member   = "allUsers"
}

resource "google_cloud_run_v2_service_iam_member" "named_invoker" {
  for_each = toset(var.public ? [] : var.invoker_members)

  name     = google_cloud_run_v2_service.service.name
  location = google_cloud_run_v2_service.service.location
  role     = "roles/run.invoker"
  member   = each.value
}
```

Note `public` defaults to `false`. Defaulting a public-internet binding to off means forgetting to
set it produces an inaccessible service rather than an exposed one.

> Adding `count` to `public_invoker` changes its address from
> `google_cloud_run_v2_service_iam_member.public_invoker` to `...public_invoker[0]`. For prod, where
> you're moving state rather than rebuilding, that needs a `terraform state mv` — see §5.2.

#### Data lifecycle

`modules/firestore/main.tf` sets no `deletion_policy`, which is why `terraform destroy` on a
populated database fails. That's not a bug to fix uniformly — it's a tier difference, and it
currently blocks both the staging rebuild in §5.1 and the Phase 4 reaper.

```hcl
# infra/modules/firestore/variables.tf
variable "data_policy" {
  type = object({
    location               = string
    deletion_policy        = string
    delete_protection      = bool
    point_in_time_recovery = bool
  })
}
```

```hcl
# infra/modules/firestore/main.tf
resource "google_firestore_database" "database" {
  for_each = toset(var.database_ids)

  project     = var.project_id
  name        = each.value
  location_id = var.data_policy.location
  type        = "FIRESTORE_NATIVE"

  # ABANDON leaves the database in place when Terraform stops managing it;
  # DELETE lets an ephemeral environment actually tear down.
  deletion_policy = var.data_policy.deletion_policy

  delete_protection_state = var.data_policy.delete_protection ? "DELETE_PROTECTION_ENABLED" : "DELETE_PROTECTION_DISABLED"

  point_in_time_recovery_enablement = var.data_policy.point_in_time_recovery ? "POINT_IN_TIME_RECOVERY_ENABLED" : "POINT_IN_TIME_RECOVERY_DISABLED"
}
```

PITR carries a real per-GB cost, which is why it's prod-only rather than on everywhere. The location
being a variable rather than `var.region` is what lets prod choose a multi-region like `nam5` for
durability while dev stays regional and cheap — worth deciding explicitly rather than inheriting.

### 2.3 Pull from the shared registry

`infra/modules/foundation/main.tf` creates a per-project Artifact Registry. Phase 1 replaced it with
a shared one in the platform project, so delete that resource and its output.

Cloud Run pulls cross-project images as the **Cloud Run Service Agent**, not as your workload's
service account — a distinction that produces a confusing `image not found` error if you miss it.
The agent is `service-<PROJECT_NUMBER>@serverless-robot-prod.iam.gserviceaccount.com`, and it needs
read access on the shared repo.

That binding lives in the platform project but can only be written once the environment project
exists, so the environment's foundation stack creates it through an aliased provider:

```hcl
# infra/stacks/foundation/registry_access.tf

provider "google" {
  alias   = "platform"
  project = var.platform_project_id
  region  = var.region
}

data "google_project" "env" {}

resource "google_artifact_registry_repository_iam_member" "pull" {
  provider = google.platform

  project    = var.platform_project_id
  location   = var.region
  repository = "personal-dashboard"
  role       = "roles/artifactregistry.reader"
  member     = "serviceAccount:service-${data.google_project.env.number}@serverless-robot-prod.iam.gserviceaccount.com"
}
```

> **Amend Phase 1 for this.** The runners currently hold `roles/artifactregistry.reader` on the
> shared repo, which lets them pull but not grant. Change that binding in
> `infra/platform/runner_iam.tf` to `roles/artifactregistry.admin` — still scoped to the one
> repository, not the platform project:
>
> ```hcl
> resource "google_artifact_registry_repository_iam_member" "runner_admin" {
>   for_each = toset(["dev", "staging", "prod"])
>
>   location   = google_artifact_registry_repository.shared.location
>   repository = google_artifact_registry_repository.shared.name
>   role       = "roles/artifactregistry.admin"
>   member     = "serviceAccount:${google_service_account.tf_runner[each.key].email}"
> }
> ```
>
> This is the one place a runner reaches outside its own tier. Keeping it repository-scoped is what
> stops it becoming a general foothold in the platform project.

### 2.4 Human access, bound to groups

There is currently no human IAM in Terraform at all — every binding in every module is for a service
account. Your requirement was different principals per tier, so this is net-new.

```hcl
# infra/modules/env-iam/variables.tf
variable "project_id" {
  type = string
}

variable "role_bindings" {
  description = "Map of IAM role to list of member strings"
  type        = map(list(string))
  default     = {}
}
```

```hcl
# infra/modules/env-iam/main.tf
locals {
  flattened = merge([
    for role, members in var.role_bindings : {
      for m in members : "${role}|${m}" => { role = role, member = m }
    }
  ]...)
}

resource "google_project_iam_member" "binding" {
  for_each = local.flattened

  project = var.project_id
  role    = each.value.role
  member  = each.value.member
}
```

Then per tier, in the environment's tfvars:

```hcl
# infra/envs/staging.tfvars
human_iam = {
  "roles/viewer"                = ["group:gcp-staging-viewers@yourdomain.com"]
  "roles/run.viewer"            = ["group:gcp-developers@yourdomain.com"]
  "roles/logging.viewer"        = ["group:gcp-developers@yourdomain.com"]
  "roles/datastore.viewer"      = ["group:gcp-developers@yourdomain.com"]
}
```

```hcl
# infra/envs/prod.tfvars
human_iam = {
  "roles/viewer"         = ["group:gcp-prod-oncall@yourdomain.com"]
  "roles/logging.viewer" = ["group:gcp-prod-oncall@yourdomain.com"]
}
```

Bind to **groups, never individuals**. An individual email in a binding means every joiner and
leaver is a Terraform change, a PR, and an apply. A group means it's a click in the admin console
and Terraform never knows. Note prod grants no write access to humans at all — service accounts do
the work, which is what you described wanting.

### 2.5 A note on the project-wide grants

You might expect this phase to tighten these:

```hcl
# infra/modules/cloud-run-job/main.tf:6-16
role = "roles/datastore.user"    # project-wide
role = "roles/run.invoker"       # project-wide
```

Leave them. They're project-scoped, and since Phase 0 the project *is* the environment boundary —
everything in that project belongs to the same environment and the same owner. Tightening them to
per-database bindings would be real work for no boundary you don't already have.

This is only true because you chose project-per-environment. Had you gone with a shared dev project
and name prefixes, these would need converting to resource-scoped bindings before that project could
host two engineers safely.

### 2.6 Let the deploy identity act as the runtime service accounts

[01-bootstrap-and-platform.md §3.4](01-bootstrap-and-platform.md) created `deployer-<tier>` to
replace `github-actions-sa`, but deliberately left out one binding. Deploying a Cloud Run revision
means setting the runtime service account on it, and that requires
`roles/iam.serviceAccountUser` on that specific SA — which doesn't exist until the component stack
creates it. So the binding belongs here, next to the SA it refers to.

Add to all three compute modules, beside the existing `google_service_account` resource:

```hcl
variable "deployer_sa" {
  description = "Deploy identity permitted to set this SA on a revision"
  type        = string
}

resource "google_service_account_iam_member" "deployer_act_as" {
  service_account_id = google_service_account.sa.name
  role               = "roles/iam.serviceAccountUser"
  member             = "serviceAccount:${var.deployer_sa}"
}
```

Scoped to the one service account, rather than the project-wide grant the old `github-oidc` module
built by string-interpolating SA addresses. A deploy identity that can act as `weather-collector-sa`
still can't act as anything else.

The value comes from the environment's tfvars:

```hcl
# infra/envs/staging.tfvars
deployer_sa = "deployer-staging@fang-dash-platform.iam.gserviceaccount.com"
```

## 3. Stack layout

```
infra/stacks/
  foundation/       APIs, Firestore, secrets, registry access, human IAM
  collector/        one Cloud Run Job + scheduler + SA
  provider/         one internal Cloud Run Service + SA
  aggregator/       public service, reads provider URLs across state
```

Each is a root module. Each is initialized against its own state prefix.

## 4. The stacks

### 4.1 `infra/stacks/foundation/`

```hcl
# infra/stacks/foundation/main.tf
terraform {
  required_version = ">= 1.13"

  backend "gcs" {}

  required_providers {
    google      = { source = "hashicorp/google",      version = "~> 5.0" }
    google-beta = { source = "hashicorp/google-beta", version = "~> 5.0" }
  }
}

provider "google" {
  project = var.project_id
  region  = var.region
}

provider "google-beta" {
  project = var.project_id
  region  = var.region
}

module "foundation" {
  source     = "../../modules/foundation"
  project_id = var.project_id
  region     = var.region
}

module "firestore" {
  source       = "../../modules/firestore"
  project_id   = var.project_id
  region       = var.region
  database_ids = var.database_ids

  depends_on = [module.foundation]
}

module "secrets" {
  source     = "../../modules/secrets"
  project_id = var.project_id
  secret_ids = var.secret_ids

  depends_on = [module.foundation]
}

module "human_iam" {
  source        = "../../modules/env-iam"
  project_id    = var.project_id
  role_bindings = var.human_iam
}
```

```hcl
# infra/stacks/foundation/outputs.tf
output "project_id" {
  value = var.project_id
}

output "region" {
  value = var.region
}

output "registry_url" {
  description = "Shared registry the component stacks pull from"
  value       = "${var.region}-docker.pkg.dev/${var.platform_project_id}/personal-dashboard"
}
```

### 4.2 `infra/stacks/collector/`

One root, instantiated once per collector. The component it deploys is an input.

```hcl
# infra/stacks/collector/main.tf
terraform {
  required_version = ">= 1.13"
  backend "gcs" {}
  required_providers {
    google = { source = "hashicorp/google", version = "~> 5.0" }
  }
}

provider "google" {
  project = var.project_id
  region  = var.region
}

data "terraform_remote_state" "foundation" {
  backend = "gcs"
  config = {
    bucket = var.state_bucket
    prefix = "${var.env_name}/foundation"
  }
}

locals {
  # An empty image_tag means "nothing has been built for this component yet" --
  # a brand-new environment, or a component whose first CI run hasn't happened.
  # Point at the placeholder rather than a registry path that 404s.
  seed_image = "us-docker.pkg.dev/cloudrun/container/hello"

  image = var.image_tag == "" ? local.seed_image : join("", [
    data.terraform_remote_state.foundation.outputs.registry_url,
    "/", var.name, ":", var.image_tag,
  ])
}

module "collector" {
  source = "../../modules/cloud-run-job"

  project_id            = var.project_id
  region                = var.region
  name                  = var.name
  sa_display_name       = "Service Account for ${var.name}"
  schedule              = var.schedule
  scheduler_description = var.scheduler_description
  image                 = local.image
  deployer_sa           = var.deployer_sa
  profile               = var.job_profile
  env_vars              = merge({ GCP_PROJECT_ID = var.project_id }, var.env_vars)

  secret_env_vars = {
    GOOGLE_MAPS_API_KEY = {
      secret_id = "google-maps-api-key"
      version   = "latest"
    }
  }

  secret_refs = ["google-maps-api-key"]
}
```

```hcl
# infra/stacks/collector/variables.tf
variable "project_id"   { type = string }
variable "region"       { type = string }
variable "env_name"     { type = string }
variable "state_bucket" { type = string }

variable "name" {
  description = "Collector name, e.g. forecast-collector"
  type        = string
}

variable "schedule" { type = string }

variable "scheduler_description" {
  type    = string
  default = ""
}

variable "image_tag" {
  description = "Image tag to deploy; empty means use the placeholder seed image"
  type        = string
  default     = ""
}

variable "deployer_sa" {
  description = "Deploy identity permitted to act as this component's runtime SA"
  type        = string
}

variable "env_vars" {
  type    = map(string)
  default = {}
}

variable "job_profile" {
  type = object({
    cpu    = string
    memory = string
  })
}
```

The per-collector differences — schedule, and the forecast collector's tuning variables — move into
a tfvars file per component:

```hcl
# infra/envs/staging/collector-forecast.tfvars
name                  = "forecast-collector"
schedule              = "0 */6 * * *"
scheduler_description = "Triggers the forecast collector job every 6 hours"

env_vars = {
  FORECAST_HORIZON_HOURS = "72"
  PRESSURE_DROP_MB       = "5"
  PRESSURE_SEVERE_MB     = "10"
  PRESSURE_WINDOW_HOURS  = "3"
}
```

### 4.3 `infra/stacks/provider/`

Same shape, wrapping `modules/cloud-run-provider`, with `port` and `profile` as inputs. Its output
is what the aggregator needs:

```hcl
# infra/stacks/provider/outputs.tf
output "service_uri" {
  value = module.provider.service_uri
}

output "service_name" {
  value = module.provider.service_name
}

output "service_location" {
  value = module.provider.service_location
}
```

### 4.4 `infra/stacks/aggregator/` — reading across state

This is where decomposition costs you something. In the monolith, the aggregator referenced
`module.weather_provider.service_uri` directly. Across separate states, it reads them:

```hcl
# infra/stacks/aggregator/main.tf
data "terraform_remote_state" "foundation" {
  backend = "gcs"
  config = {
    bucket = var.state_bucket
    prefix = "${var.env_name}/foundation"
  }
}

data "terraform_remote_state" "provider_weather" {
  backend = "gcs"
  config = {
    bucket = var.state_bucket
    prefix = "${var.env_name}/provider-weather"
  }
}

data "terraform_remote_state" "provider_pollen" {
  backend = "gcs"
  config = {
    bucket = var.state_bucket
    prefix = "${var.env_name}/provider-pollen"
  }
}

module "dashboard_api" {
  source = "../../modules/cloud-run-aggregator"

  project_id      = var.project_id
  region          = var.region
  name            = "dashboard-api"
  sa_display_name = "Service Account for Dashboard API Service"
  port            = 8080
  image           = local.image # same seed fallback as the collector stack
  profile         = local.profile
  deployer_sa     = var.deployer_sa

  env_vars = {
    WEATHER_PROVIDER_ADDR = "${trimprefix(data.terraform_remote_state.provider_weather.outputs.service_uri, "https://")}:443"
    POLLEN_PROVIDER_ADDR  = "${trimprefix(data.terraform_remote_state.provider_pollen.outputs.service_uri, "https://")}:443"
  }

  invoker_targets = [
    {
      name     = data.terraform_remote_state.provider_weather.outputs.service_name
      location = data.terraform_remote_state.provider_weather.outputs.service_location
    },
    {
      name     = data.terraform_remote_state.provider_pollen.outputs.service_name
      location = data.terraform_remote_state.provider_pollen.outputs.service_location
    },
  ]
}
```

Three consequences worth internalizing. The third is a genuine regression, and it's the price of
decomposition rather than a bug to fix.

- **Ordering is now yours to enforce.** Terraform can't see that providers must exist before the
  aggregator; if the provider state is missing, `terraform_remote_state` fails at plan time with a
  fairly opaque error. The `bin/` scripts in Phase 4 encode the order.
- **Reading remote state means reading the bucket.** It works here because both stacks live in the
  same tier bucket, which the tier's runner can read. A stack can never read across tiers — that's
  the isolation from Phase 1 doing its job.
- **The aggregator's view of the providers can go stale, silently.** In the monolith,
  `module.dashboard_api` referenced `module.weather_provider.service_uri` directly and Terraform
  propagated any change automatically. Now the URI is copied out of another state file at the
  aggregator's last apply. If a provider is destroyed and recreated its URI changes, and
  `WEATHER_PROVIDER_ADDR` keeps pointing at the old one until someone re-applies the aggregator.
  Terraform cannot see this and will not warn you.

  The CI workflows in Phase 5 paper over it by always applying the aggregator after the providers.
  But the whole point of this phase was targeted single-component deploys — and that is precisely
  the path where it breaks. Deploy just `provider-weather` after a recreate and the aggregator is
  pointing at a dead address until you remember.

  The real fix is to stop using the generated URI as the coupling point. Give each provider a stable
  internal address — a custom domain mapping, or a fixed name resolved at runtime — so the
  aggregator's configuration doesn't change when the service is recreated. Until then, treat
  "recreated a provider" as implying "re-apply the aggregator," and know that nothing enforces it.

### 4.5 Tier configuration in the environment tfvars

One profile applied to all six services would quietly collapse them back into a single deployable
unit — which is the opposite of what splitting them was for. Use a tier default plus per-component
overrides:

```hcl
# infra/stacks/provider/variables.tf  (same shape in aggregator)

variable "service_defaults" {
  description = "Tier-wide sizing, used unless a component overrides it"
  type = object({
    cpu           = string
    memory        = string
    min_instances = number
    max_instances = number
  })
}

variable "service_overrides" {
  description = "Per-component overrides, keyed by component name. Partial objects allowed."
  type        = map(map(string))
  default     = {}
}
```

```hcl
# infra/stacks/provider/main.tf
locals {
  override = lookup(var.service_overrides, var.name, {})

  profile = {
    cpu           = lookup(local.override, "cpu", var.service_defaults.cpu)
    memory        = lookup(local.override, "memory", var.service_defaults.memory)
    min_instances = tonumber(lookup(local.override, "min_instances", var.service_defaults.min_instances))
    max_instances = tonumber(lookup(local.override, "max_instances", var.service_defaults.max_instances))
  }
}
```

`service_overrides` is `map(map(string))` rather than a map of typed objects because Terraform
object types require every attribute to be present — which would defeat the purpose of a partial
override. The `tonumber` calls convert back at the point of use.

Now the three tiers:

```hcl
# infra/envs/prod.tfvars
tier = "prod"

service_defaults = {
  cpu           = "1"
  memory        = "512Mi"
  min_instances = 0      # cold starts accepted; raise per component if that changes
  max_instances = 10
}

service_overrides = {
  # Example: if the dashboard ever needs a warm instance, it goes here and
  # nowhere else -- the providers behind it stay at zero.
  # dashboard-api = { min_instances = "1", cpu = "2", memory = "1Gi" }
}

job_profile = { cpu = "1", memory = "512Mi" }

schedule_policy = {
  paused           = false
  time_zone        = "America/Chicago"
  attempt_deadline = "320s"
}

data_policy = {
  location               = "nam5"
  deletion_policy        = "ABANDON"
  delete_protection      = true
  point_in_time_recovery = true
}

public      = true
ingress     = "INGRESS_TRAFFIC_ALL"
api_domain  = "api.example.com"
deployer_sa = "deployer-prod@fang-dash-platform.iam.gserviceaccount.com"
```

```hcl
# infra/envs/staging.tfvars
tier = "staging"

service_defaults = {
  cpu           = "1"
  memory        = "512Mi"
  min_instances = 0
  max_instances = 3
}

job_profile = { cpu = "1", memory = "512Mi" }

schedule_policy = {
  paused           = false
  time_zone        = "America/Chicago"
  attempt_deadline = "320s"
}

data_policy = {
  location               = "us-central1"
  deletion_policy        = "DELETE"      # so §5.1's rebuild works
  delete_protection      = false
  point_in_time_recovery = false
}

public      = true
ingress     = "INGRESS_TRAFFIC_ALL"
api_domain  = "api-staging.example.com"
deployer_sa = "deployer-staging@fang-dash-platform.iam.gserviceaccount.com"
```

```hcl
# infra/envs/dev.tfvars
tier = "dev"

service_defaults = {
  cpu           = "1"
  memory        = "512Mi"
  min_instances = 0
  max_instances = 1      # spend cap, not a scaling target
}

job_profile = { cpu = "1", memory = "512Mi" }

schedule_policy = {
  paused           = true    # the single most important line in this file
  time_zone        = "America/Chicago"
  attempt_deadline = "320s"
}

data_policy = {
  location               = "us-central1"
  deletion_policy        = "DELETE"      # so the reaper can tear down
  delete_protection      = false
  point_in_time_recovery = false
}

public      = false        # owner-only; set invoker_members to your account
ingress     = "INGRESS_TRAFFIC_ALL"
api_domain  = ""           # no domain mapping; use the generated *.run.app URL
deployer_sa = "deployer-dev@fang-dash-platform.iam.gserviceaccount.com"
```

#### Prod must pin an immutable tag

`latest` is fine for dev and defensible for staging. In prod it makes a redeploy non-reproducible
and a rollback ambiguous. Encode it rather than relying on discipline:

```hcl
variable "tier" {
  type = string
}

variable "image_tag" {
  description = "Image tag to deploy; empty means the placeholder seed image"
  type        = string
  default     = ""

  validation {
    condition     = var.tier != "prod" || !contains(["", "latest"], var.image_tag)
    error_message = "prod must pin an immutable tag; empty and 'latest' are not allowed."
  }
}
```

> A validation block referring to *another* variable only works on Terraform 1.9 or newer — before
> that, validation could reference only the variable it belonged to. The `required_version = ">= 1.13"`
> in every stack already covers this. Verified against 1.15.8: `-var tier=prod -var image_tag=latest`
> fails at plan time, while `tier=dev` with the same tag passes.

#### Per-tier secrets

One `google-maps-api-key` per environment already, but they should be *different keys*, not copies
of one. Separate keys mean a dev environment can't exhaust prod's API quota, and a leaked dev key is
revoked without touching prod. No Terraform change — same empty container, same
`ignore_changes = [secret_data]`, different value added out-of-band per project.

## 5. Moving the state

The riskiest mechanical step in the tutorial. Two routes, and they suit the two environments
differently.

### 5.1 Staging: rebuild instead of migrate

Staging is rebuildable, and `docs/DISASTER_RECOVERY.md` is already written around that assumption.
Tearing it down and re-applying from the new stacks is far less error-prone than state surgery, and
it doubles as a genuine test of your DR runbook.

```bash
cd infra/staging
terraform destroy -var-file=../envs/staging.tfvars
```

Then apply the stacks in dependency order (§5.3).

> **You will lose the staging Firestore data.** The `weather-log` and `pollen-log` databases go with
> it. That's fine if staging holds only collected weather data that repopulates on the next
> scheduled run — check that's true for you before running this.
>
> Also: `infra/modules/firestore/main.tf` sets no `deletion_policy`, so `terraform destroy` on a
> populated database may fail outright. If it does, delete the databases by hand first with
> `gcloud firestore databases delete --database=weather-log`.

### 5.2 Prod: state surgery

Prod isn't rebuildable on a whim, so move the state. `terraform state mv` operates on local state
files, so pull down, split, push up.

```bash
cd infra/prod
terraform state pull > /tmp/prod-monolith.tfstate
cp /tmp/prod-monolith.tfstate /tmp/prod-monolith.backup.tfstate   # do not skip this
```

Split out one stack at a time:

```bash
# Foundation
terraform state mv \
  -state=/tmp/prod-monolith.tfstate \
  -state-out=/tmp/prod-foundation.tfstate \
  module.foundation module.foundation

terraform state mv \
  -state=/tmp/prod-monolith.tfstate \
  -state-out=/tmp/prod-foundation.tfstate \
  module.firestore module.firestore

terraform state mv \
  -state=/tmp/prod-monolith.tfstate \
  -state-out=/tmp/prod-foundation.tfstate \
  module.secrets module.secrets

# One collector — note the address changes from module.weather_collector to
# module.collector, because each stack instantiates the module under one name.
terraform state mv \
  -state=/tmp/prod-monolith.tfstate \
  -state-out=/tmp/prod-collector-weather.tfstate \
  module.weather_collector module.collector
```

Repeat for the remaining collectors, both providers, and the aggregator. Then push each into its
prefix:

```bash
cd ../stacks/foundation
terraform init -backend-config="bucket=fang-dash-tfstate-prod" \
               -backend-config="prefix=prod/foundation"
terraform state push /tmp/prod-foundation.tfstate

cd ../collector
terraform init -reconfigure \
               -backend-config="bucket=fang-dash-tfstate-prod" \
               -backend-config="prefix=prod/collector-weather"
terraform state push /tmp/prod-collector-weather.tfstate
```

One address changes shape rather than just moving. Adding `count` to the public invoker in §2.2
turns `google_cloud_run_v2_service_iam_member.public_invoker` into `...public_invoker[0]`:

```bash
terraform state mv \
  -state=/tmp/prod-aggregator.tfstate \
  'module.dashboard_api.google_cloud_run_v2_service_iam_member.public_invoker' \
  'module.dashboard_api.google_cloud_run_v2_service_iam_member.public_invoker[0]'
```

Miss it and Terraform plans to destroy the existing binding and create an identical one — briefly
taking your public API offline for no reason.

> **`terraform state push -force` deserves more caution than one line.** Into a genuinely empty
> prefix, a plain `push` works. If it refuses over lineage or serial mismatch, verify the target is
> empty *before* reaching for `-force`, because into a non-empty prefix it silently overwrites
> whatever was there:
>
> ```bash
> gcloud storage ls "gs://fang-dash-tfstate-prod/prod/foundation/" && \
>   echo "NOT EMPTY -- do not use -force" || echo "empty, safe to push"
> ```
>
> The alternative worth knowing about: `removed` blocks in the old root paired with `import` blocks
> in the new stack. Both are supported in Terraform 1.5+/1.7+ and you have 1.15.8. It's more typing,
> but the moves are declarative, visible in `terraform plan`, and reviewable in a pull request
> instead of being a sequence of local commands nobody can audit afterwards. For prod specifically,
> that trade is usually worth it.

After each push:

```bash
terraform plan -var-file=../../envs/prod.tfvars -var-file=../../envs/prod/collector-weather.tfvars
```

**Clean plan, or stop.** The state you pushed is the only record of those resources.

A few diffs are expected and legitimate — the new `resources` and `scaling` blocks are genuinely new
configuration, so Terraform will want to apply them. What must *not* appear is any `create` or
`destroy` of a service account, Cloud Run service, job, database, or secret. Those mean an address
didn't line up and Terraform has lost track of an existing resource.

Once everything is pushed and planning cleanly:

```bash
git rm -r infra/prod infra/staging
```

Keep `/tmp/prod-monolith.backup.tfstate` somewhere durable for a month.

### 5.3 Apply order

Whichever route you took, this is the dependency order — `terraform_remote_state` makes it your
responsibility now:

```
foundation
  ├── collector-weather      (independent)
  ├── collector-pollen       (independent)
  ├── collector-forecast     (independent)
  ├── provider-weather ──┐
  ├── provider-pollen  ──┤
  └──────────────────────┴──> aggregator
                                  └──> domain mapping (staging)
```

Collectors depend on nothing but foundation, which is exactly why a dev environment can consist of
foundation plus one collector and nothing else.

## 6. While you're here: the drift between staging and prod

Two things surfaced from reading the two roots side by side. Fix them now that there's one
definition:

- `sa_display_name` differs in three places purely by capitalization — `"...Provider Service"` in
  staging vs `"...Provider service"` in prod. Nothing depends on it; pick one.
- **Prod has no domain mapping.** Staging wires `module.dashboard_api_domain`; prod has no
  `api_domain` variable at all. That looks like an omission rather than a decision. If prod should
  have one, add the variable and instantiate the module in the aggregator stack, gated on the
  variable being set:

  ```hcl
  module "domain" {
    count  = var.api_domain == "" ? 0 : 1
    source = "../../modules/cloud-run-domain-mapping"

    domain       = var.api_domain
    service_name = module.dashboard_api.service_name
    region       = var.region
    project_id   = var.project_id
  }
  ```

  Dev environments leave `api_domain` empty and get the generated `*.run.app` URL, which is what you
  want for something that lives a day.

---

## Verification gate

```bash
# 1. Everything still formats and validates
cd infra && terraform fmt -recursive -check
for d in stacks/*/; do (cd "$d" && terraform init -backend=false && terraform validate); done
```

```bash
# 2. Prod plans clean after the state move
cd infra/stacks/foundation
terraform init -reconfigure -backend-config="bucket=fang-dash-tfstate-prod" -backend-config="prefix=prod/foundation"
terraform plan -var-file=../../envs/prod.tfvars
```
No `create` or `destroy` of existing infrastructure. Sizing changes are expected.

```bash
# 3. Independence is real — the whole point of this phase
cd infra/stacks/collector
terraform init -reconfigure -backend-config="bucket=fang-dash-tfstate-staging" -backend-config="prefix=staging/collector-forecast"
terraform plan -var-file=../../envs/staging.tfvars -var-file=../../envs/staging/collector-forecast.tfvars
```
The plan mentions only forecast-collector resources. If providers or the aggregator appear,
something is still coupled.

```bash
# 4. The spend cap landed, and nothing is warm
gcloud run services describe dashboard-api-service --region=us-central1 --project=fang-gcp \
  --format="value(
    spec.template.metadata.annotations['autoscaling.knative.dev/minScale'],
    spec.template.metadata.annotations['autoscaling.knative.dev/maxScale']
  )"
```
Expect min empty-or-`0` and max `10`. If min is `1`, an override is set that you didn't intend —
that's the expensive mistake this tier design exists to make visible.

```bash
# 5. Per-component overrides work without leaking to other components
cd infra/stacks/provider
terraform plan -var-file=../../envs/prod.tfvars -var-file=../../envs/prod/provider-weather.tfvars
```
The weather provider picks up `service_defaults`. Add a `dashboard-api` entry to
`service_overrides`, re-plan this stack, and confirm **nothing changes** — overrides must be keyed
by component, not applied tier-wide.

```bash
# 6. Firestore can actually be torn down in non-prod
gcloud firestore databases describe --database=weather-log --project=<a-dev-project> \
  --format="value(deleteProtectionState, pointInTimeRecoveryEnablement)"
```
Dev/staging: `DELETE_PROTECTION_DISABLED`. Prod: `DELETE_PROTECTION_ENABLED` with PITR enabled. If
dev shows protection on, `terraform destroy` and the Phase 4 reaper will both fail on it.

```bash
# 7. Cross-project image pull works
gcloud run jobs describe weather-collector-job --region=us-central1 --project=fang-gcp \
  --format="value(spec.template.spec.template.spec.containers[0].image)"
```
Points at `us-central1-docker.pkg.dev/fang-dash-platform/personal-dashboard/...`. If a deploy fails
with `image not found`, the Cloud Run Service Agent binding from §2.3 is missing.

```bash
# 8. Prod refuses a floating tag
cd infra/stacks/aggregator
terraform plan -var-file=../../envs/prod.tfvars -var image_tag=latest
```
Must fail with *"prod must pin an immutable tag"*. Verified against Terraform 1.15.8 — the
cross-variable validation this relies on needs 1.9 or newer.

---

**Next:** [04-ephemeral-envs.md](04-ephemeral-envs.md) — the self-service part.

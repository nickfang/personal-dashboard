terraform {
  required_providers {
    google = {
      source  = "hashicorp/google"
      version = "~> 5.0"
    }
    google-beta = {
      source  = "hashicorp/google-beta"
      version = "~> 5.0"
    }
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

locals {
  services_path = "../../services"
}

# --- Foundation ---

module "foundation" {
  source     = "../modules/foundation"
  project_id = var.project_id
  region     = var.region
}

# --- Data ---

module "firestore" {
  source       = "../modules/firestore"
  project_id   = var.project_id
  region       = var.region
  database_ids = ["weather-log", "pollen-log"]

  depends_on = [module.foundation]
}

module "secrets" {
  source     = "../modules/secrets"
  project_id = var.project_id
  secret_ids = ["google-maps-api-key", "notify-smtp-password"]

  depends_on = [module.foundation]
}

# --- Collectors (Cloud Run Jobs) ---

module "weather_collector" {
  source                = "../modules/cloud-run-job"
  project_id            = var.project_id
  region                = var.region
  name                  = "weather-collector"
  sa_display_name       = "Service Account for Weather Collector Job"
  schedule              = "0 * * * *"
  scheduler_description = "Triggers the weather collector job every hour"
  artifact_registry_url = module.foundation.artifact_registry_url
  services_path         = local.services_path

  env_vars = {
    GCP_PROJECT_ID = var.project_id
  }

  secret_env_vars = {
    GOOGLE_MAPS_API_KEY = {
      secret_id = "google-maps-api-key"
      version   = "latest"
    }
  }

  secret_refs = ["google-maps-api-key"]

  depends_on = [module.foundation, module.secrets]
}

module "pollen_collector" {
  source                = "../modules/cloud-run-job"
  project_id            = var.project_id
  region                = var.region
  name                  = "pollen-collector"
  sa_display_name       = "Service Account for Pollen Collector Job"
  schedule              = "0 6,14 * * *"
  scheduler_description = "Triggers the pollen collector job twice daily"
  artifact_registry_url = module.foundation.artifact_registry_url
  services_path         = local.services_path

  env_vars = {
    GCP_PROJECT_ID = var.project_id
  }

  secret_env_vars = {
    GOOGLE_MAPS_API_KEY = {
      secret_id = "google-maps-api-key"
      version   = "latest"
    }
  }

  secret_refs = ["google-maps-api-key"]

  depends_on = [module.foundation, module.secrets]
}

module "forecast_collector" {
  source                = "../modules/cloud-run-job"
  project_id            = var.project_id
  region                = var.region
  name                  = "forecast-collector"
  sa_display_name       = "Service Account for Forecast Collector Job"
  schedule              = "0 */6 * * *"
  scheduler_description = "Triggers the forecast collector job every 6 hours"
  artifact_registry_url = module.foundation.artifact_registry_url
  services_path         = local.services_path

  env_vars = {
    GCP_PROJECT_ID         = var.project_id
    FORECAST_HORIZON_HOURS = "72"
    PRESSURE_DROP_MB       = "5"
    PRESSURE_SEVERE_MB     = "10"
    PRESSURE_WINDOW_HOURS  = "3"
  }

  secret_env_vars = {
    GOOGLE_MAPS_API_KEY = {
      secret_id = "google-maps-api-key"
      version   = "latest"
    }
    NOTIFY_SMTP_PASSWORD = {
      secret_id = "notify-smtp-password"
      version   = "latest"
    }
  }

  secret_refs = ["google-maps-api-key", "notify-smtp-password"]

  depends_on = [module.foundation, module.secrets]
}

# --- Notifier (Cloud Run Job) ---

module "notifier" {
  source                = "../modules/cloud-run-job"
  project_id            = var.project_id
  region                = var.region
  name                  = "notifier"
  sa_display_name       = "Service Account for Notifier Job"
  schedule              = "5 * * * *"
  scheduler_description = "Triggers the notifier job hourly, offset past the weather collector"
  artifact_registry_url = module.foundation.artifact_registry_url
  services_path         = local.services_path

  env_vars = {
    GCP_PROJECT_ID = var.project_id
  }

  depends_on = [module.foundation]
}

# --- Providers (Internal gRPC Services) ---

module "weather_provider" {
  source                = "../modules/cloud-run-provider"
  project_id            = var.project_id
  region                = var.region
  name                  = "weather-provider"
  sa_display_name       = "Service Account for Weather provider service"
  port                  = 50051
  artifact_registry_url = module.foundation.artifact_registry_url
  services_path         = local.services_path

  env_vars = {
    GCP_PROJECT_ID = var.project_id
  }

  depends_on = [module.foundation]
}

module "pollen_provider" {
  source                = "../modules/cloud-run-provider"
  project_id            = var.project_id
  region                = var.region
  name                  = "pollen-provider"
  sa_display_name       = "Service Account for Pollen Provider service"
  port                  = 50052
  artifact_registry_url = module.foundation.artifact_registry_url
  services_path         = local.services_path

  env_vars = {
    GCP_PROJECT_ID = var.project_id
  }

  depends_on = [module.foundation]
}

# --- Aggregator (Public BFF) ---

module "dashboard_api" {
  source                = "../modules/cloud-run-aggregator"
  project_id            = var.project_id
  region                = var.region
  name                  = "dashboard-api"
  sa_display_name       = "Service Account for Dashboard API service"
  port                  = 8080
  artifact_registry_url = module.foundation.artifact_registry_url
  services_path         = local.services_path

  env_vars = {
    WEATHER_PROVIDER_ADDR = "${trimprefix(module.weather_provider.service_uri, "https://")}:443"
    POLLEN_PROVIDER_ADDR  = "${trimprefix(module.pollen_provider.service_uri, "https://")}:443"
  }

  invoker_targets = [
    {
      name     = module.weather_provider.service_name
      location = module.weather_provider.service_location
    },
    {
      name     = module.pollen_provider.service_name
      location = module.pollen_provider.service_location
    },
  ]

  depends_on = [module.foundation]
}

# --- CI/CD Identity ---

module "github_oidc" {
  source            = "../modules/github-oidc"
  project_id        = var.project_id
  github_repository = var.github_repository

  # Every new cloud-run-* module MUST be listed here or its first CD deploy
  # fails: github-actions-sa needs iam.serviceAccounts.actAs on the job's SA
  # to run `gcloud run jobs update`. Terraform applies cleanly without it, so
  # the omission surfaces as "my code changes aren't taking effect".
  service_account_ids = [
    module.weather_collector.service_account_id,
    module.pollen_collector.service_account_id,
    module.forecast_collector.service_account_id,
    module.notifier.service_account_id,
    module.weather_provider.service_account_id,
    module.pollen_provider.service_account_id,
    module.dashboard_api.service_account_id,
  ]
}

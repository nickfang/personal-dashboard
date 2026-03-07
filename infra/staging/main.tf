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
  secret_ids = ["google-maps-api-key"]

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

# --- Providers (Internal gRPC Services) ---

module "weather_provider" {
  source                = "../modules/cloud-run-provider"
  project_id            = var.project_id
  region                = var.region
  name                  = "weather-provider"
  sa_display_name       = "Service Account for Weather Provider Service"
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
  sa_display_name       = "Service Account for Pollen Provider Service"
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
  sa_display_name       = "Service Account for Dashboard API Service"
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

  service_account_ids = [
    module.weather_collector.service_account_id,
    module.pollen_collector.service_account_id,
    module.weather_provider.service_account_id,
    module.pollen_provider.service_account_id,
    module.dashboard_api.service_account_id,
  ]
}

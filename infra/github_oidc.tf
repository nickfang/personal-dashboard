# --- GitHub Actions Identity (OIDC) ---

# Workload Identity Pool for GitHub Actions
resource "google_iam_workload_identity_pool" "github_pool" {
  workload_identity_pool_id = "github-actions-pool"
  display_name              = "GitHub Actions Pool"
  description               = "Identity pool for GitHub Actions deployments"
}

# Workload Identity Provider for GitHub Actions
resource "google_iam_workload_identity_pool_provider" "github_provider" {
  workload_identity_pool_id          = google_iam_workload_identity_pool.github_pool.workload_identity_pool_id
  workload_identity_pool_provider_id = "github-provider"
  display_name                       = "GitHub Provider"
  description                        = "OIDC Identity Provider for GitHub Actions"

  attribute_mapping = {
    "google.subject"       = "assertion.sub"
    "attribute.actor"      = "assertion.actor"
    "attribute.repository" = "assertion.repository"
  }

  # Strictly restrict access to the specified repository
  attribute_condition = "assertion.repository == \"${var.github_repository}\""

  oidc {
    issuer_uri = "https://token.actions.githubusercontent.com"
  }
}

# Service Account specifically for GitHub Actions runners
resource "google_service_account" "github_sa" {
  account_id   = "github-actions-sa"
  display_name = "Service Account for GitHub Actions"
}

# Allow GitHub Actions to impersonate the service account via Workload Identity
resource "google_service_account_iam_member" "github_sa_impersonation" {
  service_account_id = google_service_account.github_sa.name
  role               = "roles/iam.workloadIdentityUser"
  member             = "principalSet://iam.googleapis.com/${google_iam_workload_identity_pool.github_pool.name}/attribute.repository/${var.github_repository}"
}

# --- Permissions for GitHub Actions ---

# 1. Artifact Registry Writer (To build and push Docker images)
resource "google_project_iam_member" "github_sa_artifact_writer" {
  project = var.project_id
  role    = "roles/artifactregistry.writer"
  member  = "serviceAccount:${google_service_account.github_sa.email}"
}

# 2. Cloud Run Developer (To deploy/update the Cloud Run Job)
resource "google_project_iam_member" "github_sa_cloud_run_developer" {
  project = var.project_id
  role    = "roles/run.developer"
  member  = "serviceAccount:${google_service_account.github_sa.email}"
}

# 3. Service Account User (To run the job AS the weather-collector-sa)
resource "google_service_account_iam_member" "github_sa_act_as_weather_collector_sa" {
  service_account_id = google_service_account.weather_collector_sa.name
  role               = "roles/iam.serviceAccountUser"
  member             = "serviceAccount:${google_service_account.github_sa.email}"
}

# 4. Service Account User (To run the service AS the weather-provider-sa)
resource "google_service_account_iam_member" "github_sa_act_as_weather_provider_sa" {
  service_account_id = google_service_account.weather_provider_sa.name
  role               = "roles/iam.serviceAccountUser"
  member             = "serviceAccount:${google_service_account.github_sa.email}"
}

# 5. Service Account User (To run the job AS the pollen-collector-sa)
resource "google_service_account_iam_member" "github_sa_act_as_pollen_collector_sa" {
  service_account_id = google_service_account.pollen_collector_sa.name
  role               = "roles/iam.serviceAccountUser"
  member             = "serviceAccount:${google_service_account.github_sa.email}"
}

# 6. Service Account User (To run the service AS the pollen-provider-sa)
resource "google_service_account_iam_member" "github_sa_act_as_pollen_provider_sa" {
  service_account_id = google_service_account.pollen_provider_sa.name
  role               = "roles/iam.serviceAccountUser"
  member             = "serviceAccount:${google_service_account.github_sa.email}"
}

# 7. Service Account User (To run the service AS the dashboard-api-sa)
resource "google_service_account_iam_member" "github_sa_act_as_dashboard_api_sa" {
  service_account_id = google_service_account.dashboard_api_sa.name
  role               = "roles/iam.serviceAccountUser"
  member             = "serviceAccount:${google_service_account.github_sa.email}"
}

# Service Account for the Dashboard API
resource "google_service_account" "dashboard_api_sa" {
  account_id   = "dashboard-api-sa"
  display_name = "Service Account for Dashboard API service"
}

# Allow Dashboard API to call Weather Provider
resource "google_cloud_run_v2_service_iam_member" "weather_provider_invoker" {
  name     = google_cloud_run_v2_service.weather_provider.name
  location = google_cloud_run_v2_service.weather_provider.location
  role     = "roles/run.invoker"
  member   = "serviceAccount:${google_service_account.dashboard_api_sa.email}"
}

# Bootstrap Docker Image for Dashboard API
# This resource serves as a "Bootstrap" step. It ensures an image exists so Terraform
# can successfully create the Cloud Run Service initially (Disaster Recovery).
# For day-to-day development, GitHub Actions will build and deploy new images.
resource "null_resource" "dashboard_api_bootstrap" {
  provisioner "local-exec" {
    command = <<EOT
      gcloud builds submit ../services/dashboard-api \
        --tag ${var.region}-docker.pkg.dev/${var.project_id}/${google_artifact_registry_repository.repo.repository_id}/dashboard-api:latest \
        --project ${var.project_id}
    EOT
  }

  depends_on = [google_project_service.cloudbuild, google_artifact_registry_repository.repo]
}

# Define the Cloud Run Service
resource "google_cloud_run_v2_service" "dashboard_api" {
  name     = "dashboard-api-service"
  location = var.region
  ingress  = "INGRESS_TRAFFIC_ALL" # Public BFF

  template {
    service_account = google_service_account.dashboard_api_sa.email
    containers {
      image = "${var.region}-docker.pkg.dev/${var.project_id}/${google_artifact_registry_repository.repo.repository_id}/dashboard-api:latest"
      ports {
        container_port = 8080
      }
      env {
        name  = "WEATHER_PROVIDER_ADDR"
        value = "${trimprefix(google_cloud_run_v2_service.weather_provider.uri, "https://")}:443"
      }
    }
  }

  lifecycle {
    ignore_changes = [
      template[0].containers[0].image,
      client,
      client_version,
      template[0].labels,
      template[0].annotations
    ]
  }

  depends_on = [google_project_service.run, null_resource.dashboard_api_bootstrap]
}

# Make the Dashboard API publicly accessible
resource "google_cloud_run_v2_service_iam_member" "public_invoker" {
  name     = google_cloud_run_v2_service.dashboard_api.name
  location = google_cloud_run_v2_service.dashboard_api.location
  role     = "roles/run.invoker"
  member   = "allUsers"
}

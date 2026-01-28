# Enable Cloud Run API
resource "google_project_service" "run" {
  service            = "run.googleapis.com"
  disable_on_destroy = false
}

# Enable Artifact Registry API
resource "google_project_service" "artifactregistry" {
  service            = "artifactregistry.googleapis.com"
  disable_on_destroy = false
}

# Enable Cloud Scheduler API
resource "google_project_service" "scheduler" {
  service            = "cloudscheduler.googleapis.com"
  disable_on_destroy = false
}

# Enable Secret Manager API
resource "google_project_service" "secretmanager" {
  service            = "secretmanager.googleapis.com"
  disable_on_destroy = false
}

# Enable Cloud Build API
resource "google_project_service" "cloudbuild" {
  service            = "cloudbuild.googleapis.com"
  disable_on_destroy = false
}

# Create Artifact Registry Repository for Docker images
resource "google_artifact_registry_repository" "repo" {
  provider      = google-beta
  project       = var.project_id
  location      = var.region
  repository_id = "personal-dashboard"
  description   = "Docker repository for Personal Dashboard services"
  format        = "DOCKER"

  depends_on = [google_project_service.artifactregistry]
}

# Service Account for the Cloud Run Job
resource "google_service_account" "weather_collector_sa" {
  account_id   = "weather-collector-sa"
  display_name = "Service Account for Weather Collector Job"
}

# Grant permissions to write to Firestore
resource "google_project_iam_member" "firestore_writer" {
  project = var.project_id
  role    = "roles/datastore.user"
  member  = "serviceAccount:${google_service_account.weather_collector_sa.email}"
}

# Grant permission to invoke Cloud Run jobs (Self-invocation for Scheduler)
resource "google_project_iam_member" "cloud_run_invoker" {
  project = var.project_id
  role    = "roles/run.invoker"
  member  = "serviceAccount:${google_service_account.weather_collector_sa.email}"
}

# Create the Secret in Secret Manager
resource "google_secret_manager_secret" "google_maps_key" {
  secret_id = "google-maps-api-key"
  replication {
    auto {}
  }
  depends_on = [google_project_service.secretmanager]
}

# Add the API Key Version to the Secret
resource "google_secret_manager_secret_version" "google_maps_key_version" {
  secret      = google_secret_manager_secret.google_maps_key.id
  secret_data = var.google_maps_api_key
}

# Grant the Service Account access to the Secret
resource "google_secret_manager_secret_iam_member" "secret_access" {
  secret_id = google_secret_manager_secret.google_maps_key.id
  role      = "roles/secretmanager.secretAccessor"
  member    = "serviceAccount:${google_service_account.weather_collector_sa.email}"
}

# Build and Push Docker Image using Cloud Build
# This resource serves as a "Bootstrap" step. It ensures an image exists so Terraform
# can successfully create the Cloud Run Job initially (Disaster Recovery).
# For day-to-day development, GitHub Actions will build and deploy new images.
resource "null_resource" "cloud_build" {
  provisioner "local-exec" {
    command = <<EOT
      gcloud builds submit ../services/weather-collector \
        --tag ${var.region}-docker.pkg.dev/${var.project_id}/${google_artifact_registry_repository.repo.repository_id}/weather-collector:latest \
        --project ${var.project_id}
    EOT
  }

  depends_on = [google_project_service.cloudbuild, google_artifact_registry_repository.repo]
}

# Define the Cloud Run Job
resource "google_cloud_run_v2_job" "weather_collector" {
  name     = "weather-collector-job"
  location = var.region

  template {
    template {
      service_account = google_service_account.weather_collector_sa.email
      containers {
        image = "${var.region}-docker.pkg.dev/${var.project_id}/${google_artifact_registry_repository.repo.repository_id}/weather-collector:latest"
        env {
          name = "GCP_PROJECT_ID"
          value = var.project_id
        }
        env {
          name = "GOOGLE_MAPS_API_KEY"
          value_source {
            secret_key_ref {
              secret  = google_secret_manager_secret.google_maps_key.secret_id
              version = "latest"
            }
          }
        }
      }
    }
  }

  # Lifecycle Ignore: "Bootstrap + CD" Pattern
  # We instruct Terraform to ignore changes to the 'image' field.
  # This allows GitHub Actions (CD) to deploy new versions (v2, v3...) without Terraform
  # trying to revert the job back to the 'bootstrap' image version (latest) on the next apply.
  lifecycle {
    ignore_changes = [
      template[0].template[0].containers[0].image,
      client,
      client_version,
      template[0].labels,
      template[0].annotations
    ]
  }

  depends_on = [google_project_service.run, null_resource.cloud_build, google_secret_manager_secret_version.google_maps_key_version]
}

# Create a Cloud Scheduler job to trigger the Cloud Run job hourly
resource "google_cloud_scheduler_job" "weather_cron" {
  name             = "trigger-weather-collector"
  description      = "Triggers the weather collector job every hour"
  schedule         = "0 * * * *" # Hourly at minute 0
  time_zone        = "America/Chicago"
  attempt_deadline = "320s"

  http_target {
    http_method = "POST"
    uri         = "https://${var.region}-run.googleapis.com/apis/run.googleapis.com/v1/namespaces/${var.project_id}/jobs/${google_cloud_run_v2_job.weather_collector.name}:run"

    oauth_token {
      service_account_email = google_service_account.weather_collector_sa.email
    }
  }

  depends_on = [google_project_service.scheduler]
}

# Service Account for the Pollen Collector Job
resource "google_service_account" "pollen_collector_sa" {
  account_id   = "pollen-collector-sa"
  display_name = "Service Account for Pollen Collector Job"
}

# Grant permissions to write to Firestore
resource "google_project_iam_member" "pollen_firestore_writer" {
  project = var.project_id
  role    = "roles/datastore.user"
  member  = "serviceAccount:${google_service_account.pollen_collector_sa.email}"
}

# Grant permission to invoke Cloud Run jobs (Self-invocation for Scheduler)
resource "google_project_iam_member" "pollen_collector_invoker" {
  project = var.project_id
  role    = "roles/run.invoker"
  member  = "serviceAccount:${google_service_account.pollen_collector_sa.email}"
}

# Grant the Pollen Collector access to the existing Google Maps API Key secret.
# The secret itself is defined in weather_collector.tf — we only add an IAM binding here.
resource "google_secret_manager_secret_iam_member" "pollen_secret_access" {
  secret_id = google_secret_manager_secret.google_maps_key.id
  role      = "roles/secretmanager.secretAccessor"
  member    = "serviceAccount:${google_service_account.pollen_collector_sa.email}"
}

# Build and Push Docker Image using Cloud Build
# This resource serves as a "Bootstrap" step. It ensures an image exists so Terraform
# can successfully create the Cloud Run Job initially (Disaster Recovery).
# For day-to-day development, GitHub Actions will build and deploy new images.
resource "null_resource" "pollen_collector_bootstrap" {
  provisioner "local-exec" {
    command = <<EOT
      gcloud builds submit ../services/pollen-collector \
        --tag ${var.region}-docker.pkg.dev/${var.project_id}/${google_artifact_registry_repository.repo.repository_id}/pollen-collector:latest \
        --project ${var.project_id}
    EOT
  }

  depends_on = [google_project_service.cloudbuild, google_artifact_registry_repository.repo]
}

# Define the Cloud Run Job
resource "google_cloud_run_v2_job" "pollen_collector" {
  name     = "pollen-collector-job"
  location = var.region

  template {
    template {
      service_account = google_service_account.pollen_collector_sa.email
      containers {
        image = "${var.region}-docker.pkg.dev/${var.project_id}/${google_artifact_registry_repository.repo.repository_id}/pollen-collector:latest"
        env {
          name  = "GCP_PROJECT_ID"
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

  depends_on = [google_project_service.run, null_resource.pollen_collector_bootstrap, google_secret_manager_secret_version.google_maps_key_version]
}

# Create a Cloud Scheduler job to trigger the Pollen Collector twice daily
resource "google_cloud_scheduler_job" "pollen_cron" {
  name             = "trigger-pollen-collector"
  description      = "Triggers the pollen collector job twice daily"
  schedule         = "0 6,14 * * *" # 6:00 AM and 2:00 PM Central
  time_zone        = "America/Chicago"
  attempt_deadline = "320s"

  http_target {
    http_method = "POST"
    uri         = "https://${var.region}-run.googleapis.com/apis/run.googleapis.com/v1/namespaces/${var.project_id}/jobs/${google_cloud_run_v2_job.pollen_collector.name}:run"

    oauth_token {
      service_account_email = google_service_account.pollen_collector_sa.email
    }
  }

  depends_on = [google_project_service.scheduler]
}

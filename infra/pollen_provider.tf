# Service Account for the Pollen Provider Service
resource "google_service_account" "pollen_provider_sa" {
  account_id   = "pollen-provider-sa"
  display_name = "Service Account for Pollen Provider service"
}

# Grant permissions to read from Firestore
resource "google_project_iam_member" "pollen_firestore_reader" {
  project = var.project_id
  role    = "roles/datastore.viewer"
  member  = "serviceAccount:${google_service_account.pollen_provider_sa.email}"
}

# Grant permission to invoke Cloud Run services
resource "google_project_iam_member" "pollen_provider_invoker" {
  project = var.project_id
  role    = "roles/run.invoker"
  member  = "serviceAccount:${google_service_account.pollen_provider_sa.email}"
}

# Build and Push Docker Image using Cloud Build
# This resource serves as a "Bootstrap" step. It ensures an image exists so Terraform
# can successfully create the Cloud Run Service initially (Disaster Recovery).
# For day-to-day development, GitHub Actions will build and deploy new images.
resource "null_resource" "pollen_provider_bootstrap" {
  provisioner "local-exec" {
    command = <<EOT
      gcloud builds submit ../services/pollen-provider \
        --tag ${var.region}-docker.pkg.dev/${var.project_id}/${google_artifact_registry_repository.repo.repository_id}/pollen-provider:latest \
        --project ${var.project_id}
    EOT
  }

  depends_on = [google_project_service.cloudbuild, google_artifact_registry_repository.repo]
}

# Define the Cloud Run Service
resource "google_cloud_run_v2_service" "pollen_provider" {
  name     = "pollen-provider-service"
  location = var.region

  template {
    service_account = google_service_account.pollen_provider_sa.email
    containers {
      image = "${var.region}-docker.pkg.dev/${var.project_id}/${google_artifact_registry_repository.repo.repository_id}/pollen-provider:latest"
      ports {
        container_port = 50052
        name           = "h2c" # enable HTTP/2
      }
      env {
        name  = "GCP_PROJECT_ID"
        value = var.project_id
      }
    }
  }

  # Lifecycle Ignore: "Bootstrap + CD" Pattern
  # We instruct Terraform to ignore changes to the 'image' field.
  # This allows GitHub Actions (CD) to deploy new versions (v2, v3...) without Terraform
  # trying to revert the service back to the 'bootstrap' image version (latest) on the next apply.
  lifecycle {
    ignore_changes = [
      template[0].containers[0].image,
      client,
      client_version,
      template[0].labels,
      template[0].annotations
    ]
  }

  depends_on = [google_project_service.run, null_resource.pollen_provider_bootstrap]
}

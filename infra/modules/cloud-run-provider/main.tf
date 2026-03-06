resource "google_service_account" "service_account" {
  account_id   = "${var.name}-sa"
  display_name = var.sa_display_name
}

resource "google_project_iam_member" "firestore_reader" {
  project = var.project_id
  role    = "roles/datastore.viewer"
  member  = "serviceAccount:${google_service_account.service_account.email}"
}

resource "null_resource" "bootstrap" {
  provisioner "local-exec" {
    command = <<-EOT
      gcloud builds submit ${var.services_path} \
        --config ${var.services_path}/${var.name}/cloudbuild.yaml \
        --substitutions=_IMAGE_TAG=${var.artifact_registry_url}/${var.name}:latest \
        --project ${var.project_id}
    EOT
  }
}

resource "google_cloud_run_v2_service" "service" {
  name     = "${var.name}-service"
  location = var.region

  template {
    service_account = google_service_account.service_account.email

    containers {
      image = "${var.artifact_registry_url}/${var.name}:latest"

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

  depends_on = [null_resource.bootstrap]
}

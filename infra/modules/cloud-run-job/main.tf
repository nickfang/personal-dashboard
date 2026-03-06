resource "google_service_account" "sa" {
  account_id   = "${var.name}-sa"
  display_name = var.sa_display_name
}

resource "google_project_iam_member" "firestore_writer" {
  project = var.project_id
  role    = "roles/datastore.user"
  member  = "serviceAccount:${google_service_account.sa.email}"
}

resource "google_project_iam_member" "run_invoker" {
  project = var.project_id
  role    = "roles/run.invoker"
  member  = "serviceAccount:${google_service_account.sa.email}"
}

resource "google_secret_manager_secret_iam_member" "secret_access" {
  for_each  = toset(var.secret_refs)
  secret_id = each.value
  role      = "roles/secretmanager.secretAccessor"
  member    = "serviceAccount:${google_service_account.sa.email}"
}

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

resource "google_cloud_run_v2_job" "job" {
  name     = "${var.name}-job"
  location = var.region

  template {
    template {
      service_account = google_service_account.sa.email
      containers {
        image = "${var.artifact_registry_url}/${var.name}:latest"

        dynamic "env" {
          for_each = var.env_vars
          content {
            name  = env.key
            value = env.value
          }
        }

        dynamic "env" {
          for_each = var.secret_env_vars
          content {
            name = env.key
            value_source {
              secret_key_ref {
                secret  = env.value.secret_id
                version = env.value.version
              }
            }
          }
        }
      }
    }
  }

  lifecycle {
    ignore_changes = [
      template[0].template[0].containers[0].image,
      client,
      client_version,
      template[0].labels,
      template[0].annotations
    ]
  }

  depends_on = [null_resource.bootstrap]
}

resource "google_cloud_scheduler_job" "trigger" {
  name             = "trigger-${var.name}"
  description      = var.scheduler_description
  schedule         = var.schedule
  time_zone        = "America/Chicago"
  attempt_deadline = "320s"

  http_target {
    http_method = "POST"
    uri         = "https://${var.region}-run.googleapis.com/apis/run.googleapis.com/v1/namespaces/${var.project_id}/jobs/${google_cloud_run_v2_job.job.name}:run"

    oauth_token {
      service_account_email = google_service_account.sa.email
    }
  }
}

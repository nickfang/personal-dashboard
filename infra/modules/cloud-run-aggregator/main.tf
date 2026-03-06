resource "google_service_account" "sa" {
  account_id   = "${var.name}-sa"
  display_name = var.sa_display_name
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
  ingress  = "INGRESS_TRAFFIC_ALL"

  template {
    service_account = google_service_account.sa.email

    containers {
      image = "${var.artifact_registry_url}/${var.name}:latest"

      ports {
        container_port = var.port
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

resource "google_cloud_run_v2_service_iam_member" "public_invoker" {
  name     = google_cloud_run_v2_service.service.name
  location = google_cloud_run_v2_service.service.location
  role     = "roles/run.invoker"
  member   = "allUsers"
}

resource "google_cloud_run_v2_service_iam_member" "cross_service_invoker" {
  for_each = { for t in var.invoker_targets : t.name => t }

  name     = each.value.name
  location = each.value.location
  role     = "roles/run.invoker"
  member   = "serviceAccount:${google_service_account.sa.email}"
}

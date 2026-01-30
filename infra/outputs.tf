output "github_actions_sa_email" {
  description = "The email of the service account for GitHub Actions"
  value       = google_service_account.github_sa.email
}

output "workload_identity_provider_name" {
  description = "The full identifier of the Workload Identity Provider"
  value       = google_iam_workload_identity_pool_provider.github_provider.name
}

output "artifact_registry_repo" {
  description = "The Artifact Registry repository URL"
  value       = "${var.region}-docker.pkg.dev/${var.project_id}/${google_artifact_registry_repository.repo.repository_id}"
}

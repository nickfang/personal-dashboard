output "artifact_registry_repo_id" {
  value = google_artifact_registry_repository.repo.repository_id
}

output "artifact_registry_url" {
  value = "${var.region}-docker.pkg.dev/${var.project_id}/${google_artifact_registry_repository.repo.repository_id}"
}

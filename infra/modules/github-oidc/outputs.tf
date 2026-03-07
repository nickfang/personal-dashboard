output "sa_email" {
  description = "The email of the GitHub Actions service account"
  value       = google_service_account.github_sa.email
}

output "workload_identity_provider_name" {
  description = "The full name of the Workload Identity Provider"
  value       = google_iam_workload_identity_pool_provider.github_provider.name
}

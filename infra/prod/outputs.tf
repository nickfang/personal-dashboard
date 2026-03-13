output "github_actions_sa_email" {
  description = "The email of the GitHub Actions service account"
  value       = module.github_oidc.sa_email
}

output "workload_identity_provider_name" {
  description = "The full identifier of the Workload Identity Provider"
  value       = module.github_oidc.workload_identity_provider_name
}

output "artifact_registry_url" {
  description = "The Artifact Registry repository URL"
  value       = module.foundation.artifact_registry_url
}

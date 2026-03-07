output "secret_ids_map" {
  description = "Map of secret name to full secret ID"
  value       = { for k, v in google_secret_manager_secret.secrets : k => v.id }
}

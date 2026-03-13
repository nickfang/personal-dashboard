output "service_account_email" {
  description = "Email of the service account"
  value       = google_service_account.service_account.email
}

output "service_account_id" {
  description = "Account ID of the service account"
  value       = google_service_account.service_account.account_id
}

output "service_name" {
  description = "Name of the Cloud Run service"
  value       = google_cloud_run_v2_service.service.name
}

output "service_location" {
  description = "Location of the Cloud Run service"
  value       = google_cloud_run_v2_service.service.location
}

output "service_uri" {
  description = "URI of the Cloud Run service"
  value       = google_cloud_run_v2_service.service.uri
}

output "service_account_email" {
  value = google_service_account.sa.email
}

output "service_account_id" {
  value = google_service_account.sa.account_id
}

output "service_uri" {
  value = google_cloud_run_v2_service.service.uri
}

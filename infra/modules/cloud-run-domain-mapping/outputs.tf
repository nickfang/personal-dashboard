output "dns_records" {
  description = "DNS records to configure at the domain registrar"
  value = [
    for record in google_cloud_run_domain_mapping.default.status[0].resource_records : {
      type  = record.type
      name  = record.name
      rrdata = record.rrdata
    }
  ]
}

output "mapped_domain" {
  description = "The custom domain that was mapped"
  value       = google_cloud_run_domain_mapping.default.name
}

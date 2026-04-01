variable "domain" {
  description = "The custom domain to map, e.g. api-staging.example.com"
  type        = string
}

variable "service_name" {
  description = "Name of the Cloud Run service to map the domain to"
  type        = string
}

variable "region" {
  description = "GCP region where the Cloud Run service is deployed"
  type        = string
}

variable "project_id" {
  description = "GCP project ID"
  type        = string
}

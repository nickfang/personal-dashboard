variable "project_id" {
  description = "The ID of the Google Cloud Project"
  type        = string
}

variable "region" {
  description = "The GCP region to deploy resources in"
  type        = string
  default     = "us-central1"
}

variable "google_maps_api_key" {
  description = "The Google Maps API Key for the Weather Collector"
  type        = string
  sensitive   = true
}

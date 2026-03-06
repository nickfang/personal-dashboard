variable "project_id" {
  description = "GCP project ID"
  type        = string
}

variable "region" {
  description = "GCP region"
  type        = string
}

variable "name" {
  description = "Service name, e.g. weather-provider"
  type        = string
}

variable "port" {
  description = "Container port for the service"
  type        = number
}

variable "env_vars" {
  description = "Environment variables for the container"
  type        = map(string)
  default     = {}
}

variable "artifact_registry_url" {
  description = "Artifact Registry URL for container images"
  type        = string
}

variable "services_path" {
  description = "Path to the services directory for Cloud Build"
  type        = string
}

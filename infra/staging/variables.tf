variable "project_id" {
  description = "The GCP project ID"
  type        = string
}

variable "region" {
  description = "The GCP region to deploy resources in"
  type        = string
  default     = "us-central1"
}

variable "github_repository" {
  description = "The GitHub repository (owner/repo) allowed to deploy via OIDC"
  type        = string
  default     = "nickfang/personal-dashboard"
}

variable "project_id" {
  description = "GCP project ID"
  type        = string
}

variable "region" {
  description = "GCP region for Firestore databases"
  type        = string
}

variable "database_ids" {
  description = "List of Firestore database names to create"
  type        = list(string)
}

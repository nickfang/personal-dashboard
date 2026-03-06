variable "project_id" {
  description = "GCP project ID"
  type        = string
}

variable "secret_ids" {
  description = "List of secret names to create"
  type        = list(string)
}

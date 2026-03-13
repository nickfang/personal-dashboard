variable "project_id" {
  description = "The GCP project ID"
  type        = string
}

variable "github_repository" {
  description = "The GitHub repository (e.g., \"nickfang/personal-dashboard\")"
  type        = string
}

variable "service_account_ids" {
  description = "List of service account account_ids to grant act_as permissions to the GitHub SA"
  type        = list(string)
}

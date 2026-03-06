variable "project_id" {
  type = string
}

variable "region" {
  type = string
}

variable "name" {
  type = string
}

variable "sa_display_name" {
  type = string
}

variable "scheduler_description" {
  type    = string
  default = ""
}

variable "schedule" {
  type = string
}

variable "env_vars" {
  type    = map(string)
  default = {}
}

variable "secret_env_vars" {
  type = map(object({
    secret_id = string
    version   = string
  }))
  default = {}
}

variable "secret_refs" {
  type    = list(string)
  default = []
}

variable "artifact_registry_url" {
  type = string
}

variable "services_path" {
  type = string
}

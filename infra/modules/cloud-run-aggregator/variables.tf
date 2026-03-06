variable "project_id" {
  type = string
}

variable "region" {
  type = string
}

variable "name" {
  type = string
}

variable "port" {
  type = number
}

variable "env_vars" {
  type    = map(string)
  default = {}
}

variable "artifact_registry_url" {
  type = string
}

variable "services_path" {
  type = string
}

variable "invoker_targets" {
  type = list(object({
    name     = string
    location = string
  }))
  default = []
}

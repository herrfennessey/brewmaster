variable "project_id" {
  description = "The GCP project ID"
  type        = string
  default     = "the-coffee-brewmaster"
}

variable "region" {
  description = "The GCP region for all resources"
  type        = string
  default     = "europe-west3"
}

variable "github_repository" {
  description = "The GitHub repository in 'owner/repo' format"
  type        = string
  default     = "herrfennessey/brewmaster"
}

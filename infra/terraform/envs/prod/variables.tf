variable "project_id" {
  description = "GCP project ID for the prod environment. Create this project + link billing manually before the first apply (Claude/Terraform never touch billing)."
  type        = string
  default     = "verani-webstore-prod"
}

variable "region" {
  description = "GCP region for all regional resources."
  type        = string
  default     = "europe-west1"
}

variable "env" {
  description = "Environment name, used in resource naming."
  type        = string
  default     = "prod"
}

variable "domain_root" {
  description = "Root domain. In prod the storefront is served on the apex."
  type        = string
  default     = "verani.bg"
}

variable "api_subdomain" {
  description = "Subdomain the API is served on (prefixed to domain_root)."
  type        = string
  default     = "api"
}

variable "google_client_id" {
  description = "Google OAuth client ID used for sign-in in prod. Reuses the existing client by default; give prod its own client and add https://verani.bg as an authorized JS origin."
  type        = string
  default     = "673528779465-pajifaekv8l1odrbd8mpbglo351d7r15.apps.googleusercontent.com"
}

variable "github_repo" {
  description = "GitHub repo allowed to assume the deploy service account via Workload Identity Federation, in owner/name form."
  type        = string
  default     = "adotomov/fashion-store"
}

variable "github_deploy_branch" {
  description = "Branch allowed to deploy via the GitHub Actions workflow."
  type        = string
  default     = "main"
}

variable "db_tier" {
  description = "Cloud SQL machine tier. Sized up from dev for production load."
  type        = string
  default     = "db-custom-2-7680"
}

variable "db_name" {
  description = "Application database name."
  type        = string
  default     = "fashion_store"
}

variable "db_user" {
  description = "Application database user."
  type        = string
  default     = "fashion_store"
}

variable "placeholder_image" {
  description = "Image used to create Cloud Run services before the first real deploy. CI overwrites the running image afterwards; Terraform ignores drift on it."
  type        = string
  default     = "us-docker.pkg.dev/cloudrun/container/hello"
}

variable "speedy_mode" {
  description = "Speedy logistics client mode. Must be \"real\" in prod — real shipments and tracking."
  type        = string
  default     = "real"

  validation {
    condition     = contains(["fake", "real"], var.speedy_mode)
    error_message = "speedy_mode must be either \"fake\" or \"real\"."
  }
}

variable "fulfillment_poll_interval" {
  description = "How often the shipment-tracking poller runs (Go duration string). With min_instance_count = 1 in prod the poller stays warm."
  type        = string
  default     = "15m"
}

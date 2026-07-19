variable "project_id" {
  description = "GCP project ID for the dev environment."
  type        = string
  default     = "project-538051b7-1abf-4eab-94c" # verani-webstore-dev
}

variable "region" {
  description = "GCP region for all regional resources."
  type        = string
  default     = "europe-west1"
}

variable "env" {
  description = "Environment name, used in resource naming."
  type        = string
  default     = "dev"
}

variable "domain_root" {
  description = "Root domain managed in Cloud DNS."
  type        = string
  default     = "verani.bg"
}

variable "web_subdomain" {
  description = "Subdomain the storefront frontend is served on."
  type        = string
  default     = "dev"
}

variable "api_subdomain" {
  description = "Subdomain the API is served on."
  type        = string
  default     = "api.dev"
}

variable "google_client_id" {
  description = "Existing Google OAuth client ID used for sign-in (already provisioned in this project)."
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

variable "create_domain_mappings" {
  description = "Whether to create Cloud Run domain mappings for dev.verani.bg / api.dev.verani.bg. Requires verani.bg to already be a verified domain on this Google account (Search Console) AND the verani.bg DNS zone (now owned by the prod env) to be resolving. If an apply fails on the mappings, set this back to false, finish domain verification, then re-apply."
  type        = bool
  default     = true
}

variable "db_tier" {
  description = "Cloud SQL machine tier."
  type        = string
  default     = "db-custom-1-3840"
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
  description = "Speedy logistics client mode. \"fake\" uses a local simulated client (no real Speedy API calls, no real shipments) — the default for this dev env. Set to \"real\" once live Speedy credentials are configured for the provider."
  type        = string
  default     = "fake"

  validation {
    condition     = contains(["fake", "real"], var.speedy_mode)
    error_message = "speedy_mode must be either \"fake\" or \"real\"."
  }
}

variable "fulfillment_poll_interval" {
  description = "How often the shipment-tracking poller runs (Go duration string, e.g. \"15m\", \"30s\"). Note: with Cloud Run min instances = 0 the in-process poller only ticks while an instance is warm, so tracking auto-progression is best-effort on Cloud Run."
  type        = string
  default     = "15m"
}

variable "revolut_mode" {
  description = "Revolut Merchant environment: \"sandbox\" or \"prod\". Selects the Merchant API base URL and the checkout widget mode. Dev uses \"sandbox\"."
  type        = string
  default     = "sandbox"

  validation {
    condition     = contains(["sandbox", "prod"], var.revolut_mode)
    error_message = "revolut_mode must be either \"sandbox\" or \"prod\"."
  }
}

variable "revolut_api_version" {
  description = "Pinned Revolut-Api-Version request header (date form, e.g. \"2024-09-01\"). Bump when the code is validated against a newer Merchant API version."
  type        = string
  default     = "2024-09-01"
}

variable "revolut_enabled" {
  description = "Inject the Revolut API key + webhook secret into the API service. Keep false until the secret VALUES are populated out-of-band; flip to true to activate card payments. Defaults true on dev because the fs-dev-revolut-* secrets are already populated — this keeps the flag sticky so a bare `terraform apply` can't silently revert to the mock gateway (mock tokens make the real Revolut widget show \"Something went wrong\")."
  type        = bool
  default     = true
}

variable "observability_enabled" {
  description = "Export OTel traces to Cloud Trace and custom metrics to Cloud Monitoring. Structured logging + trace correlation are always on; this only gates the OTel exporters. Dev is the first env to enable, so this defaults to true."
  type        = bool
  default     = true
}

variable "otel_trace_sample_ratio" {
  description = "Parent-based trace sampling ratio (0.0-1.0) applied to root spans."
  type        = string
  default     = "0.1"
}

variable "alert_email" {
  description = "Email address for a Cloud Monitoring notification channel wired to the alert policies. Leave empty to create the policies without notifications (view-only in the console)."
  type        = string
  default     = ""
}

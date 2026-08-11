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
  description = "Google OAuth client ID used for sign-in in prod. Dedicated prod Web-application client (project verani-webstore-prod) with https://verani.bg + https://www.verani.bg as authorized JS origins. Must match VITE_GOOGLE_CLIENT_ID in deploy-prod.yml — the backend validates the ID token's audience against this."
  type        = string
  default     = "399593875435-vkfiip6lrhh9r5d8a5jv7s3rnpc88djn.apps.googleusercontent.com"
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

variable "revolut_mode" {
  description = "Revolut Merchant environment. Must be \"prod\" here — the API also fails closed at boot if this isn't \"prod\" when APP_ENV=prod."
  type        = string
  default     = "prod"

  validation {
    condition     = var.revolut_mode == "prod"
    error_message = "revolut_mode must be \"prod\" in the prod environment."
  }
}

variable "revolut_api_version" {
  description = "Pinned Revolut-Api-Version request header (date form, e.g. \"2024-09-01\"). Bump when the code is validated against a newer Merchant API version."
  type        = string
  default     = "2024-09-01"
}

variable "revolut_enabled" {
  description = "Inject the Revolut LIVE key + webhook secret into the API service. Keep false until the secret VALUES are populated out-of-band; flip to true to activate card payments in prod."
  type        = bool
  default     = false
}

variable "observability_enabled" {
  description = "Export OTel traces to Cloud Trace and custom metrics to Cloud Monitoring. Structured logging + trace correlation are always on; this only gates the OTel exporters. Enable after cloudtrace/monitoring APIs and the runtime SA roles have propagated."
  type        = bool
  default     = false
}

variable "otel_trace_sample_ratio" {
  description = "Parent-based trace sampling ratio (0.0-1.0) applied to root spans. Kept low in prod to stay within the Cloud Trace free tier."
  type        = string
  default     = "0.1"
}

variable "alert_email" {
  description = "Email address for a Cloud Monitoring notification channel wired to the alert policies. Leave empty to create the policies without notifications (view-only in the console)."
  type        = string
  default     = ""
}

variable "spf_record" {
  description = <<-EOT
    Apex SPF policy, published as part of the single apex TXT record set.
    Authorises both senders that use info@verani.bg: Google Workspace (the human
    mailbox / send-as) and SendGrid (outbound transactional mail).

    WARNING: a domain may have only ONE SPF record. Any additional provider that
    sends outbound mail as verani.bg must have its include: mechanism added here
    too, or those messages will start failing SPF. Kept at ~all (softfail)
    rather than -all while DMARC is still p=none.
  EOT
  type        = string
  default     = "v=spf1 include:_spf.google.com include:sendgrid.net ~all"
}

variable "dmarc_record" {
  description = "DMARC policy TXT value for _dmarc.verani.bg. Starts report-only (p=none); tighten to quarantine/reject once aggregate reports show SPF+DKIM aligned. Set a rua= mailbox to actually receive those reports."
  type        = string
  default     = "v=DMARC1; p=none; rua=mailto:info@verani.bg; fo=1"
}

variable "sendgrid_dns_records" {
  description = <<-EOT
    SendGrid domain-authentication CNAMEs, as subdomain (relative to the zone) =>
    target, from the SendGrid console (Sender Authentication) after authenticating
    verani.bg. One shared SendGrid account authenticates the domain once, so these
    serve both dev and prod. dns.tf appends the trailing dot. The matching DMARC
    TXT SendGrid asks for is already published by the `dmarc` resource, and
    sendgrid.net is already in the apex SPF, so neither is repeated here.
  EOT
  type        = map(string)
  default = {
    # Branded return-path / bounce subdomain
    "em728" = "u112300343.wl094.sendgrid.net"
    # DKIM signing keys
    "s1._domainkey" = "s1.domainkey.u112300343.wl094.sendgrid.net"
    "s2._domainkey" = "s2.domainkey.u112300343.wl094.sendgrid.net"
    # Branded link (click-tracking) subdomains
    "url3200"   = "sendgrid.net"
    "112300343" = "sendgrid.net"
  }
}

variable "email_enabled" {
  description = "Inject the SendGrid API key + event-webhook verification key into the API service, switching it from the log sender to real delivery. Defaults true now that the sending domain's SPF/DKIM/DMARC records resolve and the fs-prod-sendgrid-* secrets are populated — kept sticky (like revolut_enabled on dev) so a bare `terraform apply` can't silently revert to the log sender. Both secret containers MUST have a value version or the Cloud Run deploy fails on the secret_key_ref lookup."
  type        = bool
  default     = true
}

variable "email_webhook_enabled" {
  description = "Inject the SendGrid Signed Event Webhook verification key so /webhooks/sendgrid can validate inbound delivery events (bounces/complaints). Deliberately independent of email_enabled: outbound sending only needs the API key, so this stays false until the event webhook is actually configured and the fs-prod-email-webhook-verification-key secret has a value — otherwise the Cloud Run deploy would fail resolving that secret_key_ref."
  type        = bool
  default     = false
}

variable "email_alerts_enabled" {
  description = "Create the two email-deliverability alert policies (bounce/complaint and dead-letter). Off by default: they bind to the custom OTel metric emails_failed_total on the generic_task resource, which Cloud Monitoring only accepts once that metric+resource pair has been observed. Turn on after the API has exported the metric at least once (confirm the resource type in Metrics Explorer first)."
  type        = bool
  default     = false
}

variable "email_from" {
  description = "Envelope/header From address for all outbound mail. Must be an address on a domain authenticated in SendGrid."
  type        = string
  default     = "info@verani.bg"
}

variable "email_from_name" {
  description = "Display name shown alongside email_from in recipients' inboxes."
  type        = string
  default     = "Verani"
}

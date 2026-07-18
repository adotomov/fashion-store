resource "google_service_account" "api_runtime" {
  project      = var.project_id
  account_id   = "fs-api-${var.env}"
  display_name = "fashion-store api runtime (${var.env})"
}

resource "google_service_account" "web_runtime" {
  project      = var.project_id
  account_id   = "fs-web-${var.env}"
  display_name = "fashion-store webstore-fe runtime (${var.env})"
}

resource "google_service_account" "deployer" {
  project      = var.project_id
  account_id   = "fs-deployer-${var.env}"
  display_name = "fashion-store GitHub Actions deployer (${var.env})"
}

# api runtime: Cloud SQL connector access + read its own secrets + read/write
# the media bucket (granted in storage.tf, scoped to that bucket only).
resource "google_project_iam_member" "api_runtime_cloudsql_client" {
  project = var.project_id
  role    = "roles/cloudsql.client"
  member  = "serviceAccount:${google_service_account.api_runtime.email}"
}

# api runtime: export OTel spans to Cloud Trace and custom metrics to Cloud
# Monitoring. (Cloud Logging write is granted to Cloud Run runtime SAs by
# default, so no logWriter binding is needed for structured logs.)
resource "google_project_iam_member" "api_runtime_trace_agent" {
  project = var.project_id
  role    = "roles/cloudtrace.agent"
  member  = "serviceAccount:${google_service_account.api_runtime.email}"
}

resource "google_project_iam_member" "api_runtime_metric_writer" {
  project = var.project_id
  role    = "roles/monitoring.metricWriter"
  member  = "serviceAccount:${google_service_account.api_runtime.email}"
}

resource "google_secret_manager_secret_iam_member" "api_runtime_secrets" {
  for_each = toset([
    google_secret_manager_secret.database_url.secret_id,
    google_secret_manager_secret.auth_signing_secret.secret_id,
    google_secret_manager_secret.google_client_id.secret_id,
    google_secret_manager_secret.revolut_api_key.secret_id,
    google_secret_manager_secret.revolut_webhook_secret.secret_id,
  ])

  project   = var.project_id
  secret_id = each.value
  role      = "roles/secretmanager.secretAccessor"
  member    = "serviceAccount:${google_service_account.api_runtime.email}"
}

# Deployer: push images, deploy Cloud Run revisions, and impersonate the
# runtime service accounts when deploying on their behalf. Scoped to this
# project only - no org-level or billing permissions.
resource "google_project_iam_member" "deployer_run_admin" {
  project = var.project_id
  role    = "roles/run.admin"
  member  = "serviceAccount:${google_service_account.deployer.email}"
}

resource "google_project_iam_member" "deployer_artifact_writer" {
  project = var.project_id
  role    = "roles/artifactregistry.writer"
  member  = "serviceAccount:${google_service_account.deployer.email}"
}

resource "google_project_iam_member" "deployer_cloudsql_client" {
  project = var.project_id
  role    = "roles/cloudsql.client"
  member  = "serviceAccount:${google_service_account.deployer.email}"
}

resource "google_service_account_iam_member" "deployer_can_act_as_api" {
  service_account_id = google_service_account.api_runtime.name
  role               = "roles/iam.serviceAccountUser"
  member             = "serviceAccount:${google_service_account.deployer.email}"
}

resource "google_service_account_iam_member" "deployer_can_act_as_web" {
  service_account_id = google_service_account.web_runtime.name
  role               = "roles/iam.serviceAccountUser"
  member             = "serviceAccount:${google_service_account.deployer.email}"
}

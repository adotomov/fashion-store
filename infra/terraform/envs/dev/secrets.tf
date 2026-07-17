resource "random_password" "auth_signing_secret" {
  length  = 48
  special = false
}

resource "google_secret_manager_secret" "database_url" {
  project   = var.project_id
  secret_id = "fs-${var.env}-database-url"

  replication {
    auto {}
  }

  depends_on = [google_project_service.apis]
}

resource "google_secret_manager_secret_version" "database_url" {
  secret = google_secret_manager_secret.database_url.id
  secret_data = format(
    "postgres://%s:%s@/%s?host=/cloudsql/%s&sslmode=disable",
    var.db_user,
    random_password.db_password.result,
    var.db_name,
    google_sql_database_instance.main.connection_name,
  )
}

resource "google_secret_manager_secret" "auth_signing_secret" {
  project   = var.project_id
  secret_id = "fs-${var.env}-auth-signing-secret"

  replication {
    auto {}
  }

  depends_on = [google_project_service.apis]
}

resource "google_secret_manager_secret_version" "auth_signing_secret" {
  secret      = google_secret_manager_secret.auth_signing_secret.id
  secret_data = random_password.auth_signing_secret.result
}

# Not really secret, but kept alongside the others so the api Cloud Run
# service reads all its config the same way (Secret Manager), rather than
# mixing plain env vars and secret refs for what's conceptually one config
# surface.
resource "google_secret_manager_secret" "google_client_id" {
  project   = var.project_id
  secret_id = "fs-${var.env}-google-client-id"

  replication {
    auto {}
  }

  depends_on = [google_project_service.apis]
}

resource "google_secret_manager_secret_version" "google_client_id" {
  secret      = google_secret_manager_secret.google_client_id.id
  secret_data = var.google_client_id
}

# Revolut Merchant credentials (dev = Revolut SANDBOX). The secret containers
# are managed here, but their VALUES are added out-of-band with
# `gcloud secrets versions add` — they come from the Revolut dashboard and must
# never land in Terraform state. The API only reads them once revolut_enabled
# is flipped true (see cloud_run.tf).
resource "google_secret_manager_secret" "revolut_api_key" {
  project   = var.project_id
  secret_id = "fs-${var.env}-revolut-api-key"

  replication {
    auto {}
  }

  depends_on = [google_project_service.apis]
}

resource "google_secret_manager_secret" "revolut_webhook_secret" {
  project   = var.project_id
  secret_id = "fs-${var.env}-revolut-webhook-secret"

  replication {
    auto {}
  }

  depends_on = [google_project_service.apis]
}

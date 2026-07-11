resource "random_password" "db_password" {
  length  = 24
  special = false # keep it libpq-DSN-safe without escaping
}

resource "google_sql_database_instance" "main" {
  project          = var.project_id
  name             = "fashion-store-${var.env}"
  region           = var.region
  database_version = "POSTGRES_16"

  settings {
    tier              = var.db_tier
    edition           = "ENTERPRISE"
    availability_type = "REGIONAL" # HA: synchronous standby in another zone
    disk_size         = 20
    disk_type         = "PD_SSD"
    disk_autoresize   = true

    backup_configuration {
      enabled                        = true
      point_in_time_recovery_enabled = true # WAL archiving for PITR
      start_time                     = "02:00"
      transaction_log_retention_days = 7
      backup_retention_settings {
        retained_backups = 30
      }
    }

    ip_configuration {
      ipv4_enabled = true
      # As in dev: no authorized_networks — the only path in is the Cloud SQL
      # Auth Proxy used natively by Cloud Run, gated by IAM, not raw TCP.
    }
  }

  deletion_protection = true # prod — guard against accidental teardown

  depends_on = [google_project_service.apis]
}

resource "google_sql_database" "app" {
  project  = var.project_id
  instance = google_sql_database_instance.main.name
  name     = var.db_name
}

resource "google_sql_user" "app" {
  project  = var.project_id
  instance = google_sql_database_instance.main.name
  name     = var.db_user
  password = random_password.db_password.result
}

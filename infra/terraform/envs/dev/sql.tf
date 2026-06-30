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
    availability_type = "ZONAL" # no HA standby - dev environment, keep cost down
    disk_size         = 10
    disk_type         = "PD_SSD"
    disk_autoresize   = true

    backup_configuration {
      enabled = true
      # No point-in-time recovery for dev - daily backups are enough and
      # cheaper.
      point_in_time_recovery_enabled = false
    }

    ip_configuration {
      ipv4_enabled = true
      # No authorized_networks blocks declared on purpose: the only way in
      # is the Cloud SQL Auth Proxy (used natively by Cloud Run's Cloud SQL
      # connection feature), which authenticates via IAM/Cloud SQL Admin
      # API rather than raw TCP. A public IP with an empty allowlist is not
      # reachable directly from the internet.
    }
  }

  deletion_protection = false # dev only - allows easy teardown

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

# One-off Cloud Run Job that runs goose migrations against Cloud SQL,
# reusing the same image, Cloud SQL connection, and DATABASE_URL secret as
# the api service. CI executes this before deploying a new api revision.

resource "google_cloud_run_v2_job" "migrate" {
  project  = var.project_id
  name     = "api-migrate"
  location = var.region

  deletion_protection = false

  template {
    template {
      service_account = google_service_account.api_runtime.email
      max_retries     = 0

      volumes {
        name = "cloudsql"
        cloud_sql_instance {
          instances = [google_sql_database_instance.main.connection_name]
        }
      }

      containers {
        image = var.placeholder_image
        # Cloud Run doesn't expand env vars into args, so route through a
        # shell to substitute DATABASE_URL.
        command = ["/bin/sh", "-c"]
        args    = ["goose -dir /app/db/migrations postgres \"$DATABASE_URL\" up"]

        volume_mounts {
          name       = "cloudsql"
          mount_path = "/cloudsql"
        }

        env {
          name = "DATABASE_URL"
          value_source {
            secret_key_ref {
              secret  = google_secret_manager_secret.database_url.secret_id
              version = "latest"
            }
          }
        }
      }
    }
  }

  lifecycle {
    ignore_changes = [
      template[0].template[0].containers[0].image,
    ]
  }

  depends_on = [
    google_project_service.apis,
    google_secret_manager_secret_version.database_url,
  ]
}

resource "google_cloud_run_v2_service" "api" {
  project  = var.project_id
  name     = "api"
  location = var.region

  deletion_protection = false

  template {
    service_account = google_service_account.api_runtime.email

    scaling {
      min_instance_count = 0
      max_instance_count = 3
    }

    volumes {
      name = "cloudsql"
      cloud_sql_instance {
        instances = [google_sql_database_instance.main.connection_name]
      }
    }

    containers {
      image = var.placeholder_image

      ports {
        container_port = 8080
      }

      volume_mounts {
        name       = "cloudsql"
        mount_path = "/cloudsql"
      }

      env {
        name  = "APP_NAME"
        value = "fashion-store-api"
      }
      env {
        name  = "APP_ENV"
        value = var.env
      }
      env {
        name  = "HTTP_ADDR"
        value = ":8080"
      }
      env {
        name  = "LOG_LEVEL"
        value = "info"
      }
      env {
        name  = "LOG_FORMAT"
        value = "json"
      }
      env {
        name  = "SPEEDY_MODE"
        value = var.speedy_mode
      }
      env {
        name  = "FULFILLMENT_POLL_INTERVAL"
        value = var.fulfillment_poll_interval
      }
      env {
        name = "CORS_ALLOWED_ORIGINS"
        # Includes both the eventual custom domain and the Cloud Run *.run.app
        # URL, since the frontend is reachable at the latter until the Cloud
        # DNS cutover + domain mapping (create_domain_mappings) are done.
        value = "https://${var.web_subdomain}.${var.domain_root},${google_cloud_run_v2_service.webstore_fe.uri}"
      }
      env {
        name  = "STORAGE_ENDPOINT"
        value = "https://storage.googleapis.com"
      }
      env {
        name  = "STORAGE_PROJECT_ID"
        value = var.project_id
      }
      env {
        name  = "STORAGE_BUCKET"
        value = google_storage_bucket.media.name
      }
      env {
        name  = "STORAGE_INSECURE_SKIP_TLS"
        value = "false"
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
      env {
        name = "AUTH_SIGNING_SECRET"
        value_source {
          secret_key_ref {
            secret  = google_secret_manager_secret.auth_signing_secret.secret_id
            version = "latest"
          }
        }
      }
      env {
        name = "GOOGLE_CLIENT_ID"
        value_source {
          secret_key_ref {
            secret  = google_secret_manager_secret.google_client_id.secret_id
            version = "latest"
          }
        }
      }
    }
  }

  lifecycle {
    ignore_changes = [
      template[0].containers[0].image,
    ]
  }

  depends_on = [
    google_project_service.apis,
    google_secret_manager_secret_version.database_url,
    google_secret_manager_secret_version.auth_signing_secret,
    google_secret_manager_secret_version.google_client_id,
  ]
}

resource "google_cloud_run_v2_service_iam_member" "api_public" {
  project  = var.project_id
  location = var.region
  name     = google_cloud_run_v2_service.api.name
  role     = "roles/run.invoker"
  member   = "allUsers"
}

resource "google_cloud_run_v2_service" "webstore_fe" {
  project  = var.project_id
  name     = "webstore-fe"
  location = var.region

  deletion_protection = false

  template {
    service_account = google_service_account.web_runtime.email

    scaling {
      min_instance_count = 0
      max_instance_count = 3
    }

    containers {
      image = var.placeholder_image

      ports {
        container_port = 3000
      }
    }
  }

  lifecycle {
    ignore_changes = [
      template[0].containers[0].image,
    ]
  }

  depends_on = [google_project_service.apis]
}

resource "google_cloud_run_v2_service_iam_member" "web_public" {
  project  = var.project_id
  location = var.region
  name     = google_cloud_run_v2_service.webstore_fe.name
  role     = "roles/run.invoker"
  member   = "allUsers"
}

resource "google_artifact_registry_repository" "containers" {
  project       = var.project_id
  location      = var.region
  repository_id = "fashion-store-${var.env}"
  format        = "DOCKER"
  description   = "Container images for fashion-store (${var.env})"

  depends_on = [google_project_service.apis]
}

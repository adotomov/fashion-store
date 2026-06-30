resource "google_storage_bucket" "media" {
  project                     = var.project_id
  name                        = "fashion-store-media-${var.env}"
  location                    = var.region
  uniform_bucket_level_access = true
  force_destroy               = true # dev only

  cors {
    origin          = ["https://${var.web_subdomain}.${var.domain_root}"]
    method          = ["GET"]
    response_header = ["Content-Type"]
    max_age_seconds = 3600
  }
}

# Product media is browsed directly by storefront visitors - public read,
# consistent with the local FakeGCS behavior this replaces.
resource "google_storage_bucket_iam_member" "media_public_read" {
  bucket = google_storage_bucket.media.name
  role   = "roles/storage.objectViewer"
  member = "allUsers"
}

resource "google_storage_bucket_iam_member" "media_api_writer" {
  bucket = google_storage_bucket.media.name
  role   = "roles/storage.objectAdmin"
  member = "serviceAccount:${google_service_account.api_runtime.email}"
}

# objectAdmin only grants object-level permissions (storage.objects.*), not
# bucket-level ones. The app's EnsureBucket check does a GET on the bucket
# (the bucket itself is created by Terraform, never by the app), which needs
# storage.buckets.get specifically.
resource "google_storage_bucket_iam_member" "media_api_bucket_reader" {
  bucket = google_storage_bucket.media.name
  role   = "roles/storage.legacyBucketReader"
  member = "serviceAccount:${google_service_account.api_runtime.email}"
}

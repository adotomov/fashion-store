resource "google_storage_bucket" "media" {
  project                     = var.project_id
  name                        = "fashion-store-media-${var.env}"
  location                    = var.region
  uniform_bucket_level_access = true
  force_destroy               = false # prod — never mass-delete objects on teardown

  cors {
    origin          = ["https://${var.domain_root}"]
    method          = ["GET"]
    response_header = ["Content-Type"]
    max_age_seconds = 3600
  }
}

# Product media is browsed directly by storefront visitors — public read.
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

# objectAdmin is object-level only; the app's EnsureBucket check does a
# storage.buckets.get, which needs this bucket-level role.
resource "google_storage_bucket_iam_member" "media_api_bucket_reader" {
  bucket = google_storage_bucket.media.name
  role   = "roles/storage.legacyBucketReader"
  member = "serviceAccount:${google_service_account.api_runtime.email}"
}

# Private invoices bucket — stricter access than public media (per the infra
# guide). NO allUsers grant: only the api runtime SA can read/write. Reserved
# ahead of the app splitting invoice storage onto its own bucket (today the
# app has a single STORAGE_BUCKET, so no env var is wired to this yet).
resource "google_storage_bucket" "invoices" {
  project                     = var.project_id
  name                        = "fashion-store-invoices-${var.env}"
  location                    = var.region
  uniform_bucket_level_access = true
  force_destroy               = false

  versioning {
    enabled = true
  }
}

resource "google_storage_bucket_iam_member" "invoices_api_writer" {
  bucket = google_storage_bucket.invoices.name
  role   = "roles/storage.objectAdmin"
  member = "serviceAccount:${google_service_account.api_runtime.email}"
}

resource "google_storage_bucket_iam_member" "invoices_api_bucket_reader" {
  bucket = google_storage_bucket.invoices.name
  role   = "roles/storage.legacyBucketReader"
  member = "serviceAccount:${google_service_account.api_runtime.email}"
}

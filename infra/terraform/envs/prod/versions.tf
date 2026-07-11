terraform {
  required_version = ">= 1.9"

  required_providers {
    google = {
      source  = "hashicorp/google"
      version = "~> 6.0"
    }
    random = {
      source  = "hashicorp/random"
      version = "~> 3.6"
    }
  }

  # Remote state lives in a GCS bucket in the prod project. As with dev, this
  # bucket is NOT Terraform-managed (chicken/egg) — create it once by hand:
  #   gcloud storage buckets create gs://tfstate-verani-webstore-prod \
  #     --project=<prod-project-id> --location=europe-west1 --uniform-bucket-level-access
  #   gcloud storage buckets update gs://tfstate-verani-webstore-prod --versioning
  backend "gcs" {
    bucket = "tfstate-verani-webstore-prod"
    prefix = "fashion-store/prod"
  }
}

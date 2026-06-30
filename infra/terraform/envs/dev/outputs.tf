output "api_cloud_run_url" {
  value = google_cloud_run_v2_service.api.uri
}

output "web_cloud_run_url" {
  value = google_cloud_run_v2_service.webstore_fe.uri
}

output "artifact_registry_repo" {
  value = "${var.region}-docker.pkg.dev/${var.project_id}/${google_artifact_registry_repository.containers.repository_id}"
}

output "cloud_sql_connection_name" {
  value = google_sql_database_instance.main.connection_name
}

output "media_bucket" {
  value = google_storage_bucket.media.name
}

output "deployer_service_account_email" {
  value = google_service_account.deployer.email
}

output "workload_identity_provider" {
  description = "Full resource name to use as workload_identity_provider in the GitHub Actions google-github-actions/auth step."
  value       = google_iam_workload_identity_pool_provider.github.name
}

output "dns_name_servers" {
  description = "Name servers to set at the SuperHosting registrar to delegate verani.bg to Cloud DNS. Do NOT cut over until all existing records (apex A, www CNAME, MX) are confirmed correct in this zone."
  value       = google_dns_managed_zone.root.name_servers
}

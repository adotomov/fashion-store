output "lb_ip_address" {
  description = "Static anycast IP of the external HTTPS Load Balancer — the apex/www/api A records point here."
  value       = google_compute_global_address.lb.address
}

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

output "invoices_bucket" {
  value = google_storage_bucket.invoices.name
}

output "deployer_service_account_email" {
  value = google_service_account.deployer.email
}

output "workload_identity_provider" {
  description = "Full resource name to use as workload_identity_provider in the GitHub Actions google-github-actions/auth step."
  value       = google_iam_workload_identity_pool_provider.github.name
}

output "dns_name_servers" {
  description = "Nameservers to set at the SuperHosting registrar to delegate verani.bg to Cloud DNS. Do NOT cut over until all records (apex A, www A, api A, dev/api.dev CNAME, MX) are confirmed in this zone."
  value       = google_dns_managed_zone.root.name_servers
}

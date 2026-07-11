# DNS for verani.bg is now owned by the prod environment's Cloud DNS zone
# (infra/terraform/envs/prod/dns.tf) — the apex owner holds ALL records,
# including the dev/api.dev CNAMEs that point at the domain mappings below.
# This env only creates the dev Cloud Run domain mappings; it manages no DNS
# records.
#
# Migration note: the verani.bg managed zone + records previously lived here.
# Apply the prod env first (it recreates the zone), then apply this env — the
# old zone/records are destroyed from dev state. Safe only because the
# registrar nameservers have not been cut over to Cloud DNS yet.

# --- Cloud Run domain mappings for dev.verani.bg / api.dev.verani.bg ---
# Requires verani.bg to be a verified domain on this Google account (Search
# Console) and the CNAMEs (in the prod zone) resolving. Gated behind a
# variable so the rest of the apply can succeed before that manual step.

resource "google_cloud_run_domain_mapping" "web" {
  count    = var.create_domain_mappings ? 1 : 0
  project  = var.project_id
  location = var.region
  name     = "${var.web_subdomain}.${var.domain_root}"

  metadata {
    namespace = var.project_id
  }

  spec {
    route_name = google_cloud_run_v2_service.webstore_fe.name
  }
}

resource "google_cloud_run_domain_mapping" "api" {
  count    = var.create_domain_mappings ? 1 : 0
  project  = var.project_id
  location = var.region
  name     = "${var.api_subdomain}.${var.domain_root}"

  metadata {
    namespace = var.project_id
  }

  spec {
    route_name = google_cloud_run_v2_service.api.name
  }
}

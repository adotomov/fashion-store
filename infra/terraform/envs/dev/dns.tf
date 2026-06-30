resource "google_dns_managed_zone" "root" {
  project     = var.project_id
  name        = "verani-bg"
  dns_name    = "${var.domain_root}."
  description = "verani.bg - migrated from SuperHosting/bgdns.net"

  depends_on = [google_project_service.apis]
}

# --- Records that already exist at the current DNS host (SuperHosting)    ---
# --- and must be replicated here before nameservers are cut over, or the  ---
# --- live GitHub Pages site and email break. Audited 2026-06-30 via dig.  ---

resource "google_dns_record_set" "apex_a" {
  project      = var.project_id
  managed_zone = google_dns_managed_zone.root.name
  name         = google_dns_managed_zone.root.dns_name
  type         = "A"
  ttl          = 300
  rrdatas = [
    "185.199.108.153",
    "185.199.109.153",
    "185.199.110.153",
    "185.199.111.153",
  ]
}

resource "google_dns_record_set" "www_cname" {
  project      = var.project_id
  managed_zone = google_dns_managed_zone.root.name
  name         = "www.${google_dns_managed_zone.root.dns_name}"
  type         = "CNAME"
  ttl          = 300
  rrdatas      = ["boutiqueverani.github.io."]
}

resource "google_dns_record_set" "mx" {
  project      = var.project_id
  managed_zone = google_dns_managed_zone.root.name
  name         = google_dns_managed_zone.root.dns_name
  type         = "MX"
  ttl          = 300
  rrdatas      = ["20 mx2.bgdns.net."]
}

# --- New records for the dev environment ---

resource "google_dns_record_set" "dev_cname" {
  project      = var.project_id
  managed_zone = google_dns_managed_zone.root.name
  name         = "${var.web_subdomain}.${google_dns_managed_zone.root.dns_name}"
  type         = "CNAME"
  ttl          = 300
  rrdatas      = ["ghs.googlehosted.com."]
}

resource "google_dns_record_set" "api_dev_cname" {
  project      = var.project_id
  managed_zone = google_dns_managed_zone.root.name
  name         = "${var.api_subdomain}.${google_dns_managed_zone.root.dns_name}"
  type         = "CNAME"
  ttl          = 300
  rrdatas      = ["ghs.googlehosted.com."]
}

# --- Cloud Run domain mappings ---
# Requires verani.bg to be a domain verified on this Google account
# (Search Console) first - gated behind a variable so the rest of the apply
# can succeed before that one-time manual step is done.

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

# The authoritative verani.bg Cloud DNS zone lives in the prod project (the
# apex owner) and holds ALL records — prod, dev, and email. The dev env no
# longer manages DNS. Delegate the registrar's nameservers (SuperHosting) to
# the `name_servers` output here once every record below is confirmed.

resource "google_dns_managed_zone" "root" {
  project     = var.project_id
  name        = "verani-bg"
  dns_name    = "${var.domain_root}."
  description = "verani.bg authoritative zone (prod project; apex owner)"

  depends_on = [google_project_service.apis]
}

# --- Prod: apex, www and api -> the external HTTPS Load Balancer IP ---

resource "google_dns_record_set" "apex_a" {
  project      = var.project_id
  managed_zone = google_dns_managed_zone.root.name
  name         = google_dns_managed_zone.root.dns_name
  type         = "A"
  ttl          = 300
  rrdatas      = [google_compute_global_address.lb.address]
}

resource "google_dns_record_set" "www_a" {
  project      = var.project_id
  managed_zone = google_dns_managed_zone.root.name
  name         = "www.${google_dns_managed_zone.root.dns_name}"
  type         = "A"
  ttl          = 300
  rrdatas      = [google_compute_global_address.lb.address]
}

resource "google_dns_record_set" "api_a" {
  project      = var.project_id
  managed_zone = google_dns_managed_zone.root.name
  name         = "${var.api_subdomain}.${google_dns_managed_zone.root.dns_name}"
  type         = "A"
  ttl          = 300
  rrdatas      = [google_compute_global_address.lb.address]
}

# --- Dev: dev.verani.bg / api.dev.verani.bg -> dev's Cloud Run domain
# mappings. `ghs.googlehosted.com` is a static Google endpoint, so this needs
# no cross-project reference to the dev state. ---

resource "google_dns_record_set" "dev_cname" {
  project      = var.project_id
  managed_zone = google_dns_managed_zone.root.name
  name         = "dev.${google_dns_managed_zone.root.dns_name}"
  type         = "CNAME"
  ttl          = 300
  rrdatas      = ["ghs.googlehosted.com."]
}

resource "google_dns_record_set" "api_dev_cname" {
  project      = var.project_id
  managed_zone = google_dns_managed_zone.root.name
  name         = "api.dev.${google_dns_managed_zone.root.dns_name}"
  type         = "CNAME"
  ttl          = 300
  rrdatas      = ["ghs.googlehosted.com."]
}

# --- Google site verification (Search Console) ---
# Required so the boutiqueverani@gmail.com account can create Cloud Run domain
# mappings for dev.verani.bg / api.dev.verani.bg. Apex TXT record.

resource "google_dns_record_set" "google_site_verification" {
  project      = var.project_id
  managed_zone = google_dns_managed_zone.root.name
  name         = google_dns_managed_zone.root.dns_name
  type         = "TXT"
  ttl          = 300
  rrdatas      = ["\"google-site-verification=CNKnHx-eSyIwKLglHqWzI85yrSdco7AUY-1qzbJEzXQ\""]
}

# --- Email: preserved from SuperHosting, must not break on cutover ---

resource "google_dns_record_set" "mx" {
  project      = var.project_id
  managed_zone = google_dns_managed_zone.root.name
  name         = google_dns_managed_zone.root.dns_name
  type         = "MX"
  ttl          = 300
  rrdatas      = ["20 mx2.bgdns.net."]
}

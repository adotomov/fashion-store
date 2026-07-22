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

# --- Apex TXT: site verification + SPF ---
# DNS allows only ONE TXT record set per name, so the Search Console
# verification string and the SPF policy must live in this single resource —
# adding SPF as a separate apex TXT resource would conflict and clobber one of
# them. Both strings coexist as separate rrdatas entries.
#
# The verification string is required so the boutiqueverani@gmail.com account
# can create Cloud Run domain mappings for dev.verani.bg / api.dev.verani.bg.

# Renamed from google_site_verification when SPF joined this record set. Without
# this, Terraform would destroy and recreate the record — briefly dropping the
# Search Console verification that the Cloud Run domain mappings depend on.
moved {
  from = google_dns_record_set.google_site_verification
  to   = google_dns_record_set.apex_txt
}

resource "google_dns_record_set" "apex_txt" {
  project      = var.project_id
  managed_zone = google_dns_managed_zone.root.name
  name         = google_dns_managed_zone.root.dns_name
  type         = "TXT"
  ttl          = 300
  rrdatas = [
    "\"google-site-verification=CNKnHx-eSyIwKLglHqWzI85yrSdco7AUY-1qzbJEzXQ\"",
    "\"${var.spf_record}\"",
  ]
}

# --- DMARC ---
# Starts at p=none: report-only, so a misconfiguration can be observed in the
# aggregate reports without silently dropping real customer mail. Tighten to
# quarantine and then reject once reports show SPF+DKIM passing consistently.

resource "google_dns_record_set" "dmarc" {
  project      = var.project_id
  managed_zone = google_dns_managed_zone.root.name
  name         = "_dmarc.${google_dns_managed_zone.root.dns_name}"
  type         = "TXT"
  ttl          = 300
  rrdatas      = ["\"${var.dmarc_record}\""]
}

# --- SendGrid domain authentication (DKIM + branded link CNAMEs) ---
# SendGrid generates these per account when you authenticate a sending domain,
# so the values can't be known ahead of time. Populate sendgrid_dns_records from
# the SendGrid console and apply; until then this creates nothing and email
# stays disabled (see email_enabled in the dev/prod cloud_run config).

resource "google_dns_record_set" "sendgrid" {
  for_each = var.sendgrid_dns_records

  project      = var.project_id
  managed_zone = google_dns_managed_zone.root.name
  name         = "${each.key}.${google_dns_managed_zone.root.dns_name}"
  type         = "CNAME"
  ttl          = 300
  rrdatas      = [endswith(each.value, ".") ? each.value : "${each.value}."]
}

# --- Email: preserved from SuperHosting, must not break on cutover ---
# INBOUND mail only. info@verani.bg is a SuperHosting mailbox and receives here;
# outbound transactional mail goes via SendGrid. Do not repoint or remove this.

resource "google_dns_record_set" "mx" {
  project      = var.project_id
  managed_zone = google_dns_managed_zone.root.name
  name         = google_dns_managed_zone.root.dns_name
  type         = "MX"
  ttl          = 300
  rrdatas      = ["20 mx2.bgdns.net."]
}

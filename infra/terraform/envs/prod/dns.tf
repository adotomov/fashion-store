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

# --- Google Workspace DKIM ---
# Signs outbound human mail sent as info@verani.bg via Workspace, so it aligns
# under DMARC. Selector "google" is Workspace's default. The 2048-bit key is
# longer than a single 255-char TXT character-string, so it is published as two
# quoted strings in one record — resolvers concatenate them with no separator.
# This is a public key, so it lives inline like the site-verification string.
resource "google_dns_record_set" "workspace_dkim" {
  project      = var.project_id
  managed_zone = google_dns_managed_zone.root.name
  name         = "google._domainkey.${google_dns_managed_zone.root.dns_name}"
  type         = "TXT"
  ttl          = 300
  rrdatas = [
    "\"v=DKIM1; k=rsa; p=MIIBIjANBgkqhkiG9w0BAQEFAAOCAQ8AMIIBCgKCAQEAqippy2iB94ppg/h3lR9U/Of6v1YF2wmGv2qd2oLgtuWUc7PKmJTDyiCfZrRkn28GJe5V6NuyHoeWl3P2y/GBtpoaYlJ1Bl50qwQd5N4SejVdDZDsyfm1nCsfpmOEOPzUe0A5rSGwMVEo/mIiwR4OWOXyw7DJ\" \"BIIMJejbwyUH5rTfMd3DntWpD7ExkFEsI0nXOeqgXhNRulafXyhhyG3Mlc49oNNSqMLBXVdT1tLOdJAwVPUwnD2+XonLTXJ+Hs1I8PwnYaaSGGBsYkZZ0ulFLKIbmRbnhJnG3dyBGX193+7nvJCl1QkBFmYYqnBxAxCA4CEJF1qkizVRUHxpgPzN5QIDAQAB\"",
  ]
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

# --- Email: inbound routed to Google Workspace ---
# INBOUND mail only. info@verani.bg is a Google Workspace mailbox
# (boutiqueverani@gmail.com) and receives here. Migrated off SuperHosting's
# mx2.bgdns.net on the Workspace switch — the SuperHosting mailbox no longer
# receives. Outbound transactional mail still goes via SendGrid; outbound human
# mail goes via Workspace (both send as info@verani.bg — see the SPF record).
# The single-record form (smtp.google.com) is the modern Workspace MX.

resource "google_dns_record_set" "mx" {
  project      = var.project_id
  managed_zone = google_dns_managed_zone.root.name
  name         = google_dns_managed_zone.root.dns_name
  type         = "MX"
  ttl          = 300
  rrdatas      = ["1 smtp.google.com."]
}

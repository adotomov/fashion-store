# Global external Application Load Balancer fronting the two Cloud Run
# services. This is the single public entry point for prod:
#
#   verani.bg, www.verani.bg  -> webstore-fe   (apex requires an LB — Cloud
#   api.verani.bg             -> api            Run domain mappings can't map
#                                               a bare apex)
#
# TLS is a Google-managed certificate (auto-provisioned + auto-renewed once
# the A records below resolve to the LB IP). Cloud Armor is attached to both
# backends for WAF + per-IP rate limiting.

# Static anycast IP the apex A record points at.
resource "google_compute_global_address" "lb" {
  project = var.project_id
  name    = "fs-${var.env}-lb-ip"

  depends_on = [google_project_service.apis]
}

# --- Serverless NEGs (one per Cloud Run service) ---

resource "google_compute_region_network_endpoint_group" "api" {
  project               = var.project_id
  name                  = "neg-api-${var.env}"
  region                = var.region
  network_endpoint_type = "SERVERLESS"

  cloud_run {
    service = google_cloud_run_v2_service.api.name
  }
}

resource "google_compute_region_network_endpoint_group" "web" {
  project               = var.project_id
  name                  = "neg-web-${var.env}"
  region                = var.region
  network_endpoint_type = "SERVERLESS"

  cloud_run {
    service = google_cloud_run_v2_service.webstore_fe.name
  }
}

# --- Cloud Armor security policy (WAF + rate limiting) ---

resource "google_compute_security_policy" "armor" {
  project = var.project_id
  name    = "fs-${var.env}-armor"

  # OWASP CRS preconfigured rules: block obvious SQLi and XSS.
  rule {
    action   = "deny(403)"
    priority = 900
    match {
      expr {
        expression = "evaluatePreconfiguredExpr('sqli-v33-stable') || evaluatePreconfiguredExpr('xss-v33-stable')"
      }
    }
    description = "OWASP CRS: SQLi + XSS"
  }

  # Per-IP rate limit: 600 requests / minute, 429 over the threshold.
  rule {
    action   = "throttle"
    priority = 1000
    match {
      versioned_expr = "SRC_IPS_V1"
      config {
        src_ip_ranges = ["*"]
      }
    }
    rate_limit_options {
      conform_action = "allow"
      exceed_action  = "deny(429)"
      enforce_on_key = "IP"
      rate_limit_threshold {
        count        = 600
        interval_sec = 60
      }
    }
    description = "per-IP rate limit"
  }

  # Default rule — required, lowest priority.
  rule {
    action   = "allow"
    priority = 2147483647
    match {
      versioned_expr = "SRC_IPS_V1"
      config {
        src_ip_ranges = ["*"]
      }
    }
    description = "default allow"
  }
}

# --- Backend services ---

resource "google_compute_backend_service" "api" {
  project               = var.project_id
  name                  = "backend-api-${var.env}"
  load_balancing_scheme = "EXTERNAL_MANAGED"
  security_policy       = google_compute_security_policy.armor.id

  backend {
    group = google_compute_region_network_endpoint_group.api.id
  }
}

resource "google_compute_backend_service" "web" {
  project               = var.project_id
  name                  = "backend-web-${var.env}"
  load_balancing_scheme = "EXTERNAL_MANAGED"
  security_policy       = google_compute_security_policy.armor.id

  backend {
    group = google_compute_region_network_endpoint_group.web.id
  }
}

# --- URL map: host-based routing ---

resource "google_compute_url_map" "default" {
  project = var.project_id
  name    = "fs-${var.env}-urlmap"

  # Apex + www fall through to the storefront.
  default_service = google_compute_backend_service.web.id

  host_rule {
    hosts        = ["${var.api_subdomain}.${var.domain_root}"]
    path_matcher = "api"
  }

  path_matcher {
    name            = "api"
    default_service = google_compute_backend_service.api.id
  }
}

# --- Managed TLS certificate (apex + www + api) ---

resource "google_compute_managed_ssl_certificate" "default" {
  project = var.project_id
  name    = "fs-${var.env}-cert"

  managed {
    domains = [
      var.domain_root,
      "www.${var.domain_root}",
      "${var.api_subdomain}.${var.domain_root}",
    ]
  }
}

# --- HTTPS front end ---

resource "google_compute_target_https_proxy" "default" {
  project          = var.project_id
  name             = "fs-${var.env}-https-proxy"
  url_map          = google_compute_url_map.default.id
  ssl_certificates = [google_compute_managed_ssl_certificate.default.id]
}

resource "google_compute_global_forwarding_rule" "https" {
  project               = var.project_id
  name                  = "fs-${var.env}-https"
  load_balancing_scheme = "EXTERNAL_MANAGED"
  port_range            = "443"
  target                = google_compute_target_https_proxy.default.id
  ip_address            = google_compute_global_address.lb.id
}

# --- HTTP -> HTTPS redirect ---

resource "google_compute_url_map" "redirect" {
  project = var.project_id
  name    = "fs-${var.env}-redirect"

  default_url_redirect {
    https_redirect         = true
    redirect_response_code = "MOVED_PERMANENTLY_DEFAULT"
    strip_query            = false
  }
}

resource "google_compute_target_http_proxy" "redirect" {
  project = var.project_id
  name    = "fs-${var.env}-http-proxy"
  url_map = google_compute_url_map.redirect.id
}

resource "google_compute_global_forwarding_rule" "http" {
  project               = var.project_id
  name                  = "fs-${var.env}-http"
  load_balancing_scheme = "EXTERNAL_MANAGED"
  port_range            = "80"
  target                = google_compute_target_http_proxy.redirect.id
  ip_address            = google_compute_global_address.lb.id
}

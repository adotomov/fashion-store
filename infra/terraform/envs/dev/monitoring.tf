# Cloud Monitoring: uptime check, a log-based error metric, and alert policies.
# All of it lives in Google's included operations suite (no paid aggregator).
# Alert policies are created even without an alert_email — they are then
# view-only in the console; set alert_email to also get notified.

locals {
  # Public API host fronted by the HTTPS load balancer, e.g. api.verani.bg.
  monitoring_api_host = "${var.api_subdomain}.${var.domain_root}"
  # Wire the email channel into policies only when an address is configured.
  notification_channels = var.alert_email != "" ? [google_monitoring_notification_channel.email[0].id] : []
}

resource "google_monitoring_notification_channel" "email" {
  count        = var.alert_email != "" ? 1 : 0
  project      = var.project_id
  display_name = "fashion-store alerts (${var.env})"
  type         = "email"
  labels = {
    email_address = var.alert_email
  }

  depends_on = [google_project_service.apis]
}

# Counts ERROR+ log entries from the api service (the reshaped GCP severity
# field makes this filter reliable). Backs the error-spike alert below.
resource "google_logging_metric" "api_errors" {
  project = var.project_id
  name    = "fashion_store_api_errors_${var.env}"
  filter  = "resource.type=\"cloud_run_revision\" AND resource.labels.service_name=\"api\" AND severity>=ERROR"

  metric_descriptor {
    metric_kind = "DELTA"
    value_type  = "INT64"
    unit        = "1"
  }
}

# Alerts on a spike in ERROR+ logs (covers panics, failed webhooks, amount
# mismatches, etc. — anything logged at error severity).
resource "google_monitoring_alert_policy" "api_error_logs" {
  project      = var.project_id
  display_name = "API error logs elevated (${var.env})"
  combiner     = "OR"

  conditions {
    display_name = "ERROR+ log entries > 5 in 5m"
    condition_threshold {
      filter          = "resource.type=\"cloud_run_revision\" AND metric.type=\"logging.googleapis.com/user/${google_logging_metric.api_errors.name}\""
      comparison      = "COMPARISON_GT"
      threshold_value = 5
      duration        = "300s"
      aggregations {
        alignment_period   = "300s"
        per_series_aligner = "ALIGN_DELTA"
      }
      trigger {
        count = 1
      }
    }
  }

  notification_channels = local.notification_channels
  depends_on            = [google_project_service.apis]
}

# Alerts when the api service returns 5xx responses (Cloud Run's free built-in
# request_count metric, sliced by response class — no custom metric needed).
resource "google_monitoring_alert_policy" "api_5xx" {
  project      = var.project_id
  display_name = "API 5xx responses (${var.env})"
  combiner     = "OR"

  conditions {
    display_name = "5xx request rate > 0"
    condition_threshold {
      filter          = "resource.type=\"cloud_run_revision\" AND resource.labels.service_name=\"api\" AND metric.type=\"run.googleapis.com/request_count\" AND metric.labels.response_code_class=\"5xx\""
      comparison      = "COMPARISON_GT"
      threshold_value = 0
      duration        = "300s"
      aggregations {
        alignment_period   = "60s"
        per_series_aligner = "ALIGN_RATE"
      }
      trigger {
        count = 1
      }
    }
  }

  notification_channels = local.notification_channels
  depends_on            = [google_project_service.apis]
}

# External HTTPS uptime check hitting the liveness endpoint through the load
# balancer, plus an alert when it starts failing.
resource "google_monitoring_uptime_check_config" "api_healthz" {
  project      = var.project_id
  display_name = "api /healthz (${var.env})"
  timeout      = "10s"
  period       = "300s"

  http_check {
    path         = "/healthz"
    port         = 443
    use_ssl      = true
    validate_ssl = true
  }

  monitored_resource {
    type = "uptime_url"
    labels = {
      project_id = var.project_id
      host       = local.monitoring_api_host
    }
  }

  depends_on = [google_project_service.apis]
}

resource "google_monitoring_alert_policy" "api_uptime" {
  project      = var.project_id
  display_name = "API uptime check failing (${var.env})"
  combiner     = "OR"

  conditions {
    display_name = "healthz uptime < 100%"
    condition_threshold {
      filter          = "metric.type=\"monitoring.googleapis.com/uptime_check/check_passed\" AND resource.type=\"uptime_url\" AND metric.label.check_id=\"${google_monitoring_uptime_check_config.api_healthz.uptime_check_id}\""
      comparison      = "COMPARISON_LT"
      threshold_value = 1
      duration        = "300s"
      aggregations {
        alignment_period   = "300s"
        per_series_aligner = "ALIGN_FRACTION_TRUE"
      }
      trigger {
        count = 1
      }
    }
  }

  notification_channels = local.notification_channels
  depends_on            = [google_project_service.apis]
}

# Email deliverability. Bounces and spam complaints are the two signals that
# destroy a sending domain's reputation, and they are invisible until customers
# start reporting missing order confirmations — so alert on any sustained rate
# rather than waiting for a threshold.
resource "google_monitoring_alert_policy" "email_bounces" {
  project      = var.project_id
  display_name = "Email bounces / complaints (${var.env})"
  combiner     = "OR"

  conditions {
    display_name = "bounce or complaint rate > 0"
    condition_threshold {
      filter          = "resource.type=\"cloud_run_revision\" AND metric.type=\"custom.googleapis.com/opentelemetry/emails_failed_total\" AND (metric.labels.outcome=\"bounce\" OR metric.labels.outcome=\"complaint\")"
      comparison      = "COMPARISON_GT"
      threshold_value = 0
      duration        = "300s"
      aggregations {
        alignment_period   = "300s"
        per_series_aligner = "ALIGN_RATE"
      }
      trigger {
        count = 1
      }
    }
  }

  notification_channels = local.notification_channels
  depends_on            = [google_project_service.apis]
}

# A dead-lettered email is a customer who never got their order confirmation.
# Retries are expected and not alerted on; exhausting them is not.
resource "google_monitoring_alert_policy" "email_dead_letters" {
  project      = var.project_id
  display_name = "Emails dead-lettered (${var.env})"
  combiner     = "OR"

  conditions {
    display_name = "dead-letter rate > 0"
    condition_threshold {
      filter          = "resource.type=\"cloud_run_revision\" AND metric.type=\"custom.googleapis.com/opentelemetry/emails_failed_total\" AND metric.labels.outcome=\"dead_letter\""
      comparison      = "COMPARISON_GT"
      threshold_value = 0
      duration        = "300s"
      aggregations {
        alignment_period   = "300s"
        per_series_aligner = "ALIGN_RATE"
      }
      trigger {
        count = 1
      }
    }
  }

  notification_channels = local.notification_channels
  depends_on            = [google_project_service.apis]
}

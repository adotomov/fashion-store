# Drop unmatched-route 404 noise from the storefront. Internet background
# scanners constantly probe every public host for wp-*/*.php/.env/etc.;
# react-router-serve logs each miss to stderr, which Cloud Run records at ERROR
# severity. These are harmless 404s against a stack that serves no PHP/WordPress,
# so exclude them from the _Default log sink to cut noise and log-ingestion cost.
# Live traffic is unaffected, and genuine app errors (500s / ErrorBoundary) don't
# carry this text, so they still land in the logs.
resource "google_logging_project_exclusion" "fe_route_404s" {
  project     = var.project_id
  name        = "drop-fe-route-404s"
  description = "Exclude storefront unmatched-route 404 bot noise from the _Default sink"
  filter      = "resource.type=\"cloud_run_revision\" AND resource.labels.service_name=\"webstore-fe\" AND severity>=ERROR AND \"No route matches URL\""
}

import { GA_MEASUREMENT_ID, isAnalyticsEnabled } from "../../lib/analytics/config";

// GoogleAnalytics injects the gtag.js bootstrap into the document <head>.
//
// Consent-first: before anything else we set Google Consent Mode v2 defaults to
// DENIED for every storage type. verani.bg serves the EU, so no analytics or ad
// cookies may be written until the visitor accepts the consent banner (wired in
// a later phase, which flips analytics_storage to "granted"). Until then gtag
// runs in cookieless/anonymised mode — it can still model traffic but stores
// nothing on the device. `wait_for_update` gives the banner a moment to restore
// a previously-saved choice before the first hit is sent.
//
// We also set `send_page_view: false`: this is a client-side SPA, so page views
// are sent manually on route change (see the page-view hook) rather than once at
// script load.
//
// When VITE_GA_MEASUREMENT_ID is unset (local/devbox), this renders nothing.
export function GoogleAnalytics() {
  if (!isAnalyticsEnabled) return null;

  const bootstrap = `
window.dataLayer = window.dataLayer || [];
function gtag(){dataLayer.push(arguments);}
gtag('consent', 'default', {
  ad_storage: 'denied',
  ad_user_data: 'denied',
  ad_personalization: 'denied',
  analytics_storage: 'denied',
  wait_for_update: 500
});
gtag('set', 'url_passthrough', true);
gtag('set', 'ads_data_redaction', true);
gtag('js', new Date());
gtag('config', '${GA_MEASUREMENT_ID}', { send_page_view: false });
`;

  return (
    <>
      {/* Consent defaults + config MUST run before the loader processes any
          hit, so this inline script comes first. */}
      <script dangerouslySetInnerHTML={{ __html: bootstrap }} />
      <script async src={`https://www.googletagmanager.com/gtag/js?id=${GA_MEASUREMENT_ID}`} />
    </>
  );
}

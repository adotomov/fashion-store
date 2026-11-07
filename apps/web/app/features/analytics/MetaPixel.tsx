import { META_PIXEL_ID, isPixelEnabled } from "../../lib/analytics/meta-pixel";

// MetaPixel injects the Meta Pixel (fbq) bootstrap into the document <head>,
// mirroring GoogleAnalytics.tsx.
//
// Consent-first: right after the loader stub we call fbq('consent','revoke'), so
// the Pixel initialises but withholds every event until the visitor accepts the
// cookie banner — at which point ConsentContext calls setPixelConsent(true) →
// fbq('consent','grant'). verani.bg serves the EU, so no advertising data leaves
// the browser until consent is given. This reuses the single cookie-consent
// decision that already gates GA.
//
// We also omit the standard snippet's load-time fbq('track','PageView'): this is
// a client-side SPA, so PageView is fired manually on every route change (see the
// usePageViews hook), exactly like GA's send_page_view:false.
//
// No <noscript> fallback is rendered on purpose: a plain tracking <img> fires
// unconditionally and cannot honour consent (there is no JS to collect it), which
// would contradict the consent-first model above. GA has no no-script fallback
// either; script-disabled visitors are simply not tracked.
//
// When META_PIXEL_ID is unset (local/devbox), this renders nothing.
export function MetaPixel() {
  if (!isPixelEnabled) return null;

  const bootstrap = `
!function(f,b,e,v,n,t,s)
{if(f.fbq)return;n=f.fbq=function(){n.callMethod?
n.callMethod.apply(n,arguments):n.queue.push(arguments)};
if(!f._fbq)f._fbq=n;n.push=n;n.loaded=!0;n.version='2.0';
n.queue=[];t=b.createElement(e);t.async=!0;
t.src=v;s=b.getElementsByTagName(e)[0];
s.parentNode.insertBefore(t,s)}(window, document,'script',
'https://connect.facebook.net/en_US/fbevents.js');
fbq('consent', 'revoke');
fbq('init', '${META_PIXEL_ID}');
`;

  return <script dangerouslySetInnerHTML={{ __html: bootstrap }} />;
}

import { isAnalyticsEnabled } from "./config";

// Low-level gtag.js bridge. Every helper is a safe no-op when analytics is OFF
// (no Measurement ID), during SSR (no window), or before the gtag bootstrap has
// defined `window.gtag` — so call sites never need their own guards.
//
// The `gtag` function itself is defined by the inline bootstrap injected in
// features/analytics/GoogleAnalytics.tsx; here we only ever push to it.

declare global {
  interface Window {
    dataLayer?: unknown[];
    gtag?: (...args: unknown[]) => void;
  }
}

function canSend(): boolean {
  return isAnalyticsEnabled && typeof window !== "undefined" && typeof window.gtag === "function";
}

// track sends a GA4 event. `params` is the event payload (item arrays, value,
// currency, etc.). Undefined values are harmless — GA ignores them.
export function track(eventName: string, params: Record<string, unknown> = {}): void {
  if (!canSend()) return;
  window.gtag!("event", eventName, params);
}

// setAnalyticsConsent flips Consent Mode's analytics_storage. Called by the
// consent banner: granted after the visitor accepts, denied if they withdraw.
// Until this runs, the bootstrap's "denied" default keeps GA cookieless.
export function setAnalyticsConsent(granted: boolean): void {
  if (!isAnalyticsEnabled || typeof window === "undefined" || typeof window.gtag !== "function") return;
  window.gtag("consent", "update", {
    analytics_storage: granted ? "granted" : "denied",
  });
}

// trackPageView reports a virtual page view. This is a client-side SPA, so the
// gtag config was set with send_page_view:false and we fire this on every route
// change instead (see the usePageViews hook).
export function trackPageView(path: string, title?: string): void {
  track("page_view", {
    page_path: path,
    page_location: typeof window !== "undefined" ? window.location.href : undefined,
    page_title: title ?? (typeof document !== "undefined" ? document.title : undefined),
  });
}

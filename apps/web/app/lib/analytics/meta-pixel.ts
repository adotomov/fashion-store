import { runtimeConfig } from "../config/runtime";

// Low-level Meta Pixel (fbq) bridge + typed standard-event helpers. This mirrors
// the GA layer (lib/analytics/gtag.ts + ecommerce.ts): every helper is a safe
// no-op when the Pixel is OFF (no Pixel ID), during SSR (no window), or before
// the fbq bootstrap has defined `window.fbq` — so call sites never need their
// own guards.
//
// The `fbq` function itself is defined by the inline bootstrap injected in
// features/analytics/MetaPixel.tsx; here we only ever push to it.
//
// Consent: the bootstrap calls fbq('consent','revoke') before init, so the Pixel
// initialises but withholds all data until the visitor accepts the cookie
// banner, at which point the consent context calls setPixelConsent(true) →
// fbq('consent','grant'). This reuses the single cookie-consent decision that
// already gates GA (see features/consent/ConsentContext.tsx).

// The Meta Pixel ID is set per environment via META_PIXEL_ID on the server (see
// lib/config/runtime.ts). When empty — local dev, devbox, or any environment
// that hasn't set it — the Pixel is OFF: the fbq script is never injected and
// every helper below is a no-op. A valid Pixel ID is a long numeric string; we
// sanity-check that so a mis-set value doesn't inject a broken tag.
export const META_PIXEL_ID = runtimeConfig().metaPixelId;

export const isPixelEnabled = /^\d{5,}$/.test(META_PIXEL_ID);

declare global {
  interface Window {
    fbq?: (...args: unknown[]) => void;
    _fbq?: unknown;
  }
}

function canSend(): boolean {
  return isPixelEnabled && typeof window !== "undefined" && typeof window.fbq === "function";
}

// pixelTrack fires a Meta standard event (ViewContent, AddToCart, …). `params`
// is the event payload; undefined values are harmless.
export function pixelTrack(eventName: string, params: Record<string, unknown> = {}): void {
  if (!canSend()) return;
  window.fbq!("track", eventName, params);
}

// setPixelConsent flips the Pixel's consent state. Called by the consent banner:
// grant after the visitor accepts, revoke if they withdraw. Until this runs, the
// bootstrap's "revoke" default keeps the Pixel from sending.
export function setPixelConsent(granted: boolean): void {
  if (!isPixelEnabled || typeof window === "undefined" || typeof window.fbq !== "function") return;
  window.fbq("consent", granted ? "grant" : "revoke");
}

// pixelPageView reports a PageView. This is a client-side SPA, so the bootstrap
// omits the standard snippet's load-time PageView and we fire this on every
// route change instead (see the usePageViews hook).
export function pixelPageView(): void {
  pixelTrack("PageView");
}

// ---- Typed standard-event helpers ------------------------------------------
// These take plain primitives — already computed by the GA ecommerce helpers in
// ecommerce.ts that call them — so they share no logic with the GA layer and
// stay decoupled. `value` is in major units (e.g. EUR), matching GA.

export type PixelContent = { id: string; quantity: number };

export function pixelViewContent(p: { id: string; name: string; value: number; currency: string }): void {
  pixelTrack("ViewContent", {
    content_type: "product",
    content_ids: [p.id],
    content_name: p.name,
    value: p.value,
    currency: p.currency,
  });
}

export function pixelAddToCart(p: {
  id: string;
  name: string;
  quantity: number;
  value: number;
  currency: string;
}): void {
  pixelTrack("AddToCart", {
    content_type: "product",
    content_ids: [p.id],
    content_name: p.name,
    contents: [{ id: p.id, quantity: p.quantity }],
    value: p.value,
    currency: p.currency,
  });
}

export function pixelInitiateCheckout(p: {
  contentIds: string[];
  numItems: number;
  value: number;
  currency: string;
}): void {
  pixelTrack("InitiateCheckout", {
    content_type: "product",
    content_ids: p.contentIds,
    num_items: p.numItems,
    value: p.value,
    currency: p.currency,
  });
}

export function pixelPurchase(p: {
  contentIds: string[];
  contents: PixelContent[];
  value: number;
  currency: string;
}): void {
  pixelTrack("Purchase", {
    content_type: "product",
    content_ids: p.contentIds,
    contents: p.contents,
    value: p.value,
    currency: p.currency,
  });
}

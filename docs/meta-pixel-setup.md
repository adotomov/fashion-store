# Meta Pixel — Setup Guide (operator)

How the Meta (Facebook) Pixel is wired into the storefront, and how to enable it
per environment. The Pixel powers Meta ad optimization and conversion tracking.
It mirrors the Google Analytics setup ([google-analytics-setup.md](google-analytics-setup.md));
the code lives alongside GA in `apps/web/app/lib/analytics/` and
`apps/web/app/features/analytics/`.

## What's tracked

| Event | When | Key params |
| --- | --- | --- |
| `PageView` | every page / client-side navigation | — |
| `ViewContent` | product detail page viewed | `content_ids=[productId]`, `value`, `currency` |
| `AddToCart` | item added from a product page | `content_ids`, `contents=[{id,quantity}]`, `value`, `currency` |
| `InitiateCheckout` | checkout started | `content_ids`, `num_items`, `value` (cart subtotal), `currency` |
| `Purchase` | order placed (cash, card-on-terminal, or Revolut) | `content_ids`, `contents`, `value` (order total), `currency` |

`value` is in major units (EUR). Prices mirror GA: `promotion_price ?? base_price`.

## Consent (EU)

The Pixel is loaded **consent-first**: its bootstrap calls `fbq('consent','revoke')`
before `init`, so it initialises but sends **nothing** until the visitor accepts
the existing cookie banner. Acceptance flips both GA (`analytics_storage`) and the
Pixel (`fbq('consent','grant')`) via the single decision in
`app/features/consent/ConsentContext.tsx`. Reject ⇒ Pixel stays revoked. There is
no `<noscript>` fallback pixel on purpose — a plain tracking `<img>` cannot honour
consent, so script-disabled visitors are simply not tracked.

## Enable the Pixel for an environment

The Pixel ID reaches the browser at **runtime** via the frontend config
(`app/lib/config/runtime.ts`), read from the `META_PIXEL_ID` server env var. When
it's empty the Pixel is **OFF**: no `fbq` script is injected and every event is a
no-op.

- **Prod:** set in [.devbox/project.yaml](../.devbox/project.yaml) under the `prod`
  environment → `env.webstore-fe` → `META_PIXEL_ID: "1445134643714482"` (already
  added, next to `GA_MEASUREMENT_ID`). The next prod deploy picks it up.
- **Dev:** leave `META_PIXEL_ID` unset under the `dev` environment to keep the
  Pixel off (same as GA in dev), or set a test Pixel's ID if you want one.
- **Local / devbox:** leave `VITE_META_PIXEL_ID` unset in `apps/web/.env.local`.
  To test the tag locally, set `VITE_META_PIXEL_ID=1445134643714482` (or a test
  Pixel) for the `webstore-fe` service and rebuild.

## Verify

1. **Meta Pixel Helper** (Chrome extension): open verani.bg, **accept the cookie
   banner**, and confirm `PageView` fires, then `ViewContent` on a product,
   `AddToCart`, `InitiateCheckout`, and `Purchase` after an order — each with the
   right `content_ids` / `value` / `currency`.
2. **Events Manager → Test Events**: enter the site URL and reproduce the funnel
   to see events arrive server-side with their params.
3. **Before** accepting consent: confirm **no** events are sent (the Pixel stays
   in `revoke`).

## Housekeeping

- Add the Meta Pixel to the **privacy policy** (cookies used, purpose, consent as
  the legal basis) alongside the GA entry.
- Mark `Purchase` and `InitiateCheckout` as key conversions in Events Manager for
  ad optimization.

### Quick reference

| Thing | Value |
| --- | --- |
| Pixel ID (prod) | `1445134643714482` |
| Enable var (runtime) | `META_PIXEL_ID` (`.devbox/project.yaml`, prod `webstore-fe`) |
| Enable var (build/local) | `VITE_META_PIXEL_ID` |
| Consent | revoked by default → granted on cookie-banner accept |

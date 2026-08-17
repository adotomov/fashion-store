# Google Analytics 4 — Implementation Plan

Status: **implemented** (Phases 1–6 landed; awaiting deploy + GA property setup).
This document is the engineering plan for introducing GA4 into the storefront.
The operator-facing "how to set up the GA property" steps live in
[google-analytics-setup.md](google-analytics-setup.md).

Remaining before real data flows: (1) deploy so the prod Measurement ID
(`G-T02R7XERDV`) ships in the bundle; (2) create/verify the GA4 property per the
setup guide; (3) add Bulgarian translations for the new `consent.*` keys in the
usual seed pass.

## Goals

Measure the health of the store and the full purchase funnel:

| Business question | GA4 mechanism |
| --- | --- |
| Home-page visits | `page_view` on `/` (automatic) |
| Catalog browsing | `page_view` on `/shop` + `view_item_list` |
| Each product page — number of visits | `view_item` + `page_view` on `/products/:id` |
| Shopping-cart activity | `add_to_cart`, `remove_from_cart`, `view_cart` |
| User registrations | `sign_up` (new user) / `login` (returning) |
| Session duration | automatic (`user_engagement`) — no code |
| Geolocation | automatic (GA derives country/region/city from IP) — no code |
| Full ecommerce funnel | `select_item` → `begin_checkout` → `add_shipping_info` → `add_payment_info` → `purchase` |
| Wishlist interest | `add_to_wishlist` |

## Decisions (locked)

- **Integration: `gtag.js` in-code.** A small typed analytics module fires GA
  events directly from the React app. No Google Tag Manager — all event logic
  stays in version control next to the code that triggers it.
- **Consent: banner + Google Consent Mode v2.** verani.bg serves the EU
  (Bulgaria), so analytics storage is **denied by default**; GA loads in
  anonymized/cookieless mode and is upgraded to full tracking only after the
  visitor accepts. A lightweight consent banner is part of this work.
- **Measurement ID** ships as a build-time `VITE_GA_MEASUREMENT_ID`, following
  the exact pattern already used for `VITE_REVOLUT_ENV` /
  `VITE_GOOGLE_CLIENT_ID` (GitHub Actions env → `--build-arg` → Dockerfile
  `ARG`/`ENV` → Vite inlines at build).

## Architecture notes (grounding)

- Single HTML document: [apps/web/app/root.tsx](../apps/web/app/root.tsx) —
  the one place to inject the gtag bootstrap + Consent Mode defaults, and to
  hang a route-change `page_view` effect.
- React Router v7 SPA: navigations are client-side, so `page_view` must be sent
  manually on route change (GA enhanced measurement's history-based page
  tracking is unreliable for this setup).
- Money is **EUR** in minor units ([apps/web/app/lib/money/money.ts](../apps/web/app/lib/money/money.ts)).
  GA `value` = `amount_minor / 100`, `currency: "EUR"`. (BGN is a display-only
  conversion; do not report it to GA.)
- Registration is Google-based: a brand-new user finishes with
  `completePhoneSetup` in
  [apps/web/app/features/auth/AuthContext.tsx](../apps/web/app/features/auth/AuthContext.tsx) —
  that is the `sign_up` boundary; a returning Google login is `login`.
- The web frontend sets **no CSP**, so the GA script is not blocked. (The
  restrictive CSP in `apps/api/.../middleware.go` is on the JSON API only. If a
  CSP is ever added to the web app, allowlist `*.googletagmanager.com` in
  `script-src` and `*.google-analytics.com` + `*.analytics.google.com` in
  `connect-src`/`img-src`.)
- i18n uses `t(key, englishDefault)` and works with inline defaults before the
  bg strings are seeded ([apps/web/app/features/i18n/LanguageContext.tsx](../apps/web/app/features/i18n/LanguageContext.tsx)),
  so consent-banner copy can land immediately and get Bulgarian translations in
  the usual seed-map pass.

## Event map — hook points

Each row is where the event fires in code.

| Event | Trigger / file | Key params |
| --- | --- | --- |
| `page_view` | route change effect in `root.tsx` | `page_path`, `page_title` |
| `view_item_list` | shop grid load, [routes/shop.tsx](../apps/web/app/routes/shop.tsx) | `item_list_id`, `item_list_name`, `items[]` |
| `select_item` | product-card click, [components/ecommerce/ProductCard.tsx](../apps/web/app/components/ecommerce/ProductCard.tsx) | `item_list_name`, `items[1]` |
| `view_item` | product page load, [routes/product-detail.tsx](../apps/web/app/routes/product-detail.tsx) | `currency`, `value`, `items[1]` |
| `add_to_cart` | `addItem`, [features/cart/CartContext.tsx](../apps/web/app/features/cart/CartContext.tsx) | `currency`, `value`, `items[1]` |
| `remove_from_cart` | `removeItem`, `CartContext.tsx` | `currency`, `value`, `items[1]` |
| `view_cart` | cart page load, [routes/cart.tsx](../apps/web/app/routes/cart.tsx) | `currency`, `value`, `items[]` |
| `add_to_wishlist` | [features/wishlist/WishlistContext.tsx](../apps/web/app/features/wishlist/WishlistContext.tsx) | `currency`, `value`, `items[1]` |
| `begin_checkout` | checkout mount / `details` step, [features/checkout/CheckoutFlow.tsx](../apps/web/app/features/checkout/CheckoutFlow.tsx) | `currency`, `value`, `items[]` |
| `add_shipping_info` | leaving `delivery` step, `CheckoutFlow.tsx` | `shipping_tier` (delivery method) |
| `add_payment_info` | leaving `payment` step, `CheckoutFlow.tsx` | `payment_type` |
| `purchase` | order placed / `confirmation`, `CheckoutFlow.tsx` | `transaction_id`, `value`, `currency`, `tax`, `shipping`, `items[]` |
| `sign_up` | new user in `completePhoneSetup`, `AuthContext.tsx` | `method: "Google"` |
| `login` | returning `loginWithGoogleIdToken`, `AuthContext.tsx` | `method: "Google"` |

Automatic (GA4 enhanced measurement, no code): `session_start`, `first_visit`,
`user_engagement` (session duration), `scroll`, outbound-link clicks, plus
country/region/city geolocation.

> **`purchase` de-duplication:** the Revolut card flow can re-render the
> confirmation step (payment poll, reload). Fire `purchase` once per
> `transaction_id` by remembering sent order IDs in `sessionStorage` so a
> refresh of the confirmation page does not double-count revenue.

## Work breakdown

### Phase 1 — Bootstrap, config, Consent Mode defaults
- Add `VITE_GA_MEASUREMENT_ID` to
  [apps/web/Dockerfile](../apps/web/Dockerfile) (`ARG`+`ENV`, empty default so
  local/devbox ships GA-off), and to the `env:` + `--build-arg` blocks of
  [.github/workflows/deploy-prod.yml](../.github/workflows/deploy-prod.yml) and
  [deploy-dev.yml](../.github/workflows/deploy-dev.yml) (dev can use a separate
  test property or stay empty).
- In `root.tsx` `Layout`, inject (only when the ID is set):
  1. Consent Mode v2 **defaults = denied** (`ad_storage`, `analytics_storage`,
     `ad_user_data`, `ad_personalization`) with `wait_for_update`, plus
     `url_passthrough` and `ads_data_redaction`.
  2. the gtag loader `<script async>` for the Measurement ID.
  3. `gtag('config', ID, { send_page_view: false })` — we send page views
     ourselves on route change.
- When the ID is unset (local dev), inject nothing → analytics is a no-op.

### Phase 2 — Typed analytics module
- New `apps/web/app/lib/analytics/` :
  - `gtag.ts` — thin `track(event, params)` wrapper + `setConsent(granted)`;
    all guarded so calls before the script loads (or with GA off) are safe.
  - `ecommerce.ts` — `toItem(product, opts)` mapping `StorefrontProduct` /
    variant → GA `items[]` (`item_id`, `item_name`, `price`, `quantity`,
    `item_category`, `item_variant`), and typed helpers per event
    (`viewItem`, `addToCart`, `beginCheckout`, `purchase`, …).
- Keeps every call site one-line and type-checked; centralizes the EUR/minor
  → major conversion.

### Phase 3 — page_view on navigation
- A `usePageViews()` hook (react-router `useLocation`) fires `page_view` on
  path change; mounted once in `root.tsx`. Guarded so it no-ops until GA + a
  consent decision exist.

### Phase 4 — Consent banner (Consent Mode v2)
- `apps/web/app/features/consent/` : `ConsentContext` + `ConsentBanner`.
  - Persist the choice (`accepted` / `rejected`) in a first-party cookie /
    `localStorage`; re-show only when unset.
  - Accept → `gtag('consent','update',{ analytics_storage:'granted' })` (ad_*
    stay denied — the store runs no ad tags today).
  - Banner copy via `t()` with English defaults + a "Manage/Reject" control;
    link to the existing privacy policy ([routes/legal/privacy.tsx](../apps/web/app/routes/legal/privacy.tsx)).
  - Mount `<ConsentBanner/>` in `root.tsx` inside the provider tree.
- Add bg translations for the new keys in the usual seed-map pass.

### Phase 5 — Wire ecommerce events
- Add the one-line typed calls at each hook point from the event-map table.
- Cart events live in `CartContext` so every add/remove path is covered once.
- `purchase` fires on the confirmation step with `sessionStorage`
  de-duplication.

### Phase 6 — Auth events
- `sign_up` / `login` in `AuthContext` (distinguish first-time vs returning by
  the same signal that routes a new user into `completePhoneSetup`).

### Phase 7 — Verify
- `npm run typecheck` (web) and a build.
- Local: set `VITE_GA_MEASUREMENT_ID` to a **test** property, run devbox, and
  confirm events land in GA **Realtime** + **DebugView** (see setup doc);
  confirm no events fire before consent is granted, and that
  `add_to_cart`/`begin_checkout`/`purchase` carry correct `value`/`currency`.
- Confirm `purchase` is not double-counted on a confirmation-page reload.

## Explicitly out of scope (future)
- Server-side `refund` events via the Measurement Protocol from the admin
  refund flow (revenue accuracy on returns). Noted for later.
- Google Ads / Meta Pixel conversion tags (would be the reason to revisit GTM).
- A/B testing / GA4 audiences export to BigQuery.

## Privacy / compliance checklist
- Consent denied by default; analytics storage only after opt-in (Consent Mode v2).
- No PII sent to GA (no email, name, phone, address in any event or user
  property). `transaction_id` is the order id/number only.
- Privacy policy updated to disclose GA usage, cookies, and the legal basis
  (consent) — content change tracked with the legal docs.
- Consider enabling GA's IP-anonymization-equivalent defaults and a shortened
  data-retention window (setup doc).

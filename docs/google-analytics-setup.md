# Google Analytics 4 — Setup Guide (operator)

Step-by-step instructions for creating and configuring the Google Analytics 4
(GA4) property for **verani.bg**, and for wiring the Measurement ID into the
build. This is the account/console side; the code side is described in
[analytics-plan.md](analytics-plan.md).

Do this once for **prod**, and (optionally) a second time for a separate
**dev/test** property so staging traffic never pollutes real numbers.

---

## 1. Create the GA4 property

1. Sign in at <https://analytics.google.com> with the Google account that should
   own the store's analytics (use a business/shared account, not a personal one).
2. **Admin** (bottom-left gear) → **Create** → **Property**.
3. Property name: `Verani Webstore — Prod` (make a second `… — Dev` later if you
   want a test stream).
4. Reporting time zone: **Europe/Sofia**. Currency: **Euro (EUR)** — this must
   match what the code reports so revenue totals are correct.
5. Business details: pick "Shopping / Retail" and your size — this only tunes
   suggestions, nothing functional.
6. Business objectives: choose **"Examine user behavior"** and **"Generate
   leads / Drive online sales"** so the ecommerce reports are enabled.
7. Accept the GA terms and the Data Processing Amendment (required for the EU).

## 2. Create a Web data stream

1. In the new property: **Admin → Data streams → Add stream → Web**.
2. Website URL: `https://verani.bg`. Stream name: `verani.bg`.
3. Leave **Enhanced measurement ON** (this auto-collects page views, scrolls,
   outbound clicks, site search, and — importantly — **session duration** and
   **geolocation**, with no code).
4. After creating, copy the **Measurement ID** — it looks like `G-XXXXXXXXXX`.
   **This is the only value the code needs.**

> Repeat sections 1–2 for a `Verani Webstore — Dev` property to get a second
> `G-…` for staging, if desired.

## 3. Configure data retention & privacy (EU)

1. **Admin → Data settings → Data retention**: set **Event data retention** to
   the longest allowed if you want history (14 months), or keep the default per
   your privacy policy. Turn on "Reset user data on new activity" as preferred.
2. **Admin → Data settings → Data collection**: you do **not** need Google
   Signals for basic ecommerce; leave it **off** unless you later run Google Ads
   remarketing (it adds consent obligations).
3. Because the site uses **Consent Mode v2** (analytics denied until the visitor
   accepts the cookie banner), GA will model/handle pre-consent traffic
   correctly — no extra console setting is required for this to work.

## 4. Give the Measurement ID to the build

The site reads the ID at **build time** as `VITE_GA_MEASUREMENT_ID` (same
mechanism as the Revolut/Google env vars).

- **Prod (GitHub Actions):** in
  [.github/workflows/deploy-prod.yml](../.github/workflows/deploy-prod.yml), add
  the ID under `env:` and pass it as a `--build-arg` to the webstore-fe image
  build (alongside `VITE_REVOLUT_ENV`). The ID is not secret, so it can live in
  the workflow file (or a repo variable if you prefer).

  ```yaml
  # env:
  VITE_GA_MEASUREMENT_ID: G-XXXXXXXXXX
  # docker build ...
  --build-arg VITE_GA_MEASUREMENT_ID="${{ env.VITE_GA_MEASUREMENT_ID }}" \
  ```

- **Dev:** same in `deploy-dev.yml`, using the Dev property's ID (or leave it
  unset so dev ships GA-off).
- **Local:** leave `VITE_GA_MEASUREMENT_ID` **unset** in `apps/web/.env.local`.
  With no ID, the analytics code is a no-op and nothing is sent.

Once merged, the next prod deploy bakes the ID into the frontend bundle.

## 5. Mark conversions (key events)

After the code is deployed and events start arriving (may take up to ~24h to
appear as options, faster in Realtime):

1. **Admin → Events → Key events** (formerly "Conversions").
2. Mark these as **key events**:
   - `purchase` — the primary revenue conversion.
   - `begin_checkout` — top-of-funnel intent.
   - `sign_up` — new registrations.
   - (optional) `add_to_cart`, `add_shipping_info`, `add_payment_info` if you
     want them in funnel exploration.

## 6. Verify data is flowing

Use **two** views while testing:

1. **Realtime** (Reports → Realtime): open verani.bg in a browser, **accept the
   cookie banner**, and browse a product / add to cart — you should see yourself
   and the events appear within seconds.
2. **DebugView** (Admin → DebugView): to see the full event stream with
   parameters for a single session, enable debug mode — install the
   [Google Analytics Debugger](https://chrome.google.com/webstore) Chrome
   extension, or append `?_dbg=1`-style debugging via the GA debug extension,
   then reproduce the funnel. Confirm each event carries the right params:
   - `view_item` / `add_to_cart`: `currency = EUR`, `value` = price in euros.
   - `purchase`: correct `transaction_id`, `value`, `tax`, `shipping`, `items[]`.
   - **Before** accepting consent: **no** analytics events should be recorded.

## 7. Confirm the ecommerce reports

Give it a day of real traffic, then check:

- **Reports → Life cycle → Monetisation → Ecommerce purchases** — product-level
  views, add-to-carts, and purchases.
- **Reports → Life cycle → Engagement → Pages and screens** — home/catalog/
  product page visit counts and **average engagement time** (session duration).
- **Reports → User → User attributes → Demographic details** and **Tech →** for
  **geolocation** (country/region/city) and device breakdowns.
- **Explore → Funnel exploration** — build a funnel:
  `view_item → add_to_cart → begin_checkout → add_payment_info → purchase`
  to see drop-off at each step.

## 8. Housekeeping

- Add GA4 to the **privacy policy** (cookies used, purpose, consent as the legal
  basis, data retention). This ships with the legal-doc updates.
- Grant teammates access under **Admin → Property access management** with the
  least role they need (Viewer/Analyst/Editor).
- If you later add Google Ads or Meta Pixel, that's the point to reconsider
  Google Tag Manager (see the plan's out-of-scope note).

---

### Quick reference

| Thing | Value |
| --- | --- |
| Property currency | EUR |
| Time zone | Europe/Sofia |
| Stream URL | https://verani.bg |
| Build var | `VITE_GA_MEASUREMENT_ID = G-XXXXXXXXXX` |
| Key events | `purchase`, `begin_checkout`, `sign_up` |
| Consent | denied by default (Consent Mode v2) → granted on banner accept |

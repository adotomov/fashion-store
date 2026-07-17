# Revolut Card Payments — Integration Plan

Status: **Phases 1–4 landed** (DB migrations + config/secrets; gateway reshape + service
split; webhook + admin refund endpoint + sweeper; frontend widget + confirmation poll +
admin refund/audit UI). Only Phase 5 (enablement) remains. This doc is the source of truth
for integrating Revolut Merchant card payments — plus Apple Pay / Google Pay — into the
existing storefront checkout.

Decisions (locked):

- **UX:** embedded on-site widget (`@revolut/checkout`), not a hosted redirect. Card
  data never touches our servers → PCI SAQ A.
- **Capture:** auto-capture on payment. Order becomes `paid` on the confirmed webhook.
- **Refunds:** in-app admin refunds (endpoint + webhook + admin UI), partial-capable.
- **Envs:** dev → Revolut **Sandbox**; prod → Revolut **Live**. Strict isolation, driven
  by `REVOLUT_MODE`; prod fails closed at boot if not `prod` + credentials present.

---

## 1. Why the current flow must change

Today [`checkout/application/service.go`](../apps/api/internal/modules/checkout/modules)
`PlaceOrder` runs the whole checkout **synchronously** and takes a raw PAN
(`MockRevolutGateway.Charge`). Three reasons that can't ship as-is:

1. Collecting raw card numbers on our server is PCI SAQ D. The embedded widget tokenizes
   client-side; we only ever see a per-order token.
2. Apple Pay / Google Pay are only available through the Revolut SDK.
3. Real confirmation is **asynchronous** — it arrives via webhook, not inline.

So `card_online` moves to an **initiate → pay → webhook-confirm** flow. The two non-card
methods (`cash_on_delivery`, `card_on_easybox`) keep the current synchronous path.

---

## 2. Target flow (card_online)

```
Customer            Web (SDK)          API                          Revolut
  | complete order --->|  POST /checkout ------------------------>  |
  |                    |   validate, reserve stock,
  |                    |   order = pending_payment,
  |                    |   POST /api/orders (capture=automatic) -->  | create order
  |                    |<-- {order_number, revolut_order_token} ---- | id + token
  | card/Apple/GPay -->| mount widget(token)
  |                    |----------- pay (3DS + capture) ---------->  |
  |                    |<-- success (UX hint only) --|
  | -> /order/{no} (poll "confirming…")               |<-- webhook ORDER_COMPLETED
  |                    |   verify sig, GET /api/orders/{id} (authoritative),
  |                    |   commit stock, order=paid, book shipment, invoice, clear cart
```

**The client success callback is never trusted as truth.** The order flips to `paid` only
when the signature-verified webhook — re-confirmed via `GET /api/orders/{id}` — says so.
A short frontend poll on the confirmation page absorbs webhook lag.

---

## 3. Implementation phases

### ✅ Phase 1 — migrations + config/secrets scaffolding (done)

- `20260717120000_add_revolut_payment_states.sql` — new `orders.status` values
  (`pending_payment`, `payment_failed`, `refunded`, `partially_refunded`); relaxed
  `order_payments.status`; `provider_order_id` (unique, partial), `captured_minor`,
  `refunded_minor`, `updated_at`.
- `20260717120100_create_payment_webhook_events_and_refunds.sql` —
  `payment_webhook_events` (webhook idempotency ledger) and `order_refunds`.
- `20260717130000_create_payment_transactions.sql` — **append-only payment audit ledger**
  (`initiated` / `captured` / `failed` / `refunded`), written in the same DB transaction as
  the `order_payments` state change it records, so the two can't drift. `MarkPaid` /
  `MarkPaymentFailed` guard on the status transition (a duplicate webhook/sweep appends no
  second row). Queryable via `ordersService.ListPaymentTransactions(orderID)`; admin UI
  surfacing is Phase 4. Covered by an integration test in
  [`orders/infrastructure`](../apps/api/internal/modules/orders/infrastructure/postgres_repository_test.go).
- [`internal/app/config.go`](../apps/api/internal/app/config.go) — `PaymentsConfig`
  (`REVOLUT_MODE`, `REVOLUT_API_KEY`, `REVOLUT_WEBHOOK_SECRET`, `REVOLUT_API_VERSION`),
  `RevolutBaseURL()`, and prod fail-closed guards.
- Terraform (dev + prod): `revolut_api_key` / `revolut_webhook_secret` secret containers
  (values added out-of-band), SA accessor grants, `revolut_mode` / `revolut_api_version` /
  `revolut_enabled` variables, and Cloud Run env wiring (secrets gated on
  `revolut_enabled`).

### ✅ Phase 2 — backend gateway + service split (done)

- Replaced `PaymentGateway.Charge(CardInput)` with `CreateOrder` / `GetOrder` / `Refund`;
  **deleted `CardInput`, `ChargeInput/Result`, `cardRequest`, and all PAN plumbing.**
- [`checkout/infrastructure/revolut_gateway.go`](../apps/api/internal/modules/checkout/infrastructure/revolut_gateway.go)
  — real Merchant API client (`Authorization: Bearer`, `Revolut-Api-Version`,
  `capture_mode=automatic`, `merchant_order_ext_ref=order_number`). `MockRevolutGateway`
  reimplemented on the new interface (stateful, so a simulated webhook can finalize in
  dev). Real vs mock selected on `RevolutAPIKey != ""` at
  [`modules.go`](../apps/api/cmd/api/modules.go).
- `PlaceOrder` now returns `PlaceOrderResult` (either a placed pay-on-delivery `Order` or a
  `PaymentInitiation` with the widget token). Card path: validate → reserve → open Revolut
  order → persist `pending_payment` + `order_payments.pending` → clear cart, return token.
  `FinalizePaidOrder` (idempotency-guarded: re-fetch state, verify amount, commit stock,
  `MarkPaid`, book shipment + invoice), `FailPayment` (release + `payment_failed`), and
  `RefundOrder` (Revolut refund + `RecordRefund`, rolls order to refunded /
  partially_refunded) all added.
- orders module gained the supporting persistence: `provider_order_id` / `captured_minor` /
  `refunded_minor` on `order_payments`, and repo methods `FindByProviderOrderID`,
  `MarkPaid`, `MarkPaymentFailed`, `GetOrderPaymentContext`, `RecordRefund`.

  Not yet wired to callers: `FinalizePaidOrder` / `FailPayment` / `RefundOrder` are exercised
  in Phase 3 (webhook + admin endpoint). Until then, an online-card order created in dev
  (mock gateway) stays `pending_payment` — expected.

### ✅ Phase 3 — transport + webhook + sweeper (done)

- `POST /api/v1/checkout` already returns the token payload for card orders (Phase 2).
- **Webhook** `POST /webhooks/revolut` mounted at the router root via a new
  `RootRouteRegistrar` seam ([router.go](../apps/api/internal/app/router.go)) — outside
  `/api/v1` and its auth, and untouched by the auth rate-limiter (that only wraps auth
  routes). Reads the raw body, verifies `Revolut-Signature` (`v1=<hmac_sha256_hex>` over
  `v1.{Revolut-Request-Timestamp}.{rawBody}`) with `REVOLUT_WEBHOOK_SECRET`, rejects stale
  timestamps (±5 min), fails closed when no secret is configured. Idempotency ledger
  (`payment_webhook_events`) via `Seen`/`Record` recorded only after success, so a transient
  failure returns 5xx and Revolut redelivers. `ORDER_COMPLETED` → `FinalizePaidOrder`,
  `ORDER_CANCELLED`/`ORDER_PAYMENT_FAILED` → `FailPayment`; `FinalizePaidOrder` re-fetches
  authoritative state before settling. Covered by
  [webhook_test.go](../apps/api/internal/modules/checkout/transport/http/webhook_test.go).
- `POST /api/v1/admin/orders/{id}/refund` behind `requireAdmin` → `RefundOrder`
  (`{amount_minor, reason}`; positive amount ≤ remaining refundable).
- **Sweeper** runs in the API process (`RunPaymentSweeper`, alongside the fulfillment
  poller) — every 5 min it reconciles `pending_payment` orders older than 30 min: re-fetch
  from Revolut, finalize a recovered completion, else `FailPayment` + release stock.

### ✅ Phase 4 — frontend (done)

- Added `@revolut/checkout@^1.1.25`; deleted the raw card form and the `Card` type.
- [`RevolutPaymentStep`](../apps/web/app/features/checkout/RevolutPaymentStep.tsx):
  dynamically imports the SDK (client-only, SSR-safe), instantiates with the order token +
  `VITE_REVOLUT_ENV` (`sandbox|prod`, default `sandbox`), mounts the embedded card field and
  the Apple/Google Pay payment-request button (auto-hidden when unsupported) from one
  instance. On widget success it polls `getOrderPaymentStatus(order_number)` until the order
  leaves `pending_payment`, so the UI reflects webhook-driven settlement.
- [`CheckoutFlow`](../apps/web/app/features/checkout/CheckoutFlow.tsx): `placeOrder` now
  returns a discriminated `{ kind: "placed" | "payment_required" }`; card orders render the
  widget in the payment step, pay-on-delivery is unchanged.
- Admin orders UI: **Refund** action (partial-capable modal), captured/refunded amounts, the
  full payment audit trail (`payment_transactions`), and all new statuses in the badge /
  filter / labels.
- Supporting endpoints added: public `GET /checkout/orders/{order_number}/status` (guest
  poll), admin `GET /admin/orders/{id}/transactions`, and captured/refunded on the admin
  order payment response.
- Only `VITE_REVOLUT_ENV` ships to the browser — no publishable key (the per-order token
  carries auth). It's a **build-time** var, baked into the web bundle by CI: the
  [web Dockerfile](../apps/web/Dockerfile) declares `ARG VITE_REVOLUT_ENV=sandbox` (default
  matches local devbox), and each deploy workflow passes it via `--build-arg` —
  [deploy-dev.yml](../.github/workflows/deploy-dev.yml) `sandbox`,
  [deploy-prod.yml](../.github/workflows/deploy-prod.yml) `prod`.

### ☐ Phase 5 — enablement

Populate sandbox secrets → register sandbox webhook → sandbox E2E → populate live secrets
→ register live webhook → one small real transaction → flip `revolut_enabled = true`.

---

## 4. Transactions, idempotency, failure handling

- **Stock:** reserve at initiate, commit only on the confirmed webhook. The sweeper +
  Revolut re-fetch guarantee every `pending_payment` reservation is eventually committed
  or released.
- **Idempotency:** `payment_webhook_events` insert-first (`ON CONFLICT DO NOTHING`) makes
  finalize exactly-once; one Revolut order per attempt (keyed by `order_number`) prevents
  double charges.
- **Authoritative state:** always `GET /api/orders/{id}` and compare amount+currency
  before marking `paid`; never trust webhook body values.
- **Atomicity:** finalize (commit stock + order status + payment row) in one pgx tx;
  shipment/invoice stay best-effort after commit, as today.

---

## 5. Revolut Merchant configuration checklist

Do this **twice — Sandbox (→ dev) and Live (→ prod)**; keys are not interchangeable.

1. Activate the account (Live): business details, payouts/settlement, settlement currency = EUR.
2. Generate the Merchant API **Secret key** → `fs-<env>-revolut-api-key` (sandbox vs live).
3. Enable payment methods: Cards, Apple Pay, Google Pay, 3-D Secure.
4. Apple Pay **domain registration**: `verani.bg` + `www.verani.bg` (prod), `dev.verani.bg` (dev).
5. Create a **webhook** per env:
   - dev → `https://api.dev.verani.bg/webhooks/revolut`
   - prod → `https://api.verani.bg/webhooks/revolut`
   Subscribe to `ORDER_COMPLETED`, `ORDER_AUTHORISED`, `ORDER_CANCELLED`, payment-failed,
   and refund events. Copy the signing secret → `fs-<env>-revolut-webhook-secret`.
6. Pin `REVOLUT_API_VERSION` to the version you validate against.

Populate secret values out-of-band, e.g.:

```
echo -n "<secret>" | gcloud secrets versions add fs-dev-revolut-api-key --data-file=-
```

Then set `revolut_enabled = true` for that env and redeploy.

---

## 6. Open items

- Single-currency (EUR minor units) assumed — matches `money.Money`.
- Auto-capture chosen; `ORDER_AUTHORISED` is informational unless we later switch to
  authorize-until-ship.
- Partial refunds supported in schema; UI may start full-refund-only.

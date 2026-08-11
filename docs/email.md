# Transactional email

How the store sends customer email, how to operate it, and what to check when
something doesn't arrive.

**Scope:** transactional only — welcome, order confirmation, shipping update and
payment failed. Marketing campaigns and newsletters are deliberately **not**
built. They carry GDPR consent and unsubscribe obligations that transactional
mail is exempt from, so they are a separate piece of work.

All mail is sent **From `info@verani.bg`**.

---

## How it works

Nothing is sent inline. Producers **enqueue** into an outbox and a background
dispatcher delivers it.

```
checkout / auth / fulfillment          notifications module
        │                                     │
        │  Enqueue(template, to, vars)        │
        └────────────────► email_messages ────┴──► dispatcher ──► SendGrid
                           (outbox table)            │
                                                     ├─ renders template (en/bg)
                                                     ├─ checks suppression list
                                                     └─ retries w/ backoff
```

**Why an outbox.** A SendGrid outage must never fail (or slow) a checkout, and
an order confirmation must not be lost because a single HTTP call failed. Rows
are retried with exponential backoff and dead-lettered after 6 attempts.

**Idempotency.** Every message carries a unique `dedupe_key` (e.g.
`order_confirmation:<order_id>`). Enqueueing the same event twice is a silent
no-op — this is what stops a redelivered payment webhook, or the shipment poller
re-seeing the same in-flight parcel every 15 minutes, from mailing twice.

**Crash safety.** The dispatcher claims rows with `FOR UPDATE SKIP LOCKED` and a
lease. If it dies mid-send, the lease lapses and the message becomes due again
rather than being stranded.

### The emails

| Template key | Trigger | Producer |
|---|---|---|
| `welcome` | First Google sign-in that creates the account (`IsNew`) | auth service |
| `order_confirmation` | COD order placed, or card payment settled | checkout service |
| `shipping_update` | Parcel first enters the carrier network | fulfillment poller |
| `payment_failed` | Card payment declined/cancelled/abandoned | checkout service |

### Templates

Copy lives in the **`email_templates`** table, keyed `(template_key, locale)`,
seeded in `en` and `bg`. Bodies are Go template fragments holding only the inner
content; the renderer wraps them in a shared branded layout that pulls the store
name and logo from store settings. A missing locale falls back to `en`.

Editing copy is a database change, not a deploy.

---

## Configuration

| Env var | Default | Meaning |
|---|---|---|
| `EMAIL_MODE` | `log` | `log` renders to the log and sends nothing; `sendgrid` delivers |
| `SENDGRID_API_KEY` | — | Required when `EMAIL_MODE=sendgrid` |
| `EMAIL_FROM` | `info@verani.bg` | Fixed per environment, never per message |
| `EMAIL_FROM_NAME` | `Verani` | Display name |
| `EMAIL_WEBHOOK_VERIFICATION_KEY` | — | SendGrid's base64 ECDSA **public** key; empty ⇒ webhook rejects everything |
| `EMAIL_DISPATCH_INTERVAL` | `15s` | Dispatcher poll cadence |
| `STOREFRONT_URL` | localhost | Base URL linked from email bodies |
| `PUBLIC_API_URL` | — | Absolute base for the logo image; empty ⇒ layout shows the store name as text |

Locally and in devbox `EMAIL_MODE=log` is the default, so **development needs no
SendGrid account and cannot send real mail**. Set `LOG_LEVEL=debug` to see the
fully rendered HTML and text bodies.

In GCP, secrets are injected only when `email_enabled = true` (Terraform),
mirroring how `revolut_enabled` gates the payment credentials.

---

## Domain mail setup (verani.bg)

`verani.bg` is **registered at SuperHosting.bg**, but SuperHosting only acts as
registrar — DNS is delegated to **Google Cloud DNS**. The authoritative zone
lives in the **prod** project and holds every record; it is managed by Terraform
in [`infra/terraform/envs/prod/dns.tf`](../infra/terraform/envs/prod/dns.tf).
Change mail DNS **there**, not in the SuperHosting panel (which no longer serves
the zone).

There are **two independent senders and one inbox**, all on the same address:

| Concern | Handled by | Notes |
|---|---|---|
| **Inbound** `info@verani.bg` | **Google Workspace** | Mailbox is `boutiqueverani@gmail.com` configured as `info@verani.bg`. Migrated off SuperHosting on 2026-08-09 — the SuperHosting mailbox no longer receives. |
| **Outbound — human mail** | **Google Workspace** send-as | Replies, manual correspondence. |
| **Outbound — transactional** | **SendGrid** | The app pipeline documented in this file. Kept separate because Workspace caps at ~2k/day and prohibits bulk. |

Both senders send **From `info@verani.bg`**, so both must be authorised — that's
what the SPF and DKIM records below are for.

### The DNS records

| Record | Name | Value | Purpose |
|---|---|---|---|
| `MX` | `verani.bg` | `1 smtp.google.com.` | Inbound → Google Workspace |
| `TXT` (SPF) | `verani.bg` | `v=spf1 include:_spf.google.com include:sendgrid.net ~all` | Authorises **both** Workspace and SendGrid. Shares the apex TXT set with the Google site-verification string (DNS permits one TXT set per name). |
| `TXT` (DKIM — Workspace) | `google._domainkey.verani.bg` | `v=DKIM1; k=rsa; p=…` | Signs Workspace mail so it aligns under DMARC. 2048-bit key → published as **two character-strings** in one record (over the 255-char TXT limit; resolvers concatenate them). |
| `CNAME`s (DKIM — SendGrid) | per `sendgrid_dns_records` | SendGrid-generated | SendGrid domain authentication (DKIM + branded links). Empty until the sending domain is authenticated. |
| `TXT` (DMARC) | `_dmarc.verani.bg` | `v=DMARC1; p=none; …` | Report-only for now; tighten to quarantine/reject once reports show SPF+DKIM aligned. |

**Verifying a change** (TTL is 300s):

```sh
dig +short MX verani.bg
dig +short TXT verani.bg
dig +short TXT google._domainkey.verani.bg
```

To confirm alignment end-to-end, send from Gmail (as `info@verani.bg`) to an
outside address and check the received headers (**Show original**): SPF, DKIM
(`d=verani.bg`) and DMARC should all pass.

---

## Enabling it (first-time setup)

Order matters — **do not enable sending before DNS authentication resolves**, or
early mail lands in spam and damages the domain's reputation.

1. **Sign up for SendGrid** via the GCP Marketplace (keeps billing in GCP).
2. **Authenticate the sending domain** in SendGrid. It generates CNAMEs.
3. **Publish DNS** (prod project owns the `verani.bg` zone):
   - Put the generated CNAMEs in `sendgrid_dns_records` in
     `infra/terraform/envs/prod/`.
   - Confirm `spf_record` — see the warning below.
   - `_dmarc` starts at `p=none`.
   - ⚠️ **Do not touch the `MX` record.** `info@verani.bg` *receives* at
     Google Workspace (see "Domain mail setup" above); SendGrid only sends.
4. **Populate secrets** (values never go in Terraform state):
   ```sh
   gcloud secrets versions add fs-dev-sendgrid-api-key --data-file=-
   gcloud secrets versions add fs-dev-email-webhook-verification-key --data-file=-
   ```
5. **Point SendGrid's Event Webhook** at `https://api.dev.verani.bg/webhooks/sendgrid`
   and enable *Signed Event Webhook*.
6. **Flip `email_enabled = true`** and apply.
7. Verify with [mail-tester.com](https://www.mail-tester.com) that SPF, DKIM and
   DMARC all pass and the visible From is `info@verani.bg`.

### ⚠️ SPF is a single record

A domain may publish only **one** SPF record, and it lives in the same apex TXT
record set as the Google site verification (DNS permits one TXT set per name —
this is why they share a resource in `dns.tf`).

The default is `v=spf1 include:_spf.google.com include:sendgrid.net ~all` —
`_spf.google.com` for Workspace send-as, `sendgrid.net` for transactional mail.
**Any additional provider that sends as `verani.bg` must have its `include:`
added too**, or that mail starts failing SPF.

---

## Operating it

### Suppression list

`email_suppressions` holds addresses we must never mail again. It is fed by the
event webhook:

- **Hard bounce** (`type=bounced`) → suppressed. The mailbox doesn't exist.
- **Spam complaint** → suppressed. The strongest possible signal to stop.
- **Soft bounce** (`type=blocked`) → *not* suppressed; usually transient (full
  mailbox, greylisting), and suppressing would cut off a real customer.

The dispatcher checks it immediately before each send, so an address suppressed
after a message was queued still stops that message.

To un-suppress (e.g. a customer fixed their mailbox):
```sql
DELETE FROM email_suppressions WHERE email = 'customer@example.com';
```

### Metrics & alerts

`emails_sent_total{template}`, `emails_failed_total{outcome}` (outcome:
`retry|dead_letter|bounce|complaint`), `emails_suppressed_total`. Alert policies
fire on any sustained bounce/complaint rate and on any dead letter.

### Runbook: "a customer didn't get their email"

1. Find the message:
   ```sql
   SELECT id, template_key, status, attempts, last_error, provider_message_id, sent_at
   FROM email_messages WHERE to_email = 'customer@example.com'
   ORDER BY created_at DESC;
   ```
2. Interpret `status`:
   - **no row** → the producer never enqueued. Check the API logs for
     "failed to queue …" around the order/registration time.
   - `pending` → not yet sent; check `next_attempt_at` and `last_error`.
   - `sending` → in flight, or a lapsed lease that will retry.
   - `suppressed` → the address is on the suppression list (see above).
   - `failed` → dead-lettered or reported undeliverable; `last_error` says which.
   - `sent` → we handed it to SendGrid. Check SendGrid Activity for
     `provider_message_id`, then the recipient's spam folder.
3. To force a retry of a dead letter:
   ```sql
   UPDATE email_messages
   SET status = 'pending', attempts = 0, next_attempt_at = NOW()
   WHERE id = '<id>';
   ```

### Known gaps

- Delivery-method and payment-method labels inside emails are English-only,
  while the surrounding copy is localised.
- Emails are localised to the **store's** configured locale, not per-customer —
  accounts and orders don't carry a language preference yet.
- The invoice PDF is not attached to the order confirmation.

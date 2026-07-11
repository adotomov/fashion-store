# Production Readiness — fashion-store (verani.bg)

Status: **planning**. Current state = GCP **dev** environment fully built via Terraform
(`infra/terraform/envs/dev`): Cloud Run (api + webstore-fe), Cloud SQL Postgres 16,
Secret Manager, Cloud Storage, Artifact Registry, Cloud DNS zone, GitHub Actions via
Workload Identity Federation. This doc tracks what's needed to stand up **prod** and the
security posture going into it.

---

## 1. Security scan

Scan of the current codebase (`apps/api`, `apps/web`) and dev infra. Grouped by
protected / needs-work.

### ✅ Already protected

| Area | Finding |
|---|---|
| SQL injection | All repositories use parameterized queries (`$1`, `$2`…). No `fmt.Sprintf` string-built SQL anywhere. |
| Password/credential storage | No passwords — auth is Google OAuth only. Session tokens are 256-bit random, stored **only as SHA-256 hash** (`internal/shared/tokens`). Raw token never persisted. |
| OAuth token validation | Google ID tokens verified with the official `idtoken.Validate` incl. audience (client ID) check (`internal/platform/googleauth/verifier.go`). |
| Authorization / RBAC | `RequireAuth` + `RequireAdminAccess` with method-level roles: writes need `admin`; reads allow `admin`/`audit`/`accountant` (`modules/auth/transport/http/handler.go`). |
| Session logout/refresh | `Refresh` revokes the old session on rotation; `Logout` revokes. |
| Panic safety | `Recover` middleware returns a bare 500 — no stack trace / internal detail leaked to clients. |
| Secret handling | Secrets live in Secret Manager (DB URL, signing secret, client ID). No secrets committed: tracked `.env.local` files are empty / hold only a public `VITE_API_BASE_URL`. Terraform state & tfplan are gitignored. |
| Object storage auth | GCS client attaches ADC bearer tokens for real GCS (`internal/platform/storage/client.go`). *(This was a previously-flagged gap — now closed.)* |
| Slowloris / hung conns | HTTP server sets `ReadHeaderTimeout` / `ReadTimeout` / `WriteTimeout` (`internal/app/server.go`). |
| CI credentials | GitHub Actions authenticates via Workload Identity Federation — no static SA JSON keys in GitHub secrets. |

### ⚠️ Gaps to close before / at prod

| Pri | Gap | Detail & fix |
|---|---|---|
| **High** | CORS fails **open** | In `CORS()`, an empty allowlist reflects **any** `Origin` **with** `Access-Control-Allow-Credentials: true`. Prod must always set `CORS_ALLOWED_ORIGINS` (it does via TF), but the code should fail *closed* — deny when the list is empty — so a misconfig can't silently open it. |
| **High** | No rate limiting | Nothing throttles `/auth/*` (login/refresh) or any endpoint → brute-force / abuse / credential-stuffing surface. Add app-level limiter (e.g. `golang.org/x/time/rate` per-IP on auth routes) and/or Cloud Armor rate rules. |
| **High** | No WAF in front of Cloud Run | api + web are `allUsers`-invokable directly. Put an external HTTPS Load Balancer + **Cloud Armor** in front for WAF (OWASP rules), rate limiting, geo/IP controls. |
| **Med** | No security response headers | Missing HSTS, `X-Content-Type-Options: nosniff`, `X-Frame-Options`/CSP, `Referrer-Policy`, `Permissions-Policy`. Add a small headers middleware in `internal/app`. |
| **Med** | No request body size cap | No `http.MaxBytesReader` — large-payload memory DoS. Add a global max body middleware (JSON endpoints only need a few KB; media upload route gets a higher explicit cap). |
| **Med** | Session hardening (deferred set) | 30-day TTL is long; no refresh-token **rotation with reuse detection**; no "list/revoke my sessions" endpoint; token stored in `localStorage` (XSS-exfiltratable). See `auth-token-security-followups` memory. Shorten TTL + add session-revoke endpoint for prod; httpOnly-cookie migration is a bigger, separate decision. |
| **Med** | Payments / PCI scope | `card_online` payment method exists but **no PSP (Stripe) integration is wired** yet (no webhook, no PaymentIntent). Confirm **no raw card data touches our servers** before prod. When Stripe is added: verify webhook signatures, keep keys in Secret Manager, use a separate prod webhook endpoint. |
| **Low** | Dependency scanning | No automated `govulncheck` / `npm audit` / Dependabot in CI. Add before prod and run on a schedule. |
| **Low** | `AUTH_SIGNING_SECRET` | Generated & injected but the opaque-token scheme may not use it — audit and remove if dead, or document its use. |

---

## 2. Production environment plan (GCP)

### Structure
- New `infra/terraform/envs/prod`. **Refactor shared resources into `infra/terraform/modules/*`** first — dev is currently a single copy-pasted env; prod is the moment to modularize (per `40-gcp-infrastructure-context.md`).
- **Separate GCP project** for prod (hard isolation from dev billing/IAM/data). New Cloud SQL, buckets, secrets, Artifact Registry, service accounts, WIF pool.

### Config deltas vs dev
| Setting | dev | prod |
|---|---|---|
| `min_instance_count` | 0 (cold starts) | **≥ 1** (api), avoid cold-start latency |
| Cloud SQL | zonal, single | **regional HA**, automated backups + **PITR**, `deletion_protection = true` |
| `deletion_protection` (Cloud Run/SQL) | false | **true** |
| Buckets | media only, public-read | **media (public) + invoices (private)** separated; uniform bucket-level access |
| `CORS_ALLOWED_ORIGINS` | run.app + dev domain | **`https://verani.bg`** (or store subdomain) only |
| `SPEEDY_MODE` | fake/real | **real** |
| `APP_ENV` | dev | prod |
| Ingress | direct public Cloud Run | **HTTPS LB + Cloud Armor** in front |
| CI deploy | auto on `main` | **gated** (tag / GitHub Environment protection / manual approval) |

### Least-privilege (carry over from dev)
Per-service SAs: api → Cloud SQL Client + Secret Accessor + Storage Object Admin scoped to
its buckets only. No project-wide roles.

---

## 3. Domain & DNS — verani.bg

**Current live records** (SuperHosting.bg registrar; captured 2026-06-30):
- `@` A → GitHub Pages anycast IPs (185.199.108–111.153) — the **marketing landing page**
- `www` CNAME → `boutiqueverani.github.io.`
- `@` MX → `20 mx2.bgdns.net.` (SuperHosting email)
- No SPF/DKIM/DMARC/CAA found.

**Decision (made):** **Apex takeover** — `verani.bg` becomes the store; the GitHub Pages
marketing landing is retired/relocated. Implications:
- Apex `@` A records (currently → GitHub Pages `185.199.108–111.153`) get repointed to the
  store front end. Cloud Run domain mappings don't support a bare apex via CNAME, so the
  apex needs either an **external HTTPS Load Balancer** (static anycast IP via A/AAAA — the
  recommended path, also required for Cloud Armor) or Cloud DNS's apex CNAME-flattening
  equivalent. The **LB approach is preferred** here precisely because of the apex.
- `www.verani.bg` → redirect to apex (CNAME/redirect at the LB).
- `api.verani.bg` → the API (LB backend or Cloud Run domain mapping).
- **MX / email records must be preserved** unchanged (`20 mx2.bgdns.net.`) — apex takeover
  touches A/AAAA only, never the MX.
- Retire the GitHub Pages custom-domain config once traffic is cut over (avoid a dangling
  `boutiqueverani.github.io` claim on the apex).

**Cloud DNS:** the dev Terraform already builds a `verani-bg` Cloud DNS zone with the
existing records replicated + `dev`/`api.dev` CNAMEs. Nameserver cutover to Google
(`ns-cloud-a{1..4}.googledomains.com.`) is **the user's manual step at SuperHosting** and
must only happen after every live record (incl. MX/email) is confirmed replicated.
Alternatively, keep NS at SuperHosting and just add the store's records there — lower risk,
no cutover.

**Before any cutover:** re-run the `dig` audit (DNS drifts), and re-check GitHub Pages
custom-domain verification (may rely on a TXT challenge not visible in a plain sweep).

---

## 4. SSL / TLS

**No certificate purchase needed** — GCP auto-provisions and auto-renews managed certs.
Two options depending on §2 ingress choice:

- **Cloud Run domain mapping** → Google managed cert issued automatically once DNS points
  at `ghs.googlehosted.com` **and** verani.bg is a *verified* domain (Search Console).
  Domain verification is still pending (flagged in the dev plan).
- **External HTTPS LB** (recommended, pairs with Cloud Armor) →
  `google_compute_managed_ssl_certificate`, auto-renewed. LB also does HTTP→HTTPS redirect.

**Additional:**
- Enforce **HSTS** header once HTTPS is confirmed end-to-end (with care — HSTS on apex
  affects the GitHub Pages landing too if it shares the host).
- Cloud Run / LB are HTTPS-only; ensure no mixed content from the frontend.
- Optional: add a **CAA** record pinning issuance to Google's CA.

---

## 5. CI/CD topology

Both environments deploy from `main`; the difference is the gate.

| | Dev | Prod |
|---|---|---|
| Workflow | `.github/workflows/deploy-dev.yml` | `.github/workflows/deploy-prod.yml` |
| Trigger | push to `main` (auto) | push to `main` → **waits for approval** |
| GitHub Environment | `dev` (no reviewers) | `prod` (**required reviewers** = the gate) |
| Project | `project-538051b7…` (dev) | separate `verani-webstore-prod` |
| Front door | Cloud Run domain mapping (`dev.verani.bg`) | HTTPS LB + Cloud Armor (`verani.bg`) |
| Auth | Workload Identity Federation (per-project pool, no static keys) | same |

A single push to `main` therefore auto-ships dev and *queues* prod behind the approval.

### Terraform layout
- `infra/terraform/envs/dev` — existing; now only owns Cloud Run + domain mappings (DNS moved out).
- `infra/terraform/envs/prod` — **new**: HA Cloud SQL (regional + PITR), private `invoices`
  bucket, LB-only Cloud Run ingress, warm instances, `loadbalancer.tf` (LB + Cloud Armor),
  and the **authoritative `verani.bg` Cloud DNS zone** (holds prod + dev + MX records).
- Module extraction (`modules/*`) deferred — prod authored standalone to avoid refactoring
  the live dev state.

### Rollout runbook (order matters)
1. Create the prod GCP project + link billing; create the TF state bucket
   `gs://tfstate-verani-webstore-prod` (versioned).
2. Configure GitHub Environments: `dev` (no gate), `prod` (**required reviewers**).
3. `terraform apply` **`envs/prod` first** (creates the DNS zone + LB), then **`envs/dev`**
   (drops its old zone, adds domain mappings). Fill the prod workflow `env:` placeholders
   (`PROJECT_ID`/`WORKLOAD_IDENTITY_PROVIDER` project number/`DEPLOYER_SA`) from
   `terraform output`.
4. Verify `verani.bg` in Search Console (needed for the dev Cloud Run domain mappings).
5. Add authorized JS origins to the Google OAuth client(s): `https://dev.verani.bg`,
   `https://verani.bg`.
6. Cut over nameservers at SuperHosting to the prod zone's `dns_name_servers` output — only
   after confirming apex A, www A, api A, dev/api.dev CNAME, **and MX** are all present.
7. Wait for the managed SSL cert to go ACTIVE (needs the A records resolving to the LB IP).

---

## Pre-launch checklist (condensed)

**Security**
- [x] CORS fails closed on empty allowlist *(done — `internal/app/middleware.go`)*
- [x] Rate limiting on auth *(done — per-IP token bucket on `/auth`; Cloud Armor still TODO for cross-instance)*
- [x] Security-header middleware (HSTS, nosniff, frame, referrer, CSP) *(done — `SecurityHeaders`)*
- [x] Request body size cap *(done — `MaxBodyBytes`, 16 MiB)*
- [ ] Shorten session TTL + add session-revoke endpoint
- [ ] Confirm no raw card data server-side (PCI)
- [ ] `govulncheck` + `npm audit` / Dependabot in CI
- [ ] Cloud Armor / WAF in front of Cloud Run

**Infra**
- [ ] Separate prod GCP project *(project created + billing — manual)*
- [x] `envs/prod` Terraform authored (validates) — *apply is manual*; module extraction deferred
- [x] Cloud SQL HA + backups + PITR + deletion protection *(prod `sql.tf`)*
- [x] Invoices bucket private, separate from media *(prod `storage.tf`, reserved)*
- [x] min instances ≥ 1; prod secrets generated in Secret Manager *(prod `cloud_run.tf`/`secrets.tf`)*
- [x] Gated prod deploy workflow (separate WIF) *(`deploy-prod.yml` + prod `wif.tf`)*
- [ ] LB-only ingress verified end-to-end after apply

**Domain / SSL**
- [x] Store placement decided — **apex takeover** (verani.bg = store)
- [x] Authoritative DNS zone authored with all records incl. MX *(prod `dns.tf`)*
- [ ] verani.bg verified in Search Console *(manual — needed for dev domain mappings)*
- [x] Managed SSL cert authored (LB, apex+www+api) — *goes ACTIVE after DNS resolves*
- [x] HTTP→HTTPS redirect (LB) + HSTS header (app) *(prod `loadbalancer.tf` + `SecurityHeaders`)*
- [ ] Add store origins to Google OAuth authorized JS origins *(manual)*
- [x] `VITE_API_BASE_URL` + `CORS_ALLOWED_ORIGINS` set to prod/dev hosts *(workflows + `cloud_run.tf`)*
- [ ] Nameserver cutover at SuperHosting *(manual, after records confirmed)*

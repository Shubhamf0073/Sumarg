# 07 — Security and privacy

## 1. Phase 1 floor (must hold at go-live)

| # | Control | Where |
|---|---|---|
| S1 | **Tenant isolation**: forced RLS on every tenant table, composite FKs, tenant from the session only, pools refuse the wrong role, depguard keeps tenant code away from BYPASSRLS code | `platform/db`, schema, `.golangci.yml`, integration tests (two tenants) |
| S2 | **Parent scope**: every parent request checks the student/route/trip belongs to the guardian's children; live position only while that child's trip runs | `internal/parent`, tests |
| S3 | Staff passwords Argon2id, no login oracle, backoff not lockout *(Cargo)*; driver PIN with backoff and device binding | `internal/tenant/auth` |
| S4 | Parent OTP: HMAC-stored, 5 min, 5 tries, per-mobile and per-school rate limits, SMS only to invited guardians, same response otherwise; the code waits for the worker AES-GCM-sealed in the outbox and is wiped once settled | `internal/tenant/auth`, `internal/shared/otp`, `internal/worker` |
| S5 | Sessions: token `<tenant_id>.<token>`, only sha256 stored, revoke on logout/role change; apps keep the token in the OS keystore (`expo-secure-store`) | |
| S6 | Superadmin passkeys only; installer role limited to unit installation screens | `internal/admin` |
| S7 | **Units**: per-unit secret (argon2id in DB, encrypted NVS on device), MQTT ACL to own topics, tenant from DB not topic, max packet 64 KB, per-unit rate limit, JSON schema validation, unknown units refused and counted | `internal/ingest` |
| S8 | **Firmware**: Secure Boot v2 + flash encryption (release) + NVS encryption on production units; OTA images ECDSA-signed with an offline key and verified on the unit; rollback | docs/14 §8–9 |
| S9 | TLS everywhere: Cloudflare Full (strict) for the app; Let's Encrypt on `devices.` with ISRG Root X1 pinned; HSTS | Nginx, firmware |
| S10 | Origin exposure: `devices.` is DNS-only, so the server IP is public → host firewall allows 443 to the app vhost only from Cloudflare ranges, 443 for `devices.` (firmware path only) and 8883; everything else closed | ufw, Nginx |
| S11 | Input validation from huma tags; SQL only via sqlc; photos accepted only as JPEG ≤ 64 KB (magic bytes checked), served only through the API with `Content-Type: image/jpeg` + `X-Content-Type-Options: nosniff` | ingest, api |
| S12 | Secrets: `.env`/`db.env` split *(Cargo)*; FCM service-account file outside the image (chmod 600); SMS API key in `.env` | infra |
| S13 | Audit: tenant `audit_log` + `platform_audit_log`, append-only; manual boardings, voids, support views, unit allocation all audited | |
| S14 | Backups encrypted off-site *(Cargo)*; photos excluded from off-site backups (short-lived) | `backup.sh` |
| S15 | Supply chain: approved-deps list + hooks *(Cargo S26)*; firmware libraries pinned in `platformio.ini`; app deps via `expo install` | `.claude/` |
| S16 | App: no third-party analytics/ads SDKs; crash reporting only via EAS/OS-level reports without personal data | `mobile/` |
| S17 | Store-review login: a fixed OTP is accepted **only** for school code `demo`, only for `DEMO_REVIEW_MOBILES`, only while `DEMO_REVIEW_LOGIN=on`; every use is audit-logged; the demo tenant holds fictional data only; the code is rotated after each review | `internal/tenant/auth` (M9.3) |
| S18 | Onboarding data: school spreadsheets arrive over a channel the school controls (their email/drive link to Previa's role mailbox), are imported once and deleted from laptops within 7 days; import errors reports show row numbers, not children's details | `onboarding/` |

## 2. Phase 2 hardening
X.509 client certificates per unit (mutual TLS) · NTAG 424 DNA cards with SUN authentication (cloning-resistant) ·
BLE installer tool with signed sessions · anomaly detection on taps (same card on two buses) · penetration test of
the app API · MDM-style app attestation (Play Integrity / App Attest) for driver devices.

## 3. Children's data (DPDP Act 2023 + DPDP Rules 2025)

- **Roles:** the school is the data fiduciary; **Previa is its data processor** under a data processing agreement
  signed at onboarding (purpose, retention, security, breach notice within 72 h, deletion at the end).
- **Basis:** the DPDP Rules' Fourth Schedule exempts an educational institution's tracking and monitoring done in the
  interest of the safety of enrolled children, and a transport provider engaged by the institution when it tracks the
  real-time location of children travelling to and from it — from the verifiable-parental-consent and no-tracking
  rules. Sumarg stays inside that purpose:
  - location only during trips; parents see a bus only while their child's trip runs;
  - boarding events and a photo per tap only to confirm who boarded (no face templates, no recognition);
  - no profiling, advertising, analytics SDKs or sale of data; no secondary use (the sustainability module, Phase 3,
    uses bus data, not children's).
- **Minimisation:** the unit holds first names only; the app shows a parent only their own children; crew see names,
  stop and special-needs notes for their own trip only.
- **Retention:** photos 30 days, raw positions 30 days (both school-configurable within limits), boardings for the
  package + 1 year, left students anonymised after 1 year, login OTP challenges and code-SMS rows 24 h (docs/05 §9).
- **Rights:** parents can ask the school for their child's records; the school exports them from the portal
  (Phase 1: CSV of boardings + trip times for that student).
- **Breach:** platform incident runbook (docs/06 §9) includes notifying the school within 24 h so it can meet the
  72-hour reporting duty.
- Have the DPA and the in-app privacy notice reviewed by a lawyer before the first school (docs/09 L1).

## 4. Threats specific to this product

| Threat | Control |
|---|---|
| Card swapping / proxy taps | photo per tap; unknown-card alert; Phase 2 NTAG 424 DNA |
| Stalker uses a parent's phone/app | 90-day sessions bound to install; school can revoke a guardian; live only during trips |
| Fake unit floods the broker | per-unit secret; unknown uids refused before argon2; connect rate limit per IP; max units |
| Stolen unit | `retired` status refuses connect; flash encryption keeps the secret unreadable |
| Malicious firmware | Secure Boot v2 + signed OTA; signing key offline |
| Driver disables tracking | power-cut/tamper events; unit offline alert during trip windows |
| Data loss when offline | SD journal with seq; replay; gaps detected server-side |

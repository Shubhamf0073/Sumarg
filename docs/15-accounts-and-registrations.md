# 15 — Accounts and registrations

Everything Previa must register or buy before the first school goes live, in the order that avoids waiting.
Checked against public sources in Nov 2026; portals change — follow the portal when it disagrees. Secrets that come
out of these steps go only where docs/07 S12 says (server `data/secrets/`, `.env`, EAS secrets) — never in git.

## 0. Before anything: one company identity
Every account below checks the **legal name exactly as on the PAN/GST/incorporation certificate**. Prepare once:
- PDFs: company PAN, GST certificate, certificate of incorporation, address proof, board resolution or authorisation
  letter for the signatory, signatory's PAN/Aadhaar.
- Role mailboxes on `previasoftwares.com` (not personal Gmail): `appstore@`, `play@`, `dlt@`, `iot@`, `ops@`.
  Each account gets **two admins** (owner + backup) with 2-factor auth; recovery codes in the company password
  manager.
- A public **privacy policy** page and **support** page on `sumarg.previasoftwares.com` (both stores and DLT
  reviewers look for them). Draft text: docs/07 §3; lawyer review: docs/09 L1.

## 1. Master checklist

| # | Item | Cost | Typical wait | Blocks (plan) | Owner | Status |
|---|---|---|---|---|---|---|
| R1 | D-U-N-S number (Dun & Bradstreet) | free | days; then up to 2 business days before Apple sees it | R2, R3 | signatory | |
| R2 | Apple Developer Program — organization | US$99 / year | days–weeks (manual review) | M5.4 iOS builds, TestFlight | signatory + product | |
| R3 | Google Play Console — organization | US$25 one-time | identity verification | M5.4 Android release | signatory + product | |
| R4 | Expo / EAS organization | free tier to start | same day | M5.4 | product | |
| R5 | Firebase projects (prod + dev) + APNs key | free (FCM) | same day | M4.4 push | product | |
| R6 | Google Cloud billing + Maps keys | pay-as-you-go, budget alert | same day | M2.1 routes, M4.1 maps | product + finance | |
| R7 | DLT Principal Entity (one operator portal) | ₹5,900 incl. GST / year on Jio TrueConnect | ~3–7 days | R8, R9 | signatory | |
| R8 | DLT header(s) + content templates + URL whitelisting | included | ~1–3 days each | M1.3 parent OTP | product | |
| R9 | SMS provider account, bound to the PE (telemarketer chain) | per SMS | 1–2 days | M1.3 | product + finance | |
| R10 | DoT **M2M service provider (M2MSP)** registration (Saral Sanchar) | ₹5,000 application fee | ~2–3 weeks | R11 | signatory | |
| R11 | IoT/M2M SIM contract (pooled data) | ≈ ₹150/bus/month target | 1–3 weeks after R10 | M8 bring-up, M9 installs | ops + finance | |
| R12 | Cloudflare DNS records + certificates | free | same day | M9.1 | ops | |
| R13 | Compliance consultant: WPC ETA, MTCTE, EPR (`hardware/compliance.md`) | quote | weeks | paid installs at scale | signatory | |

**Start on day 1, in parallel:** R1, R7, R10 (each has a wait). R2/R3 follow R1; R8/R9 follow R7; R11 follows R10.

## 2. D-U-N-S, Apple and Google (R1–R3)
- **D-U-N-S** is free. Look up first (Apple's enrolment flow has a lookup); if Previa has none, request it with the
  exact legal name and address. One number serves **both** stores. After it is issued, allow up to 2 business days
  before enrolling with Apple.
- Store listing name for both stores: **Sumarg: School Bus Tracker** (≤ 30 characters), developer "Previa Softwares".
- **Apple Developer Program, organization:** US$99/year; needs the D-U-N-S number, legal entity status, a website,
  and a person with authority to accept Apple's agreements. Then in App Store Connect: app record with bundle id
  `com.previasoftwares.sumarg`; an **APNs auth key (.p8)** — download once, upload to Firebase (R5), store
  the file in the password manager; privacy "nutrition" labels from docs/07 §3; a **review demo account** (§8).
- **Google Play Console, organization:** US$25 once; organization accounts need the D-U-N-S number and identity
  verification. Choose **organization**, not personal: the "12 testers for 14 days" closed-testing rule applies to
  personal accounts created after 13 Nov 2023, and organization accounts are exempt. Then: app with package
  `com.previasoftwares.sumarg`, Play App Signing, Data safety form (location while a child's trip runs,
  phone number, boarding photos viewed by the school — docs/07 §3), target audience **adults** (parents and school
  staff; the app is not directed at children), a service-account key for EAS Submit (secret → EAS).

## 3. Expo / EAS (R4)
Organization `previasoftwares` on expo.dev; project linked to `mobile/app.json`. Build credentials are managed by
EAS (iOS certificates/profiles, Android upload key). `google-services.json` and `GoogleService-Info.plist` are
uploaded as **EAS file secrets**, not committed (`.claude/settings.json` denies reading them). Free tier first; paid
plan only for heavy release weeks (docs/10 §1).

## 4. Firebase and Google Cloud (R5, R6)
- Two Firebase projects: `previa-sumarg` (prod) and `previa-sumarg-dev`. In each: Android app
  (package above, SHA-1 of the Play app-signing key and the EAS dev key) and iOS app (bundle id above); upload the
  APNs key; Cloud Messaging only — **no Analytics, no Crashlytics** (docs/02 A30).
- Server push: a service account allowed only to send FCM messages; its JSON key goes to
  `data/secrets/fcm-service-account.json` on the VPS (chmod 600, docs/07 S12). `.env`: `FCM_PROJECT_ID`.
- Google Cloud billing account with a **budget alert** (e.g. ₹2,000/month). API keys, each restricted:
  Android Maps SDK (package + SHA-1), iOS Maps SDK (bundle id), Maps JavaScript (referrer
  `sumarg.previasoftwares.com/*`), Routes API (server, restricted to the VPS IP). Names in
  `infra/.env.example`.

## 5. SMS: DLT and provider (R7–R9)
Parent login uses a 6-digit OTP by SMS (docs/02 A22); push is the normal channel and SMS is only a fallback for
guardians without the app (docs/05 §6).

1. **Principal Entity (PE)** on **one** operator portal — one registration is enough; the entity ID works across
   operators. Jio TrueConnect (`trueconnect.jio.com`) is widely used: ₹5,900 incl. GST for a year; headers and
   templates are included. Documents: PAN, GST certificate, address proof, incorporation certificate. Save the
   **PE ID**.
2. **Header** (sender ID, 6 letters) for service messages, e.g. `PREVIA` (alternates `PRVSFT`, `PRVSCH`). Since
   May 2025 operators append a category suffix (`-S` for service) automatically — templates don't change.
3. **Content templates** under **Service Implicit** — non-banks must not pick "Transactional" (reserved for banks
   and RBI-regulated wallets; a common rejection reason). Each `{#var#}` holds at most 30 characters. Link one
   template to one header. Proposed templates (English; add Gujarati/Hindi as separate Unicode templates later):

| Name | Template |
|---|---|
| OTP | `{#var#} is your Sumarg login code for {#var#}. It is valid for 5 minutes. Do not share it with anyone. - Previa Softwares` |
| Invite | `{#var#} has added you to Sumarg. Install the app and sign in with school code {#var#}: https://sumarg.previasoftwares.com/i/{#var#} - Previa Softwares` |
| Boarded (fallback) | `{#var#} boarded bus {#var#} at {#var#}. - {#var#} via Sumarg` |
| Dropped (fallback) | `{#var#} got off bus {#var#} at {#var#}. - {#var#} via Sumarg` |
| Delay (fallback) | `Bus {#var#} is running about {#var#} min late today. - {#var#} via Sumarg` |

4. **Whitelist the link and any call-back number** on the DLT portal: since 1 Oct 2024 operators drop messages with
   URLs, APKs or call-back numbers the sender has not whitelisted. Keep the invite path short (`/i/<code>`).
5. **SMS provider** (default MSG91 — docs/09 V9; any DLT-compliant provider works): open the account, then on the
   DLT portal bind the PE to the provider's **telemarketer** ID so the delivery chain is traceable. Enter PE ID,
   header and template IDs in the provider; `.env`: `SMS_PROVIDER`, `SMS_API_KEY`, `SMS_SENDER_ID`,
   `SMS_DLT_PE_ID`, `SMS_DLT_TEMPLATE_OTP`, `SMS_DLT_TEMPLATE_INVITE`, … (names in `infra/.env.example`).
6. If the app later auto-reads OTPs on Android (SMS Retriever), register an OTP template variant that ends with the
   app's 11-character hash.

## 6. IoT SIMs (R10, R11)
1. **M2MSP registration first.** DoT requires entities offering SIM-based M2M services to register as an M2M
   Service Provider on the Saral Sanchar portal (application fee ₹5,000), and operators ask for the DoT registration
   number before issuing M2M SIMs. Previa owns the bus units and runs the service, so Previa registers.
2. **What to ask operators for (Airtel IoT, Jio IoT, Vi Business IoT):**
   - M2M/IoT SIMs (13-digit M2M numbering where offered), LTE on B1/B3/B5/B8/B40/B41, data only + SMS (for the
     unit's SMS fallback, docs/12) — no voice.
   - **Pooled data** across all units, sized ~250 MB per unit per month (docs/13 §8), with overage price and
     alerts; no throttling below 64 kbps on pool exhaustion.
   - Private APN or at least a fixed APN name; static IP not needed (units connect out to `devices.` over TLS).
   - A management portal/API: activate/suspend per ICCID, usage per SIM, last-seen cell, SMS to SIM.
   - Industrial SIM form factor option (MFF2 solder-down for SB-R1; nano-SIM industrial grade for SB-C1, rated
     −25…+85 °C).
   - Business KYC with the M2MSP number; end-user (school) details if the operator asks; invoices with GST.
   - Price target ≤ ₹150 per SIM per month all-in (docs/10 §1); 12-month price lock; no minimum commitment above
     the first year's forecast (start with 50 SIMs).
3. Record ICCID ↔ unit uid at provisioning (docs/12 §9); `/admin → Units` keeps IMEI and ICCID.

## 7. DNS and certificates (R12)
| Record | Type | Proxy | Purpose |
|---|---|---|---|
| `sumarg.previasoftwares.com` | A → VPS | proxied | portal + API + app API |
| `devices.sumarg.previasoftwares.com` | A → VPS | **DNS only** | MQTT/TLS :8883 and firmware downloads (units pin ISRG Root X1, so no Cloudflare proxy) |
| `schoolfleet-mobile.previasoftwares.com` | existing | — | demo; retire after launch |

Let's Encrypt for both hosts via certbot on the VPS (docs/06); renewal reloads Nginx and the MQTT listener.

## 8. Demo school for store review and sales
`migrate seed --demo` loads a **demo school** (code `demo`) from `onboarding/demo/` through the same CSV importer
real schools use: two buses on simulated routes, sample students and a demo guardian. App review needs a working login without a real SMS: the server accepts a fixed OTP **only** for the demo
school's review mobile numbers, only when `DEMO_REVIEW_LOGIN=on` (off by default, logged on every use). The
simulator (`make sim`) keeps the demo buses moving so reviewers and prospects see live tracking.

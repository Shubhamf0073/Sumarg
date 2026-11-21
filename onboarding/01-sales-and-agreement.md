# 01 — Sales, agreement and setup

## Demo (20 minutes, on the demo school)
1. Parent app: live bus on the map, ETA to "my stop", boarded notification with the time, absence request.
2. Driver app: today's trip, start, stop list with students, SOS.
3. Portal: live board of all buses, an SOS alert arriving and being acknowledged, boarding report with photos.
4. The bus unit in hand: tap a card, show the photo and the parent notification arriving.
5. Price and what's included (unit, SIM, cards, apps, support) — docs/10 §3; the school's existing AIS-140 tracker
   stays (docs/02 A29).
Keep the simulator running (`make sim` against the demo tenant) so buses move during the demo.

## Agreement — what it must cover (lawyer review: docs/09 L1)
- Service: package length (12 months), students on transport, price per student incl. GST, minimum 20 per bus,
  top-ups for extra students, billing in advance, suspension rules (docs/05 §11).
- Hardware: units remain Previa's property; school provides installation access and power; damage beyond wear at
  cost; units returned at termination.
- **DPA**: school is the data fiduciary; Previa processes students', guardians' and crew data only to run the
  service; retention (photos 30 days, positions per docs/05 §9); sub-processors (hosting, SMS provider, Google
  FCM/Maps); breach notice; deletion/export on termination (docs/07 §3).
- Parent communication: the school informs parents (privacy notice text from Previa); the school sends invites.
- Support hours and response times; go-live date depends on data and access from the school.

## Setup in `/admin` (Previa, ~1 hour)
1. Tenants → New: legal name, display name, short name (signs notifications), board, GSTIN, state code, address,
   contact; **school code** (short, lowercase, e.g. `dpsbopal`) — it is printed on invites and typed at login.
2. Package: start date, committed students, price, GST-inclusive flag, agent.
3. Campuses (or import `campuses.csv`); settings; owner login. Steps 1–3 are one wizard in `/admin/schools/new`,
   saved at once (the school starts at `onboarding_stage = setup`, with the planned `go_live_on` if known). The owner's
   one-time set-password link (72 h) is shown once: send it to the owner's own mobile. A new link: the school page →
   "New password link". Extra admin logins come with the school's own users screen (M1.5).
4. Move the stage on the school page as the school progresses; **go-live day: move to `live`** (an empty
   `go_live_on` becomes today, IST).
5. Send the data request (`02-data-collection.md`).

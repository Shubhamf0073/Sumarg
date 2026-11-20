---
paths:
  - "frontend/**"
---
# Frontend rules (loaded only when working under frontend/)

- Screens map 1:1 to `reference/demo/` (read-only spec; `/ui-port`). Admin screens in `src/admin/` (lazy).
- State: TanStack Query only (no other state library). One hook per screen aggregate in `src/api/<module>.ts`
  using generated types from `src/api/schema.d.ts` (grep it; never read it whole; never edit — `make gen-api`).
- `openapi-fetch` client sends `X-CSRF-Token` from the `__Host-sf_csrf` cookie on every non-GET.
- Money only through `lib/money.ts` (integer paise in, ₹ string out); dates `lib/dates.ts` (IST);
  permissions `lib/perms.ts` (UI hint only; the API enforces).
- Google Maps only inside the lazy Live/Trip map component; Live opens in list view; polling only there
  (30 s, paused when hidden). No other polling.
- Keep the main chunk small: admin, maps and reports are lazy routes. No new npm package without
  `/add-dependency`.
- Biome formats on save (hook). `npm run typecheck` before finishing.

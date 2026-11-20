---
name: mobile-screen
description: Use when building or changing a screen in the Expo app (parent or driver/attendant) — route file, query hook, generated types, i18n strings, focus-aware polling, offline states.
---
# App screen

1. Find the demo screen (`grep -n "<title>" reference/demo/source/js/parent.js` or `driver.js`); read only that part.
2. Contract in docs/04 (`/parent/*` or `/crew/*`); if missing, add it there first and to the Go handler (`/api-endpoint`).
3. Route file under `mobile/app/(parent)/` or `(driver)/`; data via one hook in `mobile/src/api/<area>.ts` with the
   generated types; mutations update the cache (`setQueryData`), no refetch storms.
4. Live screens poll only while focused and the app is active; stop on blur/background.
5. Strings in `src/i18n/{en,gu,hi}.json`; 48–50 px tap targets; works at 360 px width; dark mode readable.
6. States: loading skeleton, empty, offline banner (cached data shown), 401 → login, 409 `no_trip` → "Bus not on a
   trip right now".
7. `npm run typecheck && npm run lint` in `mobile/`; a jest test for any non-trivial hook logic.

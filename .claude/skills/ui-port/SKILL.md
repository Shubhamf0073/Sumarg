---
name: ui-port
description: Use when building a school-portal or admin screen from the demo in reference/demo/source — port markup/CSS, replace the demo's localStorage state with API hooks.
---
# Port a portal screen

`reference/demo/README.md` maps demo files → portal/app areas and phases. The demo is read-only (hook-enforced).
1. `grep -n "<screen title>" reference/demo/source/js/admin*.js` and read only that function.
2. Recreate in `frontend/src/screens/<Screen>.tsx` with the same labels, order and CSS tokens from
   `reference/demo/source/css/app.css`; Modal/Form/DataTable start from `reference/reuse/eswasthay-ui.tsx` (port once).
3. One TanStack Query hook per screen aggregate with generated types; lists page on the server (`limit+1`).
4. Maps only in lazy components (Google Maps JS); live board polls 10 s only while visible.
5. Phase 2/3 menus (Maintenance, Finance, Sustainability) are hidden, not stubbed.
6. Check at 1280 px and 360 px; `npm run typecheck`.

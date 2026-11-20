# Demo (UI spec) — read-only

`source/` is the School Fleet demo from `https://schoolfleet-mobile.previasoftwares.com` (saved 19 Nov 2026). Plain
JS + one CSS file; data is fake (`js/data.js`). Port screens, labels, order and CSS tokens; replace the demo's
localStorage state (`seedState()`) with API hooks. Phase 1 scope: docs/01.

| Demo file / route | Becomes | Phase |
|---|---|---|
| `js/admin.js`, `js/admin2.js` · `#/admin/*` (Dashboard, Fleet, Routes, Students, People, Operations, Safety, Reports, Audit) | portal `/` (school owner/admin/transport manager) | 1 |
| Admin › Maintenance, Finance, Sustainability | portal | 2 / 3 |
| `js/maint.js` · `#/maint/*` (Maintenance Manager) | portal maintenance module | 2 |
| `js/driver.js` · `#/driver/*` (mobile: home, trip, stops, board, scan, noshow, incidents, emergency, messages) | app → driver/attendant tabs | 1 (messages: 2) |
| `js/driver.js` · `#/portal/*` (driver desktop portal) | not built — drivers use the app | — |
| `js/parent.js` · `#/parent/*` (home, children, tracking, eta, boarding, notifications, requests, emergency info) | app → parent tabs | 1 (chat: 2) |
| `js/landing.js` | marketing page (optional, outside the app) | — |
| `css/app.css` | design tokens: navy `#0f2f57`, route colours, 50 px tap targets | 1 |

Search the demo instead of reading it whole: `grep -n "Boarding Verification" reference/demo/source/js/*.js`.

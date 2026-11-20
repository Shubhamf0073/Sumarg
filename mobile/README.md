# Mobile app (Android + iOS) — one Expo / React Native codebase

One app, two experiences chosen at login (docs/02 A28):
- **Parent** — school code + mobile + OTP → children, live bus, ETA to my stop, boarded/dropped notifications,
  report absence, requests, announcements.
- **Driver / attendant** — school code + mobile + PIN, device-bound → today's trips, start trip with inspection,
  stop list, manual mark when a card fails (QR fallback), SOS, incidents.

```
src/api/        generated types (schema.d.ts from the backend's OpenAPI) + openapi-fetch client + query hooks
app/            expo-router screens: (auth)/ (parent)/ (driver)/ — mirrors reference/demo parent.js / driver.js
src/components/ shared UI (ported look from reference/demo css/app.css: #0f2f57 navy, 50 px tap targets)
```

Builds and store submission run on EAS (Expo's build service); Maps keys and `google-services.json` /
`GoogleService-Info.plist` come from EAS secrets, never from git. Versions in `package.json` are the intended
majors — `npx expo install --fix` aligns them to the installed Expo SDK during M5.1.

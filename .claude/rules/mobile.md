---
paths:
  - "mobile/**"
---
# Mobile app rules (Expo / React Native, one app for parents and crew)

- Screens mirror `reference/demo/source/js/parent.js` and `driver.js` (`/ui-port`, `/mobile-screen`). expo-router:
  `app/(auth)`, `app/(parent)`, `app/(driver)`; the session's principal picks the tab set.
- Data: TanStack Query hooks in `src/api/` with generated types (`npm run gen:api` after `make gen-api`); one call
  per screen; live tracking polls every 10 s **only while the screen is focused** (`useFocusEffect` + app state).
- Session token in `expo-secure-store` only (never AsyncStorage, never logs). Bearer header; on 401 → login.
- Push: `expo-notifications` **device** tokens (FCM/APNs), registered with `/auth/push-token`; deep links from
  `data.screen`. No Expo push service (children's data stays with FCM only, docs/02 A10).
- Maps: `react-native-maps` with `PROVIDER_GOOGLE`; keys come from EAS secrets via `app.json` placeholders.
- No analytics, ads or tracking SDKs; location permission only for crew (fallback when the unit is offline) and
  only during a running trip.
- Strings in `src/i18n/{en,gu,hi}.json`; no hard-coded user-facing text.
- Upgrade Expo/RN only with `npx expo install --fix`; no new npm package without `/add-dependency`.

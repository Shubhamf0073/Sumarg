---
name: simulate
description: Run or write device-simulator scenarios (virtual buses over MQTT that drive routes, tap roster cards, press SOS, lose power or signal) and check the platform's results — boardings, pushes, alerts, live positions.
---
# Simulator (ported from Previa Fleet `tools/simulator` in M3.6)

- `make sim SCENARIO=<name>`; scenarios in `tools/simulator/scenarios/*.yaml` (route, start time, speed profile,
  taps per stop, events like `sos@stop3`, `power_cut 120s`, `offline 10m`).
- The simulator provisions virtual units against the dev DB (two demo schools from `make seed`), connects with real
  secrets over TLS, and publishes exactly the v1 payloads the firmware sends.
- Assertions live next to the scenario: expected boardings (count, direction, stop), notifications queued per
  guardian, alerts (kind, severity), `seq` gaps = 0 after an offline replay, live position freshness.
- Every change on the tap/live/alert path adds or updates a scenario. Run long scenarios through `test-runner`.

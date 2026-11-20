---
name: device-protocol
description: Use for any change to the unit↔cloud MQTT protocol (topics, telemetry/status/config/command/ack payloads, event kinds, photo topic) — schema, examples, docs/13, firmware codec, ingest parser and simulator in one change.
---
# Change the device protocol (adapted from Previa Fleet `/protocol-change`)

Contract: `backend/platform/protocol/schemas/*.v1.json` (wins over docs). Rules: `docsec.sh docs/13-device-protocol.md "7. Versioning"`.

1. Decide if it is **additive** (new optional field, new event kind, new command) → stays v1. Anything that removes,
   renames, retypes or tightens a shipped field → stop and ask (needs v2 topics + N/N-1 support for 6 months).
2. Edit the schema(s) and add/adjust an example in `backend/platform/protocol/testdata/examples/`; extend Go types
   in `platform/protocol/types.go`; `go test ./platform/protocol/...` (every example must validate).
3. Update docs/13 (table rows + sample JSON) in the same commit.
4. Firmware codec (`firmware/src/net/` or `src/logic/codec_*`) with a host test that serialises the example.
5. Ingest parser/pipeline (`backend/internal/ingest/`) + simulator (`tools/simulator`) so a scenario exercises it.
6. Run the `protocol-guardian` subagent on the diff; fix findings.
Keep keys short (2–4 chars) — they cost data on every message (docs/13 §8).

// Package protocol is the Go side of the device ↔ cloud contract (docs/06).
//
// The JSON Schemas in packages/protocol/schemas are the source of truth; `make gen-protocol`
// copies them into ./schemas, where they are embedded and used by Validate. The message types
// are hand-written with the protocol's short JSON keys; TestTypes_CoverSchemaProperties fails
// if a schema property has no Go field.
package protocol

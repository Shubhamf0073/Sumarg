package protocol

import (
	"bytes"
	"embed"
	"fmt"
	"sync"

	"github.com/santhosh-tekuri/jsonschema/v6"
)

//go:embed schemas/*.json
var schemaFS embed.FS

var (
	compileOnce sync.Once
	compiled    map[string]*jsonschema.Schema
	compileErr  error
)

// Kinds are the message kinds Validate understands (protocol version 1).
var Kinds = []string{"telemetry", "status", "ack", "config", "command"}

func compileAll() {
	c := jsonschema.NewCompiler()
	compiled = map[string]*jsonschema.Schema{}
	for _, k := range Kinds {
		b, err := schemaFS.ReadFile("schemas/" + k + ".v1.json")
		if err != nil {
			compileErr = fmt.Errorf("protocol: read %s schema: %w", k, err)
			return
		}
		doc, err := jsonschema.UnmarshalJSON(bytes.NewReader(b))
		if err != nil {
			compileErr = fmt.Errorf("protocol: parse %s schema: %w", k, err)
			return
		}
		url := "mem://protocol/" + k + ".v1.json"
		if err := c.AddResource(url, doc); err != nil {
			compileErr = fmt.Errorf("protocol: add %s schema: %w", k, err)
			return
		}
		s, err := c.Compile(url)
		if err != nil {
			compileErr = fmt.Errorf("protocol: compile %s schema: %w", k, err)
			return
		}
		compiled[k] = s
	}
}

// Validate checks payload against the v1 schema for kind. Unknown fields are allowed (docs/06 §7).
func Validate(kind string, payload []byte) error {
	compileOnce.Do(compileAll)
	if compileErr != nil {
		return compileErr
	}
	s, ok := compiled[kind]
	if !ok {
		return fmt.Errorf("protocol: unknown message kind %q", kind)
	}
	inst, err := jsonschema.UnmarshalJSON(bytes.NewReader(payload))
	if err != nil {
		return fmt.Errorf("protocol: invalid JSON: %w", err)
	}
	if err := s.Validate(inst); err != nil {
		return fmt.Errorf("protocol: %s does not match schema: %w", kind, err)
	}
	return nil
}

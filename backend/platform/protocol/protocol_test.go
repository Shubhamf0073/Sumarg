package protocol

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func examplesDir(t *testing.T) string {
	t.Helper()
	return filepath.Join("testdata", "examples")
}

func TestValidate_AllExamples(t *testing.T) {
	files, err := filepath.Glob(filepath.Join(examplesDir(t), "*.json"))
	if err != nil || len(files) == 0 {
		t.Fatalf("no examples found: %v", err)
	}
	for _, f := range files {
		kind := strings.SplitN(filepath.Base(f), ".", 2)[0]
		b, err := os.ReadFile(f) // #nosec G304 -- test fixture path
		if err != nil {
			t.Fatal(err)
		}
		if err := Validate(kind, b); err != nil {
			t.Errorf("%s: %v", filepath.Base(f), err)
		}
	}
}

func readExample(t *testing.T, name string) map[string]any {
	t.Helper()
	b, err := os.ReadFile(filepath.Join(examplesDir(t), name)) // #nosec G304 -- test fixture path
	if err != nil {
		t.Fatal(err)
	}
	var m map[string]any
	if err := json.Unmarshal(b, &m); err != nil {
		t.Fatal(err)
	}
	return m
}

func TestValidate_RejectsBadDumpBurst(t *testing.T) {
	m := readExample(t, "telemetry.v1.dump-burst.json")
	ev := m["evs"].([]any)[1].(map[string]any)
	fx := ev["d"].(map[string]any)["fx"].([]any)
	fx[0] = fx[0].([]any)[:4]
	b, _ := json.Marshal(m)
	if err := Validate("telemetry", b); err == nil {
		t.Fatal("a 4-element fx item must be rejected")
	}
}

func TestValidate_UnknownFieldAllowed(t *testing.T) {
	m := readExample(t, "telemetry.v1.haul-dump.json")
	m["recs"].([]any)[0].(map[string]any)["zz"] = 1
	b, _ := json.Marshal(m)
	if err := Validate("telemetry", b); err != nil {
		t.Fatalf("unknown fields must be allowed in v1: %v", err)
	}
}

func TestValidate_UnknownKind(t *testing.T) {
	if err := Validate("nope", []byte(`{}`)); err == nil {
		t.Fatal("unknown kind must error")
	}
}

func TestParseTopic_Table(t *testing.T) {
	cases := []struct {
		in      string
		ok      bool
		tk, dev string
		ch      string
	}{
		{"v1/t/acme/d/PV-A1B2C3/telemetry", true, "acme", "PV-A1B2C3", "telemetry"},
		{"v1/t/_pool/d/PV-A1B2C3D4E5F6/status", true, "_pool", "PV-A1B2C3D4E5F6", "status"},
		{"v1/t/acme/d/PV-A1B2C3/cmd", true, "acme", "PV-A1B2C3", "cmd"},
		{"v2/t/acme/d/PV-A1B2C3/telemetry", false, "", "", ""},
		{"v1/t/acme/d/PV-XYZ/telemetry", false, "", "", ""},
		{"v1/t/acme/d/PV-A1B2C3/other", false, "", "", ""},
		{"v1/t/Acme/d/PV-A1B2C3/telemetry", false, "", "", ""},
		{"v1/t/acme/d/PV-A1B2C3/telemetry/extra", false, "", "", ""},
	}
	for _, c := range cases {
		got, err := ParseTopic(c.in)
		if (err == nil) != c.ok {
			t.Errorf("%s: err=%v, want ok=%v", c.in, err, c.ok)
			continue
		}
		if c.ok && (got.Version != 1 || got.TenantKey != c.tk || got.DeviceID != c.dev || got.Channel != c.ch) {
			t.Errorf("%s: got %+v", c.in, got)
		}
	}
}

// jsonTags collects the json tag names of a struct type.
func jsonTags(typ reflect.Type) map[string]bool {
	out := map[string]bool{}
	for i := 0; i < typ.NumField(); i++ {
		name := strings.Split(typ.Field(i).Tag.Get("json"), ",")[0]
		if name != "" && name != "-" {
			out[name] = true
		}
	}
	return out
}

func schemaProps(t *testing.T, file string, path ...any) []string {
	t.Helper()
	b, err := schemaFS.ReadFile("schemas/" + file)
	if err != nil {
		t.Fatal(err)
	}
	var node any
	if err := json.Unmarshal(b, &node); err != nil {
		t.Fatal(err)
	}
	for _, p := range path {
		switch k := p.(type) {
		case string:
			node = node.(map[string]any)[k]
		case int:
			node = node.([]any)[k]
		}
	}
	var props []string
	for k := range node.(map[string]any)["properties"].(map[string]any) {
		props = append(props, k)
	}
	return props
}

// Hand-written types must cover every schema property (drift guard instead of generated types).
func TestTypes_CoverSchemaProperties(t *testing.T) {
	checks := []struct {
		name  string
		typ   reflect.Type
		props []string
	}{
		{"telemetry", reflect.TypeOf(Telemetry{}), schemaProps(t, "telemetry.v1.json")},
		{"record", reflect.TypeOf(Record{}), schemaProps(t, "telemetry.v1.json", "$defs", "record")},
		{"event", reflect.TypeOf(Event{}), schemaProps(t, "telemetry.v1.json", "$defs", "event")},
		{"status", reflect.TypeOf(Status{}), schemaProps(t, "status.v1.json")},
		{"ack", reflect.TypeOf(Ack{}), schemaProps(t, "ack.v1.json")},
		{"config", reflect.TypeOf(Config{}), schemaProps(t, "config.v1.json")},
		{"config.dump", reflect.TypeOf(ConfigDump{}), schemaProps(t, "config.v1.json", "properties", "dump")},
		{"config.broker", reflect.TypeOf(ConfigBroker{}), schemaProps(t, "config.v1.json", "properties", "broker")},
		{"command", reflect.TypeOf(Command{}), schemaProps(t, "command.v1.json")},
		{"dump detail", reflect.TypeOf(DumpDetail{}), schemaProps(t, "telemetry.v1.json", "$defs", "event", "allOf", 0, "then", "properties", "d")},
	}
	for _, c := range checks {
		tags := jsonTags(c.typ)
		for _, p := range c.props {
			if !tags[p] {
				t.Errorf("%s: schema property %q has no Go field", c.name, p)
			}
		}
	}
}

func TestDumpDetail_Decode(t *testing.T) {
	b, err := os.ReadFile(filepath.Join(examplesDir(t), "telemetry.v1.dump-burst.json")) // #nosec G304
	if err != nil {
		t.Fatal(err)
	}
	var tm Telemetry
	if err := json.Unmarshal(b, &tm); err != nil {
		t.Fatal(err)
	}
	var d DumpDetail
	if err := json.Unmarshal(tm.Evs[1].D, &d); err != nil {
		t.Fatal(err)
	}
	if d.T0 == nil || *d.T0 != 1759470101 || len(d.Fx) != 82 || d.Fx[2][0] != 1000 {
		t.Fatalf("decoded dump detail = t0 %v, %d fixes", d.T0, len(d.Fx))
	}
}

func dumpBurstWithFx(t *testing.T, item []any) []byte {
	t.Helper()
	m := readExample(t, "telemetry.v1.dump-burst.json")
	m["evs"].([]any)[1].(map[string]any)["d"].(map[string]any)["fx"].([]any)[0] = item
	b, _ := json.Marshal(m)
	return b
}

// dt_ms and sats are integers; dt_ms is at most 600000 (10 minutes).
func TestValidate_FxIntegerFields(t *testing.T) {
	for name, item := range map[string][]any{
		"fractional dt_ms": {500.5, 23.0, 72.5, 0.9, 9},
		"fractional sats":  {500, 23.0, 72.5, 0.9, 9.5},
		"dt_ms too large":  {600001, 23.0, 72.5, 0.9, 9},
	} {
		if err := Validate("telemetry", dumpBurstWithFx(t, item)); err == nil {
			t.Errorf("%s must be rejected", name)
		}
	}
}

// Extra trailing columns in an fx item are allowed, so the tuple can grow within v1.
func TestValidate_FxExtraColumnAllowed(t *testing.T) {
	if err := Validate("telemetry", dumpBurstWithFx(t, []any{500, 23.0, 72.5, 0.9, 9, 3})); err != nil {
		t.Fatalf("a 6-column fx item must be accepted: %v", err)
	}
}

func TestValidate_BrokerHostPattern(t *testing.T) {
	m := readExample(t, "config.v1.broker.json")
	m["broker"].(map[string]any)["host"] = "mqtts://mqtt.example.com/x"
	b, _ := json.Marshal(m)
	if err := Validate("config", b); err == nil {
		t.Fatal("a broker host with a scheme or path must be rejected")
	}
}

func TestTopic_Kind(t *testing.T) {
	for ch, kind := range map[string]string{"telemetry": "telemetry", "status": "status", "ack": "ack", "config": "config", "cmd": "command"} {
		if got := (Topic{Version: 1, Channel: ch}).Kind(); got != kind {
			t.Errorf("Kind(%s) = %s, want %s", ch, got, kind)
		}
	}
}

func TestValidate_DumpDetailTypes(t *testing.T) {
	for name, d := range map[string]string{
		"dur_s as string":     `{"dur_s":"41"}`,
		"negative w_after":    `{"w_after":-1}`,
		"fractional w_before": `{"w_before":2310.5}`,
		"src not a string":    `{"src":7}`,
	} {
		msg := []byte(`{"v":1,"did":"PV-A1B2C3","fw":"0.4.0","recs":[],"evs":[{"s":1,"t":1759470142,"k":"DUMP","d":` + d + `}]}`)
		if err := Validate("telemetry", msg); err == nil {
			t.Errorf("%s must be rejected", name)
		}
	}
}

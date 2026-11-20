package protocol

import (
	"fmt"
	"regexp"
)

// Topic is a parsed MQTT topic: v{Version}/t/{TenantKey}/d/{DeviceID}/{Channel}.
type Topic struct {
	Version   int
	TenantKey string
	DeviceID  string
	Channel   string
}

var topicRE = regexp.MustCompile(`^v1/t/([a-z0-9_-]{2,32})/d/(PV-[0-9A-F]{6,12})/(telemetry|status|ack|config|cmd)$`)

// ParseTopic accepts only v1 topics from docs/06 §2 with a valid tenant key and device id.
func ParseTopic(s string) (Topic, error) {
	m := topicRE.FindStringSubmatch(s)
	if m == nil {
		return Topic{}, fmt.Errorf("protocol: unsupported topic %q", s)
	}
	return Topic{Version: 1, TenantKey: m[1], DeviceID: m[2], Channel: m[3]}, nil
}

// String renders the topic back to its MQTT form.
func (t Topic) String() string {
	return fmt.Sprintf("v%d/t/%s/d/%s/%s", t.Version, t.TenantKey, t.DeviceID, t.Channel)
}

// Kind maps the topic channel to the schema kind used by Validate ("cmd" -> "command").
func (t Topic) Kind() string {
	if t.Channel == "cmd" {
		return "command"
	}
	return t.Channel
}

package protocol

import "encoding/json"

// Telemetry is one telemetry batch (telemetry.v1.json).
type Telemetry struct {
	V    int      `json:"v"`
	DID  string   `json:"did"`
	FW   string   `json:"fw"`
	BT   int64    `json:"bt,omitempty"`
	Recs []Record `json:"recs"`
	Evs  []Event  `json:"evs,omitempty"`
}

// Record is one periodic sample. Pointers keep "absent" distinct from zero.
type Record struct {
	S  int64    `json:"s"`
	T  int64    `json:"t"`
	TQ *int     `json:"tq,omitempty"`
	La *float64 `json:"la,omitempty"`
	Lo *float64 `json:"lo,omitempty"`
	Sp *float64 `json:"sp,omitempty"`
	Hd *int     `json:"hd,omitempty"`
	Al *float64 `json:"al,omitempty"`
	Sa *int     `json:"sa,omitempty"`
	Hp *float64 `json:"hp,omitempty"`
	Ig *int     `json:"ig,omitempty"`
	Mv *int     `json:"mv,omitempty"`
	W  *int     `json:"w,omitempty"`
	Fu *int     `json:"fu,omitempty"`
	Vi *int     `json:"vi,omitempty"`
	Vb *int     `json:"vb,omitempty"`
	Dm *int     `json:"dm,omitempty"`
	Tl *float64 `json:"tl,omitempty"`
	Cq *int     `json:"cq,omitempty"`
}

// Event is an edge event; D holds kind-specific details (see DumpDetail).
type Event struct {
	S  int64           `json:"s"`
	T  int64           `json:"t"`
	K  string          `json:"k"`
	La *float64        `json:"la,omitempty"`
	Lo *float64        `json:"lo,omitempty"`
	D  json.RawMessage `json:"d,omitempty"`
}

// DumpDetail is Event.D for k = "DUMP". Fx entries are [dt_ms after T0, la, lo, hdop, sats].
// Integer fields may arrive as 41.0 (valid JSON Schema integers), which encoding/json refuses for
// int: consumers that must not fail on such payloads decode the numbers as float64 (docs/06 §3).
type DumpDetail struct {
	DurS    *int         `json:"dur_s,omitempty"`
	TiltMax *float64     `json:"tilt_max,omitempty"`
	WBefore *int         `json:"w_before,omitempty"`
	WAfter  *int         `json:"w_after,omitempty"`
	Src     string       `json:"src,omitempty"`
	T0      *int64       `json:"t0,omitempty"`
	Fx      [][5]float64 `json:"fx,omitempty"`
}

// Status is the retained birth/health/LWT message (status.v1.json).
type Status struct {
	V      int        `json:"v"`
	Online bool       `json:"online"`
	DID    string     `json:"did,omitempty"`
	FW     string     `json:"fw,omitempty"`
	HW     string     `json:"hw,omitempty"`
	IMEI   string     `json:"imei,omitempty"`
	ICCID  string     `json:"iccid,omitempty"`
	CfgVer *int       `json:"cfgVer,omitempty"`
	Boot   *int64     `json:"boot,omitempty"`
	Reset  string     `json:"reset,omitempty"`
	CSQ    *int       `json:"csq,omitempty"`
	Op     string     `json:"op,omitempty"`
	RAT    string     `json:"rat,omitempty"`
	SD     *StatusSD  `json:"sd,omitempty"`
	GPS    *StatusGPS `json:"gps,omitempty"`
	VB     *int       `json:"vb,omitempty"`
	VI     *int       `json:"vi,omitempty"`
	DIP    string     `json:"dip,omitempty"`
}

// StatusSD is Status.SD.
type StatusSD struct {
	OK      bool `json:"ok"`
	FreeMB  int  `json:"freeMb"`
	Backlog int  `json:"backlog"`
}

// StatusGPS is Status.GPS.
type StatusGPS struct {
	Fix  int `json:"fix"`
	Sats int `json:"sats"`
}

// Ack is a device's reply to a command or config (ack.v1.json).
type Ack struct {
	V   int             `json:"v"`
	ID  string          `json:"id"`
	OK  bool            `json:"ok"`
	T   int64           `json:"t"`
	Err string          `json:"err,omitempty"`
	R   json.RawMessage `json:"r,omitempty"`
}

// Config is the retained full device configuration (config.v1.json). Sub-objects whose shape is
// device-specific are kept raw; Dump and Broker are typed because cloud code builds them.
type Config struct {
	V      int             `json:"v"`
	CfgVer int             `json:"cfgVer"`
	TK     string          `json:"tk"`
	Report json.RawMessage `json:"report"`
	Motion json.RawMessage `json:"motion,omitempty"`
	Inputs json.RawMessage `json:"inputs,omitempty"`
	IMU    json.RawMessage `json:"imu,omitempty"`
	Weight json.RawMessage `json:"weight,omitempty"`
	Speed  json.RawMessage `json:"speed,omitempty"`
	Buzzer json.RawMessage `json:"buzzer,omitempty"`
	FW     json.RawMessage `json:"fw,omitempty"`
	Dump   *ConfigDump     `json:"dump,omitempty"`
	Broker *ConfigBroker   `json:"broker,omitempty"`
}

// ConfigDump is Config.Dump. FixHz defaults to DefaultDumpFixHz when absent.
type ConfigDump struct {
	FixHz *int `json:"fixHz,omitempty"`
}

// DefaultDumpFixHz is the schema default for config.dump.fixHz.
const DefaultDumpFixHz = 2

// ConfigBroker is Config.Broker: the broker for the device's next connection (always TLS).
type ConfigBroker struct {
	Host string `json:"host"`
	Port int    `json:"port"`
}

// Command is a one-shot cloud -> device command (command.v1.json).
type Command struct {
	V    int             `json:"v"`
	ID   string          `json:"id"`
	Type string          `json:"type"`
	Exp  int64           `json:"exp,omitempty"`
	P    json.RawMessage `json:"p,omitempty"`
}

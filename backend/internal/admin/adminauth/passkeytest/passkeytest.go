// Package passkeytest is a software passkey for tests: it answers the options the admin sign-in endpoints return
// the way a browser + platform authenticator would (ES256, "none" attestation, user verified, discoverable), so the
// real go-webauthn verification runs end to end. Test support only: nothing in the running binary imports it.
package passkeytest

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
)

// Authenticator holds one passkey (after Create) for one relying party.
type Authenticator struct {
	RPID   string
	Origin string

	key    *ecdsa.PrivateKey
	credID []byte
	handle []byte
	count  uint32
	// Counter, if set, replaces the sign count sent on the next Get (to simulate a cloned authenticator).
	Counter *uint32
}

// CredentialID is the passkey's id (nil before Create).
func (a *Authenticator) CredentialID() []byte { return a.credID }

var b64 = base64.RawURLEncoding

type creationOptions struct {
	PublicKey struct {
		Challenge string `json:"challenge"`
		RP        struct {
			ID string `json:"id"`
		} `json:"rp"`
		User struct {
			ID string `json:"id"`
		} `json:"user"`
	} `json:"publicKey"`
}

type requestOptions struct {
	PublicKey struct {
		Challenge string `json:"challenge"`
	} `json:"publicKey"`
}

// Create answers navigator.credentials.create(options) with a new passkey; options is the begin endpoint's body.
func (a *Authenticator) Create(options map[string]any) (map[string]any, error) {
	var o creationOptions
	if err := remarshal(options, &o); err != nil {
		return nil, err
	}
	if o.PublicKey.Challenge == "" || o.PublicKey.User.ID == "" {
		return nil, errors.New("passkeytest: creation options without challenge or user id")
	}
	handle, err := b64.DecodeString(o.PublicKey.User.ID)
	if err != nil {
		return nil, fmt.Errorf("passkeytest: user id: %w", err)
	}
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return nil, err
	}
	credID := make([]byte, 16)
	if _, err := rand.Read(credID); err != nil {
		return nil, err
	}
	a.key, a.credID, a.handle, a.count = key, credID, handle, 0

	clientData := a.clientData("webauthn.create", o.PublicKey.Challenge)
	pub, err := key.PublicKey.ECDH()
	if err != nil {
		return nil, err
	}
	raw := pub.Bytes() // 0x04 || X || Y
	cose := cborMap(
		cborInt(1), cborInt(2), // kty: EC2
		cborInt(3), cborInt(-7), // alg: ES256
		cborInt(-1), cborInt(1), // crv: P-256
		cborInt(-2), cborBytes(raw[1:33]),
		cborInt(-3), cborBytes(raw[33:65]),
	)
	authData := a.authData(0x45) // UP | UV | AT
	authData = append(authData, make([]byte, 16)...)
	authData = binary.BigEndian.AppendUint16(authData, uint16(len(credID)))
	authData = append(authData, credID...)
	authData = append(authData, cose...)
	att := cborMap(
		cborText("fmt"), cborText("none"),
		cborText("attStmt"), cborMap(),
		cborText("authData"), cborBytes(authData),
	)
	return map[string]any{
		"id": b64.EncodeToString(credID), "rawId": b64.EncodeToString(credID), "type": "public-key",
		"authenticatorAttachment": "platform", "clientExtensionResults": map[string]any{},
		"response": map[string]any{
			"clientDataJSON":    b64.EncodeToString(clientData),
			"attestationObject": b64.EncodeToString(att),
			"transports":        []string{"internal"},
		},
	}, nil
}

// Get answers navigator.credentials.get(options) with an assertion signed by the passkey.
func (a *Authenticator) Get(options map[string]any) (map[string]any, error) {
	if a.key == nil {
		return nil, errors.New("passkeytest: no passkey (call Create first)")
	}
	var o requestOptions
	if err := remarshal(options, &o); err != nil {
		return nil, err
	}
	if o.PublicKey.Challenge == "" {
		return nil, errors.New("passkeytest: request options without challenge")
	}
	a.count++
	if a.Counter != nil {
		a.count = *a.Counter
		a.Counter = nil
	}
	clientData := a.clientData("webauthn.get", o.PublicKey.Challenge)
	authData := a.authData(0x05) // UP | UV
	cdHash := sha256.Sum256(clientData)
	digest := sha256.Sum256(append(append([]byte{}, authData...), cdHash[:]...))
	sig, err := ecdsa.SignASN1(rand.Reader, a.key, digest[:])
	if err != nil {
		return nil, err
	}
	return map[string]any{
		"id": b64.EncodeToString(a.credID), "rawId": b64.EncodeToString(a.credID), "type": "public-key",
		"authenticatorAttachment": "platform", "clientExtensionResults": map[string]any{},
		"response": map[string]any{
			"clientDataJSON":    b64.EncodeToString(clientData),
			"authenticatorData": b64.EncodeToString(authData),
			"signature":         b64.EncodeToString(sig),
			"userHandle":        b64.EncodeToString(a.handle),
		},
	}, nil
}

func (a *Authenticator) clientData(typ, challenge string) []byte {
	b, _ := json.Marshal(map[string]any{"type": typ, "challenge": challenge, "origin": a.Origin, "crossOrigin": false})
	return b
}

func (a *Authenticator) authData(flags byte) []byte {
	rp := sha256.Sum256([]byte(a.RPID))
	out := append([]byte{}, rp[:]...)
	out = append(out, flags)
	return binary.BigEndian.AppendUint32(out, a.count)
}

func remarshal(in any, out any) error {
	b, err := json.Marshal(in)
	if err != nil {
		return err
	}
	return json.Unmarshal(b, out)
}

// ── the little CBOR a passkey needs (RFC 8949: ints, byte/text strings, maps) ──

func cborHead(major byte, n uint64) []byte {
	m := major << 5
	switch {
	case n < 24:
		return []byte{m | byte(n)}
	case n < 1<<8:
		return []byte{m | 24, byte(n)}
	case n < 1<<16:
		return binary.BigEndian.AppendUint16([]byte{m | 25}, uint16(n))
	default:
		return binary.BigEndian.AppendUint32([]byte{m | 26}, uint32(n))
	}
}

func cborInt(v int64) []byte {
	if v >= 0 {
		return cborHead(0, uint64(v))
	}
	return cborHead(1, uint64(-1-v))
}

func cborBytes(b []byte) []byte { return append(cborHead(2, uint64(len(b))), b...) }
func cborText(s string) []byte  { return append(cborHead(3, uint64(len(s))), s...) }

// cborMap takes key, value, key, value… already encoded.
func cborMap(kv ...[]byte) []byte {
	out := cborHead(5, uint64(len(kv)/2))
	for _, p := range kv {
		out = append(out, p...)
	}
	return out
}

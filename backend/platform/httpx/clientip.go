package httpx

import (
	"net"
	"net/netip"
	"slices"
	"strings"
)

// ClientIP is the peer's address, or, when the peer is a trusted proxy, the last X-Forwarded-For entry (the one
// that proxy wrote; anything before it came from the client). nil when neither parses.
//
// Nginx overwrites X-Forwarded-For with the real client IP; the header is believed only when the connection itself
// comes from the Nginx hop (trusted), so a client that reaches the API another way can't choose its own address.
func ClientIP(remoteAddr, xff string, trusted []netip.Prefix) *netip.Addr {
	host, _, err := net.SplitHostPort(remoteAddr)
	if err != nil {
		host = remoteAddr
	}
	peer, err := netip.ParseAddr(host)
	if err != nil {
		return nil
	}
	peer = peer.Unmap()
	if xff != "" && slices.ContainsFunc(trusted, func(p netip.Prefix) bool { return p.Contains(peer) }) {
		last := xff
		if i := strings.LastIndexByte(xff, ','); i >= 0 {
			last = xff[i+1:]
		}
		if a, err := netip.ParseAddr(strings.TrimSpace(last)); err == nil {
			a = a.Unmap()
			return &a
		}
	}
	return &peer
}

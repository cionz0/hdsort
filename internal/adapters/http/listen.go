package http

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/hex"
	"fmt"
	"net"
)

// Listen binds a loopback address. An empty address or a loopback address
// with port 0 asks for an ephemeral port on 127.0.0.1. Any other host is refused.
func Listen(address string) (net.Listener, error) {
	if address == "" {
		address = "127.0.0.1:0"
	}
	host, port, err := net.SplitHostPort(address)
	if err != nil {
		return nil, err
	}
	if host == "" {
		return nil, fmt.Errorf("listen address %s is not loopback", address)
	}
	ip := net.ParseIP(host)
	if ip == nil || !ip.IsLoopback() {
		return nil, fmt.Errorf("listen address %s is not loopback", address)
	}
	return net.Listen("tcp", net.JoinHostPort(ip.String(), port))
}

// NewToken returns a 128-bit secret from crypto/rand, hex-encoded.
func NewToken() (string, error) {
	var buf [16]byte
	if _, err := rand.Read(buf[:]); err != nil {
		return "", err
	}
	return hex.EncodeToString(buf[:]), nil
}

func tokensEqual(got, want string) bool {
	return subtle.ConstantTimeCompare([]byte(got), []byte(want)) == 1
}

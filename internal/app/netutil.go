package app

import (
	"fmt"
	"net"
)

// ResolveAddr tries preferred first, then falls back to a random local port.
func ResolveAddr(preferred string) (string, error) {
	l, err := net.Listen("tcp", preferred)
	if err == nil {
		addr := l.Addr().String()
		_ = l.Close()
		return addr, nil
	}

	fallback, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return "", fmt.Errorf("no available local port: %w", err)
	}
	addr := fallback.Addr().String()
	_ = fallback.Close()
	return addr, nil
}

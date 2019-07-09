package grpc

import (
	"errors"
	"fmt"
	"net"
	"net/url"
	"strings"
)

const (
	defaultPort = "443"
)

var (
	errMissingAddr = errors.New("registry resolver: missing address")
)

// parseTarget takes the user input target string and default port, returns formatted host and port info.
func parseTarget(target, portByDefault string) (host, port string, err error) {
	if target == "" {
		return "", "", errMissingAddr
	}
	if portByDefault == "" {
		portByDefault = defaultPort
	}
	if !strings.Contains(target, ":") {
		return target, portByDefault, nil
	}
	if !strings.Contains(target, "://") {
		if host, port, err = net.SplitHostPort(target); err == nil {
			return host, port, nil
		}
	} else if u, err := url.Parse(target); err == nil {
		host = u.Hostname()
		port = u.Port()
		if port == "" {
			port = portByDefault
		}
		return host, port, nil
	}
	return "", "", fmt.Errorf("invalid target address %v, error info: %v", target, err)
}

// formatIP returns ok = false if addr is not a valid textual representation of an IP address.
// If addr is an IPv4 address, return the addr and ok = true.
// If addr is an IPv6 address, return the addr enclosed in square brackets and ok = true.
func formatIP(addr string) (addrIP string, ok bool) {
	ip := net.ParseIP(addr)
	if ip == nil {
		return "", false
	}
	if ip.To4() != nil {
		return addr, true
	}
	return "[" + addr + "]", true
}

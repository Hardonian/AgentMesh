package server

import (
	"errors"
	"fmt"
	"net"
	"net/url"
	"strings"
)

var (
	ErrInvalidScheme   = errors.New("invalid URL scheme: only http and https are permitted")
	ErrSSRFBlocked     = errors.New("destination address is blocked by SSRF protection policy")
	ErrHostResolution  = errors.New("failed to resolve target hostname")
)

// BlockedMetadataHosts lists cloud metadata hostnames that must always be blocked.
var BlockedMetadataHosts = map[string]bool{
	"metadata.google.internal":   true,
	"169.254.169.254":            true,
	"instance-data":              true,
	"metadata.internal":          true,
	"metadata":                   true,
}

// ValidateSafeRemoteURL checks that a user-supplied URL is safe from SSRF attacks.
// It verifies:
// 1. Allowed schemes (http, https).
// 2. Prohibited cloud metadata endpoints (169.254.169.254, metadata.google.internal).
// 3. Loopback addresses (127.0.0.0/8, ::1).
// 4. Link-local addresses (169.254.0.0/16, fe80::/10).
// 5. Private RFC1918 ranges (unless allowPrivate is explicitly true for enterprise VPCs).
func ValidateSafeRemoteURL(rawURL string, allowPrivate bool) (*url.URL, error) {
	if rawURL == "" {
		return nil, errors.New("target URL cannot be empty")
	}

	parsed, err := url.Parse(rawURL)
	if err != nil {
		return nil, fmt.Errorf("failed to parse URL: %w", err)
	}

	scheme := strings.ToLower(parsed.Scheme)
	if scheme != "http" && scheme != "https" {
		return nil, fmt.Errorf("%w: %q", ErrInvalidScheme, parsed.Scheme)
	}

	hostname := strings.ToLower(parsed.Hostname())
	if hostname == "" {
		return nil, errors.New("missing hostname in target URL")
	}

	// 1. Check known cloud metadata hosts
	if BlockedMetadataHosts[hostname] {
		return nil, fmt.Errorf("%w: metadata host %q is prohibited", ErrSSRFBlocked, hostname)
	}

	// 2. Resolve IP addresses to prevent DNS rebinding / internal host resolution
	ips, err := net.LookupIP(hostname)
	if err != nil {
		// If resolution fails locally (e.g. mock test environment), check if hostname is direct IP or localhost
		if ip := net.ParseIP(hostname); ip != nil {
			ips = []net.IP{ip}
		} else if hostname == "localhost" {
			return nil, fmt.Errorf("%w: localhost resolution blocked", ErrSSRFBlocked)
		} else {
			return nil, fmt.Errorf("%w: %s (%v)", ErrHostResolution, hostname, err)
		}
	}

	for _, ip := range ips {
		if ip.IsLoopback() {
			return nil, fmt.Errorf("%w: loopback address %s is prohibited", ErrSSRFBlocked, ip)
		}
		if ip.IsLinkLocalUnicast() || ip.IsLinkLocalMulticast() {
			return nil, fmt.Errorf("%w: link-local address %s is prohibited", ErrSSRFBlocked, ip)
		}

		// Cloud metadata direct IP check
		if ip.String() == "169.254.169.254" {
			return nil, fmt.Errorf("%w: cloud metadata IP %s is prohibited", ErrSSRFBlocked, ip)
		}

		// RFC1918 / Private network check
		if !allowPrivate && (ip.IsPrivate() || isCarrierGradeNAT(ip)) {
			return nil, fmt.Errorf("%w: private network address %s is prohibited without explicit private networking grant", ErrSSRFBlocked, ip)
		}
	}

	return parsed, nil
}

func isCarrierGradeNAT(ip net.IP) bool {
	ipv4 := ip.To4()
	if ipv4 == nil {
		return false
	}
	// 100.64.0.0/10 (100.64.0.0 - 100.127.255.255)
	return ipv4[0] == 100 && (ipv4[1]&0xc0) == 64
}

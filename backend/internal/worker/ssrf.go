package worker

import (
	"fmt"
	"net"
	"syscall"
	"time"
)

// isDisallowedIP reports whether ip must never be connected to.
//
// This is the actual SSRF fix (ADR-012, levels 2 and 3 combined into one
// mechanism - see newSafeDialer below for why one mechanism covers both).
//
// The product's entire job is "fetch a URL a user gave us", which is the
// textbook shape of Server-Side Request Forgery: a user could point a
// monitor at http://169.254.169.254/ (the AWS/GCP/Azure cloud metadata
// endpoint - anyone who can reach it can often read the server's own cloud
// credentials) or http://localhost:5432 and use OUR server as a proxy into
// networks the public internet cannot otherwise reach.
//
// net.IP's own methods do the real classification work. IsPrivate,
// IsLoopback and friends were added to the standard library specifically to
// answer "is this address routable on the public internet", and they already
// handle the fiddly cases (IPv4-mapped IPv6, the full RFC 1918 / RFC 4193
// ranges) correctly. Re-implementing that by hand with CIDR literals is
// exactly the kind of code that looks right and is subtly wrong.
func isDisallowedIP(ip net.IP) bool {
	return ip.IsLoopback() ||
		ip.IsPrivate() ||
		ip.IsLinkLocalUnicast() || // covers 169.254.169.254, the cloud metadata address
		ip.IsLinkLocalMulticast() ||
		ip.IsMulticast() ||
		ip.IsUnspecified()
}

// newSafeDialer builds a *net.Dialer that refuses to connect to a disallowed
// address.
//
// WHY THE DIALER, AND NOT ONLY A CHECK AT MONITOR-CREATION TIME.
//
// A hostname's DNS answer can change between the moment a user saves a
// monitor and the moment it is actually checked - and it changes again on
// every check after that. Validating once at creation time is a classic
// TOCTOU (time-of-check to time-of-use) gap: an attacker registers a monitor
// for a hostname that resolves to a public IP at creation time, then repoints
// its DNS at 127.0.0.1 before the worker ever checks it. This is called DNS
// rebinding, and it defeats any check that only runs once, up front.
//
// Control is called by the dialer AFTER DNS resolution but BEFORE the TCP
// connection opens, and it receives the literal resolved ip:port - never the
// hostname. That is the only point in the whole request lifecycle where we
// are looking at the address we are actually about to connect to, which is
// what makes this immune to DNS rebinding.
//
// It also covers HTTP REDIRECTS for free (ADR-012 level 3): a redirect makes
// the same Transport dial again, through this same Control function, so a
// public URL that redirects to a private one is refused at the second dial
// exactly as the first request would have been. One mechanism, two problems.
func newSafeDialer() *net.Dialer {
	dialer := &net.Dialer{
		Timeout: 10 * time.Second,
	}

	dialer.Control = func(network, address string, _ syscall.RawConn) error {
		host, _, err := net.SplitHostPort(address)
		if err != nil {
			return fmt.Errorf("parsing dial address %q: %w", address, err)
		}

		ip := net.ParseIP(host)
		if ip == nil {
			// Should never happen: by this point the dialer has already
			// resolved DNS, so address is always a literal IP. Refusing
			// rather than allowing keeps the failure mode safe either way.
			return fmt.Errorf("dial address %q is not a literal ip", host)
		}

		if isDisallowedIP(ip) {
			return fmt.Errorf(
				"refusing to connect to %s: address is private, loopback, or reserved", ip)
		}

		return nil
	}

	return dialer
}

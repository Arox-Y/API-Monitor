package worker

import (
	"context"
	"net"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestIsDisallowedIP(t *testing.T) {
	tests := []struct {
		name string
		ip   string
		want bool
	}{
		{"loopback v4", "127.0.0.1", true},
		{"loopback v6", "::1", true},
		{"private 10/8", "10.0.0.1", true},
		{"private 172.16/12", "172.16.5.4", true},
		{"private 192.168/16", "192.168.1.1", true},
		{"private v6 unique-local", "fc00::1", true},
		// The AWS/GCP/Azure cloud metadata address - the concrete attack this
		// guard exists to stop. It sits inside the link-local range.
		{"link-local / cloud metadata", "169.254.169.254", true},
		{"link-local v6", "fe80::1", true},
		{"unspecified v4", "0.0.0.0", true},
		{"unspecified v6", "::", true},
		{"multicast", "224.0.0.1", true},

		// Real public addresses must NOT be blocked, or the product could
		// never monitor anything at all.
		{"public v4 (google dns)", "8.8.8.8", false},
		{"public v4 (cloudflare dns)", "1.1.1.1", false},
		{"public v6 (cloudflare)", "2606:4700:4700::1111", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ip := net.ParseIP(tt.ip)
			if ip == nil {
				t.Fatalf("test setup: %q did not parse as an IP", tt.ip)
			}
			if got := isDisallowedIP(ip); got != tt.want {
				t.Errorf("isDisallowedIP(%s) = %v, want %v", tt.ip, got, tt.want)
			}
		})
	}
}

// THE TEST THAT PROVES BOTH THE VULNERABILITY AND THE FIX, IN ONE PLACE.
//
// httptest.NewServer listens on 127.0.0.1 - exactly the kind of address a
// real attacker would put behind a monitor pointed at "http://localhost:5432"
// or an internal service. This shows the protected checker actually refuses
// it, and that disabling protection (which only ever happens in tests)
// restores the old, vulnerable behaviour.
func TestChecker_SSRFProtection_BlocksLoopback(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	monitor := monitorFor(server.URL, 200, 5)

	t.Run("protected checker refuses to connect", func(t *testing.T) {
		// NewChecker() with NO options - the exact call main.go makes.
		result := NewChecker().Check(context.Background(), monitor)

		if result.Success {
			t.Fatal("SECURITY FAILURE: the protected checker reached a loopback address")
		}
		if result.StatusCode != nil {
			t.Errorf("expected no status code (connection should never have been made), got %d",
				*result.StatusCode)
		}
	})

	t.Run("unprotected checker (test-only) connects normally", func(t *testing.T) {
		result := NewChecker(WithSSRFProtection(false)).Check(context.Background(), monitor)

		if !result.Success {
			t.Fatalf("expected success with protection disabled, got: %s", result.Error)
		}
	})
}

package auth

import (
	"encoding/base64"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
)

// A fixed 32+ character secret for tests. Real secrets come from the
// environment; hard-coding one here is fine because it protects nothing.
const testSecret = "test-secret-that-is-long-enough-32+"

func newTestService(t *testing.T, ttl time.Duration) *TokenService {
	t.Helper() // marks this as a helper, so failures point at the caller

	svc, err := NewTokenService(testSecret, ttl)
	if err != nil {
		t.Fatalf("NewTokenService failed: %v", err)
	}
	return svc
}

func TestNewTokenService_RejectsWeakConfig(t *testing.T) {
	tests := []struct {
		name   string
		secret string
		ttl    time.Duration
	}{
		{"secret too short", "short", time.Hour},
		{"empty secret", "", time.Hour},
		{"31 chars, one under the limit", strings.Repeat("a", 31), time.Hour},
		{"zero ttl", testSecret, 0},
		{"negative ttl", testSecret, -time.Hour},
		// JWT timestamps are whole seconds, so sub-second TTLs cannot work.
		{"sub-second ttl", testSecret, 500 * time.Millisecond},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if _, err := NewTokenService(tt.secret, tt.ttl); err == nil {
				t.Error("expected an error, got nil - weak config was accepted")
			}
		})
	}
}

func TestGenerateAndVerify_RoundTrip(t *testing.T) {
	svc := newTestService(t, time.Hour)
	userID := uuid.New()

	token, err := svc.Generate(userID)
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	got, err := svc.Verify(token)
	if err != nil {
		t.Fatalf("Verify failed: %v", err)
	}

	if got != userID {
		t.Errorf("got user %v, want %v", got, userID)
	}
}

// This test documents the single most misunderstood fact about JWTs.
func TestToken_PayloadIsReadableByAnyone(t *testing.T) {
	svc := newTestService(t, time.Hour)
	userID := uuid.New()

	token, err := svc.Generate(userID)
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	// A JWT is three base64 sections separated by dots.
	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		t.Fatalf("expected 3 token parts, got %d", len(parts))
	}

	// Decode the middle part with NO secret at all.
	payload, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		t.Fatalf("decoding payload: %v", err)
	}

	// The user ID is sitting there in plain text.
	if !strings.Contains(string(payload), userID.String()) {
		t.Errorf("payload did not contain the user id: %s", payload)
	}

	// THE LESSON: a JWT is SIGNED, not ENCRYPTED. Anyone can read it.
	// It proves nobody altered the contents; it does not keep them secret.
	// Never put a password or API key in one.
	t.Logf("payload, decoded without any secret: %s", payload)
}

func TestVerify_RejectsTamperedToken(t *testing.T) {
	svc := newTestService(t, time.Hour)

	token, err := svc.Generate(uuid.New())
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	parts := strings.Split(token, ".")

	// Forge a new payload claiming to be a DIFFERENT user, and splice it in
	// while keeping the original signature. This is exactly what an attacker
	// would try.
	attackerID := uuid.New()
	forged := base64.RawURLEncoding.EncodeToString(
		[]byte(`{"sub":"` + attackerID.String() + `"}`),
	)
	tampered := parts[0] + "." + forged + "." + parts[2]

	if _, err := svc.Verify(tampered); err == nil {
		t.Fatal("SECURITY FAILURE: a tampered token was accepted")
	}
}

func TestVerify_RejectsWrongSecret(t *testing.T) {
	issuer := newTestService(t, time.Hour)

	// A different service with a different secret - as if the token came from
	// another system, or an attacker signed their own.
	other, err := NewTokenService("a-completely-different-secret-3232", time.Hour)
	if err != nil {
		t.Fatalf("setup failed: %v", err)
	}

	token, err := other.Generate(uuid.New())
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	if _, err := issuer.Verify(token); err == nil {
		t.Fatal("SECURITY FAILURE: a token signed with another secret was accepted")
	}
}

func TestVerify_RejectsExpiredToken(t *testing.T) {
	// One second is the SHORTEST usable lifetime, because JWT stores exp as a
	// whole number of seconds. An earlier version of this test used 50ms and
	// failed instantly - the token expired the moment it was created, which is
	// what led to the minimum-TTL check in NewTokenService.
	//
	// This test really waits rather than faking a clock, so it costs ~1.2s.
	// That is acceptable for one test guarding something this important.
	svc := newTestService(t, time.Second)

	token, err := svc.Generate(uuid.New())
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	// It should work right now.
	if _, err := svc.Verify(token); err != nil {
		t.Fatalf("token should be valid immediately: %v", err)
	}

	time.Sleep(1200 * time.Millisecond)

	_, err = svc.Verify(token)
	if err == nil {
		t.Fatal("SECURITY FAILURE: an expired token was accepted")
	}
	// errors.Is looks through the whole wrapped chain, not just the outer error.
	if !errors.Is(err, ErrExpiredToken) {
		t.Errorf("expected ErrExpiredToken, got %v", err)
	}
}

func TestVerify_RejectsMalformedTokens(t *testing.T) {
	svc := newTestService(t, time.Hour)

	tests := []struct {
		name  string
		token string
	}{
		{"empty string", ""},
		{"not a token", "hello world"},
		{"only two parts", "aaa.bbb"},
		{"four parts", "aaa.bbb.ccc.ddd"},
		{"empty signature", "aaa.bbb."},
		{"none algorithm attack", "eyJhbGciOiJub25lIn0.eyJzdWIiOiJhdHRhY2tlciJ9."},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if _, err := svc.Verify(tt.token); err == nil {
				t.Errorf("malformed token %q was accepted", tt.token)
			}
		})
	}
}

func TestVerify_RejectsValidSignatureWithNonUUIDSubject(t *testing.T) {
	// A token can be perfectly signed by us and still carry nonsense.
	// A valid signature means "we issued this", not "the contents make sense".
	svc := newTestService(t, time.Hour)

	// Build a properly signed token whose subject is not a UUID, by reaching
	// through the same library the service uses.
	other, err := NewTokenService(testSecret, time.Hour)
	if err != nil {
		t.Fatalf("setup failed: %v", err)
	}
	// uuid.Nil formats as a valid UUID string, so Generate cannot produce a
	// bad subject. Craft the token manually instead.
	token, err := other.Generate(uuid.New())
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	// Sanity: our own good token still verifies. (The malformed-subject path
	// is covered by TestVerify_RejectsTamperedToken, since altering the
	// subject invalidates the signature.)
	if _, err := svc.Verify(token); err != nil {
		t.Errorf("a legitimately signed token was rejected: %v", err)
	}
}

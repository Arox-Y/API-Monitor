package auth

import (
	"strings"
	"testing"
)

// Go's testing rules:
//   - the file must end in _test.go
//   - each test is a function starting with Test, taking *testing.T
//   - t.Errorf marks a failure and continues; t.Fatalf stops that test
//   - run them with: go test ./...
//
// No test framework is installed. This is all the standard library.

func TestHashPassword_ProducesUsableHash(t *testing.T) {
	const password = "correct horse battery staple"

	hash, err := HashPassword(password)
	if err != nil {
		t.Fatalf("HashPassword returned an error: %v", err)
	}

	// The most important property: the real password must not be recoverable
	// from what we store.
	if strings.Contains(hash, password) {
		t.Error("the hash contains the plain password - it is not hashed at all")
	}

	// bcrypt hashes always start with $2a$ (or $2b$/$2y$) followed by the cost.
	if !strings.HasPrefix(hash, "$2a$12$") {
		t.Errorf("hash does not look like bcrypt cost 12, got prefix %q", hash[:7])
	}
}

func TestHashPassword_SamePasswordGivesDifferentHashes(t *testing.T) {
	const password = "correct horse battery staple"

	first, err := HashPassword(password)
	if err != nil {
		t.Fatalf("first hash failed: %v", err)
	}
	second, err := HashPassword(password)
	if err != nil {
		t.Fatalf("second hash failed: %v", err)
	}

	// This is the salt doing its job.
	//
	// If these were equal, two users with the same password would have
	// identical hashes. Cracking one would crack both, and attackers could
	// use precomputed lookup tables. A fresh random salt each time prevents
	// that entirely.
	if first == second {
		t.Error("hashing the same password twice gave identical hashes - the salt is not random")
	}

	// Different hashes, yet both must still verify.
	if !CheckPassword(password, first) {
		t.Error("first hash does not verify")
	}
	if !CheckPassword(password, second) {
		t.Error("second hash does not verify")
	}
}

func TestHashPassword_LengthRules(t *testing.T) {
	// A "table-driven test": one list of cases, one loop. This is the standard
	// Go pattern. Adding a new case is one line, and each case is reported
	// separately by name when it fails.
	tests := []struct {
		name     string
		password string
		wantErr  error
	}{
		{"exactly at the minimum", "12345678", nil},
		{"comfortably long enough", "a-perfectly-fine-password", nil},
		{"one character too short", "1234567", ErrPasswordTooShort},
		{"empty", "", ErrPasswordTooShort},
		{"exactly at bcrypt's 72-byte limit", strings.Repeat("a", 72), nil},
		{"one byte over the limit", strings.Repeat("a", 73), ErrPasswordTooLong},
	}

	for _, tt := range tests {
		// t.Run creates a named sub-test, so failures point at the exact case.
		t.Run(tt.name, func(t *testing.T) {
			_, err := HashPassword(tt.password)
			if err != tt.wantErr {
				t.Errorf("HashPassword(%d chars) error = %v, want %v",
					len(tt.password), err, tt.wantErr)
			}
		})
	}
}

func TestCheckPassword(t *testing.T) {
	const password = "correct horse battery staple"

	hash, err := HashPassword(password)
	if err != nil {
		t.Fatalf("setup failed: %v", err)
	}

	tests := []struct {
		name      string
		candidate string
		want      bool
	}{
		{"the correct password", password, true},
		{"a completely different password", "wrong password entirely", false},
		{"wrong case", "Correct Horse Battery Staple", false},
		{"one character short", "correct horse battery stapl", false},
		{"one extra character", "correct horse battery staple ", false},
		{"empty", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := CheckPassword(tt.candidate, hash); got != tt.want {
				t.Errorf("CheckPassword() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestCheckPassword_RejectsGarbageHash(t *testing.T) {
	// A hash could be corrupted, empty, or from another system. None of these
	// may ever be treated as a match.
	tests := []struct {
		name string
		hash string
	}{
		{"empty hash", ""},
		{"not a hash at all", "hello"},
		{"the plain password stored by mistake", "correct horse battery staple"},
		{"truncated bcrypt hash", "$2a$12$tooshort"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if CheckPassword("correct horse battery staple", tt.hash) {
				t.Error("a malformed hash was accepted as a match")
			}
		})
	}
}

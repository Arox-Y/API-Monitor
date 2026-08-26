// Package auth handles passwords and login tokens.
package auth

import (
	"errors"
	"fmt"

	"golang.org/x/crypto/bcrypt"
)

// bcryptCost controls how slow hashing is. Each +1 DOUBLES the work.
//
// Slowness is the whole point. An attacker who steals the database must try
// passwords one at a time, and cost 12 makes each attempt take roughly a
// quarter of a second. At that speed, brute force is hopeless.
//
// The trade-off: our own login endpoint also takes ~250ms. That is fine for a
// human logging in, and it is the price of the protection.
//
// 12 is the common recommendation today. bcrypt.DefaultCost is 10, which is
// on the low side for something guarding real accounts.
const bcryptCost = 12

// maxPasswordLength is bcrypt's hard limit, in bytes.
//
// bcrypt SILENTLY IGNORES anything past 72 bytes. A 100-character password
// would be truncated, and the last 28 characters would do nothing at all —
// while the user believes they chose something very strong.
//
// We reject over-long passwords instead of quietly truncating them, so nobody
// ends up with a false sense of security.
const maxPasswordLength = 72

// minPasswordLength is our own policy, not a bcrypt limit.
const minPasswordLength = 8

// ErrPasswordTooShort and ErrPasswordTooLong describe invalid input.
//
// These are "sentinel errors": named error values a caller can compare
// against with errors.Is(), so it can react to a specific failure rather than
// trying to pattern-match an error message.
var (
	ErrPasswordTooShort = errors.New("password must be at least 8 characters")
	ErrPasswordTooLong  = errors.New("password must be at most 72 characters")
)

// HashPassword turns a plain password into a bcrypt hash safe to store.
//
// The returned string already contains the algorithm, the cost and the random
// salt, for example:
//
//	$2a$12$N9qo8uLOickgx2ZMRZoMy.MH/rMH7VqYRPT2s5W4Nq0hVXqJqQKGa
//	 │   │  └── 22-char salt, then the hash itself
//	 │   └───── cost (12)
//	 └───────── algorithm (bcrypt)
//
// Because the salt travels with the hash, we never store it separately.
func HashPassword(plain string) (string, error) {
	// len() on a string counts BYTES, not characters — which is exactly what
	// we want, because bcrypt's 72 limit is a byte limit. An emoji or an
	// accented letter can take several bytes.
	if len(plain) < minPasswordLength {
		return "", ErrPasswordTooShort
	}
	if len(plain) > maxPasswordLength {
		return "", ErrPasswordTooLong
	}

	// bcrypt works on bytes, so the string is converted first.
	hashed, err := bcrypt.GenerateFromPassword([]byte(plain), bcryptCost)
	if err != nil {
		return "", fmt.Errorf("hashing password: %w", err)
	}

	return string(hashed), nil
}

// CheckPassword reports whether plain matches the stored hash.
//
// Note what is NOT happening: the hash is never turned back into a password.
// bcrypt re-hashes the candidate using the salt embedded in the stored hash
// and compares the results.
//
// It returns a plain bool rather than an error because callers only ever need
// the yes/no answer, and an unchecked error here would be a security bug.
func CheckPassword(plain, hash string) bool {
	// CompareHashAndPassword returns nil when they match.
	//
	// It also compares in constant time — it always takes the same amount of
	// time regardless of how much of the hash matched. A naive comparison
	// that stops at the first difference leaks information: an attacker can
	// measure the timing and work out the value one byte at a time.
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(plain))
	return err == nil
}

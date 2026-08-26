package service

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"regexp"
	"strings"
)

// slugify turns a project name into a URL-safe identifier.
//
//	"Acme API"          -> "acme-api"
//	"  My   Project!! "  -> "my-project"
//	"Café / Bar"         -> "caf-bar"
//
// Slugs appear in public URLs (/status/acme-api), so they must contain only
// characters that are safe and readable in a URL.
var (
	nonAlphanumeric = regexp.MustCompile(`[^a-z0-9]+`)
	trimDashes      = regexp.MustCompile(`^-+|-+$`)
)

// maxSlugLength keeps URLs reasonable and fits comfortably in the column.
const maxSlugLength = 60

func slugify(name string) string {
	s := strings.ToLower(strings.TrimSpace(name))

	// Every run of characters that is not a-z or 0-9 becomes a single dash.
	// This handles spaces, punctuation and accented letters in one step.
	s = nonAlphanumeric.ReplaceAllString(s, "-")

	// Remove leading/trailing dashes left behind, so "!Hello!" is "hello"
	// rather than "-hello-".
	s = trimDashes.ReplaceAllString(s, "")

	if len(s) > maxSlugLength {
		s = s[:maxSlugLength]
		s = trimDashes.ReplaceAllString(s, "") // truncation may leave a dash
	}

	// A name made entirely of symbols ("!!!") or another alphabet would reduce
	// to an empty string, which cannot be a URL. Fall back to something valid.
	if s == "" {
		s = "project"
	}

	return s
}

// randomSuffix returns a short random string for breaking slug collisions.
//
// Slugs are globally unique, so two users both naming a project "API" cannot
// both have /status/api. The second gets /status/api-3f9c.
//
// crypto/rand is used rather than math/rand. math/rand is predictable, and a
// guessable suffix would let someone work out the URL of a status page they
// were not given.
func randomSuffix() (string, error) {
	b := make([]byte, 2) // 2 bytes -> 4 hex characters -> 65,536 possibilities
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("generating random suffix: %w", err)
	}
	return hex.EncodeToString(b), nil
}

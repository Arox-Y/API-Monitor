package service

import (
	"regexp"
	"strings"
	"testing"
)

func TestSlugify(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{"simple two words", "Acme API", "acme-api"},
		{"already a slug", "acme-api", "acme-api"},
		{"uppercase", "PRODUCTION", "production"},
		{"extra whitespace", "  My   Project  ", "my-project"},
		{"punctuation", "My Project!!", "my-project"},
		{"leading/trailing symbols", "!!!Hello!!!", "hello"},
		{"slashes", "Café / Bar", "caf-bar"},
		{"underscores and dots", "my_project.v2", "my-project-v2"},
		{"digits kept", "API v2 2026", "api-v2-2026"},
		{"only symbols falls back", "!!!", "project"},
		{"empty falls back", "", "project"},
		{"whitespace only falls back", "   ", "project"},
		{"non-latin script falls back", "日本語", "project"},
		{"multiple dashes collapse", "a---b", "a-b"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := slugify(tt.input); got != tt.want {
				t.Errorf("slugify(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

// validSlug is what a URL-safe slug must look like: lowercase letters, digits
// and single dashes, never starting or ending with a dash.
var validSlug = regexp.MustCompile(`^[a-z0-9]+(-[a-z0-9]+)*$`)

func TestSlugify_AlwaysProducesAValidSlug(t *testing.T) {
	// Whatever a user types, the result must be usable in a URL. These are the
	// awkward inputs that break naive implementations.
	inputs := []string{
		"", "   ", "!!!", "---", "a", "Z",
		"Café / Bar", "日本語", "emoji 🎉 here",
		strings.Repeat("very long name ", 20),
		"...leading dots", "trailing dots...",
		"tabs\tand\nnewlines",
		"100% uptime!",
	}

	for _, in := range inputs {
		t.Run(strings.TrimSpace(in[:min(len(in), 20)]), func(t *testing.T) {
			got := slugify(in)

			if got == "" {
				t.Fatal("slugify produced an empty string, which cannot be a URL")
			}
			if !validSlug.MatchString(got) {
				t.Errorf("slugify(%q) = %q, which is not a valid slug", in, got)
			}
			if len(got) > maxSlugLength {
				t.Errorf("slug %q is %d chars, over the %d limit", got, len(got), maxSlugLength)
			}
		})
	}
}

func TestSlugify_TruncatesLongNames(t *testing.T) {
	long := strings.Repeat("abcdefghij", 20) // 200 characters

	got := slugify(long)
	if len(got) > maxSlugLength {
		t.Errorf("slug is %d chars, want at most %d", len(got), maxSlugLength)
	}
}

func TestRandomSuffix(t *testing.T) {
	seen := make(map[string]bool)

	for i := 0; i < 100; i++ {
		s, err := randomSuffix()
		if err != nil {
			t.Fatalf("randomSuffix failed: %v", err)
		}
		if len(s) != 4 {
			t.Errorf("suffix %q has length %d, want 4", s, len(s))
		}
		if !validSlug.MatchString(s) {
			t.Errorf("suffix %q is not URL-safe", s)
		}
		seen[s] = true
	}

	// Not a strict guarantee - collisions are possible in 65,536 values - but
	// if this drops low, the generator is not random at all.
	if len(seen) < 90 {
		t.Errorf("only %d unique values out of 100; suffix is not random enough", len(seen))
	}
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

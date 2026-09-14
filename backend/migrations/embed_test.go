package migrations

import "testing"

// This test does NOT run the migrations against a real database - that is
// verified separately by actually starting the app against an empty
// database and watching it create its own schema (see PROGRESS.md).
//
// What it catches is the class of mistake that is easy to make and easy to
// miss: a typo in the go:embed pattern, or a migration file added to the
// folder that never got embedded because of it. Both would compile fine and
// only fail much later, at startup, against a real database.
func TestEmbeddedMigrations_ContainsExpectedFiles(t *testing.T) {
	entries, err := FS.ReadDir(".")
	if err != nil {
		t.Fatalf("reading embedded migrations: %v", err)
	}

	want := map[string]bool{
		"000001_init_schema.up.sql":   false,
		"000001_init_schema.down.sql": false,
	}

	for _, entry := range entries {
		if _, expected := want[entry.Name()]; expected {
			want[entry.Name()] = true
		}
	}

	for name, found := range want {
		if !found {
			t.Errorf("expected embedded file %q was not found - check the go:embed pattern in embed.go", name)
		}
	}
}

// Every "up" migration must be pairable with a "down" file that can undo it -
// enforced here as a naming convention, since golang-migrate itself does not
// require this alignment.
func TestEmbeddedMigrations_EveryUpHasADown(t *testing.T) {
	entries, err := FS.ReadDir(".")
	if err != nil {
		t.Fatalf("reading embedded migrations: %v", err)
	}

	ups := map[string]bool{}
	downs := map[string]bool{}

	for _, entry := range entries {
		name := entry.Name()
		switch {
		case len(name) > 7 && name[len(name)-7:] == ".up.sql":
			ups[name[:len(name)-7]] = true
		case len(name) > 9 && name[len(name)-9:] == ".down.sql":
			downs[name[:len(name)-9]] = true
		}
	}

	for version := range ups {
		if !downs[version] {
			t.Errorf("migration %q has an up file but no matching down file", version)
		}
	}
}

package testutil

import (
	"strings"
	"testing"
)

func TestMemoryDSNIsUniquePerCall(t *testing.T) {
	t.Parallel()
	seen := map[string]bool{}
	for range 100 {
		dsn := MemoryDSN(t)
		if seen[dsn] {
			t.Fatalf("duplicate DSN %q: repeated runs would share a database", dsn)
		}
		seen[dsn] = true
	}
}

// Subtest names can contain characters that are special in a SQLite URI. Go appends "#01" to
// duplicate subtest names, and "#" starts the URI fragment, which would push "?mode=memory"
// into the fragment and silently open an on-disk file instead of an in-memory database.
func TestMemoryDSNSurvivesAwkwardTestNames(t *testing.T) {
	t.Parallel()
	for _, name := range []string{"same", "same", "has space", "has?question", "has#hash", "a/b/c", "100%"} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			dsn := MemoryDSN(t)
			path, query, ok := strings.Cut(dsn, "?")
			if !ok || query != "mode=memory&cache=shared" {
				t.Fatalf("DSN %q lost its query: got %q", dsn, query)
			}
			if strings.ContainsAny(path, "?#% ") {
				t.Errorf("DSN path %q contains characters that are special in a URI", path)
			}
			if !strings.HasPrefix(path, "file:") {
				t.Errorf("DSN %q must be a file: URI", dsn)
			}
		})
	}
}

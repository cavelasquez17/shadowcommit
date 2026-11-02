package testutil

import "testing"

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

func TestMemoryDSNKeepsSharedInMemoryOptions(t *testing.T) {
	t.Parallel()
	dsn := MemoryDSN(t)
	for _, want := range []string{"file:", "mode=memory", "cache=shared"} {
		if !contains(dsn, want) {
			t.Errorf("DSN %q must contain %q", dsn, want)
		}
	}
}

func contains(s, sub string) bool {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}

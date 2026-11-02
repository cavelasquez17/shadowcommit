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

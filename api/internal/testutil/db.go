// Package testutil holds helpers shared by tests in several packages. Only test code
// should import it.
package testutil

import (
	"fmt"
	"strings"
	"sync/atomic"
	"testing"
)

var dsnSeq atomic.Uint64

// MemoryDSN returns a SQLite DSN for an in-memory database that no other test, and no
// earlier run of the same test, shares. "cache=shared" is required so every connection in
// one test sees the same database, but the name must be unique: t.Name() alone is not, so
// go test -count=N reused the previous run's data.
//
// The test name is kept only to make a failing database easy to trace, and is sanitised:
// subtest names can hold "#" (Go appends "#01" to duplicates) or "?", which are special in a
// URI and would push "?mode=memory" out of the query and open an on-disk file instead.
func MemoryDSN(t testing.TB) string {
	t.Helper()
	name := strings.Map(func(r rune) rune {
		switch {
		case r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z', r >= '0' && r <= '9', r == '_', r == '-':
			return r
		}
		return '_'
	}, t.Name())
	return fmt.Sprintf("file:%s-%d?mode=memory&cache=shared", name, dsnSeq.Add(1))
}

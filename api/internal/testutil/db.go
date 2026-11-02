// Package testutil holds helpers shared by tests in several packages. Only test code
// should import it.
package testutil

import (
	"fmt"
	"sync/atomic"
	"testing"
)

var dsnSeq atomic.Uint64

// MemoryDSN returns a SQLite DSN for an in-memory database that no other test, and no
// earlier run of the same test, shares. "cache=shared" is required so every connection in
// one test sees the same database, but the name must be unique: t.Name() alone is not, so
// go test -count=N reused the previous run's data.
func MemoryDSN(t testing.TB) string {
	t.Helper()
	return fmt.Sprintf("file:%s-%d?mode=memory&cache=shared", t.Name(), dsnSeq.Add(1))
}

//go:build unix

package vfs

import (
	"errors"
	"path/filepath"
	"testing"
)

// TestLock: un segundo Lock en el mismo proceso falla con ErrLocked hasta que se suelta el primero.
func TestLock(t *testing.T) {
	fsys := OS()
	name := filepath.Join(t.TempDir(), "LOCK")
	first, err := fsys.Lock(name)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := fsys.Lock(name); !errors.Is(err, ErrLocked) {
		t.Fatalf("second Lock = %v; want ErrLocked", err)
	}
	if err := first.Close(); err != nil {
		t.Fatal(err)
	}
	again, err := fsys.Lock(name)
	if err != nil {
		t.Fatalf("Lock after release: %v", err)
	}
	again.Close()
}

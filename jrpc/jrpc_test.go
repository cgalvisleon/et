package jrpc

import (
	"net"
	"testing"
	"time"

	"github.com/cgalvisleon/et/et"
)

type EchoService struct{}

func (EchoService) Echo(arg *string, reply *string) error {
	*reply = *arg
	return nil
}

// TestGetSolverBeforeMountReturnsError guards against GetSolver dereferencing a
// nil pkg (panic) when Call/CallJson/CallItems/CallItem run before any Mount in
// this process — a real path from ettp/v2's RPC pipe handler and the jrex
// "rpc.call" JS binding, neither of which guarantees Mount was called first.
func TestGetSolverBeforeMountReturnsError(t *testing.T) {
	saved := pkg
	pkg = nil
	defer func() { pkg = saved }()

	if _, err := GetSolver("Anything.Method"); err != ErrorPackageNotMounted {
		t.Fatalf("GetSolver() error = %v, want ErrorPackageNotMounted", err)
	}
}

// TestMountRegistersRpcs guards against Mount never writing into the package
// registry (rpcs), which made GET /rpc (HttpListRouters) always report an
// empty list regardless of what was actually mounted.
func TestMountRegistersRpcs(t *testing.T) {
	savedPkg, savedRpcs := pkg, rpcs
	pkg = nil
	rpcs = make(map[string]et.Json)
	defer func() { pkg, rpcs = savedPkg, savedRpcs }()

	if _, err := Mount("localhost", 9000, &EchoService{}, "TestPkg"); err != nil {
		t.Fatalf("Mount failed: %v", err)
	}

	entry, ok := rpcs["TestPkg"]
	if !ok {
		t.Fatal("expected rpcs to contain an entry for the mounted package")
	}
	if entry.Str("name") != "TestPkg" {
		t.Fatalf("unexpected rpcs entry: %v", entry)
	}
}

// TestStartCloseReleasesListener guards against Close() being a no-op (it only
// logged before), which left the RPC listener bound forever and, on any real
// close, would have spun the Accept() loop at 100% CPU instead of exiting.
func TestStartCloseReleasesListener(t *testing.T) {
	port := freePort(t)

	if err := Start(port); err != nil {
		t.Fatalf("Start failed: %v", err)
	}
	time.Sleep(50 * time.Millisecond)

	Close()
	time.Sleep(50 * time.Millisecond)

	if err := Start(port); err != nil {
		t.Fatalf("expected to rebind port %d after Close, got: %v", port, err)
	}
	Close()
}

func freePort(t *testing.T) int {
	t.Helper()
	l, err := net.Listen("tcp", ":0")
	if err != nil {
		t.Fatalf("failed to find a free port: %v", err)
	}
	port := l.Addr().(*net.TCPAddr).Port
	l.Close()
	return port
}

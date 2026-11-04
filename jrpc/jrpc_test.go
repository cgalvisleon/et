package jrpc

import (
	"bytes"
	"encoding/gob"
	"fmt"
	"net"
	"net/rpc"
	"sync"
	"testing"
	"time"

	"github.com/cgalvisleon/et/et"
)

type EchoService struct{}

func (EchoService) Echo(arg *string, reply *string) error {
	*reply = *arg
	return nil
}

type EchoService2 struct{}

func (EchoService2) Echo(arg *string, reply *string) error {
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

// TestMountGetSolverListRoutersConcurrent guards against pkg/rpcs being read
// and written from different goroutines with no synchronization: Mount wrote
// to pkg.Solvers (via Package.Add) and to rpcs while GetSolver and
// listRouters read them with no lock at all — a real "concurrent map
// read/write" hazard once Mount can run while the server is already serving
// RPC/HTTP traffic. Run with -race.
func TestMountGetSolverListRoutersConcurrent(t *testing.T) {
	savedPkg, savedRpcs := pkg, rpcs
	pkg = nil
	rpcs = make(map[string]et.Json)
	defer func() { pkg, rpcs = savedPkg, savedRpcs }()

	var wg sync.WaitGroup
	wg.Add(3)
	go func() {
		defer wg.Done()
		if _, err := Mount("localhost", 9001, &EchoService2{}, "ConcPkg"); err != nil {
			t.Errorf("Mount failed: %v", err)
		}
	}()
	go func() {
		defer wg.Done()
		for range 200 {
			GetSolver("EchoService2.Echo")
		}
	}()
	go func() {
		defer wg.Done()
		for range 200 {
			listRouters()
		}
	}()
	wg.Wait()
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

// ResolvedService is served by this process but not mounted in its package: only the resolver knows where it is
type ResolvedService struct{}

func (ResolvedService) Hello(args et.Json, reply *et.Json) error {
	*reply = et.Json{"hello": args.Str("name")}
	return nil
}

// TestResolver: a method that the local package does not have is found through the resolver (another service)
func TestResolver(t *testing.T) {
	savedPkg := pkg
	pkg = nil
	defer func() { pkg = savedPkg; SetResolver(nil) }()

	port := freePort(t)
	if err := rpc.Register(new(ResolvedService)); err != nil {
		t.Fatal(err)
	}
	if err := Start(port); err != nil {
		t.Fatal(err)
	}
	defer Close()

	SetResolver(func(method string) (*Solver, error) {
		if method != "ResolvedService.Hello" {
			return nil, fmt.Errorf("unknown %s", method)
		}
		return &Solver{Host: "localhost", Port: port}, nil
	})
	reply, err := CallJson("ResolvedService.Hello", et.Json{"name": "eam"})
	if err != nil || reply.Str("hello") != "eam" {
		t.Fatalf("CallJson = %v, %v", reply, err)
	}
	if _, err := GetSolver("Other.Method"); err == nil {
		t.Fatal("expected the resolver error")
	}
}

// TestMethods: the mounted methods with their package, host and port, as GET /rpcs lists them
func TestMethods(t *testing.T) {
	savedPkg := pkg
	pkg = newPackage("eam", "eam-host", 4512)
	defer func() { pkg = savedPkg }()

	pkg.Add("Eam.New", []string{"et.Json"}, []string{"error"})
	methods := Methods()
	if len(methods) != 1 || methods[0].Str("packageName") != "eam" || methods[0].Str("method") != "Eam.New" || methods[0].Int("port") != 4512 {
		t.Fatalf("Methods = %v", methods)
	}
}

// TestGobLists: a Json with lists, nested objects and dates goes through gob (the args and replies of the methods)
func TestGobLists(t *testing.T) {
	now := time.Now().UTC().Truncate(time.Second)
	value := et.Json{"geometry": et.Json{"type": "Point", "coordinates": []any{-74.08, 4.6}}, "rows": []et.Json{{"a": 1}}, "at": now}
	var buf bytes.Buffer
	if err := gob.NewEncoder(&buf).Encode(value); err != nil {
		t.Fatal(err)
	}
	var out et.Json
	if err := gob.NewDecoder(&buf).Decode(&out); err != nil {
		t.Fatal(err)
	}
	if len(out.Json("geometry").Array("coordinates")) != 2 || len(out.ArrayJson("rows")) != 1 || !out.Time("at").Equal(now) {
		t.Fatalf("decoded %v", out)
	}
}

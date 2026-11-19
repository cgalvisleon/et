package store

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

/**
* openTest: Abre un store ReadWrite en dir o falla la prueba.
* @param t *testing.T, dir string
* @return *FileStore
**/
func openTest(t *testing.T, dir string) *FileStore {
	t.Helper()
	s, err := Open(filepath.Join(dir, "data"), filepath.Join(dir, "wal"), "test", ReadWrite)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	return s
}

/**
* TestOpenLock: Un segundo Open ReadWrite (y Recover) del mismo store falla mientras el primero
* está abierto, con el pid en el error; tras Close abre.
* @param t *testing.T
**/
func TestOpenLock(t *testing.T) {
	dir := t.TempDir()
	s1 := openTest(t, dir)

	_, err := Open(filepath.Join(dir, "data"), filepath.Join(dir, "wal"), "test", ReadWrite)
	if err == nil {
		t.Fatal("el segundo Open debía fallar")
	}
	if !strings.Contains(err.Error(), strconv.Itoa(os.Getpid())) {
		t.Fatalf("el error no trae el pid: %v", err)
	}

	if _, err := Recover(filepath.Join(dir, "data"), filepath.Join(dir, "wal"), "test"); err == nil {
		t.Fatal("Recover con el store abierto debía fallar")
	}

	// ReadOnly no toma el bloqueo.
	ro, err := Open(filepath.Join(dir, "data"), filepath.Join(dir, "wal"), "test", ReadOnly)
	if err != nil {
		t.Fatalf("Open ReadOnly: %v", err)
	}
	ro.Close()

	if err := s1.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}
	if err := s1.Close(); err != nil {
		t.Fatalf("segundo Close: %v", err)
	}

	s2 := openTest(t, dir)
	if err := s2.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}

	if _, err := Recover(filepath.Join(dir, "data"), filepath.Join(dir, "wal"), "test"); err != nil {
		t.Fatalf("Recover con el store cerrado: %v", err)
	}
}

/**
* TestLockSurvivesCompact: La compactación renombra el directorio de segmentos; el bloqueo
* sigue vigente y los datos se conservan al reabrir.
* @param t *testing.T
**/
func TestLockSurvivesCompact(t *testing.T) {
	dir := t.TempDir()
	s := openTest(t, dir)
	for i := range 50 {
		id := fmt.Sprintf("k%03d", i)
		if _, err := s.Insert(id, []byte("v1")); err != nil {
			t.Fatal(err)
		}
		if _, err := s.Update(id, []byte("v2")); err != nil {
			t.Fatal(err)
		}
	}
	if err := s.compact(); err != nil {
		t.Fatalf("compact: %v", err)
	}

	if _, err := Open(filepath.Join(dir, "data"), filepath.Join(dir, "wal"), "test", ReadWrite); err == nil {
		t.Fatal("Open tras la compactación debía seguir bloqueado")
	}
	if err := s.Close(); err != nil {
		t.Fatal(err)
	}

	s = openTest(t, dir)
	defer s.Close()
	if n := s.Count(); n != 50 {
		t.Fatalf("Count = %d, se esperaba 50", n)
	}
	data, ok, err := s.Get("k007")
	if err != nil || !ok || string(data) != "v2" {
		t.Fatalf("Get = %q %v %v", data, ok, err)
	}
}

/**
* TestEmptyReleasesLock: Empty borra los datos y libera el bloqueo.
* @param t *testing.T
**/
func TestEmptyReleasesLock(t *testing.T) {
	dir := t.TempDir()
	s := openTest(t, dir)
	if _, err := s.Insert("a", []byte("1")); err != nil {
		t.Fatal(err)
	}
	if err := s.Empty(); err != nil {
		t.Fatalf("Empty: %v", err)
	}

	s = openTest(t, dir)
	defer s.Close()
	if n := s.Count(); n != 0 {
		t.Fatalf("Count = %d tras Empty", n)
	}
}

/**
* TestSyncDir: syncDir sobre un directorio existente no falla; sobre uno inexistente sí.
* @param t *testing.T
**/
func TestSyncDir(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "f"), []byte("x"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := syncDir(dir); err != nil {
		t.Fatalf("syncDir: %v", err)
	}
	if err := syncDir(filepath.Join(dir, "no-existe")); err == nil {
		t.Fatal("syncDir de un directorio inexistente debía fallar")
	}

	nested := filepath.Join(dir, "a", "b", "c")
	if err := mkdirAll(nested); err != nil {
		t.Fatalf("mkdirAll: %v", err)
	}
	if !exists(nested) {
		t.Fatal("mkdirAll no creó el directorio")
	}
}

/**
* TestSnapshotCloseError: Un error del Close del snapshot se propaga, no se renombra y queda
* el temporal; también a través de la rotación de un Insert.
* @param t *testing.T
**/
func TestSnapshotCloseError(t *testing.T) {
	injected := errors.New("close inyectado")
	orig := closeFile
	closeFile = func(f *os.File) error {
		f.Close()
		return injected
	}
	t.Cleanup(func() { closeFile = orig })

	dir := t.TempDir()
	s := openTest(t, dir)
	defer s.Close()

	if _, err := s.Insert("a", []byte("1")); err != nil {
		t.Fatal(err)
	}

	s.writeMu.Lock()
	err := s.createSnapshot()
	s.writeMu.Unlock()
	if !errors.Is(err, injected) {
		t.Fatalf("createSnapshot = %v, se esperaba el error del Close", err)
	}
	if exists(s.snapshotPath()) {
		t.Fatal("no debía renombrarse el snapshot")
	}
	if !exists(s.snapshotPath() + ".tmp") {
		t.Fatal("debía quedar el temporal")
	}

	// La rotación de un segmento escribe el snapshot: Insert recibe el error.
	s.MaxSegment = 64
	if _, err := s.Insert("b", []byte(strings.Repeat("x", 80))); !errors.Is(err, injected) {
		t.Fatalf("Insert con rotación = %v, se esperaba el error del Close", err)
	}
}

/**
* TestOpenLockOtherProcess: Con el store abierto por otro proceso (este binario de prueba
* re-ejecutado), Open falla con el pid de ese proceso; cuando termina, abre.
* @param t *testing.T
**/
func TestOpenLockOtherProcess(t *testing.T) {
	dir := t.TempDir()
	cmd := exec.Command(os.Args[0], "-test.run=^TestHelperHoldLock$")
	cmd.Env = append(os.Environ(), "STORE_LOCK_HELPER_DIR="+dir)
	stdin, err := cmd.StdinPipe()
	if err != nil {
		t.Fatal(err)
	}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		t.Fatal(err)
	}
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}

	line, err := bufio.NewReader(stdout).ReadString('\n')
	if err != nil || strings.TrimSpace(line) != "ready" {
		t.Fatalf("el proceso auxiliar no abrió el store: %q %v", line, err)
	}

	_, err = Open(filepath.Join(dir, "data"), filepath.Join(dir, "wal"), "test", ReadWrite)
	if err == nil {
		t.Fatal("Open debía fallar con el store abierto por otro proceso")
	}
	if !strings.Contains(err.Error(), strconv.Itoa(cmd.Process.Pid)) {
		t.Fatalf("el error no trae el pid del otro proceso (%d): %v", cmd.Process.Pid, err)
	}

	stdin.Close()
	if err := cmd.Wait(); err != nil {
		t.Fatalf("proceso auxiliar: %v", err)
	}

	s := openTest(t, dir)
	s.Close()
}

/**
* TestHelperHoldLock: Proceso auxiliar de TestOpenLockOtherProcess: abre el store, avisa «ready»
* y lo mantiene abierto hasta que se cierre su entrada estándar.
* @param t *testing.T
**/
func TestHelperHoldLock(t *testing.T) {
	dir := os.Getenv("STORE_LOCK_HELPER_DIR")
	if dir == "" {
		t.Skip("solo como proceso auxiliar")
	}
	s := openTest(t, dir)
	fmt.Println("ready")
	io.Copy(io.Discard, os.Stdin)
	s.Close()
}

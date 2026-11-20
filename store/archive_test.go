package store

import (
	"errors"
	"fmt"
	"math/rand"
	"os"
	"path/filepath"
	"slices"
	"testing"
	"time"

	"github.com/cgalvisleon/et/store/internal/crashfs"
	"github.com/cgalvisleon/et/store/internal/page"
	"github.com/cgalvisleon/et/store/internal/vfs"
	"github.com/cgalvisleon/et/store/internal/wal"
)

// walArchive es el destino de un archivador de prueba: número de segmento → sus bytes válidos.
type walArchive map[uint64][]byte

/**
* archiveConfig es testConfig con el archivo continuo y segmentos de 256 KB (cada prueba sella varios).
* @return config
**/
func archiveConfig() config {
	cfg := testConfig()
	cfg.archive = true
	cfg.wal.SegmentSize = 256 << 10
	cfg.wal.Spares = 1
	return cfg
}

/**
* pending es PendingArchive que falla la prueba ante un error y exige rangos de lsn contiguos.
* @param t testing.TB, e *Engine
* @return []WALSegment
**/
func pending(t testing.TB, e *Engine) []WALSegment {
	t.Helper()
	segs, err := e.PendingArchive()
	mustOK(t, err)
	for i := 1; i < len(segs); i++ {
		if segs[i].Segment != segs[i-1].Segment+1 || segs[i].StartLSN != segs[i-1].EndLSN+1 {
			t.Fatalf("pending segments not contiguous: %+v then %+v", segs[i-1], segs[i])
		}
	}
	for _, s := range segs {
		if s.Size <= 0 || s.EndLSN < s.StartLSN || s.Name != wal.SegmentName(s.Segment) {
			t.Fatalf("pending segment %+v", s)
		}
	}
	return segs
}

/**
* archiveAll es un archivador: copia cada segmento pendiente (sus Size bytes) a arch y lo marca archivado.
* @param t testing.TB, e *Engine, fsys vfs.FS, arch walArchive
* @return int: segmentos archivados
**/
func archiveAll(t testing.TB, e *Engine, fsys vfs.FS, arch walArchive) int {
	t.Helper()
	segs := pending(t, e)
	for _, s := range segs {
		arch[s.Segment] = readAll(t, fsys, s.Path)[:s.Size]
		mustOK(t, e.SetArchived(s.Segment))
	}
	return len(segs)
}

/**
* putFile deja en path exactamente data (lo crea si falta), con fsync del archivo y del directorio.
* @param t testing.TB, fsys vfs.FS, path string, data []byte
**/
func putFile(t testing.TB, fsys vfs.FS, path string, data []byte) {
	t.Helper()
	f, err := fsys.OpenFile(path, os.O_RDWR|os.O_CREATE)
	mustOK(t, err)
	mustOK(t, f.Truncate(int64(len(data))))
	_, err = f.WriteAt(data, 0)
	mustOK(t, err)
	mustOK(t, f.Sync())
	mustOK(t, f.Close())
	mustOK(t, fsys.SyncDir(filepath.Dir(path)))
}

/**
* addArchived agrega a wal/ de una copia los segmentos archivados desde el primero que tiene (reemplaza el último
* de la copia, que el respaldo cortó en su registro de fin).
* @param t testing.TB, dst vfs.FS, arch walArchive
**/
func addArchived(t testing.TB, dst vfs.FS, arch walArchive) {
	t.Helper()
	first := walSegments(t, dst)[0]
	for num, data := range arch {
		if num >= first {
			putFile(t, dst, filepath.Join(testDir, wal.Dir, wal.SegmentName(num)), data)
		}
	}
}

/**
* writeRounds hace rounds tandas de n escrituras al azar con un checkpoint después de cada una.
* @param t testing.TB, e *Engine, stores []*Store, rounds, n int, rng *rand.Rand
**/
func writeRounds(t testing.TB, e *Engine, stores []*Store, rounds, n int, rng *rand.Rand) {
	t.Helper()
	for range rounds {
		for range n {
			mustOK(t, writeOne(stores, 0, rng))
		}
		mustOK(t, e.Checkpoint())
	}
}

// TestArchiveRetains: con el archivo activo y sin SetArchived ningún segmento se recicla tras varios checkpoints;
// PendingArchive los da en orden con rangos contiguos y Stats los cuenta. Tras SetArchived el próximo checkpoint los
// recicla. SetArchived del activo da ErrNotSealed; repetirlo no hace nada.
func TestArchiveRetains(t *testing.T) {
	base := crashfs.New()
	e := openTest(t, base, archiveConfig())
	defer e.Close()
	stores := fillStores(t, e, []string{"a", "b"}, 100)
	writeRounds(t, e, stores, 4, 150, rand.New(rand.NewSource(1)))

	segs := walSegments(t, base)
	got := pending(t, e)
	if segs[0] != 1 || len(got) < 3 || got[0].Segment != 1 || got[0].StartLSN != 1 ||
		got[len(got)-1].Segment != e.log.ActiveSegment()-1 {
		t.Fatalf("segments %v, pending %d (active %d)", segs, len(got), e.log.ActiveSegment())
	}
	st := e.Stats()
	if !st.Archive || st.ArchivedSegment != 0 || st.ArchivePending != int64(len(got)) ||
		st.ArchiveRetained != int64(len(got))*(256<<10) || st.ArchiveGap != 0 {
		t.Fatalf("stats %+v", st)
	}

	last := got[len(got)-1].Segment
	if err := e.SetArchived(e.log.ActiveSegment()); !errors.Is(err, ErrNotSealed) {
		t.Fatalf("SetArchived of the active segment: %v", err)
	}
	mustOK(t, e.SetArchived(last))
	mustOK(t, e.SetArchived(1)) // ya archivado: nada
	mustOK(t, e.Checkpoint())
	if segs := walSegments(t, base); segs[0] != last+1 {
		t.Fatalf("segments %v after archiving up to %d", segs, last)
	}
	if st := e.Stats(); st.ArchivedSegment != last || st.ArchivePending != 0 {
		t.Fatalf("stats after archiving: %+v", st)
	}
	if got := pending(t, e); len(got) != 0 {
		t.Fatalf("pending after archiving: %+v", got)
	}
	sealed, err := e.SwitchWAL()
	mustOK(t, err)
	if got := pending(t, e); len(got) != 1 || got[0].Segment != sealed {
		t.Fatalf("pending after SwitchWAL %d: %+v", sealed, got)
	}
	checkClean(t, e)
}

// TestArchiveOff: con el archivo apagado todo es como antes: los checkpoints reciclan, no hay archive_status y la
// API del archivo da ErrArchiveOff (SwitchWAL funciona igual).
func TestArchiveOff(t *testing.T) {
	base := crashfs.New()
	cfg := archiveConfig()
	cfg.archive = false
	e := openTest(t, base, cfg)
	defer e.Close()
	stores := fillStores(t, e, []string{"a", "b"}, 100)
	writeRounds(t, e, stores, 4, 150, rand.New(rand.NewSource(1)))
	if segs := walSegments(t, base); segs[0] == 1 || len(segs) > 1+cfg.wal.Spares+1 {
		t.Fatalf("segments %v with the archive off", segs)
	}
	if _, err := e.PendingArchive(); !errors.Is(err, ErrArchiveOff) {
		t.Fatalf("PendingArchive: %v", err)
	}
	if err := e.SetArchived(1); !errors.Is(err, ErrArchiveOff) {
		t.Fatalf("SetArchived: %v", err)
	}
	_, err := e.SwitchWAL()
	mustOK(t, err)
	names, err := base.ReadDir(testDir)
	mustOK(t, err)
	if slices.Contains(names, archiveStatusName) || e.Stats().Archive {
		t.Fatalf("archive state with the archive off: %v", names)
	}
}

// TestArchivePersist: lo archivado sobrevive a Close/Open y a un corte (tres modos de crashfs) con escrituras y
// checkpoints después de SetArchived: al reabrir no se recicla nada sin archivar, lo pendiente sigue contiguo desde
// lo archivado y el segmento en que el motor quedó escribiendo se sella al abrir, así lo que el archivador ya copió
// no cambia. Lo confirmado está.
func TestArchivePersist(t *testing.T) {
	modes := map[string]crashfs.Mode{
		"close": -1, "lose": crashfs.LoseUnsynced, "reorder": crashfs.ReorderUnsynced, "torn": crashfs.TornWrites,
	}
	for name, mode := range modes {
		t.Run(name, func(t *testing.T) {
			base := crashfs.New()
			proc := newProc(base)
			cfg := archiveConfig()
			e := openTest(t, proc, cfg)
			stores := fillStores(t, e, []string{"a"}, 100)
			rng := rand.New(rand.NewSource(2))
			writeRounds(t, e, stores, 2, 150, rng)
			k := pending(t, e)[0].Segment
			mustOK(t, e.SetArchived(k))
			writeRounds(t, e, stores, 2, 100, rng)
			a := stores[0]
			mustOK(t, a.Put("last", []byte("durable")))
			_, err := e.SwitchWAL() // el activo queda vacío: al reabrir, el WAL termina en el sellado
			mustOK(t, err)
			copied := pending(t, e) // lo que un archivador ya copió (sin SetArchived)
			lastCopy := copied[len(copied)-1]
			lastBytes := readAll(t, proc, lastCopy.Path)[:lastCopy.Size]
			want := dumpEngine(t, e)
			if mode < 0 {
				mustOK(t, e.Close())
			} else {
				proc.kill(mode, 3)
				e.Close()
			}

			r := openTest(t, newProc(base), cfg)
			defer r.Close()
			checkContents(t, r, want)
			if st := r.Stats(); st.ArchivedSegment != k || st.ArchiveGap != 0 {
				t.Fatalf("stats after reopening: %+v", st)
			}
			got := pending(t, r)
			if got[0].Segment != k+1 || got[len(got)-1].Segment < lastCopy.Segment {
				t.Fatalf("pending after reopening: first %d, last %d (copied up to %d)", got[0].Segment,
					got[len(got)-1].Segment, lastCopy.Segment)
			}
			s := mustStore(t, r, "a")
			mustOK(t, s.Put("after", []byte("reopen")))
			mustOK(t, r.Checkpoint())
			if segs := walSegments(t, base); segs[0] > k+1 {
				t.Fatalf("segments %v recycled without archiving (archived %d)", segs, k)
			}
			if again := readAll(t, base, lastCopy.Path)[:lastCopy.Size]; string(again) != string(lastBytes) {
				t.Fatalf("segment %d changed after it was copied", lastCopy.Segment)
			}
			for _, seg := range pending(t, r) {
				if seg.Segment == lastCopy.Segment && seg.Size != lastCopy.Size {
					t.Fatalf("segment %d grew after reopening: %d, was %d", seg.Segment, seg.Size, lastCopy.Size)
				}
			}
			checkClean(t, r)
		})
	}
}

// TestArchiveThrottle: sin tope (WAL_ARCHIVE_MAX_SIZE = 0) un archivador atrasado frena a los escritores
// (ErrThrottled tras LOCK_TIMEOUT, Pressure por el archivo) sin perder nada; al archivar, las escrituras siguen.
func TestArchiveThrottle(t *testing.T) {
	base := crashfs.New()
	cfg := archiveConfig()
	cfg.maxWALSize = 256 << 10 // el freno a 512 KB: dos segmentos
	cfg.lockTimeout = 100 * time.Millisecond
	e := openTest(t, base, cfg)
	defer e.Close()
	s := mustStore(t, e, "a")
	want := map[string]string{}
	var throttled error
	for i := 0; i < 3000 && throttled == nil; i++ {
		id, v := fmt.Sprintf("k%04d", i%500), fmt.Sprintf("v%d-%0500d", i, i)
		if err := s.Put(id, []byte(v)); err != nil {
			throttled = err
			break
		}
		want[id] = v
		if i%50 == 0 { // el WAL desde el checkpoint no frena: solo el archivo
			mustOK(t, e.Checkpoint())
		}
	}
	if !errors.Is(throttled, ErrThrottled) {
		t.Fatalf("the late archiver did not throttle: %v", throttled)
	}
	st := e.Stats()
	if st.Throttled == 0 || st.PressureCause != PressureArchive || st.Pressure < 1 {
		t.Fatalf("stats while throttled: %+v", st)
	}
	archiveAll(t, e, base, walArchive{})
	mustOK(t, s.Put("after", []byte("archived")))
	want["after"] = "archived"
	checkContents(t, e, contents{"a": want})
	checkClean(t, e)
}

// TestArchiveCap: con WAL_ARCHIVE_MAX_SIZE el archivador atrasado no frena: el checkpoint suelta lo retenido de más,
// anota la pérdida de continuidad (ArchiveGap, persistente) y recicla. Lo confirmado está.
func TestArchiveCap(t *testing.T) {
	base := crashfs.New()
	cfg := archiveConfig()
	cfg.maxWALSize = 256 << 10
	cfg.archiveMaxSize = 512 << 10
	cfg.lockTimeout = 100 * time.Millisecond
	e := openTest(t, base, cfg)
	s := mustStore(t, e, "a")
	want := map[string]string{}
	for i := range 3000 {
		id, v := fmt.Sprintf("k%04d", i%500), fmt.Sprintf("v%d-%0500d", i, i)
		mustOK(t, s.Put(id, []byte(v)))
		want[id] = v
		if i%50 == 0 {
			mustOK(t, e.Checkpoint())
		}
	}
	mustOK(t, e.Checkpoint())
	st := e.Stats()
	if st.ArchiveGap == 0 || st.ArchiveLost == 0 || st.ArchiveRetained > cfg.archiveMaxSize+cfg.wal.SegmentSize {
		t.Fatalf("stats with the cap: %+v", st)
	}
	if segs := walSegments(t, base); segs[0] == 1 || len(segs) > 8 {
		t.Fatalf("segments %v with the cap", segs)
	}
	if got := pending(t, e); len(got) > 0 && got[0].Segment < st.ArchiveGap {
		t.Fatalf("pending %+v before the gap %d", got[0], st.ArchiveGap)
	}
	checkContents(t, e, contents{"a": want})
	mustOK(t, e.Close())
	r := openTest(t, base, cfg)
	defer r.Close()
	if r.Stats().ArchiveGap < st.ArchiveGap {
		t.Fatalf("gap after reopening: %+v", r.Stats())
	}
	checkContents(t, r, contents{"a": want})
}

// TestArchivePITR: un respaldo, escrituras y los segmentos archivados después. La copia con esos segmentos en wal/
// llega hasta el final de lo archivado; con un objetivo (OpenAt) llega hasta ese lsn y no más. Verify limpio. Un
// objetivo antes del fin del respaldo, después del final del WAL o sin copia de respaldo da ErrRecoveryTarget sin
// cambiar nada.
func TestArchivePITR(t *testing.T) {
	base, full, mid, bad := crashfs.New(), crashfs.New(), crashfs.New(), crashfs.New()
	cfg := archiveConfig()
	e := openTest(t, base, cfg)
	defer e.Close()
	stores := fillStores(t, e, []string{"a", "b"}, 100)
	arch := walArchive{}
	archiveAll(t, e, base, arch)
	var info BackupInfo
	for _, dst := range []vfs.FS{full, mid, bad} {
		var err error
		info, err = e.backup(dst, testDir)
		mustOK(t, err)
	}
	rng := rand.New(rand.NewSource(4))
	for range 300 {
		mustOK(t, writeOne(stores, 0, rng))
	}
	mustOK(t, mustStore(t, e, "b").Put("phase", []byte("one")))
	want1, lsn1 := dumpEngine(t, e), e.Stats().LSN
	for range 300 {
		mustOK(t, writeOne(stores, 1, rng))
	}
	mustOK(t, mustStore(t, e, "b").Put("phase", []byte("two")))
	want2 := dumpEngine(t, e)
	_, err := e.SwitchWAL()
	mustOK(t, err)
	if n := archiveAll(t, e, base, arch); n < 3 {
		t.Fatalf("only %d segments archived after the backup", n)
	}
	last := e.Stats().ArchivedSegment

	addArchived(t, full, arch)
	r := restore(t, full, cfg)
	checkContents(t, r, want2)
	mustOK(t, r.Close())

	addArchived(t, mid, arch)
	at := cfg
	at.archive = false
	at.stopLSN = page.LSN(lsn1)
	r = restore(t, mid, at)
	checkContents(t, r, want1)
	mustOK(t, r.Close())
	r = openTest(t, mid, testConfig()) // reabierta, es un motor más
	checkContents(t, r, want1)
	mustOK(t, r.Close())

	addArchived(t, bad, arch)
	for _, stop := range []uint64{info.EndLSN - 1, uint64(len(arch)) << 40} {
		at.stopLSN = page.LSN(stop)
		if _, err := open(bad, testDir, at); !errors.Is(err, ErrRecoveryTarget) {
			t.Fatalf("OpenAt %d (backup end %d): %v", stop, info.EndLSN, err)
		}
	}
	at.stopLSN = page.LSN(info.EndLSN)
	r = restore(t, bad, at)
	mustOK(t, r.Close())
	at.stopLSN = page.LSN(lsn1)
	if _, err := open(bad, testDir, at); !errors.Is(err, ErrRecoveryTarget) {
		t.Fatalf("OpenAt without backup_label: %v", err)
	}
	t.Logf("backup %+v, archived up to segment %d, target lsn %d", info, last, lsn1)
}

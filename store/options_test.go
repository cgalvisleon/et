package store

import (
	"path/filepath"
	"testing"
	"time"

	"github.com/cgalvisleon/et/envar"
)

// TestOptionsApply (tarea 84): Options{} deja la configuración del entorno; cada campo fijado la cambia, con los
// negativos apagando lo que 0 apaga; MAX_WAL_SIZE y WAL_SEGMENT_SIZE recalculan los repuestos del WAL.
func TestOptionsApply(t *testing.T) {
	base := envConfig()
	same := base
	Options{}.apply(&same)
	if same.buffers != base.buffers || same.wal != base.wal || same.vacuumEvery != base.vacuumEvery ||
		same.syncOnWrite != base.syncOnWrite || same.walWriterDelay != base.walWriterDelay {
		t.Fatalf("Options{} changed the config: %+v", same)
	}
	off, on := false, true
	cfg := envConfig()
	Options{CacheMB: 2, WALSegmentMB: 4, MaxWALMB: 32, WALBuffersKB: -1, SyncOnWrite: &off, Archive: &on,
		CheckpointTimeout: 7 * time.Second, LockTimeout: time.Second, TransactionTimeout: -1, WALWriterDelay: -1,
		VacuumInterval: -1, VacuumThreshold: 9, ArchiveMaxMB: 3, SlotKeepMB: -1}.apply(&cfg)
	switch {
	case cfg.buffers != 2<<20/8192, cfg.wal.SegmentSize != 4<<20, cfg.maxWALSize != 32<<20, cfg.wal.Spares != 4,
		cfg.wal.Buffer != 0, cfg.syncOnWrite, !cfg.archive, cfg.checkpointTimeout != 7*time.Second,
		cfg.lockTimeout != time.Second, cfg.txTimeout != 0, cfg.walWriterDelay != 0, cfg.vacuumEvery != 0,
		cfg.vacuumWrites != 9, cfg.archiveMaxSize != 3<<20, cfg.slotKeepSize != 0:
		t.Fatalf("Options did not apply: %+v", cfg)
	}
}

// TestOpenWith (tarea 84): dos motores en el mismo proceso con cachés distintas (Stats.CacheSize), sin tocar las
// variables de entorno; OpenReplicaWith abre una copia como réplica con sus opciones.
func TestOpenWith(t *testing.T) {
	envar.SetInt("STORE_CACHE_SIZE", 8)
	dir := t.TempDir()
	small, err := OpenWith(filepath.Join(dir, "local"), Options{CacheMB: 1, WALSegmentMB: 1, MaxWALMB: 4})
	mustOK(t, err)
	defer small.Close()
	big, err := OpenWith(filepath.Join(dir, "data"), Options{})
	mustOK(t, err)
	if small.Stats().CacheSize != 128 || big.Stats().CacheSize != 8<<20/8192 {
		t.Fatalf("cache sizes %d and %d", small.Stats().CacheSize, big.Stats().CacheSize)
	}
	s := mustStore(t, big, "a")
	mustOK(t, s.Put("k", []byte("v")))
	_, err = big.Backup(filepath.Join(dir, "copy"))
	mustOK(t, err)
	mustOK(t, big.Close())
	r, err := OpenReplicaWith(filepath.Join(dir, "copy"), Options{CacheMB: 1})
	mustOK(t, err)
	defer r.Close()
	if st := r.Stats(); !st.Replica || st.CacheSize != 128 {
		t.Fatalf("replica stats %+v", st)
	}
	if v, ok, err := mustStore(t, r, "a").Read("k"); err != nil || !ok || string(v) != "v" {
		t.Fatalf("replica read %q %v %v", v, ok, err)
	}
}

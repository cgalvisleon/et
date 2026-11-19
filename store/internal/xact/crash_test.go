package xact

import (
	"fmt"
	"testing"

	"github.com/cgalvisleon/et/store/internal/crashfs"
	"github.com/cgalvisleon/et/store/internal/page"
)

// crashModes son los tres modos de corte de crashfs, con su nombre para los subtests.
var crashModes = map[string]crashfs.Mode{
	"LoseUnsynced":    crashfs.LoseUnsynced,
	"ReorderUnsynced": crashfs.ReorderUnsynced,
	"TornWrites":      crashfs.TornWrites,
}

// TestFlushSurvivesCrash: lo escrito con Flush sobrevive a un corte en los tres modos, también en un archivo nuevo
// (Flush hace fsync del directorio). Lo anotado después y no confirmado (aunque una expulsión lo haya escrito) queda
// con su estado nuevo o en curso, nunca otro; con LoseUnsynced, en curso.
func TestFlushSurvivesCrash(t *testing.T) {
	const dir = "/db"
	// Páginas distintas y dos archivos, con caché de dos páginas: hay expulsiones antes del Flush.
	flushed := []page.XID{1, 2, XidsPerPage + 5, 3*XidsPerPage + 7, XidsPerFile + 1}
	// Después del Flush: la misma página 0 que ya estaba en el disco, páginas nuevas y un tercer archivo.
	later := []page.XID{3, 2*XidsPerPage + 1, 4*XidsPerPage + 2, XidsPerFile + XidsPerPage + 3, 2*XidsPerFile + 1}
	statusOf := func(i int) Status {
		if i%2 == 0 {
			return StatusCommitted
		}
		return StatusAborted
	}

	for name, mode := range crashModes {
		for seed := int64(0); seed < 8; seed++ {
			t.Run(fmt.Sprintf("%s/seed%d", name, seed), func(t *testing.T) {
				fs := crashfs.New()
				fl := &fakeFlusher{}
				s := openTest(t, fs, dir, fl, 2)
				for i, xid := range flushed {
					if err := s.SetStatus(xid, statusOf(i), page.LSN(i+1)); err != nil {
						t.Fatal(err)
					}
				}
				if err := s.Flush(); err != nil {
					t.Fatal(err)
				}
				for i, xid := range later {
					if err := s.SetStatus(xid, statusOf(i), page.LSN(100+i)); err != nil {
						t.Fatal(err)
					}
				}

				fs.Crash(mode, seed)
				s.Close()
				s = openTest(t, fs, dir, fl, 2)
				defer s.Close()
				for i, xid := range flushed {
					wantStatus(t, s, xid, statusOf(i))
				}
				for i, xid := range later {
					got, _, err := s.Status(xid)
					if err != nil {
						t.Fatalf("Status(%d): %v", xid, err)
					}
					if got != StatusInProgress && (got != statusOf(i) || mode == crashfs.LoseUnsynced) {
						t.Fatalf("Status(%d) = %d after crash", xid, got)
					}
				}
			})
		}
	}
}

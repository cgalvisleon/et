package wal

import (
	"bytes"
	"errors"
	"os"
	"testing"

	"github.com/cgalvisleon/et/store/internal/page"
	"github.com/cgalvisleon/et/store/internal/vfs"
)

// TestReadAppendCopy: Read entrega los registros de un WAL con su posición, de a tandas acotadas por bytes; Append
// los escribe en otro WAL en la misma posición (también tras un Switch del primero, que la réplica sigue rotando), y
// los segmentos quedan iguales byte a byte hasta su final válido. Locate encuentra cualquier registro; un registro
// fuera de orden da ErrOutOfOrder sin tocar nada; tras reciclar, Locate y Read dan ErrRemoved.
func TestReadAppendCopy(t *testing.T) {
	fsys := vfs.OS()
	primaryDir, replicaDir := t.TempDir(), t.TempDir()
	opts := Options{SegmentSize: 2 * MaxRecordSize, Spares: 0, Preallocate: true}
	primary, _, _ := openTest(t, fsys, primaryDir, opts, Start)
	replica, _, _ := openTest(t, fsys, replicaDir, opts, Start)

	for i := 1; i <= 120; i++ {
		appendRecord(t, primary, testRecord(i))
		if i == 50 {
			if _, err := primary.Switch(); err != nil {
				t.Fatal(err)
			}
		}
	}
	if err := primary.WaitDurable(primary.Published()); err != nil {
		t.Fatal(err)
	}

	pos, copied := Start, 0
	for pos.LSN <= primary.Durable() {
		next, err := primary.Read(pos, primary.Durable(), 4096, func(at Position, raw []byte, rec *Record) error {
			if at.LSN != rec.LSN {
				t.Fatalf("position lsn %d, record lsn %d", at.LSN, rec.LSN)
			}
			copied++
			return replica.Append(at, raw)
		})
		if err != nil {
			t.Fatal(err)
		}
		if next == pos {
			t.Fatalf("Read did not advance from %+v", pos)
		}
		pos = next
	}
	if copied != 120 || replica.Published() != 120 {
		t.Fatalf("copied %d, replica published %d", copied, replica.Published())
	}
	bad := encoded(t, testRecord(500), 500)
	if err := replica.Append(Position{Segment: replica.ActiveSegment(), Offset: 0, LSN: 500}, bad); !errors.Is(err, ErrOutOfOrder) {
		t.Fatalf("out of order append: %v", err)
	}
	if err := replica.WaitDurable(replica.Published()); err != nil {
		t.Fatal(err)
	}
	if primary.ActiveSegment() != replica.ActiveSegment() {
		t.Fatalf("active segments %d and %d", primary.ActiveSegment(), replica.ActiveSegment())
	}
	for num := uint64(1); num < primary.ActiveSegment(); num++ {
		info, err := primary.SegmentInfo(num)
		if err != nil {
			t.Fatal(err)
		}
		a, _ := os.ReadFile(segPath(primaryDir, num))
		b, _ := os.ReadFile(segPath(replicaDir, num))
		if !bytes.Equal(a[:info.End.Offset], b[:info.End.Offset]) {
			t.Fatalf("segment %d differs", num)
		}
	}
	for _, lsn := range []page.LSN{1, 50, 51, 77, 120} {
		at, err := primary.Locate(lsn)
		if err != nil || at.LSN != lsn {
			t.Fatalf("Locate %d: %+v %v", lsn, at, err)
		}
		var got page.LSN
		if _, err := primary.Read(at, lsn, 1, func(_ Position, _ []byte, rec *Record) error {
			got = rec.LSN
			return nil
		}); err != nil || got != lsn {
			t.Fatalf("Read at the located %d: %d %v", lsn, got, err)
		}
	}

	if err := primary.Recycle(Position{Segment: primary.ActiveSegment(), LSN: 121}); err != nil {
		t.Fatal(err)
	}
	if _, err := primary.Locate(1); !errors.Is(err, ErrRemoved) {
		t.Fatalf("Locate after recycling: %v", err)
	}
	if _, err := primary.Read(Start, 120, 1<<20, func(Position, []byte, *Record) error { return nil }); !errors.Is(err, ErrRemoved) {
		t.Fatalf("Read after recycling: %v", err)
	}
}

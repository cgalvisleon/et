package wal

import (
	"github.com/cgalvisleon/et/envar"
	"github.com/cgalvisleon/et/store/internal/page"
)

// Options es lo configurable del WAL. EnvOptions las lee del entorno; las pruebas las fijan a mano.
type Options struct {
	// SegmentSize es el tamaño máximo de un segmento en bytes (WAL_SEGMENT_SIZE). Al menos MaxRecordSize.
	SegmentSize int64

	// Spares es cuántos segmentos reciclados (renombrados a números futuros) se guardan para reutilizar; Recycle
	// borra los demás.
	Spares int

	// Preallocate escribe cada segmento entero con ceros (y fsync) antes de poner registros en él, así un registro
	// reservado nunca pide lugar nuevo en el disco y el disco lleno sale en Room, antes de empezar una operación (ver
	// «Preasignación» en doc.go). El motor lo usa siempre; las pruebas del paquete, según el caso.
	Preallocate bool

	// StopLSN, si no es 0, es el objetivo de la recuperación de Open (recovery_target_lsn de PostgreSQL): el
	// recorrido termina tras el registro de ese lsn y lo que sigue se corta como una cola inválida.
	StopLSN page.LSN
}

// EnvOptions lee las opciones de WAL_SEGMENT_SIZE (MB, mínimo 1) y MAX_WAL_SIZE (MB): se guardan como repuesto los
// segmentos que llenan la mitad de MAX_WAL_SIZE (a confirmar), al menos uno. Siempre con preasignación.
func EnvOptions() Options {
	segment := max(envar.GetInt64("WAL_SEGMENT_SIZE", 128), MinSegmentSize>>20) << 20
	maxWAL := max(envar.GetInt64("MAX_WAL_SIZE", 1024), 1) << 20
	return Options{SegmentSize: segment, Spares: int(max(maxWAL/2/segment, 1)), Preallocate: true}
}

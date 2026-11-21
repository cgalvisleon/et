package crashfs

import (
	"cmp"
	"slices"
)

// extent es un tramo [start, end) de un archivo con lugar asignado en el disco.
type extent struct{ start, end int64 }

// extents son los tramos asignados de un archivo, ordenados, sin solaparse ni tocarse. Lo que queda fuera de ellos
// y dentro del largo del archivo es un hueco: se lee como ceros y no ocupa disco (como en ext4 o XFS).
type extents []extent

// size suma los bytes asignados.
func (x extents) size() int64 {
	var total int64
	for _, e := range x {
		total += e.end - e.start
	}
	return total
}

// missing cuenta los bytes de [start, end) que todavía no tienen lugar: lo que escribir ahí agrega al disco.
func (x extents) missing(start, end int64) int64 {
	if end <= start {
		return 0
	}
	covered := int64(0)
	for _, e := range x {
		lo, hi := max(e.start, start), min(e.end, end)
		if lo < hi {
			covered += hi - lo
		}
	}
	return end - start - covered
}

// add devuelve los tramos con [start, end) asignado, uniendo los que se tocan.
func (x extents) add(start, end int64) extents {
	if end <= start {
		return x
	}
	out := make(extents, 0, len(x)+1)
	for _, e := range x {
		switch {
		case e.end < start || e.start > end:
			out = append(out, e)
		default:
			start, end = min(start, e.start), max(end, e.end)
		}
	}
	out = append(out, extent{start, end})
	slices.SortFunc(out, func(a, b extent) int { return cmp.Compare(a.start, b.start) })
	return out
}

// cut devuelve los tramos sin lo que pasa de size (un truncado que achica).
func (x extents) cut(size int64) extents {
	out := make(extents, 0, len(x))
	for _, e := range x {
		if e.start >= size {
			continue
		}
		out = append(out, extent{e.start, min(e.end, size)})
	}
	return out
}

// clone copia los tramos.
func (x extents) clone() extents {
	return slices.Clone(x)
}

package store

import (
	"fmt"
	"strings"
)

// MaxFindings es cuántos hallazgos guarda un Report con su detalle; los demás solo se cuentan (Report.Total). Así el
// informe de un motor muy dañado no crece con las páginas.
const MaxFindings = 1000

// FindingKind es la clase de un hallazgo de Verify o Repair.
type FindingKind string

const (
	FindControl     FindingKind = "control"      // el archivo de control no se lee o no coincide con el motor
	FindCatalog     FindingKind = "catalog"      // el catálogo guardado no se lee o no coincide con el de memoria
	FindOrphanFile  FindingKind = "orphan-file"  // archivos de datos de un número sin store (reparable: se borran)
	FindUnknownFile FindingKind = "unknown-file" // un archivo en data/ que no tiene nombre de archivo de datos
	FindNoMeta      FindingKind = "no-meta"      // store sin página meta: se crea vacío al abrirlo con Store
	FindTree        FindingKind = "tree"         // el árbol no pasa btree.Verify o una página suya no se lee
	FindChain       FindingKind = "chain"        // la cadena de desborde de una versión está dañada
	FindValue       FindingKind = "value"        // un valor comprimido (en línea o en desborde) no se descomprime (formato 2)
	FindDouble      FindingKind = "double-page"  // una página es de dos dueños (árbol, cadena o lista de libres)
	FindFreeList    FindingKind = "free-list"    // la lista de libres apunta a una página que no es libre o se sale
	FindLostPage    FindingKind = "lost-page"    // páginas sin dueño (reparable: vuelven a la lista de libres)
	FindOrphanChain FindingKind = "orphan-chain" // páginas de desborde sin versión (reparable: vuelven a la lista)
	FindVisibility  FindingKind = "visibility"   // un bit del mapa de visibilidad mal puesto (reparable: se borra)
	FindFrozen      FindingKind = "frozen"       // una versión sin congelar por debajo del frozenXID del store
	FindCount       FindingKind = "count"        // Count no coincide con el recorrido (reparable: se corrige)
	FindStatus      FindingKind = "status"       // el archivo de estados no da el estado de una versión
	FindHalfDead    FindingKind = "half-dead"    // un borrado de páginas cortado a medias (reparable: se termina)
)

// Finding es un hallazgo: dónde está y qué pasa.
type Finding struct {
	Kind       FindingKind
	Store      string // "" si es del motor (control, catálogo, archivos)
	File       string // archivo afectado, relativo al directorio del motor ("data/0000000003", "control"…)
	Block      int64  // primer bloque afectado del fork principal; -1 si no es de un bloque
	Blocks     int64  // cuántos bloques seguidos desde Block (1 si es uno solo; 0 si no es de un bloque)
	Desc       string
	Repairable bool // Repair sabe arreglarlo sin perder nada sano
	Repaired   bool // Repair lo arregló (con WAL; ver «Verificación y reparación» en doc.go)
}

// Report es el resultado de Verify o Repair: los totales recorridos y los hallazgos (hasta MaxFindings).
type Report struct {
	Stores      int   // stores revisados
	Pages       int64 // bloques del fork principal de esos stores
	TreePages   int64 // páginas del árbol enganchadas en sus niveles (sin las metas ni las borradas)
	Deleted     int64 // páginas borradas del árbol que esperan reciclarse (en la pila de su meta)
	ChainPages  int64 // páginas de cadenas de desborde vivas
	FreePages   int64 // páginas en la lista de libres
	LostPages   int64 // páginas sin dueño, de cadenas huérfanas incluidas
	Entries     int64 // versiones en las hojas
	LiveIDs     int64 // ids vivos según el recorrido (lo que debería dar Count)
	OrphanFiles int   // números de archivo de datos sin store
	LeavesForm1 int64 // hojas de la forma 1 (formato 1) que la limpieza todavía no reescribió en la forma 2

	Findings    []Finding // los primeros MaxFindings hallazgos
	Total       int       // todos los hallazgos, también los que no entraron en Findings
	Repaired    int       // hallazgos que Repair arregló
	Irreparable int       // hallazgos que Repair no sabe arreglar
}

/**
* Clean dice si no hubo ningún hallazgo.
* @return bool
**/
func (r Report) Clean() bool {
	return r.Total == 0
}

/**
* Pending cuenta los hallazgos que siguen sin arreglar (todos en un Verify; en un Repair, los que no arregló).
* @return int
**/
func (r Report) Pending() int {
	return r.Total - r.Repaired
}

/**
* String da el informe para imprimir: una línea de totales y una por hallazgo.
* @return string
**/
func (r Report) String() string {
	var b strings.Builder
	fmt.Fprintf(&b, MSG_REPORT_TOTALS, r.Stores, r.Pages, r.TreePages, r.Deleted, r.ChainPages, r.FreePages, r.LostPages,
		r.Entries, r.LiveIDs, r.OrphanFiles, r.LeavesForm1)
	fmt.Fprintf(&b, MSG_REPORT_FINDINGS, r.Total, r.Repaired, r.Irreparable)
	for _, f := range r.Findings {
		b.WriteString(f.String())
		b.WriteByte('\n')
	}
	if extra := r.Total - len(r.Findings); extra > 0 {
		fmt.Fprintf(&b, MSG_REPORT_OMITTED, extra)
	}
	return b.String()
}

/**
* String da un hallazgo en una línea.
* @return string
**/
func (f Finding) String() string {
	where := f.File
	if f.Store != "" {
		where = fmt.Sprintf("store %q %s", f.Store, f.File)
	}
	if f.Block >= 0 && f.Blocks > 1 {
		where += fmt.Sprintf(" blocks %d-%d", f.Block, f.Block+f.Blocks-1)
	} else if f.Block >= 0 {
		where += fmt.Sprintf(" block %d", f.Block)
	}
	state := "irreparable"
	switch {
	case f.Repaired:
		state = "repaired"
	case f.Repairable:
		state = "repairable"
	}
	return fmt.Sprintf("[%s] %s: %s (%s)", f.Kind, where, f.Desc, state)
}

/**
* add anota un hallazgo: lo cuenta siempre y lo guarda si todavía entra.
* @param f Finding
* @return int: su posición en Findings, o -1 si no entró
**/
func (r *Report) add(f Finding) int {
	r.Total++
	if !f.Repairable {
		r.Irreparable++
	}
	if f.Repaired {
		r.Repaired++
	}
	if len(r.Findings) >= MaxFindings {
		return -1
	}
	r.Findings = append(r.Findings, f)
	return len(r.Findings) - 1
}

/**
* repaired marca como arreglado el hallazgo de la posición i (la que devolvió add).
* @param i int
**/
func (r *Report) repaired(i int) {
	r.Repaired++
	if i >= 0 {
		r.Findings[i].Repaired = true
	}
}

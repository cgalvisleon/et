package main

import (
	"bytes"
	"fmt"
	"maps"
	"slices"

	"github.com/cgalvisleon/et/store"
)

// summary es el resultado de comprobar un motor contra el registro.
type summary struct {
	confirmed int // confirmaciones anotadas en el registro (todas deben estar)
	ops       int // operaciones con efecto posible
	keys      int // claves vivas encontradas
	uncertain int // operaciones después de la última confirmada de cada escritor (pueden estar o no)
	survived  int // de esas, cuántas sobrevivieron (el prefijo elegido)
	repaired  int // hallazgos esperables tras un corte (páginas perdidas, cadenas o archivos huérfanos) que arregló Repair
	problems  []string
}

/**
* ok dice si no hubo problemas.
* @return bool
**/
func (s summary) ok() bool {
	return len(s.problems) == 0
}

/**
* String da una línea con los totales y, debajo, cada problema.
* @return string
**/
func (s summary) String() string {
	var b bytes.Buffer
	fmt.Fprintf(&b, "confirmadas=%d ops=%d claves=%d inciertas=%d sobrevivieron=%d huerfanos_reparados=%d problemas=%d",
		s.confirmed, s.ops, s.keys, s.uncertain, s.survived, s.repaired, len(s.problems))
	for _, p := range s.problems {
		b.WriteString("\n  - " + p)
	}
	return b.String()
}

/**
* readStore lee el contenido del store name (vacío si no existe) como id → seq de la escritura que dejó el valor,
* comprobando cada valor byte por byte contra el que se escribió.
* @param e *store.Engine
* @param rec *record
* @param g int
* @return map[string]int64, bool (si el store existe), []string (problemas)
**/
func readStore(e *store.Engine, rec *record, g int) (map[string]int64, bool, []string) {
	name := storeName(g)
	out := map[string]int64{}
	if !slices.Contains(e.Stores(), name) {
		return out, false, nil
	}
	s, err := e.Store(name)
	if err != nil {
		return out, true, []string{fmt.Sprintf("%s: Store: %v", name, err)}
	}
	var problems []string
	err = s.ForEach(func(id string, data []byte) (bool, error) {
		wg, seq, key, ok := parseValue(data)
		switch {
		case !ok || wg != g || key != id:
			problems = append(problems, fmt.Sprintf("%s/%s: valor ajeno o roto (%q…)", name, id, data[:min(len(data), 40)]))
		case !bytes.Equal(data, value(rec.seed, rec.profile, g, seq, id)):
			problems = append(problems, fmt.Sprintf("%s/%s: valor de la op %d con bytes distintos (largo %d)", name, id, seq, len(data)))
		}
		out[id] = seq
		return len(problems) < 20, nil
	}, true, 0, 0)
	if err != nil {
		problems = append(problems, fmt.Sprintf("%s: ForEach: %v", name, err))
	}
	return out, true, problems
}

/**
* checkEngine compara el motor abierto con el registro: lo confirmado y durable está con su último valor, lo
* posterior (incierto) sobrevivió como un prefijo de las operaciones de cada escritor (un lote entero o nada) y,
* con full, Count coincide con el contenido y Engine.Verify da limpio.
* @param e *store.Engine
* @param rec *record
* @param full bool: false en un motor de solo lectura (Count puede incluir la operación que falló; Verify no corre)
* @return summary
**/
func checkEngine(e *store.Engine, rec *record, full bool) summary {
	var sum summary
	for g, h := range rec.hist {
		sum.confirmed += h.count
		sum.ops += len(h.ops)
		actual, exists, problems := readStore(e, rec, g)
		sum.problems = append(sum.problems, problems...)
		sum.keys += len(actual)

		state := map[string]int64{}
		for _, o := range h.ops[:h.confirmed+1] {
			apply(state, o)
		}
		later := h.ops[h.confirmed+1:]
		sum.uncertain += len(later)
		match := maps.Equal(state, actual)
		for i := 0; !match && i < len(later); i++ {
			apply(state, later[i])
			if match = maps.Equal(state, actual); match {
				sum.survived += i + 1
			}
		}
		if !match {
			sum.problems = append(sum.problems, describeMismatch(rec, g, h, actual))
		}
		if full && exists {
			if s, err := e.Store(storeName(g)); err == nil && s.Count() != int64(len(actual)) {
				sum.problems = append(sum.problems, fmt.Sprintf("%s: Count=%d pero el recorrido da %d", storeName(g), s.Count(), len(actual)))
			}
		}
	}
	if full {
		sum.problems = append(sum.problems, verifyEngine(e, &sum)...)
	}
	return sum
}

/**
* verifyEngine corre Engine.Verify. Tras un corte acepta solo lo que doc.go da por esperable (páginas perdidas,
* cadenas de desborde huérfanas y archivos huérfanos: escritos antes de un registro que no llegó a ser durable), lo
* arregla con Repair y exige que el Verify siguiente dé limpio, como checkAfterCrash en las pruebas del motor.
* @param e *store.Engine
* @param sum *summary: anota cuántos hallazgos esperables reparó
* @return []string: los problemas
**/
func verifyEngine(e *store.Engine, sum *summary) []string {
	rep, err := e.Verify()
	if err != nil {
		return []string{fmt.Sprintf("Engine.Verify: %v", err)}
	}
	if rep.Clean() {
		return nil
	}
	for _, f := range rep.Findings {
		switch f.Kind {
		case store.FindLostPage, store.FindOrphanChain, store.FindOrphanFile:
		default:
			return []string{"Engine.Verify no da limpio:\n" + rep.String()}
		}
	}
	sum.repaired += rep.Total
	if _, err := e.Repair(); err != nil {
		return []string{fmt.Sprintf("Engine.Repair: %v", err)}
	}
	if rep, err = e.Verify(); err != nil || !rep.Clean() {
		return []string{fmt.Sprintf("Engine.Verify tras Repair: %v\n%s", err, rep.String())}
	}
	return nil
}

/**
* describeMismatch explica por qué el contenido de un store no es ningún prefijo válido: las diferencias con el
* estado tras la última confirmada.
* @param rec *record
* @param g int
* @param h *history
* @param actual map[string]int64
* @return string
**/
func describeMismatch(rec *record, g int, h *history, actual map[string]int64) string {
	state := map[string]int64{}
	for _, o := range h.ops[:h.confirmed+1] {
		apply(state, o)
	}
	var diffs []string
	for _, k := range slices.Sorted(maps.Keys(state)) {
		if a, ok := actual[k]; !ok {
			diffs = append(diffs, fmt.Sprintf("%s falta (esperado de la op %d)", k, state[k]))
		} else if a != state[k] {
			diffs = append(diffs, fmt.Sprintf("%s tiene la op %d, esperado la %d", k, a, state[k]))
		}
	}
	for _, k := range slices.Sorted(maps.Keys(actual)) {
		if _, ok := state[k]; !ok {
			diffs = append(diffs, fmt.Sprintf("%s sobra (op %d)", k, actual[k]))
		}
	}
	last := int64(0)
	if h.confirmed >= 0 {
		last = h.ops[h.confirmed].seq
	}
	return fmt.Sprintf("%s: el contenido no es ningún prefijo válido (última confirmada op %d, %d inciertas después); frente a lo confirmado: %v",
		storeName(g), last, len(h.ops)-h.confirmed-1, diffs[:min(len(diffs), 12)])
}

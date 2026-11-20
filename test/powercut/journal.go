package main

import (
	"bufio"
	"fmt"
	"math/rand/v2"
	"os"
	"strconv"
	"strings"
	"sync"
)

// El registro de confirmaciones vive fuera del disco que se corta. Una línea por evento, escrita con una sola
// llamada a Write (sin búfer), así un kill -9 no deja confirmaciones a medias:
//
//	H <seed> <workers> <profile>     cabecera
//	I <g> <seq> <op> [args…]         intento: se anota antes de ejecutar la operación
//	C <g> <seq>                      confirmada y durable (y con ella todas las anteriores del mismo escritor)
//	F <g> <seq>                      rechazada sin efecto (ErrBusy, ErrThrottled, ErrConflict, ErrDiskFull limpio)
//
// Operaciones (minúscula = diferida, sin esperar el fsync):
//
//	P k / p k        Put             D k / d k        Delete
//	B ±k… / b ±k…    lote: +k es PutIn, -k es DeleteIn; B = Commit, b = CommitDeferred
//	E                Empty           X                DropStore (y Store de nuevo)

// journal es el registro de confirmaciones compartido por los escritores.
type journal struct {
	mu sync.Mutex
	f  *os.File
}

/**
* openJournal abre (o crea) el registro para agregar al final.
* @param path string
* @return *journal, error
**/
func openJournal(path string) (*journal, error) {
	f, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
	if err != nil {
		return nil, err
	}
	return &journal{f: f}, nil
}

/**
* line escribe una línea entera con un solo Write.
* @param format string
* @param args ...any
**/
func (j *journal) line(format string, args ...any) {
	j.mu.Lock()
	defer j.mu.Unlock()
	if _, err := j.f.WriteString(fmt.Sprintf(format, args...) + "\n"); err != nil {
		panic("powercut: no se pudo escribir el registro: " + err.Error())
	}
}

/**
* close sincroniza y cierra el registro.
**/
func (j *journal) close() {
	j.f.Sync()
	j.f.Close()
}

// op es una operación de un escritor tal como quedó en el registro.
type op struct {
	seq  int64
	kind byte     // P p D d B b E X
	keys []string // para B/b, con su signo
}

// history es lo que el registro dice de un escritor: sus operaciones con efecto posible, en orden, y la última
// confirmada.
type history struct {
	ops       []op
	confirmed int // índice en ops de la última confirmada; -1 si ninguna
	count     int // confirmaciones anotadas
}

// record es el registro entero ya leído.
type record struct {
	seed    uint64
	workers int
	profile string
	hist    []*history
}

/**
* readRecord lee el registro. Ignora una última línea incompleta (no debería haberla: cada línea es un Write).
* @param path string
* @return *record, error
**/
func readRecord(path string) (*record, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	rec := &record{}
	failed := map[[2]int64]bool{}
	var all []struct {
		g int
		o op
	}
	confirmedSeq := map[int]int64{}
	countC := map[int]int{}
	sc := bufio.NewScanner(f)
	sc.Buffer(make([]byte, 1<<20), 1<<20)
	for sc.Scan() {
		fs := strings.Fields(sc.Text())
		if len(fs) == 0 {
			continue
		}
		switch fs[0] {
		case "H":
			if len(fs) < 4 {
				continue
			}
			rec.seed, _ = strconv.ParseUint(fs[1], 10, 64)
			rec.workers, _ = strconv.Atoi(fs[2])
			rec.profile = fs[3]
		case "I":
			if len(fs) < 4 || len(fs[3]) != 1 {
				continue
			}
			g, _ := strconv.Atoi(fs[1])
			seq, _ := strconv.ParseInt(fs[2], 10, 64)
			all = append(all, struct {
				g int
				o op
			}{g, op{seq: seq, kind: fs[3][0], keys: fs[4:]}})
		case "C", "F":
			if len(fs) < 3 {
				continue
			}
			g, _ := strconv.Atoi(fs[1])
			seq, _ := strconv.ParseInt(fs[2], 10, 64)
			if fs[0] == "F" {
				failed[[2]int64{int64(g), seq}] = true
			} else {
				countC[g]++
				if seq > confirmedSeq[g] {
					confirmedSeq[g] = seq
				}
			}
		}
	}
	if err := sc.Err(); err != nil {
		return nil, err
	}
	if rec.workers <= 0 {
		return nil, fmt.Errorf("registro sin cabecera: %s", path)
	}
	rec.hist = make([]*history, rec.workers)
	for i := range rec.hist {
		rec.hist[i] = &history{confirmed: -1}
	}
	for _, it := range all {
		if it.g < 0 || it.g >= rec.workers || failed[[2]int64{int64(it.g), it.o.seq}] {
			continue
		}
		h := rec.hist[it.g]
		h.ops = append(h.ops, it.o)
		if c, ok := confirmedSeq[it.g]; ok && it.o.seq <= c {
			h.confirmed = len(h.ops) - 1
		}
	}
	for g, h := range rec.hist {
		h.count = countC[g]
	}
	return rec, nil
}

/**
* apply aplica una operación al estado de un store del modelo (id → seq de la escritura que dejó su valor).
* @param state map[string]int64
* @param o op
**/
func apply(state map[string]int64, o op) {
	switch o.kind {
	case 'P', 'p':
		state[o.keys[0]] = o.seq
	case 'D', 'd':
		delete(state, o.keys[0])
	case 'B', 'b':
		for _, k := range o.keys {
			if k[0] == '+' {
				state[k[1:]] = o.seq
			} else {
				delete(state, k[1:])
			}
		}
	case 'E', 'X':
		clear(state)
	}
}

/**
* storeName da el store que usa el escritor g (cada escritor tiene el suyo, así su orden es el del WAL).
* @param g int
* @return string
**/
func storeName(g int) string {
	return fmt.Sprintf("s%d", g)
}

/**
* value da el valor determinista que escribe el escritor g en su operación seq para la clave key. Empieza con
* «g/seq/key|», así la verificación sabe qué escritura dejó un valor y comprueba el resto byte por byte.
* @param seed uint64
* @param profile string: "load" (valores chicos y alguno de desborde) o "big" (para llenar el disco)
* @param g int
* @param seq int64
* @param key string
* @return []byte
**/
func value(seed uint64, profile string, g int, seq int64, key string) []byte {
	h := uint64(14695981039346656037)
	for _, c := range []byte(key) {
		h = (h ^ uint64(c)) * 1099511628211
	}
	r := rand.New(rand.NewPCG(seed^h, uint64(g)<<48^uint64(seq)))
	n := 16 + r.IntN(300)
	switch {
	case profile == "big":
		n = 2000 + r.IntN(6000)
	case r.IntN(12) == 0:
		n = 2000 + r.IntN(30000) // desborde
	}
	head := fmt.Sprintf("%d/%d/%s|", g, seq, key)
	out := make([]byte, len(head)+n)
	copy(out, head)
	for i := len(head); i < len(out); i += 8 {
		x := r.Uint64()
		for b := 0; b < 8 && i+b < len(out); b++ {
			out[i+b] = 'a' + byte(x>>(8*b))%26
		}
	}
	return out
}

/**
* parseValue lee la cabecera «g/seq/key|» de un valor.
* @param data []byte
* @return g int, seq int64, key string, ok bool
**/
func parseValue(data []byte) (int, int64, string, bool) {
	s := string(data[:min(len(data), 128)])
	i := strings.IndexByte(s, '|')
	if i < 0 {
		return 0, 0, "", false
	}
	parts := strings.SplitN(s[:i], "/", 3)
	if len(parts) != 3 {
		return 0, 0, "", false
	}
	g, err1 := strconv.Atoi(parts[0])
	seq, err2 := strconv.ParseInt(parts[1], 10, 64)
	if err1 != nil || err2 != nil {
		return 0, 0, "", false
	}
	return g, seq, parts[2], true
}

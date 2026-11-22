package main

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"math/rand/v2"
	"sort"
	"strconv"
)

/**
* model es el estado esperado de los stores de datos: por store y por índice de clave, si está viva y su generación
* (cuántas veces se escribió). El valor se deriva de (semilla, store, índice, generación), así que no se guarda.
* Cada índice tiene un solo dueño (índice % escritores), que es el único que lo cambia: el modelo no necesita candados
* entre escritores; el verificador lo lee con la carga detenida (gate).
**/
type model struct {
	seed     uint64
	keys     int
	base     int // tamaño medio de un valor normal
	largePct int // % de valores grandes (desborde)
	gen      [][]uint32
	live     [][]bool
}

/**
* newModel crea el modelo vacío.
* @param seed uint64, stores, keys, base, largePct int
* @return *model
**/
func newModel(seed uint64, stores, keys, base, largePct int) *model {
	m := &model{seed: seed, keys: keys, base: base, largePct: largePct}
	m.gen = make([][]uint32, stores)
	m.live = make([][]bool, stores)
	for s := range m.gen {
		m.gen[s] = make([]uint32, keys)
		m.live[s] = make([]bool, keys)
	}
	return m
}

/**
* keyOf da el id del índice i: «k» + 7 dígitos, así el orden de los ids es el de los índices.
* @param i int
* @return string
**/
func keyOf(i int) string {
	return fmt.Sprintf("k%07d", i)
}

/**
* indexOf es la inversa de keyOf.
* @param id string
* @return int, bool
**/
func indexOf(id string) (int, bool) {
	if len(id) != 8 || id[0] != 'k' {
		return 0, false
	}
	i, err := strconv.Atoi(id[1:])
	return i, err == nil
}

/**
* mix mezcla los cuatro números en una semilla (splitmix64).
* @param a, b, c, d uint64
* @return uint64
**/
func mix(a, b, c, d uint64) uint64 {
	x := a ^ b*0x9e3779b97f4a7c15 ^ c*0xbf58476d1ce4e5b9 ^ d*0x94d049bb133111eb
	x ^= x >> 30
	x *= 0xbf58476d1ce4e5b9
	x ^= x >> 27
	x *= 0x94d049bb133111eb
	return x ^ x>>31
}

/**
* valueLen da el largo del valor de (s, i, g): base ±25 %, o entre 4 y 24 KB para el largePct % (cadena de desborde).
* @param s, i int, g uint32
* @return int
**/
func (m *model) valueLen(s, i int, g uint32) int {
	h := mix(m.seed, uint64(s), uint64(i), uint64(g))
	if int(h%100) < m.largePct {
		return 4096 + int(h>>8%20480)
	}
	return m.base*3/4 + int(h>>8%uint64(m.base/2+1))
}

/**
* value genera el valor de (s, i, g): 12 bytes de cabecera (s, i, g) y el resto pseudoaleatorio.
* @param s, i int, g uint32
* @return []byte
**/
func (m *model) value(s, i int, g uint32) []byte {
	n := m.valueLen(s, i, g)
	b := make([]byte, n)
	binary.LittleEndian.PutUint32(b[0:], uint32(s))
	binary.LittleEndian.PutUint32(b[4:], uint32(i))
	binary.LittleEndian.PutUint32(b[8:], g)
	r := rand.New(rand.NewPCG(mix(m.seed, uint64(s), uint64(i), uint64(g)), 7))
	for p := 12; p < n; p += 8 {
		var w [8]byte
		binary.LittleEndian.PutUint64(w[:], r.Uint64())
		copy(b[p:], w[:])
	}
	return b
}

/**
* expect compara lo leído con el modelo de (s, i).
* @param s, i int, data []byte, found bool
* @return error: nil si coincide
**/
func (m *model) expect(s, i int, data []byte, found bool) error {
	if found != m.live[s][i] {
		return fmt.Errorf("store %d clave %s: existe=%v, el modelo dice %v (gen %d)", s, keyOf(i), found, m.live[s][i], m.gen[s][i])
	}
	if found && !bytes.Equal(data, m.value(s, i, m.gen[s][i])) {
		return fmt.Errorf("store %d clave %s: valor distinto al de la generación %d (%d bytes leídos)", s, keyOf(i), m.gen[s][i], len(data))
	}
	return nil
}

/**
* liveStats cuenta las claves vivas y sus bytes (id + valor) de todos los stores.
* @return int64, int64
**/
func (m *model) liveStats() (count, bytes int64) {
	for s := range m.live {
		for i, ok := range m.live[s] {
			if ok {
				count++
				bytes += int64(8 + m.valueLen(s, i, m.gen[s][i]))
			}
		}
	}
	return count, bytes
}

/**
* liveCount cuenta las claves vivas del store s.
* @param s int
* @return int64
**/
func (m *model) liveCount(s int) int64 {
	var n int64
	for _, ok := range m.live[s] {
		if ok {
			n++
		}
	}
	return n
}

/**
* sortedNames devuelve una copia ordenada.
* @param names []string
* @return []string
**/
func sortedNames(names []string) []string {
	out := append([]string(nil), names...)
	sort.Strings(out)
	return out
}

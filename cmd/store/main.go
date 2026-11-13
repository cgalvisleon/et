package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"github.com/cgalvisleon/et/store"
)

/**
* main: Ejemplo de uso del paquete store: abrir, escribir, leer, recorrer y cerrar.
* Sin -dir trabaja en un directorio temporal que borra al terminar.
**/
func main() {
	dir := flag.String("dir", "", "directorio de datos (vacío = temporal, se borra al terminar)")
	flag.Parse()

	root := *dir
	if root == "" {
		tmp, err := os.MkdirTemp("", "et-store-")
		check(err)
		defer os.RemoveAll(tmp)
		root = tmp
	}
	fmt.Println("datos en:", root)

	// ---- Abrir ----
	pathData := filepath.Join(root, "data") // segmentos con los registros
	pathWal := filepath.Join(root, "wal")   // snapshot, compactación y cuarentena
	s, err := store.Open(pathData, pathWal, "usuarios", store.ReadWrite)
	check(err)
	defer s.Close()

	// Sync recibe cada cambio aplicado, en orden de LSN (lo usaría una réplica).
	s.Sync(func(c store.Change) {
		fmt.Printf("  [sync] lsn=%d op=%s id=%s\n", c.LSN, c.Op, c.ID)
	})

	// ---- Escribir ----
	ok, err := s.Insert("u1", []byte(`{"name":"Ana"}`)) // solo si "u1" NO existe
	show("Insert u1", ok, err)

	ok, err = s.Insert("u1", []byte(`{"name":"Otra"}`)) // ya existe: false
	show("Insert u1 otra vez", ok, err)

	ok, err = s.Update("u1", []byte(`{"name":"Ana M"}`)) // solo si existe y el valor cambió
	show("Update u1", ok, err)

	ok, err = s.Update("u1", []byte(`{"name":"Ana M"}`)) // mismo valor: false, no escribe
	show("Update u1 igual", ok, err)

	ok, err = s.Update("u9", []byte(`{"name":"Nadie"}`)) // no existe: false
	show("Update u9", ok, err)

	// Escrituras diferidas: no esperan el fsync; WaitDurable las confirma juntas.
	for _, id := range []string{"u2", "u3", "u4"} {
		_, err := s.Deferred().Insert(id, []byte(`{"name":"`+id+`"}`))
		check(err)
	}
	check(s.WaitDurable())
	fmt.Println("Deferred u2..u4 + WaitDurable: ok")

	// ---- Leer ----
	data, found, err := s.Get("u1")
	check(err)
	fmt.Printf("Get u1 → found=%v data=%s\n", found, data)

	data, version, found, err := s.GetVersion("u1") // version = LSN del último registro
	check(err)
	fmt.Printf("GetVersion u1 → found=%v version=%d data=%s\n", found, version, data)

	// Concurrencia optimista: escribir solo si nadie cambió u1 desde que se leyó.
	if _, now, _, _ := s.GetVersion("u1"); now == version {
		ok, err = s.Update("u1", []byte(`{"name":"Ana María"}`))
		show("Update u1 (versión sin cambios)", ok, err)
	}

	fmt.Println("IsExist u2:", s.IsExist("u2"), "· Count:", s.Count())
	fmt.Println("Keys desc:", s.Keys(false, 0, 10))

	// ---- Recorrer ----
	fmt.Println("ForEach asc:")
	err = s.ForEach(func(id string, data []byte) (bool, error) { // en orden de clave
		fmt.Printf("  %s = %s\n", id, data)
		return true, nil // false = detenerse
	}, true, 0, 100) // asc, offset, limit
	check(err)

	// ---- Borrar ----
	ok, err = s.Delete("u1") // solo si existe
	show("Delete u1", ok, err)

	ok, err = s.Delete("u1") // ya no existe: false
	show("Delete u1 otra vez", ok, err)

	_, found, err = s.Get("u1")
	check(err)
	fmt.Println("Get u1 tras Delete → found:", found)

	// ---- Mantenimiento y estado ----
	check(s.Compact()) // reescribe solo lo vivo
	fmt.Println("Compact: ok")
	fmt.Println("Estado:", s.ToString())
}

/**
* show: Imprime el resultado de una escritura.
* @param label string, ok bool, err error
**/
func show(label string, ok bool, err error) {
	check(err)
	fmt.Printf("%s → %v\n", label, ok)
}

/**
* check: Termina el programa si err no es nil.
* @param err error
**/
func check(err error) {
	if err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}

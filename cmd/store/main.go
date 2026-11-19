// Ejemplo de uso de et/store: abre un store, escribe, lee y borra registros, y lo cierra.
//
//	go run ./cmd/store
package main

import (
	"fmt"
	"log"
	"os"

	"github.com/cgalvisleon/et/store"
)

func main() {
	dir, err := os.MkdirTemp("", "et-store-")
	if err != nil {
		log.Fatal(err)
	}
	defer os.RemoveAll(dir)

	s, err := store.Open(dir)
	if err != nil {
		log.Fatal(err)
	}

	// Put guarda siempre: crea o reemplaza.
	check(s.Put("customer:001", []byte(`{"name":"Ana"}`)))

	// Read devuelve el valor y si existe.
	data, ok, err := s.Read("customer:001")
	check(err)
	fmt.Printf("Read customer:001 → %s (existe: %v)\n", data, ok)

	// Insert solo guarda si el id no existe.
	ok, err = s.Insert("customer:001", []byte(`{"name":"Otra"}`))
	check(err)
	fmt.Println("Insert customer:001 (ya existía) →", ok)

	ok, err = s.Insert("customer:002", []byte(`{"name":"Beto"}`))
	check(err)
	fmt.Println("Insert customer:002 (nuevo) →", ok)

	// Update solo guarda si el id existe y el valor cambió.
	ok, err = s.Update("customer:001", []byte(`{"name":"Ana María"}`))
	check(err)
	fmt.Println("Update customer:001 →", ok)

	// Delete solo borra si el id existe; Remove borra sin preguntar.
	ok, err = s.Delete("customer:002")
	check(err)
	fmt.Println("Delete customer:002 →", ok)
	check(s.Remove("customer:001"))

	_, ok, err = s.Read("customer:001")
	check(err)
	fmt.Println("Read customer:001 después de Remove → existe:", ok)

	check(s.Close())
}

/**
* check: Termina el programa si hubo un error.
* @param err error
**/
func check(err error) {
	if err != nil {
		log.Fatal(err)
	}
}

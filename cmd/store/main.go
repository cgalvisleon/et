// Ejemplo de uso de et/store: abre un store, guarda un registro con Put y lo cierra.
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

	if err := s.Put("customer:001", []byte(`{"name":"Ana"}`)); err != nil {
		log.Fatal(err)
	}
	fmt.Println("customer:001 guardado en", dir)

	if err := s.Close(); err != nil {
		log.Fatal(err)
	}
}

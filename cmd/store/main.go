// Ejemplo de uso de et/store: abrir un motor, un store, las operaciones sueltas de la fase 1A y los lotes y
// snapshots de la fase 1B.
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

	// Un motor es un directorio; cada store es un árbol propio dentro de él.
	engine, err := store.Open(dir)
	if err != nil {
		log.Fatal(err)
	}
	users, err := engine.Store("users")
	check(err)

	// Put guarda siempre; Read devuelve la versión más nueva confirmada.
	check(users.Put("ana", []byte(`{"name":"Ana"}`)))
	data, found, err := users.Read("ana")
	check(err)
	fmt.Printf("read ana: %s (found %v)\n", data, found)

	// Insert, Update y Delete dicen si escribieron.
	inserted, err := users.Insert("ana", []byte(`{"name":"Otra"}`))
	check(err)
	fmt.Println("insert ana again:", inserted)

	updated, err := users.Update("ana", []byte(`{"name":"Ana María"}`))
	check(err)
	fmt.Println("update ana:", updated)

	deleted, err := users.Delete("nadie")
	check(err)
	fmt.Println("delete nadie:", deleted)

	// Remove borra sin preguntar.
	check(users.Put("beto", []byte(`{"name":"Beto"}`)))
	check(users.Remove("beto"))
	fmt.Println("count:", users.Count(), "stores:", engine.Stores())

	// Un lote es una transacción en uno o varios stores: ve sus propios cambios y confirma todo junto.
	orders, err := engine.Store("orders")
	check(err)
	snap := engine.Snapshot() // foto fija de todos los stores, tomada antes del lote
	tx := engine.NewBatch()
	check(users.PutIn(tx, "carla", []byte(`{"name":"Carla"}`)))
	check(orders.PutIn(tx, "o-1", []byte(`{"user":"carla"}`)))
	_, found, err = users.ReadIn(tx, "carla")
	check(err)
	_, outside, err := users.Read("carla")
	check(err)
	fmt.Println("batch sees carla:", found, "- others see it:", outside)
	check(tx.Commit())

	// Rollback deshace sin dejar rastro.
	tx = engine.NewBatch()
	check(orders.PutIn(tx, "o-2", []byte(`{}`)))
	check(tx.Rollback())
	_, found, err = orders.Read("o-2")
	check(err)
	fmt.Println("o-2 after rollback:", found)

	// El snapshot sigue viendo la base como cuando se tomó.
	_, found, err = users.ReadAt(snap, "carla")
	check(err)
	fmt.Println("snapshot sees carla:", found)
	check(snap.Close())

	// Vacuum quita las versiones que ya nadie puede ver (también corre de fondo, cada VACUUM_INTERVAL).
	check(engine.Vacuum())

	// Close hace el checkpoint final; al reabrir, los datos siguen ahí.
	check(engine.Close())
	engine, err = store.Open(dir)
	check(err)
	users, err = engine.Store("users")
	check(err)
	data, _, err = users.Read("ana")
	check(err)
	fmt.Printf("after reopen: %s\n", data)
	check(engine.Close())
}

// check termina el programa si hubo un error.
func check(err error) {
	if err != nil {
		log.Fatal(err)
	}
}

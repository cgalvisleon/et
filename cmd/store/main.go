// Ejemplo de uso de et/store.
//
// El paquete se está reescribiendo sobre la arquitectura de PostgreSQL (F-23): el motor nuevo todavía no tiene API
// pública, que llega en la ola 4. Hasta entonces este programa solo lo avisa; el ejemplo (Open, Put, Read, lotes,
// snapshots, Close) vuelve con la API nueva. El diseño está en store/doc.go.
//
//	go run ./cmd/store
package main

import "fmt"

func main() {
	fmt.Println("et/store se está reescribiendo (F-23): el ejemplo vuelve con la API nueva. Ver store/doc.go.")
}

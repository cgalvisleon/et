// storecheck revisa (o repara) un motor de et/store e imprime el informe.
//
//	go run ./cmd/storecheck -dir <dir> [-repair]
//
// Abre el motor (con su recuperación), corre Engine.Verify o, con -repair, Engine.Repair, imprime el informe y lo
// cierra. Sale con 0 si no queda nada pendiente, 1 si quedan hallazgos sin arreglar y 2 si no pudo revisar.
package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/cgalvisleon/et/store"
)

func main() {
	dir := flag.String("dir", "", "directorio del motor")
	repair := flag.Bool("repair", false, "arreglar lo reparable (con WAL)")
	flag.Parse()
	if *dir == "" {
		flag.Usage()
		os.Exit(2)
	}
	if _, err := os.Stat(*dir); err != nil {
		fail(err) // Open crearía un motor vacío
	}

	engine, err := store.Open(*dir)
	if err != nil {
		fail(err)
	}
	check := engine.Verify
	if *repair {
		check = engine.Repair
	}
	report, err := check()
	fmt.Print(report.String())
	if cerr := engine.Close(); err == nil {
		err = cerr
	}
	if err != nil {
		fail(err)
	}
	if report.Pending() > 0 {
		os.Exit(1)
	}
}

/**
* fail imprime el error y sale con 2.
* @param err error
**/
func fail(err error) {
	fmt.Fprintln(os.Stderr, "storecheck:", err)
	os.Exit(2)
}

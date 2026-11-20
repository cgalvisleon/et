//go:build race

package overflow

// raceEnabled dice si las pruebas corren con -race (el detector multiplica la memoria de cada byte escrito).
const raceEnabled = true

package mem

import (
	"sync/atomic"

	"github.com/cgalvisleon/et/et"
	"github.com/cgalvisleon/et/logs"
)

type Peticiones struct {
	peticiones atomic.Int64
	Capacity   int /* Capacidad de peticiones del ACS */
	TimeWait   int /* Tiempo de espera para la siguiente petición */
	SizeStack  int /* Tamaño de la pila de peticiones */
}

/**
* NewPeticiones: Crea un limitador de concurrencia.
* @param capacity int
* @param timeWait int
* @return *Peticiones
**/
func NewPeticiones(capacity, timeWait int) *Peticiones {
	return &Peticiones{
		Capacity:  capacity,
		TimeWait:  timeWait,
		SizeStack: capacity * timeWait,
	}
}

/**
* Ejecucion: Ejecuta una función bajo control de concurrencia.
* @param executeFc func(params et.Json) (et.Items, error)
* @param params et.Json
* @return et.Items, error
**/
func (c *Peticiones) Ejecucion(executeFc func(params et.Json) (et.Items, error), params et.Json) (et.Items, error) {
	limit := int64(c.SizeStack)
	for {
		current := c.peticiones.Load()
		if current >= limit {
			return et.Items{}, logs.Alertf(`Se ha superado el límite de peticiones %d`, c.SizeStack)
		}
		if c.peticiones.CompareAndSwap(current, current+1) {
			break
		}
	}
	defer c.peticiones.Add(-1)

	return executeFc(params)
}

/**
* GetPeticiones: Retorna el número de peticiones activas.
* @return int
**/
func (c *Peticiones) GetPeticiones() int {
	return int(c.peticiones.Load())
}

/**
* GetConfig: Retorna la configuración del limitador.
* @return et.Json
**/
func (c *Peticiones) GetConfig() et.Json {
	return et.Json{
		"peticiones": c.peticiones.Load(),
		"capacity":   c.Capacity,
		"timeWait":   c.TimeWait,
		"sizeStack":  c.SizeStack,
	}
}


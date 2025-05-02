package mem

import (
	"sync"

	"github.com/cgalvisleon/et/et"
	"github.com/cgalvisleon/et/logs"
)

type Peticiones struct {
	mutex      sync.RWMutex
	Peticiones int
	Capacity   int /* Capacidad de peticiones del ACS */
	TimeWait   int /* Tiempo de espera para la siguiente petición */
	SizeStack  int /* Tamaño de la pila de peticiones */
}

func NewPeticiones(capacity, timeWait int) *Peticiones {
	return &Peticiones{
		Peticiones: 0,
		Capacity:   capacity,
		TimeWait:   timeWait,
		SizeStack:  capacity * timeWait,
	}
}

func (c *Peticiones) Ejecucion(executeFc func(params et.Json) (et.Items, error), params et.Json) (et.Items, error) {
	turno := c.Peticiones + 1
	if turno > c.SizeStack {
		return et.Items{}, logs.Alertf(`Se ha superado el límite de peticiones %d`, c.SizeStack)
	}

	c.mutex.Lock()
	c.Peticiones++
	c.mutex.Unlock()

	// Ejecutar la petición
	result, err := executeFc(params)
	if err != nil {
		return et.Items{}, err
	}

	c.mutex.Lock()
	c.Peticiones--
	c.mutex.Unlock()

	return result, nil
}

func (c *Peticiones) GetPeticiones() int {
	c.mutex.RLock()
	defer c.mutex.RUnlock()

	return c.Peticiones
}

func (c *Peticiones) GetConfig() et.Json {
	return et.Json{
		"peticiones": c.Peticiones,
		"capacity":   c.Capacity,
		"timeWait":   c.TimeWait,
		"sizeStack":  c.SizeStack,
	}
}

var execute *Peticiones

func init() {
	execute = NewPeticiones(10, 1)
}

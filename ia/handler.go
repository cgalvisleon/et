package ia

import (
	"sync"

	"github.com/cgalvisleon/et/envar"
	"github.com/cgalvisleon/et/instances"
)

func New(store instances.Store) *Agents {
	result := &Agents{
		agents:  make(map[string]*Agent),
		mu:      sync.RWMutex{},
		isDebug: envar.GetBool("DEBUG", false),
	}

	if store != nil {
		result.getInstance = store.Get
		result.setInstance = store.Set
	}

	return result
}

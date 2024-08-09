package mem

import (
	"sync"

	"github.com/cgalvisleon/et/logs"
)

type Mem struct {
	items map[string]*Item
	locks map[string]*sync.RWMutex
}

func Load() (*Mem, error) {
	result := &Mem{
		items: make(map[string]*Item),
		locks: make(map[string]*sync.RWMutex),
	}

	logs.Logf("Mem", "Load memory cache")

	return result, nil
}

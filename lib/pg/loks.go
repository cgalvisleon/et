package lib

import "sync"

/**
* wg return a wait group
* @param tag string
* @return *sync.WaitGroup
**/
func (d *Postgres) wg(tag string) *sync.WaitGroup {
	if d.wgs[tag] == nil {
		d.wgs[tag] = &sync.WaitGroup{}
	}

	return d.wgs[tag]
}

/**
* wgAdd add a delta to a wait group
* @param tag string
* @param delta int
* @return *sync.WaitGroup
**/
func (d *Postgres) wgAdd(tag string, delta int) *sync.WaitGroup {
	result := d.wg(tag)
	result.Add(delta)

	return result
}

/**
* lock return a lock
* @param tag string
* @return *sync.RWMutex
**/
func (d *Postgres) lock(tag string) *sync.RWMutex {
	if d.locks[tag] == nil {
		d.locks[tag] = &sync.RWMutex{}
	}

	return d.locks[tag]
}

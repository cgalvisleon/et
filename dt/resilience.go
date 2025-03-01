package dt

import "time"

type Step struct {
	Step  int
	Limit int
}

var Resilencia map[string]*Step

func init() {
	Resilencia = make(map[string]*Step)
}

/**
* Resilience
* @param key string
* @return int
**/
func Resilience(key string, limit int) int {
	if Resilencia[key] == nil {
		Resilencia[key] = &Step{Step: 0, Limit: limit}
	}

	Resilencia[key].Step++

	clean := func() {
		ResilienceReset(key)
	}

	duration := 24 * 7 * time.Hour
	if duration != 0 {
		go time.AfterFunc(duration, clean)
	}

	return Resilencia[key].Step
}

/**
* ResilienceReset
* @param key string
**/
func ResilienceReset(key string) {
	delete(Resilencia, key)
}

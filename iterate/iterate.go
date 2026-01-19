package iterate

import (
	"time"

	"github.com/cgalvisleon/et/logs"
	"github.com/cgalvisleon/et/timezone"
)

type Iterate struct {
	iterates map[string]time.Time
}

/**
* start
* @param tag string
**/
func (s *Iterate) start(tag string) {
	s.iterates[tag] = timezone.Now()
}

/**
* segment
* @param tag string, msg string, isDebug bool
* @return time.Duration
**/
func (s *Iterate) segment(tag, msg string, isDebug bool) time.Duration {
	if isDebug {
		return 0
	}

	start, ok := s.iterates[tag]
	if !ok {
		return 0
	}

	end := timezone.Now()
	elapsed := end.Sub(start)
	s.iterates[tag] = end
	if msg == "" {
		logs.Infof(`%s:elapsed: %d`, tag, elapsed.Milliseconds())
	} else {
		logs.Infof(`%s:elapsed: %d | %s`, tag, elapsed.Milliseconds(), msg)
	}

	return elapsed
}

/**
* end
* @param tag string, msg string, isDebug bool
* @return time.Duration
**/
func (s *Iterate) end(tag, msg string, isDebug bool) time.Duration {
	result := s.segment(tag, msg, isDebug)
	delete(s.iterates, tag)
	return result
}

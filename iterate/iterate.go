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
	s.iterates[tag] = timezone.NowTime()
}

/**
* segment
* @param tag string, msg string, isDebug bool
**/
func (s *Iterate) segment(tag, msg string, isDebug bool) {
	if isDebug {
		return
	}

	start, ok := s.iterates[tag]
	if !ok {
		return
	}

	end := timezone.NowTime()
	elapsed := end.Sub(start)
	s.iterates[tag] = end
	if msg == "" {
		logs.Infof(`%s:elapsed: %d`, tag, elapsed.Milliseconds())
	} else {
		logs.Infof(`%s:elapsed: %d | %s`, tag, elapsed.Milliseconds(), msg)
	}
}

/**
* end
* @param tag string, msg string, isDebug bool
**/
func (s *Iterate) end(tag, msg string, isDebug bool) {
	s.segment(tag, msg, isDebug)
	delete(s.iterates, tag)
}

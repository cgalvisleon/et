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
* Start
* @param tag string
**/
func (s *Iterate) Start(tag string) {
	s.iterates[tag] = timezone.NowTime()
}

/**
* Segment
* @param tag string, msg string, isDebug bool
**/
func (s *Iterate) Segment(tag, msg string, isDebug bool) {
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
* End
* @param tag string, msg string, isDebug bool
**/
func (s *Iterate) End(tag, msg string, isDebug bool) {
	s.Segment(tag, msg, isDebug)
	delete(s.iterates, tag)
}

var iterate *Iterate

func init() {
	iterate = &Iterate{
		iterates: make(map[string]time.Time),
	}
}

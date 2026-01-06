package iterate

import "time"

var iterate *Iterate

func init() {
	iterate = &Iterate{
		iterates: make(map[string]time.Time),
	}
}

/**
* Start
* @param tag string
**/
func Start(tag string) {
	iterate.start(tag)
}

/**
* Segment
* @param tag string, msg string, isDebug bool
**/
func Segment(tag, msg string, isDebug bool) {
	iterate.segment(tag, msg, isDebug)
}

/**
* End
* @param tag string, msg string, isDebug bool
**/
func End(tag, msg string, isDebug bool) {
	iterate.end(tag, msg, isDebug)
}

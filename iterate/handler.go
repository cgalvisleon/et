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
* @return time.Duration
**/
func Segment(tag, msg string, isDebug bool) time.Duration {
	return iterate.segment(tag, msg, isDebug)
}

/**
* End
* @param tag string, msg string, isDebug bool
* @return time.Duration
**/
func End(tag, msg string, isDebug bool) time.Duration {
	return iterate.end(tag, msg, isDebug)
}

package iterate

import "time"

var iterate *Iterate

func init() {
	iterate = &Iterate{
		iterates: make(map[string]time.Time, 0),
	}
}

/**
* Start
* @param tag string
**/
func Start(tag string) {
	if iterate == nil {
		return
	}
	iterate.start(tag)
}

/**
* Segment
* @param tag string, msg string, isDebug bool
* @return time.Duration
**/
func Segment(tag, msg string, isDebug bool) time.Duration {
	if iterate == nil {
		return 0
	}
	return iterate.segment(tag, msg, isDebug)
}

/**
* End
* @param tag string, msg string, isDebug bool
* @return time.Duration
**/
func End(tag, msg string, isDebug bool) time.Duration {
	if iterate == nil {
		return 0
	}
	return iterate.end(tag, msg, isDebug)
}

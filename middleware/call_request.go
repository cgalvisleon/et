package middleware

import (
	"time"

	"github.com/cgalvisleon/et/cache"
	"github.com/cgalvisleon/et/config"
	"github.com/cgalvisleon/et/strs"
	"github.com/cgalvisleon/et/timezone"
)

type Request struct {
	Tag     string
	Day     int
	Hour    int
	Minute  int
	Seccond int
	Limit   int
}

/**
* callRequests, create new Request
* @param tag string
* @return Request
**/
func callRequests(tag string) Request {
	now := timezone.NowTime().Unix()
	return Request{
		Tag:     tag,
		Day:     cache.More(strs.Format(`%s-%d`, tag, now/86400), 86400),
		Hour:    cache.More(strs.Format(`%s-%d`, tag, now/3600), 3600),
		Minute:  cache.More(strs.Format(`%s-%d`, tag, now/60), 60),
		Seccond: cache.More(strs.Format(`%s-%d`, tag, now/1), 1),
		Limit:   config.Int("REQUESTS_LIMIT", 400),
	}
}

var items map[string]int64 = make(map[string]int64)

/**
* localRequests, create new Request
* @param tag string
* @return Request
**/
func localRequests(tag string) Request {
	now := timezone.NowTime().Unix()
	return Request{
		Tag:     tag,
		Day:     more(strs.Format(`%s-%d`, tag, now/86400), 86400),
		Hour:    more(strs.Format(`%s-%d`, tag, now/3600), 3600),
		Minute:  more(strs.Format(`%s-%d`, tag, now/60), 60),
		Seccond: more(strs.Format(`%s-%d`, tag, now/1), 1),
		Limit:   config.Int("REQUESTS_LIMIT", 400),
	}
}

/**
* more, create new Request
* @param tag string
* @param expiration time.Duration
* @return int
**/
func more(tag string, expiration time.Duration) int {
	value, ok := items[tag]
	if ok {
		value++
	} else {
		value = 1
	}

	items[tag] = value

	clean := func() {
		delete(items, tag)
	}

	duration := expiration * time.Second
	if duration != 0 {
		go time.AfterFunc(duration, clean)
	}

	return 0
}

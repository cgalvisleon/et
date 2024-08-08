package middleware

import (
	"time"

	"github.com/cgalvisleon/et/cache"
	"github.com/cgalvisleon/et/envar"
	"github.com/cgalvisleon/et/strs"
)

type Request struct {
	Tag     string
	Day     int64
	Hour    int64
	Minute  int64
	Seccond int64
	Limit   int64
}

func callRequests(tag string) Request {
	return Request{
		Tag:     tag,
		Day:     cache.More(strs.Format(`%s-%d`, tag, time.Now().Unix()/86400), 86400),
		Hour:    cache.More(strs.Format(`%s-%d`, tag, time.Now().Unix()/3600), 3600),
		Minute:  cache.More(strs.Format(`%s-%d`, tag, time.Now().Unix()/60), 60),
		Seccond: cache.More(strs.Format(`%s-%d`, tag, time.Now().Unix()/1), 1),
		Limit:   envar.GetInt64(400, "REQUESTS_LIMIT"),
	}
}

var items map[string]int64 = make(map[string]int64)

func localRequests(tag string) Request {
	return Request{
		Tag:     tag,
		Day:     more(strs.Format(`%s-%d`, tag, time.Now().Unix()/86400), 86400),
		Hour:    more(strs.Format(`%s-%d`, tag, time.Now().Unix()/3600), 3600),
		Minute:  more(strs.Format(`%s-%d`, tag, time.Now().Unix()/60), 60),
		Seccond: more(strs.Format(`%s-%d`, tag, time.Now().Unix()/1), 1),
		Limit:   envar.GetInt64(400, "REQUESTS_LIMIT"),
	}
}

func more(tag string, expiration time.Duration) int64 {
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

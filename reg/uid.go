package reg

import (
	"fmt"
	"strconv"

	"github.com/cgalvisleon/et/cache"
	"github.com/cgalvisleon/et/timezone"
	"github.com/google/uuid"
)

/**
* GenUId
* @param tag string
* @return string
**/
func GenUId(tag string) string {
	stamp := timezone.NowTime().UnixMicro()
	key := fmt.Sprintf(`%s:%d`, tag, stamp)

	n := -1
	ns, err := cache.Get(key, "-1")
	if err != nil {
		return fmt.Sprintf(`%s:%s`, tag, uuid.NewString())
	}

	if ns != "-1" {
		n, err = strconv.Atoi(ns)
		if err != nil {
			return fmt.Sprintf(`%s:%s`, tag, uuid.NewString())
		}
	}

	n++
	cache.SetDuration(key, n, 1)
	result := fmt.Sprintf(`%v%03v`, stamp, n)
	val, err := strconv.ParseInt(result, 10, 64)
	if err != nil {
		return fmt.Sprintf(`%s:%s`, tag, uuid.NewString())
	}

	return fmt.Sprintf(`%s:%x`, tag, val)
}

/**
* GetUId
* @param tag string, id string
* @return string
**/
func GetUId(tag, id string) string {
	if !map[string]bool{"": true, "*": true, "new": true}[id] {
		return id
	}

	return GenUId(tag)
}

/**
* GenUIndex
* @return int64
**/
func GenUIndex() int64 {
	stamp := timezone.NowTime().UnixMicro()
	key := fmt.Sprintf(`%s:%d`, "index", stamp)

	n := -1
	ns, err := cache.Get(key, "-1")
	if err != nil {
		return timezone.NowTime().UnixNano()
	}

	if ns != "-1" {
		n, err = strconv.Atoi(ns)
		if err != nil {
			return timezone.NowTime().UnixNano()
		}
	}

	n++
	cache.SetDuration(key, n, 1)
	result := fmt.Sprintf(`%v%03v`, stamp, n)
	val, err := strconv.ParseInt(result, 10, 64)
	if err != nil {
		return timezone.NowTime().UnixNano()
	}

	return val

}

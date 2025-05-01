package cache

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/cgalvisleon/et/strs"
	"github.com/cgalvisleon/et/timezone"
	"github.com/google/uuid"
)

/**
* GenIndex
* @return int64
**/
func GenIndex() int64 {
	stamp := timezone.NowTime().UnixMicro()
	key := fmt.Sprintf(`%s:%d`, "index", stamp)

	n := -1
	ns, err := Get(key, "-1")
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
	SetDuration(key, n, 1)
	result := strs.Format(`%v%03v`, stamp, n)
	val, err := strconv.ParseInt(result, 10, 64)
	if err != nil {
		return timezone.NowTime().UnixNano()
	}

	return val

}

/**
* GenRecordId
* @param tag string
* @return string
**/
func GenRecordId(tag string) string {
	stamp := timezone.NowTime().UnixMicro()
	key := fmt.Sprintf(`%s:%d`, tag, stamp)

	n := -1
	ns, err := Get(key, "-1")
	if err != nil {
		return strs.Format(`%s:%s`, tag, uuid.NewString())
	}

	if ns != "-1" {
		n, err = strconv.Atoi(ns)
		if err != nil {
			return strs.Format(`%s:%s`, tag, uuid.NewString())
		}
	}

	n++
	SetDuration(key, n, 1)
	result := strs.Format(`%v%03v`, stamp, n)
	val, err := strconv.ParseInt(result, 10, 64)
	if err != nil {
		return strs.Format(`%s:%s`, tag, uuid.NewString())
	}

	return strs.Format(`%s:%x`, tag, val)
}

/**
* GetRecordId
* @param tag string, id string
* @return string
**/
func GetRecordId(tag, id string) string {
	if !map[string]bool{"": true, "*": true, "new": true}[id] {
		split := strings.Split(id, ":")
		if len(split) == 1 {
			return strs.Format(`%s:%s`, tag, id)
		} else if len(split) > 1 && split[0] != tag {
			return strs.Format(`%s:%s`, tag, strings.Join(split[1:], ":"))
		}

		return id
	}

	return GenRecordId(tag)
}

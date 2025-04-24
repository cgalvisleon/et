package cache

import (
	"strconv"
	"strings"

	"github.com/cgalvisleon/et/strs"
	"github.com/cgalvisleon/et/timezone"
	"github.com/google/uuid"
)

/**
* RecordId
* @param tag string
* @return string
**/
func RecordId(tag string) string {
	result := strs.Format(`%s:%d`, tag, timezone.NowTime().UnixMicro())

	n := -1
	ns, err := Get(result, "-1")
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
	SetDuration(result, n, 1)
	return strs.Format(`%s%03v`, result, n)
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

	return RecordId(tag)
}

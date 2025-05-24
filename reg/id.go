package reg

import (
	"strings"

	"github.com/cgalvisleon/et/strs"
	"github.com/cgalvisleon/et/timezone"
	"github.com/google/uuid"
)

/**
* GenKey
* @params args ...interface{}
* @return string
**/
func GenKey(args ...interface{}) string {
	var keys []string
	for _, arg := range args {
		keys = append(keys, strs.Format(`%v`, arg))
	}

	return strings.Join(keys, ":")
}

/**
* Id
* @params tag string
* @return string
**/
func GenId(tag string) string {
	return strs.Format(`%s:%s`, tag, uuid.NewString())
}

/**
* GetId
* @params tag, id string
* @return string
**/
func GetId(tag, id string) string {
	if !map[string]bool{"": true, "*": true, "new": true}[id] {
		return id
	}

	return GenId(tag)
}

/**
* GenIndex
* @return int64
**/
func GenIndex() int64 {
	return timezone.NowTime().UnixNano()
}

package reg

import "github.com/cgalvisleon/et/cache"

/**
* Id
* @params tag string
* @return string
**/
func GenId(tag string) string {
	return cache.GenRecordId(tag)
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

	return cache.GetRecordId(tag, id)
}

/**
* GenIndex
* @return int64
**/
func GenIndex() int64 {
	return cache.GenIndex()
}

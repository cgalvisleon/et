package slices

import "github.com/cgalvisleon/et/et"

func SliceFindIndex(item et.Json, list []et.Json, key string) int {
	result := -1
	for i, element := range list {
		if item[key] == element[key] {
			return i
		}
	}

	return result
}

func NotInSlice(la, lb []et.Json, key string) []string {
	var result []string = []string{}
	for _, item := range la {
		idx := SliceFindIndex(item, lb, key)
		if idx == -1 {
			result = append(result, item.Key(key))
		}
	}

	return result
}

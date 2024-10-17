package et

// SliceFindIndex find the index of an item in a slice
func SliceFindIndex(item Json, list []Json, key string) int {
	result := -1
	for i, element := range list {
		if item[key] == element[key] {
			return i
		}
	}

	return result
}

// InSlice return the items that are in the slice
func NotInSlice(la, lb []Json, key string) []string {
	var result []string = []string{}
	for _, item := range la {
		idx := SliceFindIndex(item, lb, key)
		if idx == -1 {
			result = append(result, item.Key(key))
		}
	}

	return result
}

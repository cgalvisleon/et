package utility

type Num float64

func DivNum(a, b float64) float64 {
	if b == 0 {
		return 0
	}

	return a / b
}

func DivInt(a, b int64) int64 {
	if b == 0 {
		return 0
	}

	return a / b
}

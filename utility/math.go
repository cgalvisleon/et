package utility

type Num float64

/**
* DivNum return the division of two numbers
* @param a float64, b float64
* @return float64
**/
func DivNum(a, b float64) float64 {
	if b == 0 {
		return 0
	}

	return a / b
}

/**
* DivInt return the division of two numbers
* @param a int64, b int64
* @return int64
**/
func DivInt(a, b int64) int64 {
	if b == 0 {
		return 0
	}

	return a / b
}

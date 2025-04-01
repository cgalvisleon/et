package et

import "strconv"

type KeyValue struct {
	Ok    bool   `json:"ok"`
	Value []byte `json:"value"`
	Index int    `json:"index"`
}

/**
* NewKeyValue
* @param value interface{}
* @return *KeyValue
**/
func NewKeyValue(value interface{}) *KeyValue {
	return &KeyValue{
		Ok:    true,
		Value: []byte(value.(string)),
		Index: 0,
	}
}

/**
* Str
* @return string
**/
func (s KeyValue) Str() string {
	return string(s.Value)
}

/**
* Int
* @return int, error
**/
func (s KeyValue) Int() (int, error) {
	str := string(s.Value)
	result, err := strconv.Atoi(str)
	if err != nil {
		return -1, err
	}

	return result, nil
}

/**
* Int64
* @return int64, error
**/
func (s KeyValue) Int64() (int64, error) {
	str := string(s.Value)
	result, err := strconv.ParseInt(str, 10, 64)
	if err != nil {
		return -1, err
	}

	return result, nil
}

/**
* Float64
* @return float64, error
**/
func (s KeyValue) Float64() (float64, error) {
	str := string(s.Value)
	result, err := strconv.ParseFloat(str, 64)
	if err != nil {
		return -1, err
	}

	return result, nil
}

/**
* Bool
* @return bool, error
**/
func (s KeyValue) Bool() (bool, error) {
	str := string(s.Value)
	result, err := strconv.ParseBool(str)
	if err != nil {
		return false, err
	}

	return result, nil
}

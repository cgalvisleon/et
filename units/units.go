package units

import (
	"fmt"
	"strconv"

	"github.com/cgalvisleon/et/et"
)

type TypeUnity int

const (
	UnityUnit TypeUnity = iota
	UnityKilometer
	UnityMeter
	UnityCentimeter
	UnityMillimeter
	UnityMiligram
	UnityGram
	UnityKilogram
	UnityTon
	UnityPound
	UnityOunce
	UnityMilliliter
	UnityCentiliter
	UnityLiter
	UnityCubicMeter
)

var UnityMap = map[string]TypeUnity{
	"und": UnityUnit,
	"km":  UnityKilometer,
	"m":   UnityMeter,
	"cm":  UnityCentimeter,
	"mm":  UnityMillimeter,
	"mg":  UnityMiligram,
	"g":   UnityGram,
	"kg":  UnityKilogram,
	"t":   UnityTon,
	"lb":  UnityPound,
	"oz":  UnityOunce,
	"ml":  UnityMilliliter,
	"cl":  UnityCentiliter,
	"l":   UnityLiter,
	"m3":  UnityCubicMeter,
}

/**
* StrToUnity
* @param s string
* @return TypeUnity
**/
func StrToUnity(s string) TypeUnity {
	if val, ok := UnityMap[s]; ok {
		return val
	}

	return UnityUnit
}

/**
* Str
* @return string
**/
func (s TypeUnity) Str() string {
	switch s {
	case UnityKilometer:
		return "km"
	case UnityMeter:
		return "m"
	case UnityCentimeter:
		return "cm"
	case UnityMillimeter:
		return "mm"
	case UnityMiligram:
		return "mg"
	case UnityGram:
		return "g"
	case UnityKilogram:
		return "kg"
	case UnityTon:
		return "t"
	case UnityPound:
		return "lb"
	case UnityOunce:
		return "oz"
	case UnityMilliliter:
		return "ml"
	case UnityCentiliter:
		return "cl"
	case UnityLiter:
		return "l"
	case UnityCubicMeter:
		return "m3"
	default:
		return "und"
	}
}

type Quantity struct {
	Value float64   `json:"value"`
	Unity TypeUnity `json:"unity"`
}

/**
* NewQuantity
* @param val float64, unit TypeUnity
* @return *Quantity
**/
func NewQuantity(val float64, unit TypeUnity) *Quantity {
	result := &Quantity{
		Value: val,
		Unity: unit,
	}

	return result
}

/**
* Load
* @param val interface{}
**/
func (s *Quantity) Load(val interface{}) error {
	switch v := val.(type) {
	case float64:
		s.Value = v
		s.Unity = UnityUnit
	case int:
		s.Value = float64(v)
		s.Unity = UnityUnit
	case int64:
		s.Value = float64(v)
		s.Unity = UnityUnit
	case float32:
		s.Value = float64(v)
		s.Unity = UnityUnit
	case map[string]interface{}:
		obj := et.Json{
			"value": v["value"],
			"unity": v["unity"],
		}
		value, err := strconv.ParseFloat(obj.Str("value"), 64)
		if err != nil {
			return err
		}
		s.Value = value
		s.Unity = StrToUnity(obj.Str("unity"))
	case et.Json:
		value, err := strconv.ParseFloat(v.Str("value"), 64)
		if err != nil {
			return err
		}
		s.Value = value
		s.Unity = StrToUnity(v.Str("unity"))
	case string:
		value, err := strconv.ParseFloat(v, 64)
		if err != nil {
			return err
		}
		s.Value = value
		s.Unity = UnityUnit
	}

	return nil
}

func (s *Quantity) To(un TypeUnity) error {
	multiply := func(v float64) error {
		s.Value = s.Value * v
		s.Unity = un
		return nil
	}

	if s.Unity == UnityKilometer {
		switch un {
		case UnityMeter:
			return multiply(1000)
		case UnityCentimeter:
			return multiply(100000)
		case UnityMillimeter:
			return multiply(1000000)
		}
	}

	if s.Unity == UnityMeter {
		switch un {
		case UnityKilometer:
			return multiply(0.001)
		case UnityCentimeter:
			return multiply(100)
		case UnityMillimeter:
			return multiply(1000)
		}
	}

	if s.Unity == UnityCentimeter {
		switch un {
		case UnityKilometer:
			return multiply(0.00001)
		case UnityMeter:
			return multiply(0.01)
		case UnityMillimeter:
			return multiply(10)
		}
	}

	if s.Unity == UnityMillimeter {
		switch un {
		case UnityKilometer:
			return multiply(0.000001)
		case UnityMeter:
			return multiply(0.001)
		case UnityCentimeter:
			return multiply(0.1)
		}
	}

	if s.Unity == UnityMiligram {
		switch un {
		case UnityGram:
			return multiply(0.001)
		case UnityKilogram:
			return multiply(0.000001)
		case UnityTon:
			return multiply(0.000000001)
		case UnityPound:
			return multiply(0.00000220462)
		case UnityOunce:
			return multiply(0.000035274)
		case UnityMilliliter:
			return multiply(0.001)
		case UnityCentiliter:
			return multiply(0.00001)
		case UnityLiter:
			return multiply(0.000001)
		case UnityCubicMeter:
			return multiply(0.000000001)
		}
	}

	if s.Unity == UnityGram {
		switch un {
		case UnityMiligram:
			return multiply(1000)
		case UnityKilogram:
			return multiply(0.001)
		case UnityTon:
			return multiply(0.000001)
		case UnityPound:
			return multiply(0.00220462)
		case UnityOunce:
			return multiply(0.035274)
		case UnityMilliliter:
			return multiply(1)
		case UnityCentiliter:
			return multiply(0.01)
		case UnityLiter:
			return multiply(0.001)
		case UnityCubicMeter:
			return multiply(0.000001)
		}
	}

	if s.Unity == UnityKilogram {
		switch un {
		case UnityMiligram:
			return multiply(1000000)
		case UnityGram:
			return multiply(1000)
		case UnityTon:
			return multiply(0.001)
		case UnityPound:
			return multiply(2.20462)
		case UnityOunce:
			return multiply(35.274)
		case UnityMilliliter:
			return multiply(1000)
		case UnityCentiliter:
			return multiply(10)
		case UnityLiter:
			return multiply(1)
		case UnityCubicMeter:
			return multiply(0.001)
		}
	}

	if s.Unity == UnityTon {
		switch un {
		case UnityMiligram:
			return multiply(1000000000)
		case UnityGram:
			return multiply(1000000)
		case UnityKilogram:
			return multiply(1000)
		case UnityPound:
			return multiply(2204.62)
		case UnityOunce:
			return multiply(35274)
		case UnityMilliliter:
			return multiply(1000000)
		case UnityCentiliter:
			return multiply(10000)
		case UnityLiter:
			return multiply(1000)
		case UnityCubicMeter:
			return multiply(1)
		}
	}

	if s.Unity == UnityPound {
		switch un {
		case UnityMiligram:
			return multiply(453592)
		case UnityGram:
			return multiply(453.592)
		case UnityKilogram:
			return multiply(0.453592)
		case UnityTon:
			return multiply(0.000453592)
		case UnityOunce:
			return multiply(16)
		case UnityMilliliter:
			return multiply(453.592)
		case UnityCentiliter:
			return multiply(4.53592)
		case UnityLiter:
			return multiply(0.453592)
		case UnityCubicMeter:
			return multiply(0.000453592)
		}
	}

	if s.Unity == UnityOunce {
		switch un {
		case UnityMiligram:
			return multiply(28349.5)
		case UnityGram:
			return multiply(28.3495)
		case UnityKilogram:
			return multiply(0.0283495)
		case UnityTon:
			return multiply(0.0000283495)
		case UnityPound:
			return multiply(0.0625)
		case UnityMilliliter:
			return multiply(28.3495)
		case UnityCentiliter:
			return multiply(0.283495)
		case UnityLiter:
			return multiply(0.0283495)
		case UnityCubicMeter:
			return multiply(0.0000283495)
		}
	}

	if s.Unity == UnityMilliliter {
		switch un {
		case UnityMiligram:
			return multiply(1000)
		case UnityGram:
			return multiply(1)
		case UnityKilogram:
			return multiply(0.001)
		case UnityTon:
			return multiply(0.000001)
		case UnityPound:
			return multiply(0.00220462)
		case UnityOunce:
			return multiply(0.035274)
		case UnityCentiliter:
			return multiply(0.01)
		case UnityLiter:
			return multiply(0.001)
		case UnityCubicMeter:
			return multiply(0.000001)
		}
	}

	if s.Unity == UnityCentiliter {
		switch un {
		case UnityMiligram:
			return multiply(10000)
		case UnityGram:
			return multiply(10)
		case UnityKilogram:
			return multiply(0.01)
		case UnityTon:
			return multiply(0.00001)
		case UnityPound:
			return multiply(0.0220462)
		case UnityOunce:
			return multiply(0.35274)
		case UnityMilliliter:
			return multiply(100)
		case UnityLiter:
			return multiply(0.1)
		case UnityCubicMeter:
			return multiply(0.0001)
		}
	}

	if s.Unity == UnityLiter {
		switch un {
		case UnityMiligram:
			return multiply(1000000)
		case UnityGram:
			return multiply(1000)
		case UnityKilogram:
			return multiply(1)
		case UnityTon:
			return multiply(0.001)
		case UnityPound:
			return multiply(2.20462)
		case UnityOunce:
			return multiply(35.274)
		case UnityMilliliter:
			return multiply(1000)
		case UnityCentiliter:
			return multiply(10)
		case UnityCubicMeter:
			return multiply(0.001)
		}
	}

	if s.Unity == UnityCubicMeter {
		switch un {
		case UnityMiligram:
			return multiply(1000000000)
		case UnityGram:
			return multiply(1000000)
		case UnityKilogram:
			return multiply(1000)
		case UnityTon:
			return multiply(1)
		case UnityPound:
			return multiply(2204.62)
		case UnityOunce:
			return multiply(35274)
		case UnityMilliliter:
			return multiply(1000000)
		case UnityCentiliter:
			return multiply(10000)
		case UnityLiter:
			return multiply(1000)
		}
	}

	return fmt.Errorf(`invalid conversion from %s to %s`, s.Unity.Str(), un.Str())
}

/**
* ToStr
* @return string
**/
func (s *Quantity) ToStr() string {
	return fmt.Sprintf(`%f %s`, s.Value, s.Unity.Str())
}

/**
* ToJson
* @return et.Json
**/
func (s *Quantity) ToJson() et.Json {
	return et.Json{
		"value": s.Value,
		"unity": s.Unity.Str(),
	}
}

/**
* Load
* @param val any
* @return *Quantity, error
**/
func Load(val any) (*Quantity, error) {
	quantity := &Quantity{}
	err := quantity.Load(val)
	if err != nil {
		return nil, err
	}

	return quantity, nil
}

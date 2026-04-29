package main

import (
	"github.com/cgalvisleon/et/et"
	"github.com/cgalvisleon/et/logs"
)

func main() {
	itemA := []et.Json{
		{"name": "cesar", "age": 30},
		{"name": "maria", "age": 25,
			"citas": []et.Json{
				{"date": "2022-01-01", "time": "10:00:00", "code": "CITA-001"},
				{"date": "2022-01-02", "time": "11:00:00", "code": "CITA-002"},
			}},
		{"name": "juan", "age": 35,
			"citas": []et.Json{
				{"date": "2022-01-03", "time": "12:00:00", "code": "CITA-003"},
			}},
	}

	itemB := []et.Json{
		{"name": "pedro", "age": 30, "identity": et.Json{"type": "CC", "number": "123458"}},
		{"name": "ana", "age": 2, "identity": et.Json{"type": "CC", "number": "123459"}},
		{"name": "juan", "age": 35, "identity": et.Json{"type": "CC", "number": "123460"}},
	}

	result := et.From(et.From(itemA, "A").
		Where(et.NotNull("citas")).
		Order("name", true).
		Join(itemB, "B", map[string]string{"A.age": "B.age"}).
		All()).
		Where(et.Eq("age", 25)).
		All()

	logs.Log("JQL:", et.ToString(result))
}

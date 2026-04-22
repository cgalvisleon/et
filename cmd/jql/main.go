package main

import (
	"github.com/cgalvisleon/et/et"
	"github.com/cgalvisleon/et/jql"
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

	// result := sql.From(sql.From(items).
	// 	Where(sql.NotNull("citas")).
	// 	Order("name", true).
	// 	Join("name", "citas", "age").
	// 	Run(nil)).
	// 	Where(sql.Eq("age", 25)).
	// 	Run(nil)

	result := jql.From(itemA, "a").
		// Where(sql.NotNull("citas")).
		Order("name", true).
		Join(itemB, "b", map[string]string{"a.age": "b.age"}).
		// Select("citas", "age", "name", "identity->type:tipo").
		Select("a.name", "a.age", "b.identity->number:number").
		Run(nil)

	// result := sql.JoinToArray(sql.JoinToKeyValue([]et.Json{
	// 	{"name": "cesasr"},
	// }, "age", 30), []et.Json{
	// 	{"date": "2022-01-01", "time": "10:00:00", "code": "CITA-001"},
	// 	{"date": "2022-01-02", "time": "11:00:00", "code": "CITA-002"},
	// })

	logs.Log("JQL:", et.ToString(result))
}

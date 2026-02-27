package main

import (
	"github.com/cgalvisleon/et/et"
	"github.com/cgalvisleon/et/logs"
	"github.com/cgalvisleon/et/sql"
)

func main() {
	items := []et.Json{
		{"name": "cesar", "age": 30, "identity": et.Json{"type": "CC", "number": "123456"}},
		{"name": "maria", "age": 25, "citas": []et.Json{
			{"date": "2022-01-01", "time": "10:00:00", "code": "CITA-001"},
			{"date": "2022-01-02", "time": "11:00:00", "code": "CITA-002"},
		}},
		{"name": "juan", "age": 35, "citas": []et.Json{
			{"date": "2022-01-03", "time": "12:00:00", "code": "CITA-003"},
		}},
	}

	result := sql.From(sql.From(items).
		Where(sql.NotNull("citas")).
		Order("name", true).
		Join("name", "citas", "age").
		Run(nil)).
		Where(sql.Eq("age", 25)).
		Run(nil)

	// result := sql.From(items).
	// 	Where(sql.NotNull("citas")).
	// 	Order("name", true).
	// 	Join("citas", "age", "name").
	// 	// Select("name", "age", "identity->type", "identity->number").
	// 	Run(nil)

	// result := sql.JoinToArray(sql.JoinToKeyValue([]et.Json{
	// 	{"name": "cesasr"},
	// }, "age", 30), []et.Json{
	// 	{"date": "2022-01-01", "time": "10:00:00", "code": "CITA-001"},
	// 	{"date": "2022-01-02", "time": "11:00:00", "code": "CITA-002"},
	// })

	logs.Log("JQL:", et.ToString(result))
}

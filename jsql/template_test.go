package jsql

import (
	"testing"
	"time"

	"github.com/cgalvisleon/et/et"
)

/**
* TestTemplateRender: a template puts each field of the row in its place, with its formats (also chained), a path into
* an object or the first item of a list, and "" for a missing value
**/
func TestTemplateRender(t *testing.T) {
	date := time.Date(2026, 11, 3, 7, 25, 0, 0, time.UTC)
	item := et.Json{
		"code":   "lum-001",
		"name":   "Luminaria LED",
		"power":  1234.5,
		"price":  1500000,
		"date":   date,
		"text":   "2026-11-03T07:25:00Z",
		"clase":  et.Json{"name": "Luminaria"},
		"roles":  []et.Json{{"name": "admin"}, {"name": "user"}},
		"empty":  nil,
		"amount": "-98765.432",
	}
	for _, tc := range []struct {
		template string
		want     string
	}{
		{`Código: {{code}}, {{name}}; Clase de equipo: {{clase.name}}`, "Código: lum-001, Luminaria LED; Clase de equipo: Luminaria"},
		{`{{code|upper}} {{name|lower}}`, "LUM-001 luminaria led"},
		{`{{power|number:1}} W`, "1.234,5 W"},
		{`{{power|number}}`, "1.235"},
		{`{{price|currency}}`, "$ 1.500.000"},
		{`{{amount|number:2}}`, "-98.765,43"},
		{`{{date|date}} / {{date|datetime}} / {{text|date:"02/01/2006 15:04"}}`, "2026-11-03 / 2026-11-03 07:25 / 03/11/2026 07:25"},
		{`{{empty|default:"Sin clase"}}|{{missing}}|`, "Sin clase||"},
		{`{{missing|date|default:"—"}}`, "—"},
		{`{{roles.name}}`, "admin"},
		{`{{ code }} sin campos`, "lum-001 sin campos"},
		{`{{name|default:"a|b"}}`, "Luminaria LED"},
	} {
		parts, err := parseTemplate(tc.template)
		if err != nil {
			t.Fatalf("%s: %v", tc.template, err)
		}
		if got := renderTemplate(parts, item); got != tc.want {
			t.Errorf("%s: %q, want %q", tc.template, got, tc.want)
		}
	}
}

/**
* TestTemplateParse: an empty field is not valid; a quoted argument may hold | and :
**/
func TestTemplateParse(t *testing.T) {
	if _, err := parseTemplate("{{ }}"); err == nil {
		t.Error("empty field accepted")
	}
	parts, err := parseTemplate(`{{a|default:"x|y:z"|upper}}`)
	if err != nil {
		t.Fatal(err)
	}
	formats := parts[0].formats
	if len(formats) != 2 || formats[0].arg != "x|y:z" || formats[1].name != "upper" {
		t.Errorf("formats = %+v", formats)
	}
}

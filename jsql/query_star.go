package jsql

import (
	"fmt"
	"slices"
	"strings"
)

// STAR: en selects, todas las columnas de los from; se le suman otros campos (rollups, atributos, detalles, maestros…)
const STAR = "*"

/**
* HasStar: Si selects trae "*"
* @return bool
**/
func (s *Query) HasStar() bool {
	return slices.Contains(s.Selects, STAR)
}

/**
* hiddenIn: Si una columna de un from está oculta en la consulta: por su nombre o con el alias del from (A.password)
* @param from *From, column string
* @return bool
**/
func (s *Query) hiddenIn(from *From, column string) bool {
	return slices.Contains(s.Hiddens, column) || slices.Contains(s.Hiddens, fmt.Sprintf("%s.%s", from.As, column))
}

/**
* StarFields: Los campos que trae "*": las columnas COLUMN de cada from (alias.nombre con varios from), sin las ocultas
* de la consulta (por nombre o con alias) ni las del modelo, ni el campo fuente
* @return []string
**/
func (s *Query) StarFields() []string {
	result := []string{}
	for _, from := range s.Froms {
		model := from.Model
		if model == nil {
			continue
		}
		for _, col := range model.Columns {
			if col.TypeColumn != COLUMN || col.Name == model.SourceField {
				continue
			}
			if slices.Contains(model.Hiddens, col.Name) || s.hiddenIn(from, col.Name) {
				continue
			}
			name := col.Name
			if len(s.Froms) > 1 {
				name = fmt.Sprintf("%s.%s", from.As, col.Name)
			}
			result = append(result, name)
		}
	}

	return result
}

/**
* SourceHiddens: Las claves que se quitan del campo fuente de un from: las ocultas de la consulta (sin el alias de ese
* from, A.password → password) y las del modelo
* @param from *From
* @return []string
**/
func (s *Query) SourceHiddens(from *From) []string {
	result := []string{}
	prefix := from.As + "."
	for _, hidden := range s.Hiddens {
		result = append(result, strings.TrimPrefix(hidden, prefix))
	}
	if from.Model != nil {
		result = append(result, from.Model.Hiddens...)
	}

	return result
}

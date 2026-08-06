package csv

import (
	"encoding/csv"
	"fmt"
	"io"
	"net/http"
	"os"
	"sort"

	"github.com/cgalvisleon/et/et"
)

/**
* Column: Defines how a Json key is rendered as a CSV column.
**/
type Column struct {
	Key   string
	Title string
}

/**
* Csv: Holds the data, column definitions and delimiter used to render a CSV document.
**/
type Csv struct {
	Columns []Column
	Rows    []et.Json
	Comma   rune
}

/**
* NewCsv: Builds a Csv document from a list of Json rows and the column definitions.
* If columns is empty, the columns are derived from the union of keys present in data, sorted alphabetically,
* using the key as the title. A Column with an empty Title falls back to its Key.
* @param data []et.Json, columns []Column
* @return *Csv
**/
func NewCsv(data []et.Json, columns ...Column) *Csv {
	cols := columns
	if len(cols) == 0 {
		keys := []string{}
		seen := map[string]bool{}
		for _, row := range data {
			for key := range row {
				if !seen[key] {
					seen[key] = true
					keys = append(keys, key)
				}
			}
		}
		sort.Strings(keys)

		cols = make([]Column, 0, len(keys))
		for _, key := range keys {
			cols = append(cols, Column{Key: key, Title: key})
		}
	}

	for i, col := range cols {
		if col.Title == "" {
			cols[i].Title = col.Key
		}
	}

	return &Csv{
		Columns: cols,
		Rows:    data,
		Comma:   ',',
	}
}

/**
* build: Renders the header and data rows as a matrix of strings.
* @return [][]string
**/
func (s *Csv) build() [][]string {
	result := make([][]string, 0, len(s.Rows)+1)

	header := make([]string, len(s.Columns))
	for i, col := range s.Columns {
		header[i] = col.Title
	}
	result = append(result, header)

	for _, row := range s.Rows {
		record := make([]string, len(s.Columns))
		for i, col := range s.Columns {
			value, ok := row[col.Key]
			if !ok || value == nil {
				record[i] = ""
				continue
			}
			record[i] = fmt.Sprintf("%v", value)
		}
		result = append(result, record)
	}

	return result
}

/**
* writer: Creates a csv.Writer configured with this document's delimiter, writes all rows and flushes it.
* @param w io.Writer
* @return error
**/
func (s *Csv) writer(w io.Writer) error {
	writer := csv.NewWriter(w)
	if s.Comma != 0 {
		writer.Comma = s.Comma
	}

	if err := writer.WriteAll(s.build()); err != nil {
		return err
	}

	writer.Flush()
	return writer.Error()
}

/**
* ToFile: Exports the document to a csv file at the given path.
* @param path string
* @return error
**/
func (s *Csv) ToFile(path string) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()

	return s.writer(f)
}

/**
* ToWriter: Writes the document as csv data into the given writer.
* @param w io.Writer
* @return error
**/
func (s *Csv) ToWriter(w io.Writer) error {
	return s.writer(w)
}

/**
* ToHttp: Writes the document as a downloadable csv attachment to the http response.
* @param w http.ResponseWriter, filename string
* @return error
**/
func (s *Csv) ToHttp(w http.ResponseWriter, filename string) error {
	w.Header().Set("Content-Type", "text/csv")
	w.Header().Set("Content-Disposition", "attachment; filename=\""+filename+"\"")

	return s.writer(w)
}

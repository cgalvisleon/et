package csv

import (
	"bytes"
	"encoding/csv"
	"fmt"
	"io"
	"mime/multipart"
	"os"

	"github.com/cgalvisleon/et/et"
	"github.com/cgalvisleon/et/msg"
)

/**
* CsvReader: Wraps parsed CSV rows for reading.
**/
type CsvReader struct {
	rows [][]string
}

/**
* ReadCsv: Parses CSV data from raw bytes for reading. If comma is zero, the default comma (',') is used.
* @param data []byte, comma rune
* @return *CsvReader, error
**/
func ReadCsv(data []byte, comma rune) (*CsvReader, error) {
	r := csv.NewReader(bytes.NewReader(data))
	r.FieldsPerRecord = -1
	if comma != 0 {
		r.Comma = comma
	}

	rows, err := r.ReadAll()
	if err != nil {
		return nil, err
	}

	return &CsvReader{rows: rows}, nil
}

/**
* ReadCsvMultipart: Parses CSV data from a multipart file for reading. If comma is zero, the default comma (',') is used.
* @param fileData multipart.File, comma rune
* @return *CsvReader, error
**/
func ReadCsvMultipart(fileData multipart.File, comma rune) (*CsvReader, error) {
	byteArray, err := io.ReadAll(fileData)
	if err != nil {
		return nil, err
	}

	return ReadCsv(byteArray, comma)
}

/**
* ReadCsvFile: Parses CSV data from a file path for reading. If comma is zero, the default comma (',') is used.
* @param path string, comma rune
* @return *CsvReader, error
**/
func ReadCsvFile(path string, comma rune) (*CsvReader, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	return ReadCsv(data, comma)
}

/**
* GetRows: Returns the parsed CSV rows as a list of strings.
* @return [][]string, error
**/
func (s *CsvReader) GetRows() ([][]string, error) {
	if len(s.rows) == 0 {
		return nil, fmt.Errorf(msg.MSG_SHEET_NOT_DATA, "csv")
	}

	return s.rows, nil
}

/**
* GetData: Returns the CSV rows as a list of Json objects, using the first row as headers. If columns is empty, all columns are returned.
* @param columns []string
* @return []et.Json, error
**/
func (s *CsvReader) GetData(columns []string) ([]et.Json, error) {
	rows, err := s.GetRows()
	if err != nil {
		return nil, err
	}

	result := []et.Json{}
	if len(rows) == 0 {
		return result, nil
	}

	headers := rows[0]
	selected := columns
	if len(selected) == 0 {
		selected = headers
	}

	for _, row := range rows[1:] {
		item := et.Json{}
		for _, col := range selected {
			idx := IndexOf(headers, col)
			if idx == -1 || idx >= len(row) {
				continue
			}
			item[col] = row[idx]
		}
		result = append(result, item)
	}

	return result, nil
}

/**
* IndexOf: Returns the index of value inside list, or -1 if not found.
* @param list []string, value string
* @return int
**/
func IndexOf(list []string, value string) int {
	for i, v := range list {
		if v == value {
			return i
		}
	}
	return -1
}

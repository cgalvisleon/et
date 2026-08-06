package xls

import (
	"bytes"
	"fmt"
	"io"
	"mime/multipart"

	"github.com/cgalvisleon/et/et"
	"github.com/cgalvisleon/et/msg"
	"github.com/xuri/excelize/v2"
)

/**
* XlsReader: Wraps an opened Excel workbook for reading sheet data.
**/
type XlsReader struct {
	file *excelize.File
}

/**
* ReadXls: Opens an Excel workbook from raw bytes for reading.
* @param data []byte
* @return *XlsReader, error
**/
func ReadXls(data []byte) (*XlsReader, error) {
	f, err := excelize.OpenReader(bytes.NewReader(data))
	if err != nil {
		return nil, err
	}

	return &XlsReader{file: f}, nil
}

/**
* ReadXlsMultipart: Opens an Excel workbook from a multipart file for reading.
* @param fileData multipart.File
* @return *XlsReader, error
**/
func ReadXlsMultipart(fileData multipart.File) (*XlsReader, error) {
	byteArray, err := io.ReadAll(fileData)
	if err != nil {
		return nil, err
	}

	return ReadXls(byteArray)
}

/**
* ReadXlsFile: Opens an Excel workbook from a file for reading.
* @param path string
* @return *XlsReader, error
**/
func ReadXlsFile(path string) (*XlsReader, error) {
	f, err := excelize.OpenFile(path)
	if err != nil {
		return nil, err
	}

	return &XlsReader{file: f}, nil
}

/**
* Close: Releases the resources held by the opened workbook.
* @return error
**/
func (s *XlsReader) Close() error {
	return s.file.Close()
}

/**
* GetRows: Returns the rows of a sheet as a list of strings.
* @param nameSheet string
* @return [][]string, error
**/
func (s *XlsReader) GetRows(nameSheet string) ([][]string, error) {
	result, err := s.file.GetRows(nameSheet)
	if err != nil {
		return nil, err
	}

	if len(result) == 0 {
		return nil, fmt.Errorf(msg.MSG_SHEET_NOT_DATA, nameSheet)
	}
	return result, nil
}

/**
* GetSheet: Returns the rows of a sheet as a list of Json objects. If columns is empty, all columns of the sheet are returned.
* @param nameSheet string, columns []string
* @return []et.Json, error
**/
func (s *XlsReader) GetSheet(nameSheet string, columns []string) ([]et.Json, error) {
	rows, err := s.GetRows(nameSheet)
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
* indexOf: Returns the index of value inside list, or -1 if not found.
* @param list []string
* @param value string
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

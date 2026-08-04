package xls

import (
	"io"
	"net/http"
	"sort"

	"github.com/cgalvisleon/et/et"
	"github.com/xuri/excelize/v2"
)

/**
* Column: Defines how a Json key is rendered as a worksheet column.
**/
type Column struct {
	Key   string
	Title string
	Width float64
	Align string
}

/**
* Sheet: Holds the data and column definitions for a single Excel worksheet.
**/
type Sheet struct {
	Name    string
	Columns []Column
	Rows    []et.Json
}

/**
* Xls: Aggregates the sheets that make up an Excel workbook.
**/
type Xls struct {
	Sheets []*Sheet
}

/**
* NewXls: Builds an Xls workbook with an initial sheet from a list of Json rows, a sheet name and the column definitions.
* @param data []et.Json, nameSheet string, columns []Column
* @return *Xls
**/
func NewXls(data []et.Json, nameSheet string, columns ...Column) *Xls {
	result := &Xls{
		Sheets: []*Sheet{},
	}
	result.Add(data, nameSheet, columns)

	return result
}

/**
* Add: Appends a new sheet to the workbook from a list of Json rows, a sheet name and the column definitions.
* If columns is empty, the columns are derived from the union of keys present in data, sorted alphabetically,
* using the key as the title and the default width. A Column with an empty Title falls back to its Key.
* @param data []et.Json, nameSheet string, columns ...Column
* @return *Xls
**/
func (s *Xls) Add(data []et.Json, nameSheet string, columns []Column) *Xls {
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

	s.Sheets = append(s.Sheets, &Sheet{
		Name:    nameSheet,
		Columns: cols,
		Rows:    data,
	})

	return s
}

/**
* build: Creates the excelize.File populated with all the workbook's sheets.
* @return *excelize.File, error
**/
func (s *Xls) build() (*excelize.File, error) {
	f := excelize.NewFile()

	defaultSheet := f.GetSheetName(0)
	usedDefault := false

	for _, sheet := range s.Sheets {
		sheetName := sheet.Name
		if sheetName == "" {
			sheetName = "Sheet1"
		}

		if sheetName == defaultSheet && !usedDefault {
			usedDefault = true
		} else if _, err := f.NewSheet(sheetName); err != nil {
			return nil, err
		}

		for col, column := range sheet.Columns {
			colName, err := excelize.ColumnNumberToName(col + 1)
			if err != nil {
				return nil, err
			}

			if err := f.SetCellValue(sheetName, colName+"1", column.Title); err != nil {
				return nil, err
			}

			if column.Width > 0 {
				if err := f.SetColWidth(sheetName, colName, colName, column.Width); err != nil {
					return nil, err
				}
			}

			if column.Align != "" {
				styleId, err := f.NewStyle(&excelize.Style{
					Alignment: &excelize.Alignment{Horizontal: column.Align},
				})
				if err != nil {
					return nil, err
				}

				endCell, err := excelize.CoordinatesToCellName(col+1, len(sheet.Rows)+1)
				if err != nil {
					return nil, err
				}

				if err := f.SetCellStyle(sheetName, colName+"1", endCell, styleId); err != nil {
					return nil, err
				}
			}
		}

		for rowIdx, row := range sheet.Rows {
			for col, column := range sheet.Columns {
				cell, err := excelize.CoordinatesToCellName(col+1, rowIdx+2)
				if err != nil {
					return nil, err
				}
				if err := f.SetCellValue(sheetName, cell, row[column.Key]); err != nil {
					return nil, err
				}
			}
		}
	}

	if !usedDefault && len(s.Sheets) > 0 {
		if err := f.DeleteSheet(defaultSheet); err != nil {
			return nil, err
		}
	}

	f.SetActiveSheet(0)

	return f, nil
}

/**
* ToFile: Exports the workbook to an xlsx file at the given path.
* @param path string
* @return error
**/
func (s *Xls) ToFile(path string) error {
	f, err := s.build()
	if err != nil {
		return err
	}
	defer f.Close()

	return f.SaveAs(path)
}

/**
* ToWriter: Writes the workbook as xlsx data into the given writer.
* @param w io.Writer
* @return error
**/
func (s *Xls) ToWriter(w io.Writer) error {
	f, err := s.build()
	if err != nil {
		return err
	}
	defer f.Close()

	return f.Write(w)
}

/**
* ToHttp: Writes the workbook as a downloadable xlsx attachment to the http response.
* @param w http.ResponseWriter, filename string
* @return error
**/
func (s *Xls) ToHttp(w http.ResponseWriter, filename string) error {
	f, err := s.build()
	if err != nil {
		return err
	}
	defer f.Close()

	w.Header().Set("Content-Type", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet")
	w.Header().Set("Content-Disposition", "attachment; filename=\""+filename+"\"")

	return f.Write(w)
}

package config

import (
	"fmt"

	"github.com/xuri/excelize/v2"
)

func (d FileCheckTableDefinition) read(
	file *excelize.File,
) ([]FileCheckRule, error) {
	rows, err := file.GetRows(d.Sheet)
	if err != nil {
		return nil, fmt.Errorf(
			"read sheet %q: %w",
			d.Sheet,
			err,
		)
	}

	if len(rows) == 0 {
		return nil, fmt.Errorf(
			"sheet %q is empty",
			d.Sheet,
		)
	}

	headers := indexHeaders(rows[0])

	var errs ValidationErrors

	newFileCol, err := requireColumn(headers, d.NewFileCol)
	if err != nil {
		errs = append(
			errs,
			fmt.Errorf("sheet %q: %w", d.Sheet, err),
		)
	}

	oldFileCol, err := requireColumn(headers, d.OldFileCol)
	if err != nil {
		errs = append(
			errs,
			fmt.Errorf("sheet %q: %w", d.Sheet, err),
		)
	}

	if len(errs) > 0 {
		return nil, errs
	}

	rules := make([]FileCheckRule, 0, len(rows)-1)

	for index, row := range rows[1:] {
		excelRow := index + 2

		newFile := getCell(row, newFileCol)
		oldFile := getCell(row, oldFileCol)

		if newFile == "" && oldFile == "" {
			continue
		}

		rowIsInvalid := false

		if newFile == "" {
			errs = append(
				errs,
				fmt.Errorf(
					"sheet %q, row %d: column %q is required",
					d.Sheet,
					excelRow,
					d.NewFileCol,
				),
			)

			rowIsInvalid = true
		}

		if oldFile == "" {
			errs = append(
				errs,
				fmt.Errorf(
					"sheet %q, row %d: column %q is required",
					d.Sheet,
					excelRow,
					d.OldFileCol,
				),
			)

			rowIsInvalid = true
		}

		if rowIsInvalid {
			continue
		}

		rules = append(rules, FileCheckRule{
			ExcelRow: excelRow,
			NewFile: newFile,
			OldFile: oldFile,
		})
	}

	if len(errs) > 0 {
		return nil, errs
	}

	return rules, nil
}

package config

import (
	"fmt"

	"github.com/xuri/excelize/v2"
)

func (d FileTransferTableDefinition) read(
	file *excelize.File,
) ([]FileTransferMap, error) {
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

	srcCol, err := requireColumn(headers, d.SrcCol)
	if err != nil {
		errs = append(
			errs,
			fmt.Errorf("sheet %q: %w", d.Sheet, err),
		)
	}

	dstCol, err := requireColumn(headers, d.DstCol)
	if err != nil {
		errs = append(
			errs,
			fmt.Errorf("sheet %q: %w", d.Sheet, err),
		)
	}

	if len(errs) > 0 {
		return nil, errs
	}

	mappings := make([]FileTransferMap, 0, len(rows)-1)

	for index, row := range rows[1:] {
		excelRow := index + 2

		src := getCell(row, srcCol)
		dst := getCell(row, dstCol)

		if src == "" && dst == "" {
			continue
		}

		rowIsInvalid := false

		if src == "" {
			errs = append(
				errs,
				fmt.Errorf(
					"sheet %q, row %d: column %q is required",
					d.Sheet,
					excelRow,
					d.SrcCol,
				),
			)

			rowIsInvalid = true
		}

		if dst == "" {
			errs = append(
				errs,
				fmt.Errorf(
					"sheet %q, row %d: column %q is required",
					d.Sheet,
					excelRow,
					d.DstCol,
				),
			)

			rowIsInvalid = true
		}

		if rowIsInvalid {
			continue
		}

		mappings = append(mappings, FileTransferMap{
			ExcelRow: excelRow,
			Src:      src,
			Dest:     dst,
		})
	}

	if len(errs) > 0 {
		return nil, errs
	}

	return mappings, nil
}

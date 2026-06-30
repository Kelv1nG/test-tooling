package sheetsearch

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/xuri/excelize/v2"
)

type ExactTextMatch struct {
	Sheet string
	Cell  string
	Value string
}

func ResolveSheetName(
	workbook *excelize.File,
	selector string,
) (string, error) {
	if workbook == nil {
		return "", fmt.Errorf("workbook is nil")
	}

	trimmed := strings.TrimSpace(selector)
	if trimmed == "" {
		return "", fmt.Errorf("sheet is required")
	}

	if sheetIndex, err := strconv.Atoi(trimmed); err == nil {
		sheets := workbook.GetSheetList()
		if sheetIndex < 1 || sheetIndex > len(sheets) {
			return "", fmt.Errorf(
				"sheet index %d is out of range; workbook has %d sheet(s)",
				sheetIndex,
				len(sheets),
			)
		}

		return sheets[sheetIndex-1], nil
	}

	if _, err := workbook.GetRows(trimmed); err != nil {
		return "", fmt.Errorf("read sheet %q: %w", trimmed, err)
	}

	return trimmed, nil
}

func FindExactText(
	workbook *excelize.File,
	sheetSelector string,
	expected string,
) (ExactTextMatch, bool, error) {
	if workbook == nil {
		return ExactTextMatch{}, false, fmt.Errorf("workbook is nil")
	}

	if expected == "" {
		return ExactTextMatch{}, false, fmt.Errorf("expected text is required")
	}

	sheetName, err := ResolveSheetName(workbook, sheetSelector)
	if err != nil {
		return ExactTextMatch{}, false, err
	}

	rows, err := workbook.GetRows(sheetName)
	if err != nil {
		return ExactTextMatch{}, false, fmt.Errorf("read sheet %q: %w", sheetName, err)
	}

	for rowIndex, row := range rows {
		for columnIndex, value := range row {
			if value != expected {
				continue
			}

			cell, err := excelize.CoordinatesToCellName(columnIndex+1, rowIndex+1)
			if err != nil {
				return ExactTextMatch{}, false, err
			}

			return ExactTextMatch{
				Sheet: sheetName,
				Cell:  cell,
				Value: value,
			}, true, nil
		}
	}

	return ExactTextMatch{}, false, nil
}

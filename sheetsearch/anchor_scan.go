package sheetsearch

import (
	"errors"
	"fmt"

	"github.com/xuri/excelize/v2"
)

var (
	ErrAnchorNotFound  = errors.New("anchor not found")
	ErrMultipleAnchors = errors.New("multiple exact anchor matches found")
)

const lastNonEmptyBeforeBlankSelector = "last_non_empty_before_blank"

type AnchorScanOptions struct {
	Sheet     string
	Anchor    string
	Direction string
	Select    string
}

type AnchorScanMatch struct {
	Sheet      string
	AnchorCell string
	Cell       string
	Value      string
}

func FindAnchorScanValue(
	workbook *excelize.File,
	options AnchorScanOptions,
) (AnchorScanMatch, bool, error) {
	if workbook == nil {
		return AnchorScanMatch{}, false, fmt.Errorf("workbook is nil")
	}
	if options.Anchor == "" {
		return AnchorScanMatch{}, false, fmt.Errorf("anchor is required")
	}
	if !validScanDirection(options.Direction) {
		return AnchorScanMatch{}, false, fmt.Errorf("direction must be one of up, down, left, right")
	}
	if options.Select != lastNonEmptyBeforeBlankSelector {
		return AnchorScanMatch{}, false, fmt.Errorf("scan result selector must be %q", lastNonEmptyBeforeBlankSelector)
	}

	sheetName, err := ResolveSheetName(workbook, options.Sheet)
	if err != nil {
		return AnchorScanMatch{}, false, err
	}

	rows, err := workbook.GetRows(sheetName)
	if err != nil {
		return AnchorScanMatch{}, false, fmt.Errorf("read sheet %q: %w", sheetName, err)
	}

	anchor, err := findUniqueAnchor(rows, sheetName, options.Anchor)
	if err != nil {
		return AnchorScanMatch{}, false, err
	}

	selected, found := scanLastNonEmptyBeforeBlank(rows, anchor, options.Direction)
	anchorCell, err := excelize.CoordinatesToCellName(anchor.column, anchor.row)
	if err != nil {
		return AnchorScanMatch{}, false, fmt.Errorf("resolve anchor cell: %w", err)
	}
	if !found {
		return AnchorScanMatch{
			Sheet:      sheetName,
			AnchorCell: anchorCell,
		}, false, nil
	}

	cell, err := excelize.CoordinatesToCellName(selected.column, selected.row)
	if err != nil {
		return AnchorScanMatch{}, false, fmt.Errorf("resolve selected cell: %w", err)
	}

	return AnchorScanMatch{
		Sheet:      sheetName,
		AnchorCell: anchorCell,
		Cell:       cell,
		Value:      selected.value,
	}, true, nil
}

type scanCell struct {
	row    int
	column int
	value  string
}

func findUniqueAnchor(
	rows [][]string,
	sheetName string,
	anchor string,
) (scanCell, error) {
	matches := make([]scanCell, 0, 1)

	for rowIndex, row := range rows {
		for columnIndex, value := range row {
			if value != anchor {
				continue
			}

			matches = append(matches, scanCell{
				row:    rowIndex + 1,
				column: columnIndex + 1,
				value:  value,
			})
		}
	}

	switch len(matches) {
	case 0:
		return scanCell{}, fmt.Errorf("%w: sheet=%q anchor=%q", ErrAnchorNotFound, sheetName, anchor)
	case 1:
		return matches[0], nil
	default:
		return scanCell{}, fmt.Errorf("%w: sheet=%q anchor=%q matches=%d", ErrMultipleAnchors, sheetName, anchor, len(matches))
	}
}

func scanLastNonEmptyBeforeBlank(
	rows [][]string,
	anchor scanCell,
	direction string,
) (scanCell, bool) {
	current := anchor
	var selected scanCell
	found := false

	for {
		next, ok := moveScanCell(current, direction)
		if !ok {
			break
		}

		value := scanCellValue(rows, next.row, next.column)
		if value == "" {
			break
		}

		next.value = value
		selected = next
		found = true
		current = next
	}

	return selected, found
}

func moveScanCell(
	cell scanCell,
	direction string,
) (scanCell, bool) {
	switch direction {
	case "up":
		cell.row--
	case "down":
		cell.row++
	case "left":
		cell.column--
	case "right":
		cell.column++
	default:
		return scanCell{}, false
	}

	if cell.row < 1 || cell.column < 1 {
		return scanCell{}, false
	}

	return cell, true
}

func scanCellValue(
	rows [][]string,
	row int,
	column int,
) string {
	if row < 1 || column < 1 {
		return ""
	}
	if row > len(rows) {
		return ""
	}
	if column > len(rows[row-1]) {
		return ""
	}

	return rows[row-1][column-1]
}

func validScanDirection(direction string) bool {
	switch direction {
	case "up", "down", "left", "right":
		return true
	default:
		return false
	}
}

package headersearch

import (
	"fmt"

	"github.com/xuri/excelize/v2"
)

type mergeIndex struct {
	cells     map[cellKey]CellPosition
	maxRow    int
	maxColumn int
}

type cellKey struct {
	row    int
	column int
}

func newMergeIndex(
	workbook *excelize.File,
	sheet string,
) (mergeIndex, error) {
	mergeCells, err := workbook.GetMergeCells(sheet)
	if err != nil {
		return mergeIndex{}, fmt.Errorf(
			"read merged cells for sheet %q: %w",
			sheet,
			err,
		)
	}

	index := mergeIndex{
		cells: make(map[cellKey]CellPosition),
	}

	for _, merged := range mergeCells {
		startColumn, startRow, err := excelize.CellNameToCoordinates(
			merged.GetStartAxis(),
		)
		if err != nil {
			return mergeIndex{}, fmt.Errorf(
				"resolve merged range start %q: %w",
				merged.GetStartAxis(),
				err,
			)
		}

		endColumn, endRow, err := excelize.CellNameToCoordinates(
			merged.GetEndAxis(),
		)
		if err != nil {
			return mergeIndex{}, fmt.Errorf(
				"resolve merged range end %q: %w",
				merged.GetEndAxis(),
				err,
			)
		}

		start, err := newCellPosition(startRow, startColumn)
		if err != nil {
			return mergeIndex{}, err
		}

		if endRow > index.maxRow {
			index.maxRow = endRow
		}
		if endColumn > index.maxColumn {
			index.maxColumn = endColumn
		}

		for row := startRow; row <= endRow; row++ {
			for column := startColumn; column <= endColumn; column++ {
				index.cells[cellKey{row: row, column: column}] = start
			}
		}
	}

	return index, nil
}

func (m mergeIndex) topLeft(
	row int,
	column int,
) (CellPosition, bool) {
	position, ok := m.cells[cellKey{row: row, column: column}]
	return position, ok
}

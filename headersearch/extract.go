package headersearch

import (
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/xuri/excelize/v2"
)

var (
	ErrAnchorNotFound    = errors.New("anchor not found")
	ErrMultipleAnchors   = errors.New("multiple exact anchor matches found")
	ErrInvalidDirection  = errors.New("invalid parent direction")
	ErrInvalidHeaderSpan = errors.New("invalid header span")
)

func ExtractHeaders(
	workbook *excelize.File,
	options ExtractOptions,
) (HeaderTable, error) {
	sheetName, err := validateExtractOptions(workbook, options)
	if err != nil {
		return HeaderTable{}, err
	}

	ctx, err := newSheetContext(workbook, sheetName)
	if err != nil {
		return HeaderTable{}, err
	}

	anchor, err := findExactAnchor(ctx, options.Anchor)
	if err != nil {
		return HeaderTable{}, err
	}

	leafSpan, err := findLeafSpan(ctx, options, anchor)
	if err != nil {
		return HeaderTable{}, err
	}

	layerCount, err := determineHeaderLayers(ctx, options, leafSpan)
	if err != nil {
		return HeaderTable{}, err
	}

	headers, err := buildHeaderPaths(ctx, options, leafSpan, layerCount)
	if err != nil {
		return HeaderTable{}, err
	}

	if len(headers) == 0 {
		return HeaderTable{}, fmt.Errorf(
			"%w: sheet=%q anchor=%q",
			ErrInvalidHeaderSpan,
			sheetName,
			options.Anchor,
		)
	}

	return HeaderTable{
		Sheet:           sheetName,
		Anchor:          options.Anchor,
		AnchorPosition:  anchor,
		ParentDirection: options.ParentDirection,
		Headers:         headers,
	}, nil
}

func validateExtractOptions(
	workbook *excelize.File,
	options ExtractOptions,
) (string, error) {
	if workbook == nil {
		return "", fmt.Errorf("workbook is nil")
	}

	sheetName, err := resolveSheetName(workbook, options.Sheet)
	if err != nil {
		return "", err
	}

	if options.Anchor == "" {
		return "", fmt.Errorf("anchor is required")
	}

	if !options.ParentDirection.Valid() {
		return "", ErrInvalidDirection
	}

	if options.MaxHeaderDepth < 1 {
		return "", fmt.Errorf("max header depth must be at least 1")
	}

	return sheetName, nil
}

func resolveSheetName(
	workbook *excelize.File,
	selector string,
) (string, error) {
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

type sheetContext struct {
	workbook *excelize.File
	sheet    string
	rows     [][]string
	bounds   sheetBounds
	merges   mergeIndex
}

type sheetBounds struct {
	maxRow    int
	maxColumn int
}

func newSheetContext(
	workbook *excelize.File,
	sheet string,
) (sheetContext, error) {
	rows, err := workbook.GetRows(sheet)
	if err != nil {
		return sheetContext{}, fmt.Errorf("read sheet %q: %w", sheet, err)
	}

	merges, err := newMergeIndex(workbook, sheet)
	if err != nil {
		return sheetContext{}, err
	}

	bounds := sheetBounds{maxRow: len(rows)}
	for _, row := range rows {
		if len(row) > bounds.maxColumn {
			bounds.maxColumn = len(row)
		}
	}

	if merges.maxRow > bounds.maxRow {
		bounds.maxRow = merges.maxRow
	}
	if merges.maxColumn > bounds.maxColumn {
		bounds.maxColumn = merges.maxColumn
	}

	return sheetContext{
		workbook: workbook,
		sheet:    sheet,
		rows:     rows,
		bounds:   bounds,
		merges:   merges,
	}, nil
}

func findExactAnchor(
	ctx sheetContext,
	anchor string,
) (CellPosition, error) {
	matches := make([]CellPosition, 0, 1)

	for rowIndex, row := range ctx.rows {
		for columnIndex, value := range row {
			if value != anchor {
				continue
			}

			position, err := newCellPosition(rowIndex+1, columnIndex+1)
			if err != nil {
				return CellPosition{}, err
			}
			matches = append(matches, position)
		}
	}

	switch len(matches) {
	case 0:
		return CellPosition{}, fmt.Errorf(
			"%w: sheet=%q anchor=%q",
			ErrAnchorNotFound,
			ctx.sheet,
			anchor,
		)
	case 1:
		return matches[0], nil
	default:
		return CellPosition{}, fmt.Errorf(
			"%w: sheet=%q anchor=%q matches=%d",
			ErrMultipleAnchors,
			ctx.sheet,
			anchor,
			len(matches),
		)
	}
}

func findLeafSpan(
	ctx sheetContext,
	options ExtractOptions,
	anchor CellPosition,
) ([]CellPosition, error) {
	negative, positive, err := siblingDirections(options.ParentDirection)
	if err != nil {
		return nil, err
	}

	before := make([]CellPosition, 0)
	for distance := 1; ; distance++ {
		next, ok := move(anchor, negative, distance)
		if !ok || !ctx.inBounds(next) {
			break
		}

		value, err := ctx.resolvedCellValue(next)
		if err != nil {
			return nil, err
		}
		if value == "" {
			break
		}

		before = append(before, next)
	}

	span := make([]CellPosition, 0, len(before)+1)
	for index := len(before) - 1; index >= 0; index-- {
		span = append(span, before[index])
	}
	span = append(span, anchor)

	for distance := 1; ; distance++ {
		next, ok := move(anchor, positive, distance)
		if !ok || !ctx.inBounds(next) {
			break
		}

		value, err := ctx.resolvedCellValue(next)
		if err != nil {
			return nil, err
		}
		if value == "" {
			break
		}

		span = append(span, next)
	}

	if len(span) == 0 {
		return nil, fmt.Errorf(
			"%w: sheet=%q anchor=%q",
			ErrInvalidHeaderSpan,
			ctx.sheet,
			options.Anchor,
		)
	}

	return span, nil
}

func determineHeaderLayers(
	ctx sheetContext,
	options ExtractOptions,
	leafSpan []CellPosition,
) (int, error) {
	layerCount := 1

	for distance := 1; distance < options.MaxHeaderDepth; distance++ {
		allEmpty := true

		for _, leaf := range leafSpan {
			position, ok := move(leaf, options.ParentDirection, distance)
			if !ok || !ctx.inBounds(position) {
				continue
			}

			value, err := ctx.resolvedCellValue(position)
			if err != nil {
				return 0, err
			}

			if value != "" {
				allEmpty = false
				break
			}
		}

		if allEmpty {
			break
		}

		layerCount++
	}

	return layerCount, nil
}

func buildHeaderPaths(
	ctx sheetContext,
	options ExtractOptions,
	leafSpan []CellPosition,
	layerCount int,
) ([]ColumnHeader, error) {
	headers := make([]ColumnHeader, 0, len(leafSpan))

	for _, leaf := range leafSpan {
		path := make([]string, 0, layerCount)

		for distance := layerCount - 1; distance >= 0; distance-- {
			position := leaf
			var ok bool
			if distance > 0 {
				position, ok = move(leaf, options.ParentDirection, distance)
				if !ok || !ctx.inBounds(position) {
					continue
				}
			}

			value, err := ctx.resolvedCellValue(position)
			if err != nil {
				return nil, err
			}

			if value == "" {
				continue
			}

			if len(path) > 0 && path[len(path)-1] == value {
				continue
			}

			path = append(path, value)
		}

		if len(path) == 0 {
			return nil, fmt.Errorf(
				"%w: sheet=%q anchor=%q leaf=%s",
				ErrInvalidHeaderSpan,
				ctx.sheet,
				options.Anchor,
				leaf.Axis,
			)
		}

		headers = append(headers, ColumnHeader{
			LeafPosition: leaf,
			Path:         path,
		})
	}

	return headers, nil
}

func (ctx sheetContext) rawCellValue(
	row int,
	column int,
) string {
	if row < 1 || column < 1 {
		return ""
	}
	if row > len(ctx.rows) {
		return ""
	}
	if column > len(ctx.rows[row-1]) {
		return ""
	}

	return ctx.rows[row-1][column-1]
}

func (ctx sheetContext) resolvedCellValue(
	position CellPosition,
) (string, error) {
	if start, ok := ctx.merges.topLeft(position.Row, position.Column); ok {
		return ctx.rawCellValue(start.Row, start.Column), nil
	}

	return ctx.rawCellValue(position.Row, position.Column), nil
}

func (ctx sheetContext) inBounds(position CellPosition) bool {
	if position.Row < 1 || position.Column < 1 {
		return false
	}
	if ctx.bounds.maxRow > 0 && position.Row > ctx.bounds.maxRow {
		return false
	}
	if ctx.bounds.maxColumn > 0 && position.Column > ctx.bounds.maxColumn {
		return false
	}

	return true
}

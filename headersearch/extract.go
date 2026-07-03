package headersearch

import (
	"errors"
	"fmt"

	"github.com/xuri/excelize/v2"

	"tooling/sheetsearch"
)

var (
	ErrAnchorNotFound    = errors.New("anchor not found")
	ErrMultipleAnchors   = errors.New("multiple exact anchor matches found")
	ErrInvalidDirection  = errors.New("invalid parent direction")
	ErrInvalidHeaderSpan = errors.New("invalid header span")
)

// ExtractHeaders locates a known leaf header, expands across its contiguous
// header span, then walks parent layers to build the full header paths.
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

	parentLayerOffsets, err := findParentLayerOffsets(ctx, options, leafSpan)
	if err != nil {
		return HeaderTable{}, err
	}

	headers, err := buildHeaderPaths(ctx, options, leafSpan, parentLayerOffsets)
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

	sheetName, err := sheetsearch.ResolveSheetName(workbook, options.Sheet)
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

	// excelize.GetRows only reflects stored row values, so include merged ranges
	// in the bounds or a blank merged cell can look like the edge of the sheet.
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

	// Match exactly and require a single hit so similar labels cannot silently
	// choose the wrong table or header layer.
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

	// The anchor defines the table we care about; blank cells on either side are
	// treated as boundaries so neighboring tables are not pulled into the span.
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

func findParentLayerOffsets(
	ctx sheetContext,
	options ExtractOptions,
	leafSpan []CellPosition,
) ([]int, error) {
	maxParentLayers := options.MaxHeaderDepth

	offsets := make([]int, 0, maxParentLayers)

	// Header depth counts non-empty parent layers. Fully blank spacer layers are
	// skipped so a visual gap between headers and values does not hide headers.
	for distance := 1; len(offsets) < maxParentLayers; distance++ {
		anyInBounds := false
		anyValue := false

		for _, leaf := range leafSpan {
			position, ok := move(leaf, options.ParentDirection, distance)
			if !ok || !ctx.inBounds(position) {
				continue
			}
			anyInBounds = true

			value, err := ctx.resolvedCellValue(position)
			if err != nil {
				return nil, err
			}

			if value != "" {
				anyValue = true
			}
		}

		if !anyInBounds {
			break
		}

		if anyValue {
			offsets = append(offsets, distance)
		}
	}

	return offsets, nil
}

func buildHeaderPaths(
	ctx sheetContext,
	options ExtractOptions,
	leafSpan []CellPosition,
	parentLayerOffsets []int,
) ([]ColumnHeader, error) {
	headers := make([]ColumnHeader, 0, len(leafSpan))

	for _, leaf := range leafSpan {
		path := make([]string, 0, len(parentLayerOffsets)+1)

		// Offsets are discovered nearest-first, but paths should read from the
		// outermost parent down to the leaf header.
		for index := len(parentLayerOffsets) - 1; index >= 0; index-- {
			position, ok := move(leaf, options.ParentDirection, parentLayerOffsets[index])
			if !ok || !ctx.inBounds(position) {
				continue
			}

			value, err := ctx.resolvedCellValue(position)
			if err != nil {
				return nil, err
			}

			if value == "" {
				continue
			}

			// Merged cells and repeated visual labels can surface the same text in
			// adjacent layers; keep the semantic path compact.
			if len(path) > 0 && path[len(path)-1] == value {
				continue
			}

			path = append(path, value)
		}

		// Some sheets use the anchor row or column for values beneath the actual
		// headers, so callers can opt out of appending that layer.
		if !options.IgnoreAnchorLayer {
			value, err := ctx.resolvedCellValue(leaf)
			if err != nil {
				return nil, err
			}
			if value != "" && (len(path) == 0 || path[len(path)-1] != value) {
				path = append(path, value)
			}
		}

		if len(path) == 0 {
			if options.IgnoreAnchorLayer {
				continue
			}
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
	// Any cell inside a merged range should resolve to the visible value stored
	// in the range's top-left cell.
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

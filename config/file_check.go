package config

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/xuri/excelize/v2"

	"tooling/headersearch"
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

	headerSheetCol, anchorCol, parentDirectionCol, maxHeaderDepthCol, requireOrderCol, columnErrs := d.headerCheckColumns(headers)
	errs = append(errs, columnErrs...)
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

		rule := FileCheckRule{
			ExcelRow: excelRow,
			NewFile:  newFile,
			OldFile:  oldFile,
		}

		if d.hasHeaderCheckColumns() {
			headerCheck, err := parseHeaderCheckConfig(
				d,
				excelRow,
				getCell(row, headerSheetCol),
				getCell(row, anchorCol),
				getCell(row, parentDirectionCol),
				getCell(row, maxHeaderDepthCol),
				getCell(row, requireOrderCol),
			)
			if err != nil {
				errs = append(errs, err)
				continue
			}

			rule.HeaderCheck = headerCheck
		}

		rules = append(rules, rule)
	}

	if len(errs) > 0 {
		return nil, errs
	}

	return rules, nil
}

func (d FileCheckTableDefinition) hasAnyHeaderCheckColumns() bool {
	return strings.TrimSpace(d.HeaderSheetCol) != "" ||
		strings.TrimSpace(d.AnchorCol) != "" ||
		strings.TrimSpace(d.ParentDirectionCol) != "" ||
		strings.TrimSpace(d.MaxHeaderDepthCol) != "" ||
		strings.TrimSpace(d.RequireOrderCol) != ""
}

func (d FileCheckTableDefinition) hasHeaderCheckColumns() bool {
	return strings.TrimSpace(d.HeaderSheetCol) != "" &&
		strings.TrimSpace(d.AnchorCol) != "" &&
		strings.TrimSpace(d.ParentDirectionCol) != "" &&
		strings.TrimSpace(d.MaxHeaderDepthCol) != "" &&
		strings.TrimSpace(d.RequireOrderCol) != ""
}

func (d FileCheckTableDefinition) headerCheckColumns(
	headers map[string]int,
) (
	headerSheetCol int,
	anchorCol int,
	parentDirectionCol int,
	maxHeaderDepthCol int,
	requireOrderCol int,
	errs []error,
) {
	if !d.hasHeaderCheckColumns() {
		return 0, 0, 0, 0, 0, nil
	}

	var err error

	headerSheetCol, err = requireColumn(headers, d.HeaderSheetCol)
	if err != nil {
		errs = append(errs, fmt.Errorf("sheet %q: %w", d.Sheet, err))
	}

	anchorCol, err = requireColumn(headers, d.AnchorCol)
	if err != nil {
		errs = append(errs, fmt.Errorf("sheet %q: %w", d.Sheet, err))
	}

	parentDirectionCol, err = requireColumn(headers, d.ParentDirectionCol)
	if err != nil {
		errs = append(errs, fmt.Errorf("sheet %q: %w", d.Sheet, err))
	}

	maxHeaderDepthCol, err = requireColumn(headers, d.MaxHeaderDepthCol)
	if err != nil {
		errs = append(errs, fmt.Errorf("sheet %q: %w", d.Sheet, err))
	}

	requireOrderCol, err = requireColumn(headers, d.RequireOrderCol)
	if err != nil {
		errs = append(errs, fmt.Errorf("sheet %q: %w", d.Sheet, err))
	}

	return headerSheetCol, anchorCol, parentDirectionCol, maxHeaderDepthCol, requireOrderCol, errs
}

func parseHeaderCheckConfig(
	definition FileCheckTableDefinition,
	excelRow int,
	sheet string,
	anchor string,
	parentDirection string,
	maxHeaderDepth string,
	requireOrder string,
) (HeaderCheckConfig, error) {
	if sheet == "" && anchor == "" && parentDirection == "" && maxHeaderDepth == "" && requireOrder == "" {
		return HeaderCheckConfig{}, nil
	}

	if sheet == "" {
		return HeaderCheckConfig{}, fmt.Errorf(
			"sheet %q, row %d: column %q is required when header verification is configured",
			definition.Sheet,
			excelRow,
			definition.HeaderSheetCol,
		)
	}

	if anchor == "" {
		return HeaderCheckConfig{}, fmt.Errorf(
			"sheet %q, row %d: column %q is required when header verification is configured",
			definition.Sheet,
			excelRow,
			definition.AnchorCol,
		)
	}

	if parentDirection == "" {
		return HeaderCheckConfig{}, fmt.Errorf(
			"sheet %q, row %d: column %q is required when header verification is configured",
			definition.Sheet,
			excelRow,
			definition.ParentDirectionCol,
		)
	}

	if !headersearch.Direction(parentDirection).Valid() {
		return HeaderCheckConfig{}, fmt.Errorf(
			"sheet %q, row %d: column %q must be one of up, down, left, right",
			definition.Sheet,
			excelRow,
			definition.ParentDirectionCol,
		)
	}

	if maxHeaderDepth == "" {
		return HeaderCheckConfig{}, fmt.Errorf(
			"sheet %q, row %d: column %q is required when header verification is configured",
			definition.Sheet,
			excelRow,
			definition.MaxHeaderDepthCol,
		)
	}

	depth, err := strconv.Atoi(maxHeaderDepth)
	if err != nil || depth < 1 {
		return HeaderCheckConfig{}, fmt.Errorf(
			"sheet %q, row %d: column %q must be a whole number greater than 0",
			definition.Sheet,
			excelRow,
			definition.MaxHeaderDepthCol,
		)
	}

	requireOrderValue := false
	if requireOrder != "" {
		parsed, err := strconv.ParseBool(requireOrder)
		if err != nil {
			return HeaderCheckConfig{}, fmt.Errorf(
				"sheet %q, row %d: column %q must be true or false",
				definition.Sheet,
				excelRow,
				definition.RequireOrderCol,
			)
		}
		requireOrderValue = parsed
	}

	return HeaderCheckConfig{
		Sheet:           sheet,
		Anchor:          anchor,
		ParentDirection: parentDirection,
		MaxHeaderDepth:  depth,
		RequireOrder:    requireOrderValue,
	}, nil
}

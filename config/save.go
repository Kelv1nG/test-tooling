package config

import (
	"fmt"
	"slices"
	"strings"

	"github.com/xuri/excelize/v2"
)

func (l *Loader) SaveTransferWorkbook(
	path string,
	maps []FileTransferMap,
) error {
	if l == nil {
		return fmt.Errorf("configuration loader is nil")
	}

	file, err := excelize.OpenFile(path)
	if err != nil {
		return fmt.Errorf("open workbook %q: %w", path, err)
	}
	defer file.Close()

	if err := l.definitions.FileTransfer.save(file, maps); err != nil {
		return err
	}

	if err := file.Save(); err != nil {
		return fmt.Errorf("save workbook %q: %w", path, err)
	}

	return nil
}

func (l *Loader) SaveCheckWorkbook(
	path string,
	rules []FileCheckRule,
) error {
	if l == nil {
		return fmt.Errorf("configuration loader is nil")
	}

	file, err := excelize.OpenFile(path)
	if err != nil {
		return fmt.Errorf("open workbook %q: %w", path, err)
	}
	defer file.Close()

	if err := l.definitions.FileCheck.save(file, rules); err != nil {
		return err
	}

	if err := file.Save(); err != nil {
		return fmt.Errorf("save workbook %q: %w", path, err)
	}

	return nil
}

func (d FileTransferTableDefinition) save(
	file *excelize.File,
	maps []FileTransferMap,
) error {
	headers, err := sheetHeaders(file, d.Sheet)
	if err != nil {
		return err
	}

	srcCol, err := requireColumn(headers, d.SrcCol)
	if err != nil {
		return fmt.Errorf("sheet %q: %w", d.Sheet, err)
	}

	dstCol, err := requireColumn(headers, d.DstCol)
	if err != nil {
		return fmt.Errorf("sheet %q: %w", d.Sheet, err)
	}

	existingMaps, err := d.read(file)
	if err != nil {
		return err
	}

	var errs ValidationErrors
	submittedRows := make([]FileTransferMap, 0, len(maps))
	submittedExistingRows := make(map[int]struct{}, len(maps))

	for _, mapping := range maps {
		src := strings.TrimSpace(mapping.Src)
		dst := strings.TrimSpace(mapping.Dest)

		if mapping.ExcelRow < 0 {
			errs = append(errs, fmt.Errorf("sheet %q: invalid excel row %d", d.Sheet, mapping.ExcelRow))
			continue
		}

		if src == "" {
			errs = append(errs, fmt.Errorf("sheet %q, row %d: column %q is required", d.Sheet, mapping.ExcelRow, d.SrcCol))
		}

		if dst == "" {
			errs = append(errs, fmt.Errorf("sheet %q, row %d: column %q is required", d.Sheet, mapping.ExcelRow, d.DstCol))
		}

		if src == "" || dst == "" {
			continue
		}

		submittedRow := FileTransferMap{
			ExcelRow: mapping.ExcelRow,
			Src:      src,
			Dest:     dst,
		}

		submittedRows = append(submittedRows, submittedRow)
		if mapping.ExcelRow >= 2 {
			submittedExistingRows[mapping.ExcelRow] = struct{}{}
		}
	}

	if len(errs) > 0 {
		return errs
	}

	removedRows := make([]int, 0)
	for _, existing := range existingMaps {
		if _, keep := submittedExistingRows[existing.ExcelRow]; keep {
			continue
		}

		removedRows = append(removedRows, existing.ExcelRow)
	}

	slices.Sort(removedRows)
	slices.Reverse(removedRows)

	for _, row := range removedRows {
		if err := file.RemoveRow(d.Sheet, row); err != nil {
			errs = append(errs, fmt.Errorf("sheet %q: remove row %d: %w", d.Sheet, row, err))
		}
	}

	if len(errs) > 0 {
		return errs
	}

	appendStartRow := 2
	if len(existingMaps) > len(removedRows) {
		appendStartRow += len(existingMaps) - len(removedRows)
	}

	newRowOffset := 0
	for _, mapping := range submittedRows {
		targetRow := mapping.ExcelRow
		if mapping.ExcelRow >= 2 {
			targetRow -= removedRowsBefore(removedRows, mapping.ExcelRow)
		} else {
			targetRow = appendStartRow + newRowOffset
			if err := file.InsertRows(d.Sheet, targetRow, 1); err != nil {
				errs = append(errs, fmt.Errorf("sheet %q: insert row %d: %w", d.Sheet, targetRow, err))
				continue
			}
			newRowOffset++
		}

		srcCell, err := excelize.CoordinatesToCellName(srcCol+1, targetRow)
		if err != nil {
			errs = append(errs, fmt.Errorf("sheet %q, row %d: resolve source cell: %w", d.Sheet, targetRow, err))
			continue
		}

		dstCell, err := excelize.CoordinatesToCellName(dstCol+1, targetRow)
		if err != nil {
			errs = append(errs, fmt.Errorf("sheet %q, row %d: resolve destination cell: %w", d.Sheet, targetRow, err))
			continue
		}

		if err := file.SetCellStr(d.Sheet, srcCell, mapping.Src); err != nil {
			errs = append(errs, fmt.Errorf("sheet %q, cell %s: %w", d.Sheet, srcCell, err))
		}

		if err := file.SetCellStr(d.Sheet, dstCell, mapping.Dest); err != nil {
			errs = append(errs, fmt.Errorf("sheet %q, cell %s: %w", d.Sheet, dstCell, err))
		}
	}

	if len(errs) > 0 {
		return errs
	}

	return nil
}

func (d FileCheckTableDefinition) save(
	file *excelize.File,
	rules []FileCheckRule,
) error {
	headers, err := sheetHeaders(file, d.Sheet)
	if err != nil {
		return err
	}

	newFileCol, err := requireColumn(headers, d.NewFileCol)
	if err != nil {
		return fmt.Errorf("sheet %q: %w", d.Sheet, err)
	}

	oldFileCol, err := requireColumn(headers, d.OldFileCol)
	if err != nil {
		return fmt.Errorf("sheet %q: %w", d.Sheet, err)
	}

	var errs ValidationErrors

	for _, rule := range rules {
		newFile := strings.TrimSpace(rule.NewFile)
		oldFile := strings.TrimSpace(rule.OldFile)

		if rule.ExcelRow < 2 {
			errs = append(errs, fmt.Errorf("sheet %q: invalid excel row %d", d.Sheet, rule.ExcelRow))
			continue
		}

		if newFile == "" {
			errs = append(errs, fmt.Errorf("sheet %q, row %d: column %q is required", d.Sheet, rule.ExcelRow, d.NewFileCol))
		}

		if oldFile == "" {
			errs = append(errs, fmt.Errorf("sheet %q, row %d: column %q is required", d.Sheet, rule.ExcelRow, d.OldFileCol))
		}

		if newFile == "" || oldFile == "" {
			continue
		}

		newFileCell, err := excelize.CoordinatesToCellName(newFileCol+1, rule.ExcelRow)
		if err != nil {
			errs = append(errs, fmt.Errorf("sheet %q, row %d: resolve new file cell: %w", d.Sheet, rule.ExcelRow, err))
			continue
		}

		oldFileCell, err := excelize.CoordinatesToCellName(oldFileCol+1, rule.ExcelRow)
		if err != nil {
			errs = append(errs, fmt.Errorf("sheet %q, row %d: resolve old file cell: %w", d.Sheet, rule.ExcelRow, err))
			continue
		}

		if err := file.SetCellStr(d.Sheet, newFileCell, newFile); err != nil {
			errs = append(errs, fmt.Errorf("sheet %q, cell %s: %w", d.Sheet, newFileCell, err))
		}

		if err := file.SetCellStr(d.Sheet, oldFileCell, oldFile); err != nil {
			errs = append(errs, fmt.Errorf("sheet %q, cell %s: %w", d.Sheet, oldFileCell, err))
		}
	}

	if len(errs) > 0 {
		return errs
	}

	return nil
}

func sheetHeaders(
	file *excelize.File,
	sheet string,
) (map[string]int, error) {
	rows, err := file.GetRows(sheet)
	if err != nil {
		return nil, fmt.Errorf("read sheet %q: %w", sheet, err)
	}

	if len(rows) == 0 {
		return nil, fmt.Errorf("sheet %q is empty", sheet)
	}

	return indexHeaders(rows[0]), nil
}

func removedRowsBefore(
	removedRows []int,
	row int,
) int {
	count := 0

	for _, removedRow := range removedRows {
		if removedRow >= row {
			continue
		}

		count++
	}

	return count
}

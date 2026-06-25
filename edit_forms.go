package main

import (
	"fmt"
	"strconv"
	"strings"

	"tooling/config"
)

func parseTransferRowsForm(values map[string][]string) ([]transferRowView, error) {
	excelRows := values["transferExcelRow"]
	srcs := values["transferSrc"]
	dests := values["transferDest"]

	if len(excelRows) == 0 && len(srcs) == 0 && len(dests) == 0 {
		return nil, fmt.Errorf("no transfer rows were submitted")
	}

	if len(excelRows) != len(srcs) || len(srcs) != len(dests) {
		return nil, fmt.Errorf("submitted transfer rows were incomplete")
	}

	rows := make([]transferRowView, 0, len(srcs))
	var errs config.ValidationErrors

	for index := range srcs {
		excelRow, err := strconv.Atoi(strings.TrimSpace(excelRows[index]))
		if err != nil {
			errs = append(errs, fmt.Errorf("transfer row %d is missing its workbook position", index+1))
			excelRow = 0
		}

		src := strings.TrimSpace(srcs[index])
		dest := strings.TrimSpace(dests[index])

		rows = append(rows, transferRowView{
			Index:      index + 1,
			ExcelRow:   excelRow,
			Src:        src,
			SrcExists:  fileExistsOrFalse(src),
			Dest:       dest,
			DestExists: fileExistsOrFalse(dest),
		})

		if src == "" {
			errs = append(errs, fmt.Errorf("transfer row %d requires a source path", index+1))
		}

		if dest == "" {
			errs = append(errs, fmt.Errorf("transfer row %d requires a destination path", index+1))
		}
	}

	if len(errs) > 0 {
		return rows, errs
	}

	return rows, nil
}

func parseCheckRowsForm(values map[string][]string) ([]checkRowView, error) {
	excelRows := values["checkExcelRow"]
	newFiles := values["checkNewFile"]
	oldFiles := values["checkOldFile"]

	if len(excelRows) == 0 && len(newFiles) == 0 && len(oldFiles) == 0 {
		return nil, fmt.Errorf("no check rows were submitted")
	}

	if len(excelRows) != len(newFiles) || len(newFiles) != len(oldFiles) {
		return nil, fmt.Errorf("submitted check rows were incomplete")
	}

	rows := make([]checkRowView, 0, len(newFiles))
	var errs config.ValidationErrors

	for index := range newFiles {
		excelRow, err := strconv.Atoi(strings.TrimSpace(excelRows[index]))
		if err != nil {
			errs = append(errs, fmt.Errorf("check row %d is missing its workbook position", index+1))
			excelRow = 0
		}

		newFile := strings.TrimSpace(newFiles[index])
		oldFile := strings.TrimSpace(oldFiles[index])

		rows = append(rows, checkRowView{
			Index:     index + 1,
			ExcelRow:  excelRow,
			NewFile:   newFile,
			NewExists: fileExistsOrFalse(newFile),
			OldFile:   oldFile,
			OldExists: fileExistsOrFalse(oldFile),
		})

		if newFile == "" {
			errs = append(errs, fmt.Errorf("check row %d requires a new file path", index+1))
		}

		if oldFile == "" {
			errs = append(errs, fmt.Errorf("check row %d requires an old file path", index+1))
		}
	}

	if len(errs) > 0 {
		return rows, errs
	}

	return rows, nil
}

func buildTransferMappingsFromRows(rows []transferRowView) []config.FileTransferMap {
	maps := make([]config.FileTransferMap, 0, len(rows))

	for _, row := range rows {
		maps = append(maps, config.FileTransferMap{
			ExcelRow: row.ExcelRow,
			Src:      row.Src,
			Dest:     row.Dest,
		})
	}

	return maps
}

func buildCheckRulesFromRows(rows []checkRowView) []config.FileCheckRule {
	rules := make([]config.FileCheckRule, 0, len(rows))

	for _, row := range rows {
		rules = append(rules, config.FileCheckRule{
			ExcelRow: row.ExcelRow,
			NewFile:  row.NewFile,
			OldFile:  row.OldFile,
		})
	}

	return rules
}

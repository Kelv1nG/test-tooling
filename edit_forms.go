package main

import (
	"fmt"
	"strconv"
	"strings"

	"tooling/config"
	"tooling/templates"
)

func parseTransferRowsForm(values map[string][]string) ([]templates.TransferRowView, error) {
	excelRows := values["transferExcelRow"]
	srcs := values["transferSrc"]
	dests := values["transferDest"]

	if len(excelRows) == 0 && len(srcs) == 0 && len(dests) == 0 {
		return nil, fmt.Errorf("no transfer rows were submitted")
	}

	if len(excelRows) != len(srcs) || len(srcs) != len(dests) {
		return nil, fmt.Errorf("submitted transfer rows were incomplete")
	}

	rows := make([]templates.TransferRowView, 0, len(srcs))
	var errs config.ValidationErrors

	for index := range srcs {
		excelRowText := strings.TrimSpace(excelRows[index])
		excelRow := 0
		if excelRowText != "" {
			parsedRow, err := strconv.Atoi(excelRowText)
			if err != nil {
				errs = append(errs, fmt.Errorf("transfer row %d has an invalid workbook position", index+1))
			} else {
				excelRow = parsedRow
			}
		}

		src := strings.TrimSpace(srcs[index])
		dest := strings.TrimSpace(dests[index])

		rows = append(rows, templates.TransferRowView{
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

func parseCheckRowsForm(values map[string][]string) ([]templates.CheckRowView, error) {
	excelRows := values["checkExcelRow"]
	newFiles := values["checkNewFile"]
	oldFiles := values["checkOldFile"]

	if len(excelRows) == 0 && len(newFiles) == 0 && len(oldFiles) == 0 {
		return nil, fmt.Errorf("no check rows were submitted")
	}

	if len(excelRows) != len(newFiles) || len(newFiles) != len(oldFiles) {
		return nil, fmt.Errorf("submitted check rows were incomplete")
	}

	rows := make([]templates.CheckRowView, 0, len(newFiles))
	var errs config.ValidationErrors

	for index := range newFiles {
		excelRow, err := strconv.Atoi(strings.TrimSpace(excelRows[index]))
		if err != nil {
			errs = append(errs, fmt.Errorf("check row %d is missing its workbook position", index+1))
			excelRow = 0
		}

		newFile := strings.TrimSpace(newFiles[index])
		oldFile := strings.TrimSpace(oldFiles[index])

		rows = append(rows, templates.CheckRowView{
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

func buildTransferMappingsFromRows(rows []templates.TransferRowView) []config.FileTransferMap {
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

func buildCheckRulesFromRows(rows []templates.CheckRowView) []config.FileCheckRule {
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

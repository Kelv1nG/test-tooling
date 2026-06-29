package main

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"tooling/config"
	"tooling/templates"
)

func parseTransferRowsForm(
	values map[string][]string,
	referenceDate time.Time,
) ([]templates.TransferRowView, error) {
	form := transferRowsForm{
		excelRows: values["transferExcelRow"],
		srcs:      values["transferSrc"],
		dests:     values["transferDest"],
	}

	if form.isEmpty() {
		return nil, fmt.Errorf("no transfer rows were submitted")
	}

	if err := form.validateLengths(); err != nil {
		return nil, fmt.Errorf("submitted transfer rows were incomplete")
	}

	maps := make([]config.FileTransferMap, 0, len(form.srcs))
	var errs config.ValidationErrors

	for index := range form.srcs {
		excelRow, err := parseOptionalExcelRow(
			form.excelRows[index],
			index+1,
			"transfer row",
		)
		if err != nil {
			errs = append(errs, err)
		}

		src := strings.TrimSpace(form.srcs[index])
		dest := strings.TrimSpace(form.dests[index])

		maps = append(maps, config.FileTransferMap{
			ExcelRow: excelRow,
			Src:      src,
			Dest:     dest,
		})

		errs = append(errs, validateTransferPath(src, index+1, "source")...)
		errs = append(errs, validateTransferPath(dest, index+1, "destination")...)
	}

	rows := buildTransferRows(maps, referenceDate)

	if len(errs) > 0 {
		return rows, errs
	}

	return rows, nil
}

func parseCheckRowsForm(values map[string][]string) ([]templates.CheckRowView, error) {
	form := checkRowsForm{
		excelRows: values["checkExcelRow"],
		newFiles:  values["checkNewFile"],
		oldFiles:  values["checkOldFile"],
	}

	if form.isEmpty() {
		return nil, fmt.Errorf("no check rows were submitted")
	}

	if err := form.validateLengths(); err != nil {
		return nil, fmt.Errorf("submitted check rows were incomplete")
	}

	rows := make([]templates.CheckRowView, 0, len(form.newFiles))
	var errs config.ValidationErrors

	for index := range form.newFiles {
		excelRow, err := parseRequiredExcelRow(
			form.excelRows[index],
			index+1,
			"check row",
		)
		if err != nil {
			errs = append(errs, err)
		}

		newFile := strings.TrimSpace(form.newFiles[index])
		oldFile := strings.TrimSpace(form.oldFiles[index])

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

type transferRowsForm struct {
	excelRows []string
	srcs      []string
	dests     []string
}

func (f transferRowsForm) isEmpty() bool {
	return len(f.excelRows) == 0 && len(f.srcs) == 0 && len(f.dests) == 0
}

func (f transferRowsForm) validateLengths() error {
	if len(f.excelRows) != len(f.srcs) || len(f.srcs) != len(f.dests) {
		return fmt.Errorf("mismatched transfer row fields")
	}

	return nil
}

type checkRowsForm struct {
	excelRows []string
	newFiles  []string
	oldFiles  []string
}

func (f checkRowsForm) isEmpty() bool {
	return len(f.excelRows) == 0 && len(f.newFiles) == 0 && len(f.oldFiles) == 0
}

func (f checkRowsForm) validateLengths() error {
	if len(f.excelRows) != len(f.newFiles) || len(f.newFiles) != len(f.oldFiles) {
		return fmt.Errorf("mismatched check row fields")
	}

	return nil
}

func parseOptionalExcelRow(
	value string,
	rowIndex int,
	rowLabel string,
) (int, error) {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return 0, nil
	}

	row, err := strconv.Atoi(trimmed)
	if err != nil {
		return 0, fmt.Errorf("%s %d has an invalid workbook position", rowLabel, rowIndex)
	}

	return row, nil
}

func parseRequiredExcelRow(
	value string,
	rowIndex int,
	rowLabel string,
) (int, error) {
	row, err := strconv.Atoi(strings.TrimSpace(value))
	if err != nil {
		return 0, fmt.Errorf("%s %d is missing its workbook position", rowLabel, rowIndex)
	}

	return row, nil
}

func validateTransferPath(
	value string,
	rowIndex int,
	field string,
) []error {
	if value == "" {
		return []error{fmt.Errorf("transfer row %d requires a %s path", rowIndex, field)}
	}

	if err := config.ValidatePathTemplate(value); err != nil {
		return []error{
			fmt.Errorf(
				"transfer row %d has an invalid %s path template: %v",
				rowIndex,
				field,
				err,
			),
		}
	}

	return nil
}

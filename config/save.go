package config

import (
	"encoding/json"
	"fmt"
	"slices"
	"strconv"
	"strings"

	"github.com/xuri/excelize/v2"

	"tooling/headersearch"
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
	configs []FileCheckConfig,
) error {
	if l == nil {
		return fmt.Errorf("configuration loader is nil")
	}

	file, err := excelize.OpenFile(path)
	if err != nil {
		return fmt.Errorf("open workbook %q: %w", path, err)
	}
	defer file.Close()

	if err := l.definitions.FileCheck.save(file, configs); err != nil {
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
	configs []FileCheckConfig,
) error {
	if err := d.validateConfigs(configs); err != nil {
		return err
	}

	if err := d.writeConfigs(file, configs); err != nil {
		return err
	}

	if err := d.Rules.writeRules(file, configs); err != nil {
		return err
	}

	return nil
}

func (d FileCheckTableDefinition) validateConfigs(
	configs []FileCheckConfig,
) error {
	var errs ValidationErrors
	seenIDs := make(map[string]struct{}, len(configs))

	for index, check := range configs {
		rowNumber := index + 1
		id := strings.TrimSpace(check.ID)
		fileTemplate := strings.TrimSpace(check.File)

		if id == "" {
			errs = append(errs, fmt.Errorf("check config %d requires a check id", rowNumber))
		} else if _, exists := seenIDs[id]; exists {
			errs = append(errs, fmt.Errorf("check config %d has duplicate check id %q", rowNumber, id))
		} else {
			seenIDs[id] = struct{}{}
		}

		if fileTemplate == "" {
			errs = append(errs, fmt.Errorf("check config %d requires a file path", rowNumber))
		}
		if check.requiresCompareOffset() && check.CompareOffsetMonths == 0 {
			errs = append(errs, fmt.Errorf("check config %d requires a non-zero compare offset because it has a header comparison rule", rowNumber))
		}

		seenRuleIDs := make(map[string]struct{}, len(check.Rules))
		for ruleIndex, rule := range check.Rules {
			if err := validateVerificationRule(check.ID, ruleIndex+1, rule); err != nil {
				errs = append(errs, err)
			}
			if rule.ID == "" {
				continue
			}
			if _, exists := seenRuleIDs[rule.ID]; exists {
				errs = append(errs, fmt.Errorf("check config %q has duplicate rule id %q", check.ID, rule.ID))
				continue
			}
			seenRuleIDs[rule.ID] = struct{}{}
		}
	}

	if len(errs) > 0 {
		return errs
	}

	return nil
}

func validateVerificationRule(
	checkID string,
	ruleIndex int,
	rule VerificationRule,
) error {
	var errs ValidationErrors

	if strings.TrimSpace(rule.ID) == "" {
		errs = append(errs, fmt.Errorf("check config %q rule %d requires a rule id", checkID, ruleIndex))
	}
	if !rule.Type.valid() {
		errs = append(errs, fmt.Errorf("check config %q rule %d has invalid rule type %q", checkID, ruleIndex, rule.Type))
	}

	switch rule.Type {
	case VerificationRuleTypeHeaderCompare:
		errs = append(errs, validateHeaderCompareRule(checkID, ruleIndex, rule.HeaderCompare)...)
	case VerificationRuleTypeExactText:
		errs = append(errs, validateExactTextRule(checkID, ruleIndex, rule.ExactText)...)
	}

	if len(errs) > 0 {
		return errs
	}

	return nil
}

func validateHeaderCompareRule(
	checkID string,
	ruleIndex int,
	rule HeaderCheckConfig,
) []error {
	var errs []error

	if strings.TrimSpace(rule.Sheet) == "" {
		errs = append(errs, fmt.Errorf("check config %q rule %d requires a sheet", checkID, ruleIndex))
	}
	if strings.TrimSpace(rule.Anchor) == "" {
		errs = append(errs, fmt.Errorf("check config %q rule %d requires an anchor", checkID, ruleIndex))
	}
	if !headersearch.Direction(rule.ParentDirection).Valid() {
		errs = append(errs, fmt.Errorf("check config %q rule %d direction must be one of up, down, left, right", checkID, ruleIndex))
	}
	if rule.MaxHeaderDepth < 1 {
		errs = append(errs, fmt.Errorf("check config %q rule %d requires max depth greater than 0", checkID, ruleIndex))
	}

	return errs
}

func validateExactTextRule(
	checkID string,
	ruleIndex int,
	rule ExactMatchCheckConfig,
) []error {
	var errs []error

	if strings.TrimSpace(rule.Sheet) == "" {
		errs = append(errs, fmt.Errorf("check config %q rule %d requires a sheet", checkID, ruleIndex))
	}
	if strings.TrimSpace(rule.ExpectedText) == "" {
		errs = append(errs, fmt.Errorf("check config %q rule %d requires expected text", checkID, ruleIndex))
	} else if err := ValidateTemplateText(rule.ExpectedText); err != nil {
		errs = append(errs, fmt.Errorf("check config %q rule %d has invalid expected-text template: %v", checkID, ruleIndex, err))
	}

	return errs
}

func (d FileCheckTableDefinition) writeConfigs(
	file *excelize.File,
	configs []FileCheckConfig,
) error {
	headers, err := sheetHeaders(file, d.Sheet)
	if err != nil {
		return err
	}

	idCol, err := requireColumn(headers, d.IDCol)
	if err != nil {
		return fmt.Errorf("sheet %q: %w", d.Sheet, err)
	}
	fileCol, err := requireColumn(headers, d.FileCol)
	if err != nil {
		return fmt.Errorf("sheet %q: %w", d.Sheet, err)
	}
	compareOffsetMonthsCol, err := requireColumn(headers, d.CompareOffsetMonthsCol)
	if err != nil {
		return fmt.Errorf("sheet %q: %w", d.Sheet, err)
	}

	if err := clearDataRows(file, d.Sheet); err != nil {
		return err
	}

	var errs ValidationErrors
	for index, check := range configs {
		row := index + 2
		if err := setStringCell(file, d.Sheet, idCol, row, check.ID); err != nil {
			errs = append(errs, err)
		}
		if err := setStringCell(file, d.Sheet, fileCol, row, check.File); err != nil {
			errs = append(errs, err)
		}
		if err := setStringCell(file, d.Sheet, compareOffsetMonthsCol, row, strconv.Itoa(check.CompareOffsetMonths)); err != nil {
			errs = append(errs, err)
		}
	}

	if len(errs) > 0 {
		return errs
	}

	return nil
}

func (d FileCheckRulesTableDefinition) writeRules(
	file *excelize.File,
	configs []FileCheckConfig,
) error {
	headers, err := sheetHeaders(file, d.Sheet)
	if err != nil {
		return err
	}

	checkIDCol, err := requireColumn(headers, d.CheckIDCol)
	if err != nil {
		return fmt.Errorf("sheet %q: %w", d.Sheet, err)
	}
	ruleIDCol, err := requireColumn(headers, d.RuleIDCol)
	if err != nil {
		return fmt.Errorf("sheet %q: %w", d.Sheet, err)
	}
	ruleNameCol, err := requireColumn(headers, d.RuleNameCol)
	if err != nil {
		return fmt.Errorf("sheet %q: %w", d.Sheet, err)
	}
	ruleTypeCol, err := requireColumn(headers, d.RuleTypeCol)
	if err != nil {
		return fmt.Errorf("sheet %q: %w", d.Sheet, err)
	}
	enabledCol, err := requireColumn(headers, d.EnabledCol)
	if err != nil {
		return fmt.Errorf("sheet %q: %w", d.Sheet, err)
	}
	configCol, err := requireColumn(headers, d.ConfigCol)
	if err != nil {
		return fmt.Errorf("sheet %q: %w", d.Sheet, err)
	}

	if err := clearDataRows(file, d.Sheet); err != nil {
		return err
	}

	var errs ValidationErrors
	targetRow := 2
	for _, check := range configs {
		for _, rule := range check.Rules {
			ruleConfig, err := marshalRuleConfig(rule)
			if err != nil {
				errs = append(errs, fmt.Errorf("sheet %q, row %d: %w", d.Sheet, targetRow, err))
				targetRow++
				continue
			}

			if err := setStringCell(file, d.Sheet, checkIDCol, targetRow, check.ID); err != nil {
				errs = append(errs, err)
			}
			if err := setStringCell(file, d.Sheet, ruleIDCol, targetRow, rule.ID); err != nil {
				errs = append(errs, err)
			}
			if err := setStringCell(file, d.Sheet, ruleNameCol, targetRow, rule.Name); err != nil {
				errs = append(errs, err)
			}
			if err := setStringCell(file, d.Sheet, ruleTypeCol, targetRow, string(rule.Type)); err != nil {
				errs = append(errs, err)
			}
			if err := setStringCell(file, d.Sheet, enabledCol, targetRow, strconv.FormatBool(rule.Enabled)); err != nil {
				errs = append(errs, err)
			}
			if err := setStringCell(file, d.Sheet, configCol, targetRow, ruleConfig); err != nil {
				errs = append(errs, err)
			}

			targetRow++
		}
	}

	if len(errs) > 0 {
		return errs
	}

	return nil
}

func marshalRuleConfig(rule VerificationRule) (string, error) {
	var payload any
	switch rule.Type {
	case VerificationRuleTypeHeaderCompare:
		payload = rule.HeaderCompare
	case VerificationRuleTypeExactText:
		payload = rule.ExactText
	default:
		return "", fmt.Errorf("unsupported rule type %q", rule.Type)
	}

	data, err := json.Marshal(payload)
	if err != nil {
		return "", fmt.Errorf("marshal rule config: %w", err)
	}

	return string(data), nil
}

func setStringCell(
	file *excelize.File,
	sheet string,
	column int,
	row int,
	value string,
) error {
	cell, err := excelize.CoordinatesToCellName(column+1, row)
	if err != nil {
		return fmt.Errorf("sheet %q, row %d: resolve cell: %w", sheet, row, err)
	}
	if err := file.SetCellStr(sheet, cell, value); err != nil {
		return fmt.Errorf("sheet %q, cell %s: %w", sheet, cell, err)
	}
	return nil
}

func clearDataRows(
	file *excelize.File,
	sheet string,
) error {
	rows, err := file.GetRows(sheet)
	if err != nil {
		return fmt.Errorf("read sheet %q: %w", sheet, err)
	}

	for row := len(rows); row >= 2; row-- {
		if err := file.RemoveRow(sheet, row); err != nil {
			return fmt.Errorf("sheet %q: remove row %d: %w", sheet, row, err)
		}
	}

	return nil
}

func (c HeaderCheckConfig) hasValues() bool {
	return strings.TrimSpace(c.Sheet) != "" ||
		strings.TrimSpace(c.Anchor) != "" ||
		strings.TrimSpace(c.ParentDirection) != "" ||
		c.MaxHeaderDepth > 0 ||
		c.RequireOrder
}

func (c ExactMatchCheckConfig) hasValues() bool {
	return strings.TrimSpace(c.Sheet) != "" ||
		strings.TrimSpace(c.ExpectedText) != ""
}

func (t VerificationRuleType) valid() bool {
	return t == VerificationRuleTypeHeaderCompare || t == VerificationRuleTypeExactText
}

func (r VerificationRule) requiresCompareOffset() bool {
	return r.Enabled && r.Type == VerificationRuleTypeHeaderCompare
}

func (c FileCheckConfig) requiresCompareOffset() bool {
	for _, rule := range c.Rules {
		if rule.requiresCompareOffset() {
			return true
		}
	}

	return false
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

package main

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"tooling/config"
	"tooling/sheetsearch"
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
		return []templates.TransferRowView{}, nil
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

func parseCheckRowsForm(
	values map[string][]string,
	referenceDate time.Time,
) ([]templates.CheckRowView, error) {
	form := checkConfigsForm{
		excelRows:           values["checkExcelRow"],
		ids:                 values["checkID"],
		files:               values["checkFile"],
		compareOffsetMonths: values["checkCompareOffsetMonths"],
		rules: checkRulesForm{
			parentIndexes:    values["ruleParentIndex"],
			excelRows:        values["ruleExcelRow"],
			ids:              values["ruleID"],
			names:            values["ruleName"],
			types:            values["ruleType"],
			enableds:         values["ruleEnabled"],
			sheets:           values["ruleSheet"],
			anchors:          values["ruleAnchor"],
			parentDirections: values["ruleParentDirection"],
			maxHeaderDepths:  values["ruleMaxHeaderDepth"],
			requireOrders:    values["ruleRequireOrder"],
			scanSelects:      values["ruleScanSelect"],
			expectedTexts:    values["ruleExpectedText"],
			compareAs:        values["ruleCompareAs"],
		},
	}

	if form.isEmpty() {
		return []templates.CheckRowView{}, nil
	}

	if err := form.validateLengths(); err != nil {
		return nil, fmt.Errorf("submitted check configs were incomplete")
	}

	rows := make([]templates.CheckRowView, 0, len(form.files))
	rowOffsetsByIndex := make(map[int]int, len(form.files))
	usedCheckIDs := collectUsedCheckIDs(form.ids)
	usedRuleIDsByParentIndex := collectUsedRuleIDsByParentIndex(form.rules)
	var errs config.ValidationErrors

	for index := range form.files {
		excelRow, err := parseRequiredExcelRow(
			form.excelRows[index],
			index+1,
			"check row",
		)
		if err != nil {
			errs = append(errs, err)
		}

		id := strings.TrimSpace(form.ids[index])
		fileTemplate := strings.TrimSpace(form.files[index])
		compareOffsetMonths, err := parseCheckCompareOffsetMonths(form.compareOffsetMonths[index], index+1)
		if err != nil {
			errs = append(errs, err)
		}

		if id == "" {
			id = nextGeneratedCheckID(usedCheckIDs, index+1)
		} else {
			usedCheckIDs[id] = struct{}{}
		}
		if fileTemplate == "" {
			errs = append(errs, fmt.Errorf("check config %d requires a file path", index+1))
		}

		if err := config.ValidatePathTemplate(fileTemplate); err != nil {
			errs = append(errs, fmt.Errorf("check config %d has an invalid file path template: %v", index+1, err))
		}

		row := templates.CheckRowView{
			Index:               index + 1,
			ExcelRow:            excelRow,
			ID:                  id,
			File:                fileTemplate,
			CompareOffsetMonths: compareOffsetMonths,
		}
		rows = append(rows, row)
		rowOffsetsByIndex[row.Index] = len(rows) - 1
	}

	for index := range form.rules.ids {
		parentIndex, err := strconv.Atoi(strings.TrimSpace(form.rules.parentIndexes[index]))
		if err != nil {
			errs = append(errs, fmt.Errorf("verification rule %d has an invalid parent config", index+1))
			continue
		}

		parentOffset, ok := rowOffsetsByIndex[parentIndex]
		if !ok {
			errs = append(errs, fmt.Errorf("verification rule %d references missing check config %d", index+1, parentIndex))
			continue
		}

		parent := &rows[parentOffset]
		rule, ruleErrs := parseCheckRuleForm(form.rules, index, parent.ID)
		errs = append(errs, ruleErrs...)
		if rule.ID == "" {
			usedRuleIDs := usedRuleIDsByParentIndex[parentIndex]
			if usedRuleIDs == nil {
				usedRuleIDs = map[string]struct{}{}
				usedRuleIDsByParentIndex[parentIndex] = usedRuleIDs
			}
			rule.ID = nextGeneratedRuleID(usedRuleIDs, len(parent.Rules)+1)
		}
		parent.Rules = append(parent.Rules, rule)
	}

	for index := range rows {
		if checkRowRequiresCompareOffset(rows[index]) && rows[index].CompareOffsetMonths == 0 {
			errs = append(errs, fmt.Errorf("check config %d requires a non-zero compare offset when a header comparison rule is enabled", rows[index].Index))
		}
		applyCheckPathStatus(&rows[index], referenceDate)
	}

	if len(errs) > 0 {
		return rows, errs
	}

	return rows, nil
}

func filterSingleCheckRowsForm(
	values map[string][]string,
	targetIndex int,
) (map[string][]string, error) {
	if targetIndex < 1 {
		return nil, fmt.Errorf("save target must be a positive check config index")
	}

	targetOffset := targetIndex - 1
	filtered := make(map[string][]string)
	for _, field := range checkConfigFieldNames() {
		source := values[field]
		if targetOffset >= len(source) {
			return nil, fmt.Errorf("save target check config %d was not submitted", targetIndex)
		}
		filtered[field] = []string{source[targetOffset]}
	}

	parentIndexes := values["ruleParentIndex"]
	ruleFields := checkRuleFieldNames()
	for _, field := range ruleFields {
		if len(values[field]) != len(parentIndexes) {
			return nil, fmt.Errorf("submitted check configs were incomplete")
		}
	}

	for ruleIndex, parentIndexValue := range parentIndexes {
		parentIndex, err := strconv.Atoi(strings.TrimSpace(parentIndexValue))
		if err != nil {
			continue
		}
		if parentIndex != targetIndex {
			continue
		}

		for _, field := range ruleFields {
			value := values[field][ruleIndex]
			if field == "ruleParentIndex" {
				value = "1"
			}
			filtered[field] = append(filtered[field], value)
		}
	}

	return filtered, nil
}

func checkConfigFieldNames() []string {
	return []string{
		"checkExcelRow",
		"checkID",
		"checkFile",
		"checkCompareOffsetMonths",
	}
}

func checkRuleFieldNames() []string {
	return []string{
		"ruleParentIndex",
		"ruleExcelRow",
		"ruleID",
		"ruleName",
		"ruleType",
		"ruleEnabled",
		"ruleSheet",
		"ruleAnchor",
		"ruleParentDirection",
		"ruleMaxHeaderDepth",
		"ruleRequireOrder",
		"ruleScanSelect",
		"ruleExpectedText",
		"ruleCompareAs",
	}
}

func collectUsedCheckIDs(ids []string) map[string]struct{} {
	used := make(map[string]struct{}, len(ids))
	for _, id := range ids {
		id = strings.TrimSpace(id)
		if id == "" {
			continue
		}
		used[id] = struct{}{}
	}

	return used
}

func nextGeneratedCheckID(
	used map[string]struct{},
	seed int,
) string {
	if seed < 1 {
		seed = 1
	}

	for counter := seed; ; counter++ {
		id := fmt.Sprintf("CHECK-%03d", counter)
		if _, exists := used[id]; exists {
			continue
		}
		used[id] = struct{}{}
		return id
	}
}

func collectUsedRuleIDsByParentIndex(
	form checkRulesForm,
) map[int]map[string]struct{} {
	usedByParent := make(map[int]map[string]struct{})

	for index, id := range form.ids {
		id = strings.TrimSpace(id)
		if id == "" {
			continue
		}

		parentIndex, err := strconv.Atoi(strings.TrimSpace(form.parentIndexes[index]))
		if err != nil {
			continue
		}

		used := usedByParent[parentIndex]
		if used == nil {
			used = map[string]struct{}{}
			usedByParent[parentIndex] = used
		}
		used[id] = struct{}{}
	}

	return usedByParent
}

func nextGeneratedRuleID(
	used map[string]struct{},
	seed int,
) string {
	if seed < 1 {
		seed = 1
	}

	for counter := seed; ; counter++ {
		id := fmt.Sprintf("R%03d", counter)
		if _, exists := used[id]; exists {
			continue
		}
		used[id] = struct{}{}
		return id
	}
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

type checkConfigsForm struct {
	excelRows           []string
	ids                 []string
	files               []string
	compareOffsetMonths []string
	rules               checkRulesForm
}

func (f checkConfigsForm) isEmpty() bool {
	return len(f.excelRows) == 0 &&
		len(f.ids) == 0 &&
		len(f.files) == 0 &&
		len(f.compareOffsetMonths) == 0
}

func (f checkConfigsForm) validateLengths() error {
	if len(f.excelRows) != len(f.ids) ||
		len(f.ids) != len(f.files) ||
		len(f.files) != len(f.compareOffsetMonths) {
		return fmt.Errorf("mismatched check config fields")
	}

	return f.rules.validateLengths()
}

func parseCheckCompareOffsetMonths(value string, rowIndex int) (int, error) {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return 0, nil
	}

	months, err := strconv.Atoi(trimmed)
	if err != nil {
		return 0, fmt.Errorf("check config %d has an invalid compare offset", rowIndex)
	}

	return months, nil
}

type checkRulesForm struct {
	parentIndexes    []string
	excelRows        []string
	ids              []string
	names            []string
	types            []string
	enableds         []string
	sheets           []string
	anchors          []string
	parentDirections []string
	maxHeaderDepths  []string
	requireOrders    []string
	scanSelects      []string
	expectedTexts    []string
	compareAs        []string
}

func (f checkRulesForm) validateLengths() error {
	length := len(f.ids)
	if len(f.parentIndexes) != length ||
		len(f.excelRows) != length ||
		len(f.names) != length ||
		len(f.types) != length ||
		len(f.enableds) != length ||
		len(f.sheets) != length ||
		len(f.anchors) != length ||
		len(f.parentDirections) != length ||
		len(f.maxHeaderDepths) != length ||
		len(f.requireOrders) != length ||
		len(f.scanSelects) != length ||
		len(f.expectedTexts) != length ||
		len(f.compareAs) != length {
		return fmt.Errorf("mismatched verification rule fields")
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

func parseCheckRuleForm(
	form checkRulesForm,
	index int,
	checkID string,
) (templates.CheckRuleView, []error) {
	excelRow, err := parseOptionalExcelRow(
		form.excelRows[index],
		index+1,
		"verification rule",
	)
	var errs []error
	if err != nil {
		errs = append(errs, err)
	}

	enabled, err := strconv.ParseBool(strings.TrimSpace(form.enableds[index]))
	if err != nil {
		errs = append(errs, fmt.Errorf("verification rule %d has an invalid enabled value", index+1))
		enabled = false
	}

	requireOrder, err := strconv.ParseBool(strings.TrimSpace(form.requireOrders[index]))
	if err != nil {
		errs = append(errs, fmt.Errorf("verification rule %d has an invalid order requirement", index+1))
		requireOrder = false
	}

	rule := templates.CheckRuleView{
		Index:           index + 1,
		ExcelRow:        excelRow,
		ID:              strings.TrimSpace(form.ids[index]),
		CheckID:         checkID,
		Name:            strings.TrimSpace(form.names[index]),
		Type:            strings.TrimSpace(form.types[index]),
		Enabled:         enabled,
		Sheet:           strings.TrimSpace(form.sheets[index]),
		Anchor:          strings.TrimSpace(form.anchors[index]),
		ParentDirection: strings.TrimSpace(form.parentDirections[index]),
		MaxHeaderDepth:  strings.TrimSpace(form.maxHeaderDepths[index]),
		RequireOrder:    requireOrder,
		ScanSelect:      strings.TrimSpace(form.scanSelects[index]),
		ExpectedText:    form.expectedTexts[index],
		CompareAs:       strings.TrimSpace(form.compareAs[index]),
	}

	if rule.Type == "" {
		errs = append(errs, fmt.Errorf("verification rule %d requires a rule type", index+1))
	}

	switch config.VerificationRuleType(rule.Type) {
	case config.VerificationRuleTypeHeaderCompare:
		errs = append(errs, validateHeaderCheckForm(index+1, rule.Sheet, rule.Anchor, rule.ParentDirection, rule.MaxHeaderDepth, rule.RequireOrder)...)
	case config.VerificationRuleTypeExactText:
		errs = append(errs, validateExactMatchForm(index+1, rule.Sheet, rule.ExpectedText)...)
	case config.VerificationRuleTypeAnchorScan:
		errs = append(errs, validateAnchorScanForm(index+1, rule.Sheet, rule.Anchor, rule.ParentDirection, rule.ScanSelect, rule.ExpectedText, rule.CompareAs)...)
	default:
		if rule.Type != "" {
			errs = append(errs, fmt.Errorf("verification rule %d type must be header_compare, exact_text, or anchor_scan_match", index+1))
		}
	}

	return rule, errs
}

func validateHeaderCheckForm(
	rowIndex int,
	sheet string,
	anchor string,
	parentDirection string,
	maxHeaderDepth string,
	requireOrder bool,
) []error {
	if sheet == "" && anchor == "" && parentDirection == "" && maxHeaderDepth == "" && !requireOrder {
		return nil
	}

	var errs []error

	if sheet == "" {
		errs = append(errs, fmt.Errorf("check row %d requires a verification sheet", rowIndex))
	}
	if anchor == "" {
		errs = append(errs, fmt.Errorf("check row %d requires a verification anchor", rowIndex))
	}
	if parentDirection == "" {
		errs = append(errs, fmt.Errorf("check row %d requires a verification direction", rowIndex))
	} else if !sheetsearch.Direction(parentDirection).Valid() {
		errs = append(errs, fmt.Errorf("check row %d has an invalid verification direction", rowIndex))
	}
	if maxHeaderDepth == "" {
		errs = append(errs, fmt.Errorf("check row %d requires a verification max depth", rowIndex))
	} else if depth, err := strconv.Atoi(maxHeaderDepth); err != nil || depth < 1 {
		errs = append(errs, fmt.Errorf("check row %d requires a verification max depth greater than 0", rowIndex))
	}

	return errs
}

func validateExactMatchForm(
	rowIndex int,
	sheet string,
	expectedText string,
) []error {
	if sheet == "" && strings.TrimSpace(expectedText) == "" {
		return nil
	}

	var errs []error

	if sheet == "" {
		errs = append(errs, fmt.Errorf("check row %d requires an exact-match sheet", rowIndex))
	}
	if strings.TrimSpace(expectedText) == "" {
		errs = append(errs, fmt.Errorf("check row %d requires exact-match text", rowIndex))
	} else if err := config.ValidateTemplateText(expectedText); err != nil {
		errs = append(errs, fmt.Errorf("check row %d has an invalid exact-match template: %v", rowIndex, err))
	}

	return errs
}

func validateAnchorScanForm(
	rowIndex int,
	sheet string,
	anchor string,
	direction string,
	scanSelect string,
	expectedText string,
	compareAs string,
) []error {
	if sheet == "" &&
		anchor == "" &&
		direction == "" &&
		scanSelect == "" &&
		strings.TrimSpace(expectedText) == "" &&
		compareAs == "" {
		return nil
	}

	var errs []error

	if sheet == "" {
		errs = append(errs, fmt.Errorf("check row %d requires an anchor-scan sheet", rowIndex))
	}
	if anchor == "" {
		errs = append(errs, fmt.Errorf("check row %d requires an anchor-scan anchor", rowIndex))
	}
	if direction == "" {
		errs = append(errs, fmt.Errorf("check row %d requires an anchor-scan direction", rowIndex))
	} else if !sheetsearch.Direction(direction).Valid() {
		errs = append(errs, fmt.Errorf("check row %d has an invalid anchor-scan direction", rowIndex))
	}
	if scanSelect == "" {
		errs = append(errs, fmt.Errorf("check row %d requires an anchor-scan result selector", rowIndex))
	} else if scanSelect != config.AnchorScanSelectLastNonEmptyBeforeBlank {
		errs = append(errs, fmt.Errorf("check row %d has an invalid anchor-scan result selector", rowIndex))
	}
	if strings.TrimSpace(expectedText) == "" {
		errs = append(errs, fmt.Errorf("check row %d requires anchor-scan expected text", rowIndex))
	} else if err := config.ValidateTemplateText(expectedText); err != nil {
		errs = append(errs, fmt.Errorf("check row %d has an invalid anchor-scan template: %v", rowIndex, err))
	}
	if compareAs == "" {
		errs = append(errs, fmt.Errorf("check row %d requires an anchor-scan comparison mode", rowIndex))
	} else if compareAs != config.AnchorScanCompareExactText && compareAs != config.AnchorScanCompareDate {
		errs = append(errs, fmt.Errorf("check row %d has an invalid anchor-scan comparison mode", rowIndex))
	}

	return errs
}

func checkRowRequiresCompareOffset(row templates.CheckRowView) bool {
	for _, rule := range row.Rules {
		if rule.Enabled && config.VerificationRuleType(rule.Type) == config.VerificationRuleTypeHeaderCompare {
			return true
		}
	}

	return false
}

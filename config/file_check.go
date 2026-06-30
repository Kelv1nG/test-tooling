package config

import (
	"encoding/json"
	"fmt"
	"io"
	"strconv"
	"strings"

	"github.com/xuri/excelize/v2"

	"tooling/headersearch"
)

func (d FileCheckTableDefinition) read(
	file *excelize.File,
) ([]FileCheckConfig, error) {
	rows, err := file.GetRows(d.Sheet)
	if err != nil {
		return nil, fmt.Errorf("read sheet %q: %w", d.Sheet, err)
	}
	if len(rows) == 0 {
		return nil, fmt.Errorf("sheet %q is empty", d.Sheet)
	}

	headers := indexHeaders(rows[0])
	var errs ValidationErrors

	idCol, err := requireColumn(headers, d.IDCol)
	if err != nil {
		errs = append(errs, fmt.Errorf("sheet %q: %w", d.Sheet, err))
	}
	newFileCol, err := requireColumn(headers, d.NewFileCol)
	if err != nil {
		errs = append(errs, fmt.Errorf("sheet %q: %w", d.Sheet, err))
	}
	oldFileCol, err := requireColumn(headers, d.OldFileCol)
	if err != nil {
		errs = append(errs, fmt.Errorf("sheet %q: %w", d.Sheet, err))
	}
	if len(errs) > 0 {
		return nil, errs
	}

	configs := make([]FileCheckConfig, 0, len(rows)-1)
	configIndexByID := make(map[string]int)
	for index, row := range rows[1:] {
		excelRow := index + 2
		id := strings.TrimSpace(getCell(row, idCol))
		newFile := strings.TrimSpace(getCell(row, newFileCol))
		oldFile := strings.TrimSpace(getCell(row, oldFileCol))

		if id == "" && newFile == "" && oldFile == "" {
			continue
		}

		rowIsInvalid := false
		if id == "" {
			errs = append(errs, fmt.Errorf("sheet %q, row %d: column %q is required", d.Sheet, excelRow, d.IDCol))
			rowIsInvalid = true
		}
		if newFile == "" {
			errs = append(errs, fmt.Errorf("sheet %q, row %d: column %q is required", d.Sheet, excelRow, d.NewFileCol))
			rowIsInvalid = true
		}
		if _, exists := configIndexByID[id]; id != "" && exists {
			errs = append(errs, fmt.Errorf("sheet %q, row %d: duplicate check id %q", d.Sheet, excelRow, id))
			rowIsInvalid = true
		}
		if rowIsInvalid {
			continue
		}

		configs = append(configs, FileCheckConfig{
			ExcelRow: excelRow,
			ID:       id,
			NewFile:  newFile,
			OldFile:  oldFile,
		})
		configIndexByID[id] = len(configs) - 1
	}

	rules, err := d.Rules.read(file)
	if err != nil {
		errs = append(errs, err)
	}
	for _, rule := range rules {
		configIndex, ok := configIndexByID[rule.CheckID]
		if !ok {
			errs = append(errs, fmt.Errorf("sheet %q, row %d: check id %q does not exist in %q", d.Rules.Sheet, rule.ExcelRow, rule.CheckID, d.Sheet))
			continue
		}
		configs[configIndex].Rules = append(configs[configIndex].Rules, rule)
	}

	for index := range configs {
		if configs[index].requiresOldFile() && strings.TrimSpace(configs[index].OldFile) == "" {
			errs = append(errs, fmt.Errorf("sheet %q, row %d: column %q is required when a header comparison rule exists", d.Sheet, configs[index].ExcelRow, d.OldFileCol))
		}
	}

	if len(errs) > 0 {
		return nil, errs
	}

	return configs, nil
}

func (d FileCheckRulesTableDefinition) read(
	file *excelize.File,
) ([]VerificationRule, error) {
	rows, err := file.GetRows(d.Sheet)
	if err != nil {
		return nil, fmt.Errorf("read sheet %q: %w", d.Sheet, err)
	}
	if len(rows) == 0 {
		return nil, fmt.Errorf("sheet %q is empty", d.Sheet)
	}

	headers := indexHeaders(rows[0])
	var errs ValidationErrors

	checkIDCol, err := requireColumn(headers, d.CheckIDCol)
	if err != nil {
		errs = append(errs, fmt.Errorf("sheet %q: %w", d.Sheet, err))
	}
	ruleIDCol, err := requireColumn(headers, d.RuleIDCol)
	if err != nil {
		errs = append(errs, fmt.Errorf("sheet %q: %w", d.Sheet, err))
	}
	ruleNameCol, err := requireColumn(headers, d.RuleNameCol)
	if err != nil {
		errs = append(errs, fmt.Errorf("sheet %q: %w", d.Sheet, err))
	}
	ruleTypeCol, err := requireColumn(headers, d.RuleTypeCol)
	if err != nil {
		errs = append(errs, fmt.Errorf("sheet %q: %w", d.Sheet, err))
	}
	enabledCol, err := requireColumn(headers, d.EnabledCol)
	if err != nil {
		errs = append(errs, fmt.Errorf("sheet %q: %w", d.Sheet, err))
	}
	configCol, err := requireColumn(headers, d.ConfigCol)
	if err != nil {
		errs = append(errs, fmt.Errorf("sheet %q: %w", d.Sheet, err))
	}
	if len(errs) > 0 {
		return nil, errs
	}

	rules := make([]VerificationRule, 0, len(rows)-1)
	seenRuleIDs := make(map[string]struct{}, len(rows)-1)
	for index, row := range rows[1:] {
		excelRow := index + 2
		raw := rawVerificationRule{
			checkID:    strings.TrimSpace(getCell(row, checkIDCol)),
			ruleID:     strings.TrimSpace(getCell(row, ruleIDCol)),
			ruleName:   strings.TrimSpace(getCell(row, ruleNameCol)),
			ruleType:   strings.TrimSpace(getCell(row, ruleTypeCol)),
			enabled:    strings.TrimSpace(getCell(row, enabledCol)),
			configJSON: getCell(row, configCol),
		}

		if raw.empty() {
			continue
		}

		rule, err := parseVerificationRule(d, excelRow, raw)
		if err != nil {
			errs = append(errs, err)
			continue
		}

		ruleKey := rule.CheckID + "\x00" + rule.ID
		if _, exists := seenRuleIDs[ruleKey]; exists {
			errs = append(errs, fmt.Errorf("sheet %q, row %d: duplicate rule id %q for check id %q", d.Sheet, excelRow, rule.ID, rule.CheckID))
			continue
		}

		seenRuleIDs[ruleKey] = struct{}{}
		rules = append(rules, rule)
	}

	if len(errs) > 0 {
		return nil, errs
	}

	return rules, nil
}

type rawVerificationRule struct {
	checkID    string
	ruleID     string
	ruleName   string
	ruleType   string
	enabled    string
	configJSON string
}

func (r rawVerificationRule) empty() bool {
	return r.checkID == "" &&
		r.ruleID == "" &&
		r.ruleName == "" &&
		r.ruleType == "" &&
		r.enabled == "" &&
		strings.TrimSpace(r.configJSON) == ""
}

func parseVerificationRule(
	definition FileCheckRulesTableDefinition,
	excelRow int,
	raw rawVerificationRule,
) (VerificationRule, error) {
	var errs ValidationErrors

	if raw.checkID == "" {
		errs = append(errs, fmt.Errorf("sheet %q, row %d: column %q is required", definition.Sheet, excelRow, definition.CheckIDCol))
	}
	if raw.ruleID == "" {
		errs = append(errs, fmt.Errorf("sheet %q, row %d: column %q is required", definition.Sheet, excelRow, definition.RuleIDCol))
	}

	ruleType, err := parseVerificationRuleType(raw.ruleType)
	if err != nil {
		errs = append(errs, fmt.Errorf("sheet %q, row %d: column %q %v", definition.Sheet, excelRow, definition.RuleTypeCol, err))
	}

	enabled := true
	if raw.enabled != "" {
		enabled, err = strconv.ParseBool(raw.enabled)
		if err != nil {
			errs = append(errs, fmt.Errorf("sheet %q, row %d: column %q must be true or false", definition.Sheet, excelRow, definition.EnabledCol))
		}
	}

	rule := VerificationRule{
		ExcelRow: excelRow,
		ID:       raw.ruleID,
		CheckID:  raw.checkID,
		Name:     raw.ruleName,
		Type:     ruleType,
		Enabled:  enabled,
	}
	if rule.Name == "" && rule.ID != "" {
		rule.Name = rule.ID
	}

	switch ruleType {
	case VerificationRuleTypeHeaderCompare:
		headerCheck, headerErrs := parseHeaderCompareRule(definition, excelRow, raw)
		errs = append(errs, headerErrs...)
		rule.HeaderCompare = headerCheck
	case VerificationRuleTypeExactText:
		exactText, exactErrs := parseExactTextRule(definition, excelRow, raw)
		errs = append(errs, exactErrs...)
		rule.ExactText = exactText
	}

	if len(errs) > 0 {
		return rule, errs
	}

	return rule, nil
}

func parseVerificationRuleType(value string) (VerificationRuleType, error) {
	switch VerificationRuleType(strings.TrimSpace(value)) {
	case VerificationRuleTypeHeaderCompare:
		return VerificationRuleTypeHeaderCompare, nil
	case VerificationRuleTypeExactText:
		return VerificationRuleTypeExactText, nil
	default:
		return "", fmt.Errorf("must be one of %s or %s", VerificationRuleTypeHeaderCompare, VerificationRuleTypeExactText)
	}
}

func parseHeaderCompareRule(
	definition FileCheckRulesTableDefinition,
	excelRow int,
	raw rawVerificationRule,
) (HeaderCheckConfig, []error) {
	var errs []error
	var headerCheck HeaderCheckConfig

	errs = append(errs, decodeRuleConfigJSON(definition, excelRow, raw.configJSON, &headerCheck, VerificationRuleTypeHeaderCompare)...)
	if len(errs) > 0 {
		return headerCheck, errs
	}

	headerCheck.Sheet = strings.TrimSpace(headerCheck.Sheet)
	headerCheck.Anchor = strings.TrimSpace(headerCheck.Anchor)
	headerCheck.ParentDirection = strings.TrimSpace(headerCheck.ParentDirection)

	if headerCheck.Sheet == "" {
		errs = append(errs, missingRuleConfigFieldError(definition, excelRow, VerificationRuleTypeHeaderCompare, "sheet"))
	}
	if headerCheck.Anchor == "" {
		errs = append(errs, missingRuleConfigFieldError(definition, excelRow, VerificationRuleTypeHeaderCompare, "anchor"))
	}
	if headerCheck.ParentDirection == "" {
		errs = append(errs, missingRuleConfigFieldError(definition, excelRow, VerificationRuleTypeHeaderCompare, "parent_direction"))
	} else if !headersearch.Direction(headerCheck.ParentDirection).Valid() {
		errs = append(errs, fmt.Errorf("sheet %q, row %d: field %q in column %q must be one of up, down, left, right", definition.Sheet, excelRow, "parent_direction", definition.ConfigCol))
	}
	if headerCheck.MaxHeaderDepth < 1 {
		errs = append(errs, fmt.Errorf("sheet %q, row %d: field %q in column %q must be a whole number greater than 0", definition.Sheet, excelRow, "max_header_depth", definition.ConfigCol))
	}

	return headerCheck, errs
}

func parseExactTextRule(
	definition FileCheckRulesTableDefinition,
	excelRow int,
	raw rawVerificationRule,
) (ExactMatchCheckConfig, []error) {
	var errs []error
	var exactText ExactMatchCheckConfig

	errs = append(errs, decodeRuleConfigJSON(definition, excelRow, raw.configJSON, &exactText, VerificationRuleTypeExactText)...)
	if len(errs) > 0 {
		return exactText, errs
	}

	exactText.Sheet = strings.TrimSpace(exactText.Sheet)

	if exactText.Sheet == "" {
		errs = append(errs, missingRuleConfigFieldError(definition, excelRow, VerificationRuleTypeExactText, "sheet"))
	}
	if strings.TrimSpace(exactText.ExpectedText) == "" {
		errs = append(errs, missingRuleConfigFieldError(definition, excelRow, VerificationRuleTypeExactText, "expected_text"))
	} else if err := ValidateTemplateText(exactText.ExpectedText); err != nil {
		errs = append(errs, fmt.Errorf("sheet %q, row %d: field %q in column %q has an invalid expected-text template: %v", definition.Sheet, excelRow, "expected_text", definition.ConfigCol, err))
	}

	return exactText, errs
}

func decodeRuleConfigJSON(
	definition FileCheckRulesTableDefinition,
	excelRow int,
	rawConfig string,
	target any,
	ruleType VerificationRuleType,
) []error {
	if strings.TrimSpace(rawConfig) == "" {
		return []error{
			fmt.Errorf("sheet %q, row %d: column %q is required for %s rules", definition.Sheet, excelRow, definition.ConfigCol, ruleType),
		}
	}

	decoder := json.NewDecoder(strings.NewReader(rawConfig))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(target); err != nil {
		return []error{
			fmt.Errorf("sheet %q, row %d: column %q contains invalid JSON for %s rules: %v", definition.Sheet, excelRow, definition.ConfigCol, ruleType, err),
		}
	}

	var extra struct{}
	if err := decoder.Decode(&extra); err != io.EOF {
		if err == nil {
			err = fmt.Errorf("contains more than one JSON value")
		}
		return []error{
			fmt.Errorf("sheet %q, row %d: column %q contains invalid JSON for %s rules: %v", definition.Sheet, excelRow, definition.ConfigCol, ruleType, err),
		}
	}

	return nil
}

func missingRuleConfigFieldError(
	definition FileCheckRulesTableDefinition,
	excelRow int,
	ruleType VerificationRuleType,
	field string,
) error {
	return fmt.Errorf("sheet %q, row %d: field %q in column %q is required for %s rules", definition.Sheet, excelRow, field, definition.ConfigCol, ruleType)
}

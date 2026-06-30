package main

import (
	"fmt"
	"strconv"
	"time"

	"tooling/config"
	"tooling/templates"
)

func buildTransferRows(
	mappings []config.FileTransferMap,
	referenceDate time.Time,
) []templates.TransferRowView {
	rows := make([]templates.TransferRowView, 0, len(mappings))

	for index, mapping := range mappings {
		row := templates.TransferRowView{
			Index:    index + 1,
			ExcelRow: mapping.ExcelRow,
			Src:      mapping.Src,
			Dest:     mapping.Dest,
		}

		resolved, err := config.ResolveFileTransferMap(mapping, referenceDate)
		if err != nil {
			row.Status = "Invalid template"
			row.Badge = "rose"
			row.Detail = err.Error()
			rows = append(rows, row)
			continue
		}

		row.SrcExists = fileExistsOrFalse(resolved.Src)
		row.DestExists = fileExistsOrFalse(resolved.Dest)
		rows = append(rows, row)
	}

	return rows
}

func applyTransferResultsToRows(
	rows []templates.TransferRowView,
	results []templates.TransferResultView,
) []templates.TransferRowView {
	if len(rows) == 0 || len(results) == 0 {
		return rows
	}

	resultsByIndex := make(map[int]templates.TransferResultView, len(results))
	for _, result := range results {
		resultsByIndex[result.Index] = result
	}

	for index := range rows {
		result, exists := resultsByIndex[rows[index].Index]
		if !exists {
			continue
		}

		rows[index].Status = result.Status
		rows[index].Badge = result.Badge
		rows[index].Detail = result.Detail
	}

	return rows
}

func buildCheckRows(
	checks []config.FileCheckConfig,
	referenceDate time.Time,
) []templates.CheckRowView {
	rows := make([]templates.CheckRowView, 0, len(checks))

	for index, check := range checks {
		row := templates.CheckRowView{
			Index:               index + 1,
			ExcelRow:            check.ExcelRow,
			ID:                  check.ID,
			File:                check.File,
			CompareOffsetMonths: check.CompareOffsetMonths,
			Rules:               buildCheckRuleRows(check.Rules),
		}

		applyCheckPathStatus(&row, referenceDate)
		rows = append(rows, row)
	}

	return rows
}

func buildCheckRuleRows(
	rules []config.VerificationRule,
) []templates.CheckRuleView {
	rows := make([]templates.CheckRuleView, 0, len(rules))

	for index, rule := range rules {
		row := templates.CheckRuleView{
			Index:    index + 1,
			ExcelRow: rule.ExcelRow,
			ID:       rule.ID,
			CheckID:  rule.CheckID,
			Name:     rule.Name,
			Type:     string(rule.Type),
			Enabled:  rule.Enabled,
		}

		switch rule.Type {
		case config.VerificationRuleTypeHeaderCompare:
			row.Sheet = rule.HeaderCompare.Sheet
			row.Anchor = rule.HeaderCompare.Anchor
			row.ParentDirection = rule.HeaderCompare.ParentDirection
			row.MaxHeaderDepth = formatHeaderMaxDepth(rule.HeaderCompare.MaxHeaderDepth)
			row.RequireOrder = rule.HeaderCompare.RequireOrder
		case config.VerificationRuleTypeExactText:
			row.Sheet = rule.ExactText.Sheet
			row.ExpectedText = rule.ExactText.ExpectedText
		}

		rows = append(rows, row)
	}

	return rows
}

func buildTransferMappings(
	rows []templates.TransferRowView,
) []config.FileTransferMap {
	mappings := make([]config.FileTransferMap, 0, len(rows))

	for _, row := range rows {
		mappings = append(mappings, config.FileTransferMap{
			ExcelRow: row.ExcelRow,
			Src:      row.Src,
			Dest:     row.Dest,
		})
	}

	return mappings
}

func buildCheckConfigs(
	rows []templates.CheckRowView,
) []config.FileCheckConfig {
	checks := make([]config.FileCheckConfig, 0, len(rows))

	for _, row := range rows {
		checks = append(checks, buildCheckConfig(row))
	}

	return checks
}

func buildCheckConfig(row templates.CheckRowView) config.FileCheckConfig {
	return config.FileCheckConfig{
		ExcelRow:            row.ExcelRow,
		ID:                  row.ID,
		File:                row.File,
		CompareOffsetMonths: row.CompareOffsetMonths,
		Rules:               buildVerificationRules(row.ID, row.Rules),
	}
}

func buildVerificationRules(
	checkID string,
	rows []templates.CheckRuleView,
) []config.VerificationRule {
	rules := make([]config.VerificationRule, 0, len(rows))

	for _, row := range rows {
		rule := config.VerificationRule{
			ExcelRow: row.ExcelRow,
			ID:       row.ID,
			CheckID:  checkID,
			Name:     row.Name,
			Type:     config.VerificationRuleType(row.Type),
			Enabled:  row.Enabled,
		}

		switch rule.Type {
		case config.VerificationRuleTypeHeaderCompare:
			rule.HeaderCompare = config.HeaderCheckConfig{
				Sheet:           row.Sheet,
				Anchor:          row.Anchor,
				ParentDirection: row.ParentDirection,
				MaxHeaderDepth:  parseHeaderMaxDepth(row.MaxHeaderDepth),
				RequireOrder:    row.RequireOrder,
			}
		case config.VerificationRuleTypeExactText:
			rule.ExactText = config.ExactMatchCheckConfig{
				Sheet:        row.Sheet,
				ExpectedText: row.ExpectedText,
			}
		}

		rules = append(rules, rule)
	}

	return rules
}

func formatHeaderMaxDepth(value int) string {
	if value < 1 {
		return ""
	}

	return strconv.Itoa(value)
}

func parseHeaderMaxDepth(value string) int {
	depth, err := strconv.Atoi(value)
	if err != nil || depth < 1 {
		return 0
	}

	return depth
}

func applyCheckPathStatus(
	row *templates.CheckRowView,
	referenceDate time.Time,
) {
	if row == nil {
		return
	}

	currentFile, err := config.ResolvePathTemplate(row.File, referenceDate)
	if err != nil {
		row.Status = "Invalid template"
		row.Badge = "rose"
		row.Detail = fmt.Sprintf("file template: %v", err)
		return
	}
	row.FileExists = fileExistsOrFalse(currentFile)

	if !checkRowRequiresCompareOffset(*row) || row.CompareOffsetMonths == 0 {
		row.CompareExists = false
		return
	}

	compareDate := addMonthsClamped(referenceDate, row.CompareOffsetMonths)
	compareFile, err := config.ResolvePathTemplate(row.File, compareDate)
	if err != nil {
		row.Status = "Invalid template"
		row.Badge = "rose"
		row.Detail = fmt.Sprintf("compare file template: %v", err)
		return
	}
	row.CompareExists = fileExistsOrFalse(compareFile)
}

package main

import (
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
	rules []config.FileCheckRule,
) []templates.CheckRowView {
	rows := make([]templates.CheckRowView, 0, len(rules))

	for index, rule := range rules {
		rows = append(rows, templates.CheckRowView{
			Index:                 index + 1,
			ExcelRow:              rule.ExcelRow,
			NewFile:               rule.NewFile,
			NewExists:             fileExistsOrFalse(rule.NewFile),
			OldFile:               rule.OldFile,
			OldExists:             fileExistsOrFalse(rule.OldFile),
			HeaderSheet:           rule.HeaderCheck.Sheet,
			HeaderAnchor:          rule.HeaderCheck.Anchor,
			HeaderParentDirection: rule.HeaderCheck.ParentDirection,
			HeaderMaxDepth:        formatHeaderMaxDepth(rule.HeaderCheck.MaxHeaderDepth),
			RequireOrder:          rule.HeaderCheck.RequireOrder,
		})
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

func buildCheckRules(
	rows []templates.CheckRowView,
) []config.FileCheckRule {
	rules := make([]config.FileCheckRule, 0, len(rows))

	for _, row := range rows {
		rules = append(rules, buildCheckRule(row))
	}

	return rules
}

func buildCheckRule(row templates.CheckRowView) config.FileCheckRule {
	return config.FileCheckRule{
		ExcelRow: row.ExcelRow,
		NewFile:  row.NewFile,
		OldFile:  row.OldFile,
		HeaderCheck: config.HeaderCheckConfig{
			Sheet:           row.HeaderSheet,
			Anchor:          row.HeaderAnchor,
			ParentDirection: row.HeaderParentDirection,
			MaxHeaderDepth:  parseHeaderMaxDepth(row.HeaderMaxDepth),
			RequireOrder:    row.RequireOrder,
		},
	}
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

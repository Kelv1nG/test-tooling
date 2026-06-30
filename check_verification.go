package main

import (
	"fmt"
	"strings"
	"time"

	"github.com/xuri/excelize/v2"

	"tooling/config"
	"tooling/headersearch"
	"tooling/sheetsearch"
	"tooling/templates"
)

func runCheckVerification(
	rows []templates.CheckRowView,
	referenceDate time.Time,
) ([]templates.CheckRowView, templates.CheckSummaryView) {
	summary := templates.CheckSummaryView{HasRun: true}

	for index := range rows {
		runCheckConfigVerification(&rows[index], referenceDate, &summary)
	}

	return rows, summary
}

func runCheckConfigVerification(
	row *templates.CheckRowView,
	referenceDate time.Time,
	summary *templates.CheckSummaryView,
) {
	if row == nil {
		return
	}

	if len(row.Rules) == 0 {
		row.Status = "Not configured"
		row.Badge = "slate"
		row.Detail = "Add at least one verification rule."
		summary.Skipped++
		return
	}

	enabledRules := 0
	for index := range row.Rules {
		row.Rules[index].Status = ""
		row.Rules[index].Badge = ""
		row.Rules[index].Detail = ""

		if !row.Rules[index].Enabled {
			row.Rules[index].Status = "Disabled"
			row.Rules[index].Badge = "slate"
			row.Rules[index].Detail = "Rule is disabled."
			summary.Skipped++
			continue
		}

		enabledRules++
	}

	if enabledRules == 0 {
		row.Status = "Not configured"
		row.Badge = "slate"
		row.Detail = "All verification rules are disabled."
		return
	}

	newFile, err := config.ResolvePathTemplate(row.NewFile, referenceDate)
	if err != nil {
		markEnabledRulesErrored(row, summary, fmt.Sprintf("resolve new file template: %v", err))
		return
	}

	newWorkbook, err := excelize.OpenFile(newFile)
	if err != nil {
		markEnabledRulesErrored(row, summary, fmt.Sprintf("open new file: %v", err))
		return
	}
	defer newWorkbook.Close()

	var oldWorkbook *excelize.File
	var oldWorkbookErr error
	defer func() {
		if oldWorkbook != nil {
			_ = oldWorkbook.Close()
		}
	}()

	for index := range row.Rules {
		rule := &row.Rules[index]
		if !rule.Enabled {
			continue
		}

		summary.Attempted++

		switch config.VerificationRuleType(rule.Type) {
		case config.VerificationRuleTypeHeaderCompare:
			if oldWorkbook == nil && oldWorkbookErr == nil {
				oldWorkbook, oldWorkbookErr = openOldWorkbook(row.OldFile, referenceDate)
			}
			if oldWorkbookErr != nil {
				markRuleError(rule, summary, oldWorkbookErr.Error())
				continue
			}
			runHeaderComparisonRule(rule, oldWorkbook, newWorkbook, summary)
		case config.VerificationRuleTypeExactText:
			runExactTextRule(rule, newWorkbook, referenceDate, summary)
		default:
			markRuleError(rule, summary, fmt.Sprintf("unsupported rule type %q", rule.Type))
		}
	}

	applyCheckConfigStatus(row)
}

func openOldWorkbook(
	oldFileTemplate string,
	referenceDate time.Time,
) (*excelize.File, error) {
	if strings.TrimSpace(oldFileTemplate) == "" {
		return nil, fmt.Errorf("old file is required for header comparison")
	}

	oldFile, err := config.ResolvePathTemplate(oldFileTemplate, referenceDate)
	if err != nil {
		return nil, fmt.Errorf("resolve old file template: %w", err)
	}

	workbook, err := excelize.OpenFile(oldFile)
	if err != nil {
		return nil, fmt.Errorf("open old file: %w", err)
	}

	return workbook, nil
}

func runHeaderComparisonRule(
	rule *templates.CheckRuleView,
	oldWorkbook *excelize.File,
	newWorkbook *excelize.File,
	summary *templates.CheckSummaryView,
) {
	options, err := extractOptionsFromRule(*rule)
	if err != nil {
		markRuleError(rule, summary, err.Error())
		return
	}

	oldHeaders, err := headersearch.ExtractHeaders(oldWorkbook, options)
	if err != nil {
		markRuleError(rule, summary, fmt.Sprintf("old file extraction failed: %v", err))
		return
	}

	newHeaders, err := headersearch.ExtractHeaders(newWorkbook, options)
	if err != nil {
		markRuleError(rule, summary, fmt.Sprintf("new file extraction failed: %v", err))
		return
	}

	result := headersearch.CompareHeaders(
		oldHeaders,
		newHeaders,
		headersearch.CompareOptions{RequireOrder: rule.RequireOrder},
	)
	if result.Equal {
		markRuleMatched(rule, summary, "Headers match.")
		return
	}

	markRuleChanged(rule, summary, formatHeaderDifference(result.Difference))
}

func runExactTextRule(
	rule *templates.CheckRuleView,
	newWorkbook *excelize.File,
	referenceDate time.Time,
	summary *templates.CheckSummaryView,
) {
	expectedText, err := config.ResolveTemplateText(rule.ExpectedText, referenceDate)
	if err != nil {
		markRuleError(rule, summary, fmt.Sprintf("resolve exact-match template: %v", err))
		return
	}

	match, found, err := sheetsearch.FindExactText(
		newWorkbook,
		rule.Sheet,
		expectedText,
	)
	if err != nil {
		markRuleError(rule, summary, fmt.Sprintf("exact-match search failed: %v", err))
		return
	}

	if found {
		markRuleMatched(rule, summary, fmt.Sprintf("Exact text found at %s.", match.Cell))
		return
	}

	markRuleChanged(rule, summary, "Exact text not found in the new file.")
}

func extractOptionsFromRule(
	rule templates.CheckRuleView,
) (headersearch.ExtractOptions, error) {
	depth := parseHeaderMaxDepth(rule.MaxHeaderDepth)
	if depth < 1 {
		return headersearch.ExtractOptions{}, fmt.Errorf("max depth must be greater than 0")
	}

	direction := headersearch.Direction(strings.TrimSpace(rule.ParentDirection))
	if !direction.Valid() {
		return headersearch.ExtractOptions{}, fmt.Errorf("direction must be one of up, down, left, right")
	}

	return headersearch.ExtractOptions{
		Sheet:           strings.TrimSpace(rule.Sheet),
		Anchor:          strings.TrimSpace(rule.Anchor),
		ParentDirection: direction,
		MaxHeaderDepth:  depth,
	}, nil
}

func markEnabledRulesErrored(
	row *templates.CheckRowView,
	summary *templates.CheckSummaryView,
	detail string,
) {
	for index := range row.Rules {
		if !row.Rules[index].Enabled {
			continue
		}

		summary.Attempted++
		markRuleError(&row.Rules[index], summary, detail)
	}

	applyCheckConfigStatus(row)
}

func markRuleMatched(
	rule *templates.CheckRuleView,
	summary *templates.CheckSummaryView,
	detail string,
) {
	rule.Status = "Matched"
	rule.Badge = "emerald"
	rule.Detail = detail
	summary.Matched++
}

func markRuleChanged(
	rule *templates.CheckRuleView,
	summary *templates.CheckSummaryView,
	detail string,
) {
	rule.Status = "Changed"
	rule.Badge = "amber"
	rule.Detail = detail
	summary.Changed++
}

func markRuleError(
	rule *templates.CheckRuleView,
	summary *templates.CheckSummaryView,
	detail string,
) {
	rule.Status = "Error"
	rule.Badge = "rose"
	rule.Detail = detail
	summary.Errors++
}

func applyCheckConfigStatus(row *templates.CheckRowView) {
	statusCounts := map[string]int{}
	for _, rule := range row.Rules {
		statusCounts[rule.Status]++
	}

	switch {
	case statusCounts["Error"] > 0:
		row.Status = "Error"
		row.Badge = "rose"
	case statusCounts["Changed"] > 0:
		row.Status = "Changed"
		row.Badge = "amber"
	case statusCounts["Matched"] > 0:
		row.Status = "Matched"
		row.Badge = "emerald"
	default:
		row.Status = "Not configured"
		row.Badge = "slate"
	}

	row.Detail = fmt.Sprintf(
		"%d matched, %d changed, %d errors, %d skipped.",
		statusCounts["Matched"],
		statusCounts["Changed"],
		statusCounts["Error"],
		statusCounts["Disabled"]+statusCounts[""],
	)
}

func formatHeaderDifference(
	difference headersearch.HeaderDifference,
) string {
	parts := make([]string, 0, 3)

	if len(difference.Missing) > 0 {
		parts = append(parts, fmt.Sprintf("%d missing from new file", len(difference.Missing)))
	}

	if len(difference.Unexpected) > 0 {
		parts = append(parts, fmt.Sprintf("%d unexpected in new file", len(difference.Unexpected)))
	}

	if difference.Reordered {
		parts = append(parts, "same headers, different order")
	}

	if len(parts) == 0 {
		return "Header comparison found differences."
	}

	return strings.Join(parts, "; ") + "."
}

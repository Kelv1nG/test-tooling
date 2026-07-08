package main

import (
	"fmt"
	"math"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/xuri/excelize/v2"

	"tooling/config"
	"tooling/headersearch"
	"tooling/sheetsearch"
	"tooling/templates"
)

const maxConcurrentCheckVerifications = 5

func runCheckVerification(
	rows []templates.CheckRowView,
	referenceDate time.Time,
) ([]templates.CheckRowView, templates.CheckSummaryView) {
	return runCheckVerificationWithProgress(rows, referenceDate, nil)
}

type checkVerificationProgress struct {
	Index     int
	Row       templates.CheckRowView
	Summary   templates.CheckSummaryView
	Completed int
	Total     int
}

func runCheckVerificationWithProgress(
	rows []templates.CheckRowView,
	referenceDate time.Time,
	progress func(checkVerificationProgress),
) ([]templates.CheckRowView, templates.CheckSummaryView) {
	summary := templates.CheckSummaryView{HasRun: true}
	results := make([]templates.CheckRowView, len(rows))
	rowResults := make(chan checkVerificationRowResult, len(rows))
	sem := make(chan struct{}, maxConcurrentCheckVerifications)
	var wg sync.WaitGroup

	for index := range rows {
		wg.Add(1)
		go func(index int, row templates.CheckRowView) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() {
				<-sem
			}()

			rowSummary := templates.CheckSummaryView{}
			runCheckConfigVerification(&row, referenceDate, &rowSummary)
			rowResults <- checkVerificationRowResult{
				index:   index,
				row:     row,
				summary: rowSummary,
			}
		}(index, rows[index])
	}

	go func() {
		wg.Wait()
		close(rowResults)
	}()

	completed := 0
	for result := range rowResults {
		results[result.index] = result.row
		addCheckSummary(&summary, result.summary)
		completed++

		if progress != nil {
			progress(checkVerificationProgress{
				Index:     result.index,
				Row:       result.row,
				Summary:   summary,
				Completed: completed,
				Total:     len(rows),
			})
		}
	}

	return results, summary
}

type checkVerificationRowResult struct {
	index   int
	row     templates.CheckRowView
	summary templates.CheckSummaryView
}

func addCheckSummary(
	target *templates.CheckSummaryView,
	addition templates.CheckSummaryView,
) {
	target.Attempted += addition.Attempted
	target.Matched += addition.Matched
	target.Changed += addition.Changed
	target.Errors += addition.Errors
	target.Skipped += addition.Skipped
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

	currentFile, err := config.ResolvePathTemplateSingleMatch(row.File, referenceDate)
	if err != nil {
		markEnabledRulesErrored(row, summary, fmt.Sprintf("resolve current file pattern: %v", err))
		return
	}

	currentWorkbook, err := excelize.OpenFile(currentFile)
	if err != nil {
		markEnabledRulesErrored(row, summary, fmt.Sprintf("open current file: %v", err))
		return
	}
	defer currentWorkbook.Close()

	var compareWorkbook *excelize.File
	var compareWorkbookErr error
	defer func() {
		if compareWorkbook != nil {
			_ = compareWorkbook.Close()
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
			if compareWorkbook == nil && compareWorkbookErr == nil {
				compareWorkbook, compareWorkbookErr = openCompareWorkbook(row.File, row.CompareOffsetMonths, referenceDate)
			}
			if compareWorkbookErr != nil {
				markRuleError(rule, summary, compareWorkbookErr.Error())
				continue
			}
			runHeaderComparisonRule(rule, compareWorkbook, currentWorkbook, summary)
		case config.VerificationRuleTypeExactText:
			runExactTextRule(rule, currentWorkbook, referenceDate, summary)
		case config.VerificationRuleTypeAnchorScan:
			runAnchorScanRule(rule, currentWorkbook, referenceDate, summary)
		default:
			markRuleError(rule, summary, fmt.Sprintf("unsupported rule type %q", rule.Type))
		}
	}

	applyCheckConfigStatus(row)
}

func openCompareWorkbook(
	fileTemplate string,
	compareOffsetMonths int,
	referenceDate time.Time,
) (*excelize.File, error) {
	if compareOffsetMonths == 0 {
		return nil, fmt.Errorf("compare offset months must be non-zero for header comparison")
	}

	compareDate := addMonthsClamped(referenceDate, compareOffsetMonths)
	compareFile, err := config.ResolvePathTemplateSingleMatch(fileTemplate, compareDate)
	if err != nil {
		return nil, fmt.Errorf("resolve compare file pattern: %w", err)
	}

	workbook, err := excelize.OpenFile(compareFile)
	if err != nil {
		return nil, fmt.Errorf("open compare file: %w", err)
	}

	return workbook, nil
}

func runHeaderComparisonRule(
	rule *templates.CheckRuleView,
	compareWorkbook *excelize.File,
	currentWorkbook *excelize.File,
	summary *templates.CheckSummaryView,
) {
	options, err := extractOptionsFromRule(*rule)
	if err != nil {
		markRuleError(rule, summary, err.Error())
		return
	}

	compareHeaders, err := headersearch.ExtractHeaders(compareWorkbook, options)
	if err != nil {
		markRuleError(rule, summary, fmt.Sprintf("compare file extraction failed: %v", err))
		return
	}

	currentHeaders, err := headersearch.ExtractHeaders(currentWorkbook, options)
	if err != nil {
		markRuleError(rule, summary, fmt.Sprintf("current file extraction failed: %v", err))
		return
	}

	result := headersearch.CompareHeaders(
		compareHeaders,
		currentHeaders,
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

	markRuleNotFound(rule, summary, "Exact text not found in the current file.")
}

func runAnchorScanRule(
	rule *templates.CheckRuleView,
	newWorkbook *excelize.File,
	referenceDate time.Time,
	summary *templates.CheckSummaryView,
) {
	expectedText, err := config.ResolveTemplateText(rule.ExpectedText, referenceDate)
	if err != nil {
		markRuleError(rule, summary, fmt.Sprintf("resolve anchor-scan template: %v", err))
		return
	}

	match, found, err := sheetsearch.FindAnchorScanValue(
		newWorkbook,
		sheetsearch.AnchorScanOptions{
			Sheet:     rule.Sheet,
			Anchor:    rule.Anchor,
			Direction: sheetsearch.Direction(strings.TrimSpace(rule.ParentDirection)),
			Select:    rule.ScanSelect,
		},
	)
	if err != nil {
		markRuleError(rule, summary, fmt.Sprintf("anchor-scan search failed: %v", err))
		return
	}

	if !found {
		markRuleChanged(
			rule,
			summary,
			fmt.Sprintf("No non-empty value found %s from anchor at %s.", rule.ParentDirection, match.AnchorCell),
		)
		return
	}

	matched, detail, err := compareAnchorScanValue(match, expectedText, rule.CompareAs)
	if err != nil {
		markRuleError(rule, summary, err.Error())
		return
	}

	if matched {
		markRuleMatched(rule, summary, detail)
		return
	}

	markRuleChanged(rule, summary, detail)
}

func compareAnchorScanValue(
	match sheetsearch.AnchorScanMatch,
	expected string,
	compareAs string,
) (bool, string, error) {
	switch compareAs {
	case config.AnchorScanCompareExactText:
		if match.Value == expected {
			return true, fmt.Sprintf("Scanned value at %s matched expected text.", match.Cell), nil
		}

		return false, fmt.Sprintf("Scanned value at %s was %q; expected %q.", match.Cell, match.Value, expected), nil
	case config.AnchorScanCompareDate:
		expectedDate, err := parseComparableDate(expected)
		if err != nil {
			return false, "", fmt.Errorf("expected date %q is not recognizable: %v", expected, err)
		}

		actualDate, err := parseComparableDate(match.Value)
		if err != nil {
			// A bad value in the workbook means the rule changed; only an
			// unparseable configured expectation is a rule error.
			return false,
				fmt.Sprintf(
					"Scanned value at %s was %q, which is not a recognizable date; expected %s.",
					match.Cell,
					match.Value,
					formatComparableDate(expectedDate),
				),
				nil
		}

		if sameDate(actualDate, expectedDate) {
			return true,
				fmt.Sprintf(
					"Scanned value at %s matched expected date %s.",
					match.Cell,
					formatComparableDate(expectedDate),
				),
				nil
		}

		return false,
			fmt.Sprintf(
				"Scanned value at %s was %s; expected %s.",
				match.Cell,
				formatComparableDate(actualDate),
				formatComparableDate(expectedDate),
			),
			nil
	default:
		return false, "", fmt.Errorf("unsupported anchor-scan comparison mode %q", compareAs)
	}
}

// parseComparableDate accepts the common text formats produced by templates,
// user-entered report dates, and raw Excel serial dates.
func parseComparableDate(value string) (time.Time, error) {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return time.Time{}, fmt.Errorf("date is empty")
	}

	for _, layout := range comparableDateLayouts() {
		date, err := time.ParseInLocation(layout, trimmed, time.UTC)
		if err != nil {
			continue
		}

		return dateOnly(date), nil
	}

	if date, ok := parseExcelSerialDate(trimmed); ok {
		return date, nil
	}

	return time.Time{}, fmt.Errorf("unsupported date format")
}

func comparableDateLayouts() []string {
	return []string{
		"2006-01-02",
		"2006/01/02",
		"1/2/2006",
		"01/02/2006",
		"1/2/06",
		"01/02/06",
		"1-2-2006",
		"01-02-2006",
		"1-2-06",
		"01-02-06",
		"Jan 2, 2006",
		"January 2, 2006",
		"2 Jan 2006",
		"02 Jan 2006",
		"2-Jan-2006",
		"02-Jan-2006",
		"20060102",
	}
}

func parseExcelSerialDate(value string) (time.Time, bool) {
	serial, err := strconv.ParseFloat(value, 64)
	if err != nil || serial < 1 || serial > 60000 {
		return time.Time{}, false
	}

	// Excel's 1900 date system treats 1900 as a leap year for Lotus 1-2-3
	// compatibility. Serial 60 is the fake 1900-02-29, and later dates need
	// to be shifted back by one day.
	days := int(math.Floor(serial))
	if days == 60 {
		return time.Time{}, false
	}
	if days > 60 {
		days--
	}

	base := time.Date(1899, time.December, 31, 0, 0, 0, 0, time.UTC)
	return base.AddDate(0, 0, days), true
}

func dateOnly(date time.Time) time.Time {
	return time.Date(date.Year(), date.Month(), date.Day(), 0, 0, 0, 0, time.UTC)
}

func sameDate(left time.Time, right time.Time) bool {
	return left.Year() == right.Year() &&
		left.Month() == right.Month() &&
		left.Day() == right.Day()
}

func formatComparableDate(date time.Time) string {
	return date.Format("2006-01-02")
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
		Sheet:             strings.TrimSpace(rule.Sheet),
		Anchor:            strings.TrimSpace(rule.Anchor),
		ParentDirection:   direction,
		MaxHeaderDepth:    depth,
		IgnoreAnchorLayer: true,
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

func markRuleNotFound(
	rule *templates.CheckRuleView,
	summary *templates.CheckSummaryView,
	detail string,
) {
	rule.Status = "Not found"
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
	changedCount := statusCounts["Changed"] + statusCounts["Not found"]

	switch {
	case statusCounts["Error"] > 0:
		row.Status = "Error"
		row.Badge = "rose"
	case changedCount > 0:
		row.Status = "Changed"
		row.Badge = "amber"
	case statusCounts["Matched"] > 0:
		row.Status = "Matched"
		row.Badge = "emerald"
	default:
		row.Status = "Not configured"
		row.Badge = "slate"
	}

	detail := fmt.Sprintf(
		"%d matched, %d changed, %d errors, %d skipped.",
		statusCounts["Matched"],
		changedCount,
		statusCounts["Error"],
		statusCounts["Disabled"]+statusCounts[""],
	)

	if highlights := checkConfigHighlights(row.Rules); len(highlights) > 0 {
		detail += "\n" + strings.Join(highlights, "\n")
	}

	row.Detail = detail
}

func checkConfigHighlights(
	rules []templates.CheckRuleView,
) []string {
	highlights := make([]string, 0)

	for _, rule := range rules {
		if rule.Status != "Changed" && rule.Status != "Not found" && rule.Status != "Error" {
			continue
		}

		label := rule.Name
		if strings.TrimSpace(label) == "" {
			label = rule.ID
		}
		if strings.TrimSpace(label) == "" {
			label = fmt.Sprintf("Rule %d", rule.Index)
		}

		highlights = append(highlights, fmt.Sprintf("%s:", label))
		for _, line := range splitDetailLines(rule.Detail) {
			highlights = append(highlights, line)
		}
	}

	return highlights
}

func splitDetailLines(detail string) []string {
	lines := make([]string, 0)

	for _, line := range strings.Split(detail, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		lines = append(lines, line)
	}

	return lines
}

func formatHeaderDifference(
	difference headersearch.HeaderDifference,
) string {
	lines := make([]string, 0, len(difference.Missing)+len(difference.Unexpected)+3)

	if len(difference.Missing) > 0 || len(difference.Unexpected) > 0 {
		lines = append(lines, "column changes:")
	}

	for _, path := range difference.Missing {
		lines = append(lines, "-- "+formatChangedColumn(path))
	}

	for _, path := range difference.Unexpected {
		lines = append(lines, "++ "+formatChangedColumn(path))
	}

	if difference.Reordered {
		lines = append(lines, "column order changed")
	}

	if len(lines) == 0 {
		return "Header comparison found differences."
	}

	return strings.Join(lines, "\n")
}

func formatChangedColumn(path []string) string {
	if len(path) == 0 {
		return "(blank column)"
	}
	if len(path) == 1 {
		return path[0]
	}

	return strings.Join(path, " > ")
}

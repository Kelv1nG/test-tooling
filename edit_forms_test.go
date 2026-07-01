package main

import (
	"strings"
	"testing"
	"time"
)

func TestParseTransferRowsFormBuildsRowsFromTemplates(t *testing.T) {
	referenceDate := time.Date(2026, time.February, 3, 0, 0, 0, 0, time.UTC)

	rows, err := parseTransferRowsForm(map[string][]string{
		"transferExcelRow": []string{"2"},
		"transferSrc":      []string{`/tmp/report_{yyyy}_{mm}_{dd}.csv`},
		"transferDest":     []string{`/tmp/archive_{mmm}.csv`},
	}, referenceDate)
	if err != nil {
		t.Fatalf("parseTransferRowsForm returned error: %v", err)
	}

	if len(rows) != 1 {
		t.Fatalf("expected 1 row, got %d", len(rows))
	}

	if rows[0].Src != `/tmp/report_{yyyy}_{mm}_{dd}.csv` {
		t.Fatalf("unexpected source template: %q", rows[0].Src)
	}

	if rows[0].Dest != `/tmp/archive_{mmm}.csv` {
		t.Fatalf("unexpected destination template: %q", rows[0].Dest)
	}
}

func TestParseTransferRowsFormRejectsUnsupportedTemplate(t *testing.T) {
	referenceDate := time.Date(2026, time.February, 3, 0, 0, 0, 0, time.UTC)

	_, err := parseTransferRowsForm(map[string][]string{
		"transferExcelRow": []string{"2"},
		"transferSrc":      []string{`/tmp/report_{offset}.csv`},
		"transferDest":     []string{`/tmp/archive.csv`},
	}, referenceDate)
	if err == nil {
		t.Fatal("expected template validation error, got nil")
	}

	if !strings.Contains(err.Error(), "invalid source path template") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestParseCheckRowsFormAllowsExactMatchWithoutCompareOffset(t *testing.T) {
	referenceDate := time.Date(2026, time.June, 30, 0, 0, 0, 0, time.UTC)

	rows, err := parseCheckRowsForm(map[string][]string{
		"checkExcelRow":            []string{"2"},
		"checkID":                  []string{"CHK-001"},
		"checkFile":                []string{`/tmp/report_{yyyy}_{mm}_{dd}.xlsx`},
		"checkCompareOffsetMonths": []string{"0"},
		"ruleParentIndex":          []string{"1"},
		"ruleExcelRow":             []string{"2"},
		"ruleID":                   []string{"R001"},
		"ruleName":                 []string{"Performance phrase"},
		"ruleType":                 []string{"exact_text"},
		"ruleEnabled":              []string{"true"},
		"ruleSheet":                []string{"Report"},
		"ruleAnchor":               []string{""},
		"ruleParentDirection":      []string{""},
		"ruleMaxHeaderDepth":       []string{""},
		"ruleRequireOrder":         []string{"false"},
		"ruleScanSelect":           []string{""},
		"ruleExpectedText":         []string{"Actual performance from 10/5/2021 to {mm}/{dd}/{yy}"},
		"ruleCompareAs":            []string{""},
	}, referenceDate)
	if err != nil {
		t.Fatalf("parseCheckRowsForm returned error: %v", err)
	}

	if len(rows) != 1 {
		t.Fatalf("expected 1 row, got %d", len(rows))
	}

	if rows[0].CompareOffsetMonths != 0 {
		t.Fatalf("expected compare offset to stay 0, got %d", rows[0].CompareOffsetMonths)
	}
	if len(rows[0].Rules) != 1 {
		t.Fatalf("expected 1 rule, got %d", len(rows[0].Rules))
	}
	if rows[0].Rules[0].Sheet != "Report" {
		t.Fatalf("unexpected exact-match sheet: %q", rows[0].Rules[0].Sheet)
	}
	if rows[0].Rules[0].ExpectedText != "Actual performance from 10/5/2021 to {mm}/{dd}/{yy}" {
		t.Fatalf("unexpected expected text: %q", rows[0].Rules[0].ExpectedText)
	}
}

func TestParseCheckRowsFormGeneratesMissingCheckID(t *testing.T) {
	referenceDate := time.Date(2026, time.June, 30, 0, 0, 0, 0, time.UTC)

	rows, err := parseCheckRowsForm(map[string][]string{
		"checkExcelRow":            []string{"0"},
		"checkID":                  []string{""},
		"checkFile":                []string{`/tmp/report.xlsx`},
		"checkCompareOffsetMonths": []string{"0"},
	}, referenceDate)
	if err != nil {
		t.Fatalf("parseCheckRowsForm returned error: %v", err)
	}

	if len(rows) != 1 {
		t.Fatalf("expected 1 row, got %d", len(rows))
	}
	if rows[0].ID != "CHECK-001" {
		t.Fatalf("expected generated check id, got %q", rows[0].ID)
	}
}

func TestParseCheckRowsFormRequiresCompareOffsetForHeaderComparison(t *testing.T) {
	referenceDate := time.Date(2026, time.June, 30, 0, 0, 0, 0, time.UTC)

	_, err := parseCheckRowsForm(map[string][]string{
		"checkExcelRow":            []string{"2"},
		"checkID":                  []string{"CHK-001"},
		"checkFile":                []string{`/tmp/report.xlsx`},
		"checkCompareOffsetMonths": []string{"0"},
		"ruleParentIndex":          []string{"1"},
		"ruleExcelRow":             []string{"2"},
		"ruleID":                   []string{"R001"},
		"ruleName":                 []string{"Headers unchanged"},
		"ruleType":                 []string{"header_compare"},
		"ruleEnabled":              []string{"true"},
		"ruleSheet":                []string{"Report"},
		"ruleAnchor":               []string{"Fund Name"},
		"ruleParentDirection":      []string{"up"},
		"ruleMaxHeaderDepth":       []string{"1"},
		"ruleRequireOrder":         []string{"false"},
		"ruleScanSelect":           []string{""},
		"ruleExpectedText":         []string{""},
		"ruleCompareAs":            []string{""},
	}, referenceDate)
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	if !strings.Contains(err.Error(), "requires a non-zero compare offset when a header comparison rule is enabled") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestParseCheckRowsFormAllowsAnchorScanWithoutCompareOffset(t *testing.T) {
	referenceDate := time.Date(2026, time.June, 30, 0, 0, 0, 0, time.UTC)

	rows, err := parseCheckRowsForm(map[string][]string{
		"checkExcelRow":            []string{"2"},
		"checkID":                  []string{"CHK-001"},
		"checkFile":                []string{`/tmp/report_{yyyy}_{mm}_{dd}.xlsx`},
		"checkCompareOffsetMonths": []string{"0"},
		"ruleParentIndex":          []string{"1"},
		"ruleExcelRow":             []string{"2"},
		"ruleID":                   []string{"R001"},
		"ruleName":                 []string{"Reporting date"},
		"ruleType":                 []string{"anchor_scan_match"},
		"ruleEnabled":              []string{"true"},
		"ruleSheet":                []string{"Report"},
		"ruleAnchor":               []string{"Reporting dates"},
		"ruleParentDirection":      []string{"down"},
		"ruleMaxHeaderDepth":       []string{""},
		"ruleRequireOrder":         []string{"false"},
		"ruleScanSelect":           []string{"last_non_empty_before_blank"},
		"ruleExpectedText":         []string{"{mm}/{dd}/{yy}"},
		"ruleCompareAs":            []string{"date"},
	}, referenceDate)
	if err != nil {
		t.Fatalf("parseCheckRowsForm returned error: %v", err)
	}

	if len(rows) != 1 {
		t.Fatalf("expected 1 row, got %d", len(rows))
	}
	if rows[0].CompareOffsetMonths != 0 {
		t.Fatalf("expected compare offset to stay 0, got %d", rows[0].CompareOffsetMonths)
	}
	if len(rows[0].Rules) != 1 {
		t.Fatalf("expected 1 rule, got %d", len(rows[0].Rules))
	}

	rule := rows[0].Rules[0]
	if rule.Anchor != "Reporting dates" {
		t.Fatalf("anchor = %q", rule.Anchor)
	}
	if rule.ParentDirection != "down" {
		t.Fatalf("direction = %q", rule.ParentDirection)
	}
	if rule.ScanSelect != "last_non_empty_before_blank" {
		t.Fatalf("scan selector = %q", rule.ScanSelect)
	}
	if rule.CompareAs != "date" {
		t.Fatalf("compare_as = %q", rule.CompareAs)
	}
}

func TestParseCheckRowsFormGeneratesMissingRuleID(t *testing.T) {
	referenceDate := time.Date(2026, time.June, 30, 0, 0, 0, 0, time.UTC)

	rows, err := parseCheckRowsForm(map[string][]string{
		"checkExcelRow":            []string{"2"},
		"checkID":                  []string{"CHK-001"},
		"checkFile":                []string{`/tmp/report_{yyyy}_{mm}_{dd}.xlsx`},
		"checkCompareOffsetMonths": []string{"0"},
		"ruleParentIndex":          []string{"1", "1"},
		"ruleExcelRow":             []string{"2", "0"},
		"ruleID":                   []string{"R001", ""},
		"ruleName":                 []string{"Performance phrase", "Second phrase"},
		"ruleType":                 []string{"exact_text", "exact_text"},
		"ruleEnabled":              []string{"true", "true"},
		"ruleSheet":                []string{"Report", "Report"},
		"ruleAnchor":               []string{"", ""},
		"ruleParentDirection":      []string{"", ""},
		"ruleMaxHeaderDepth":       []string{"", ""},
		"ruleRequireOrder":         []string{"false", "false"},
		"ruleScanSelect":           []string{"", ""},
		"ruleExpectedText":         []string{"Actual performance from 10/5/2021 to {mm}/{dd}/{yy}", "Report date {mm}/{dd}/{yy}"},
		"ruleCompareAs":            []string{"", ""},
	}, referenceDate)
	if err != nil {
		t.Fatalf("parseCheckRowsForm returned error: %v", err)
	}

	if len(rows) != 1 || len(rows[0].Rules) != 2 {
		t.Fatalf("expected 1 row with 2 rules, got %+v", rows)
	}
	if rows[0].Rules[1].ID != "R002" {
		t.Fatalf("generated rule id = %q, want R002", rows[0].Rules[1].ID)
	}
}

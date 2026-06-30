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

func TestParseCheckRowsFormAllowsExactMatchWithoutOldFile(t *testing.T) {
	referenceDate := time.Date(2026, time.June, 30, 0, 0, 0, 0, time.UTC)

	rows, err := parseCheckRowsForm(map[string][]string{
		"checkExcelRow":       []string{"2"},
		"checkID":             []string{"CHK-001"},
		"checkNewFile":        []string{`/tmp/report_{yyyy}_{mm}_{dd}.xlsx`},
		"checkOldFile":        []string{""},
		"ruleParentIndex":     []string{"1"},
		"ruleExcelRow":        []string{"2"},
		"ruleID":              []string{"R001"},
		"ruleName":            []string{"Performance phrase"},
		"ruleType":            []string{"exact_text"},
		"ruleEnabled":         []string{"true"},
		"ruleSheet":           []string{"Report"},
		"ruleAnchor":          []string{""},
		"ruleParentDirection": []string{""},
		"ruleMaxHeaderDepth":  []string{""},
		"ruleRequireOrder":    []string{"false"},
		"ruleExpectedText":    []string{"Actual performance from 10/5/2021 to {mm}/{dd}/{yy}"},
	}, referenceDate)
	if err != nil {
		t.Fatalf("parseCheckRowsForm returned error: %v", err)
	}

	if len(rows) != 1 {
		t.Fatalf("expected 1 row, got %d", len(rows))
	}

	if rows[0].OldFile != "" {
		t.Fatalf("expected old file to stay empty, got %q", rows[0].OldFile)
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

func TestParseCheckRowsFormRequiresOldFileForHeaderComparison(t *testing.T) {
	referenceDate := time.Date(2026, time.June, 30, 0, 0, 0, 0, time.UTC)

	_, err := parseCheckRowsForm(map[string][]string{
		"checkExcelRow":       []string{"2"},
		"checkID":             []string{"CHK-001"},
		"checkNewFile":        []string{`/tmp/report.xlsx`},
		"checkOldFile":        []string{""},
		"ruleParentIndex":     []string{"1"},
		"ruleExcelRow":        []string{"2"},
		"ruleID":              []string{"R001"},
		"ruleName":            []string{"Headers unchanged"},
		"ruleType":            []string{"header_compare"},
		"ruleEnabled":         []string{"true"},
		"ruleSheet":           []string{"Report"},
		"ruleAnchor":          []string{"Fund Name"},
		"ruleParentDirection": []string{"up"},
		"ruleMaxHeaderDepth":  []string{"1"},
		"ruleRequireOrder":    []string{"false"},
		"ruleExpectedText":    []string{""},
	}, referenceDate)
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	if !strings.Contains(err.Error(), "requires an old file path when a header comparison rule is enabled") {
		t.Fatalf("unexpected error: %v", err)
	}
}

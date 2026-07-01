package config

import (
	"testing"

	"github.com/xuri/excelize/v2"
)

func TestFileCheckReadRulesFromJSONConfig(t *testing.T) {
	file := newFileCheckWorkbook(t)
	definition := testFileCheckDefinition()

	setRow(t, file, "File Checks", 1, []string{"Check ID", "File", "Compare Offset Months"})
	setRow(t, file, "File Checks", 2, []string{"CHK-001", "/tmp/report_{yyyy}_{mm}_{dd}.xlsx", "-1"})
	setRow(t, file, "File Check Rules", 1, []string{"Check ID", "Rule ID", "Rule Name", "Rule Type", "Enabled", "Rule Config"})
	setRow(t, file, "File Check Rules", 2, []string{
		"CHK-001",
		"R001",
		"Headers unchanged",
		"header_compare",
		"true",
		`{"sheet":"Report","anchor":"Fund Name","parent_direction":"up","max_header_depth":2,"require_order":true}`,
	})
	setRow(t, file, "File Check Rules", 3, []string{
		"CHK-001",
		"R002",
		"Performance phrase",
		"exact_text",
		"true",
		`{"sheet":"Report","expected_text":"Actual performance from 10/5/2021 to {mm}/{dd}/{yy}"}`,
	})
	setRow(t, file, "File Check Rules", 4, []string{
		"CHK-001",
		"R003",
		"Reporting date",
		"anchor_scan_match",
		"true",
		`{"sheet":"Report","anchor":"Reporting dates","direction":"down","select":"last_non_empty_before_blank","expected_text":"{mm}/{dd}/{yy}","compare_as":"date"}`,
	})

	configs, err := definition.read(file)
	if err != nil {
		t.Fatalf("read returned error: %v", err)
	}
	if len(configs) != 1 {
		t.Fatalf("expected 1 check config, got %d", len(configs))
	}
	if len(configs[0].Rules) != 3 {
		t.Fatalf("expected 3 rules, got %d", len(configs[0].Rules))
	}

	headerRule := configs[0].Rules[0]
	if headerRule.HeaderCompare.Sheet != "Report" {
		t.Fatalf("header sheet = %q, want Report", headerRule.HeaderCompare.Sheet)
	}
	if headerRule.HeaderCompare.Anchor != "Fund Name" {
		t.Fatalf("header anchor = %q, want Fund Name", headerRule.HeaderCompare.Anchor)
	}
	if headerRule.HeaderCompare.MaxHeaderDepth != 2 {
		t.Fatalf("header depth = %d, want 2", headerRule.HeaderCompare.MaxHeaderDepth)
	}
	if !headerRule.HeaderCompare.RequireOrder {
		t.Fatal("expected header rule to require order")
	}

	exactRule := configs[0].Rules[1]
	if exactRule.ExactText.ExpectedText != "Actual performance from 10/5/2021 to {mm}/{dd}/{yy}" {
		t.Fatalf("exact text = %q", exactRule.ExactText.ExpectedText)
	}

	anchorScanRule := configs[0].Rules[2]
	if anchorScanRule.AnchorScan.Anchor != "Reporting dates" {
		t.Fatalf("anchor scan anchor = %q", anchorScanRule.AnchorScan.Anchor)
	}
	if anchorScanRule.AnchorScan.Direction != "down" {
		t.Fatalf("anchor scan direction = %q", anchorScanRule.AnchorScan.Direction)
	}
	if anchorScanRule.AnchorScan.CompareAs != "date" {
		t.Fatalf("anchor scan compare_as = %q", anchorScanRule.AnchorScan.CompareAs)
	}
}

func TestFileCheckSaveWritesJSONConfig(t *testing.T) {
	file := newFileCheckWorkbook(t)
	definition := testFileCheckDefinition()

	setRow(t, file, "File Checks", 1, []string{"Check ID", "File", "Compare Offset Months"})
	setRow(t, file, "File Check Rules", 1, []string{"Check ID", "Rule ID", "Rule Name", "Rule Type", "Enabled", "Rule Config"})

	err := definition.save(file, []FileCheckConfig{
		{
			ID:                  "CHK-001",
			File:                "/tmp/report_{yyyy}_{mm}_{dd}.xlsx",
			CompareOffsetMonths: -1,
			Rules: []VerificationRule{
				{
					ID:      "R001",
					Name:    "Headers unchanged",
					Type:    VerificationRuleTypeHeaderCompare,
					Enabled: true,
					HeaderCompare: HeaderCheckConfig{
						Sheet:           "Report",
						Anchor:          "Fund Name",
						ParentDirection: "up",
						MaxHeaderDepth:  2,
						RequireOrder:    true,
					},
				},
				{
					ID:      "R002",
					Name:    "Reporting date",
					Type:    VerificationRuleTypeAnchorScan,
					Enabled: true,
					AnchorScan: AnchorScanMatchConfig{
						Sheet:        "Report",
						Anchor:       "Reporting dates",
						Direction:    "down",
						Select:       AnchorScanSelectLastNonEmptyBeforeBlank,
						ExpectedText: "{mm}/{dd}/{yy}",
						CompareAs:    AnchorScanCompareDate,
					},
				},
			},
		},
	})
	if err != nil {
		t.Fatalf("save returned error: %v", err)
	}

	configJSON, err := file.GetCellValue("File Check Rules", "F2")
	if err != nil {
		t.Fatalf("GetCellValue returned error: %v", err)
	}

	const expected = `{"sheet":"Report","anchor":"Fund Name","parent_direction":"up","max_header_depth":2,"require_order":true}`
	if configJSON != expected {
		t.Fatalf("config JSON = %q, want %q", configJSON, expected)
	}

	anchorScanJSON, err := file.GetCellValue("File Check Rules", "F3")
	if err != nil {
		t.Fatalf("GetCellValue returned error: %v", err)
	}

	const expectedAnchorScan = `{"sheet":"Report","anchor":"Reporting dates","direction":"down","select":"last_non_empty_before_blank","expected_text":"{mm}/{dd}/{yy}","compare_as":"date"}`
	if anchorScanJSON != expectedAnchorScan {
		t.Fatalf("anchor scan config JSON = %q, want %q", anchorScanJSON, expectedAnchorScan)
	}
}

func testFileCheckDefinition() FileCheckTableDefinition {
	return FileCheckTableDefinition{
		Sheet:                  "File Checks",
		IDCol:                  "Check ID",
		FileCol:                "File",
		CompareOffsetMonthsCol: "Compare Offset Months",
		Rules: FileCheckRulesTableDefinition{
			Sheet:       "File Check Rules",
			CheckIDCol:  "Check ID",
			RuleIDCol:   "Rule ID",
			RuleNameCol: "Rule Name",
			RuleTypeCol: "Rule Type",
			EnabledCol:  "Enabled",
			ConfigCol:   "Rule Config",
		},
	}
}

func newFileCheckWorkbook(t *testing.T) *excelize.File {
	t.Helper()

	file := excelize.NewFile()
	defaultSheet := file.GetSheetName(file.GetActiveSheetIndex())
	if err := file.SetSheetName(defaultSheet, "File Checks"); err != nil {
		t.Fatalf("SetSheetName returned error: %v", err)
	}
	if _, err := file.NewSheet("File Check Rules"); err != nil {
		t.Fatalf("NewSheet returned error: %v", err)
	}

	return file
}

func setRow(t *testing.T, file *excelize.File, sheet string, row int, values []string) {
	t.Helper()

	for col, value := range values {
		cell, err := excelize.CoordinatesToCellName(col+1, row)
		if err != nil {
			t.Fatalf("CoordinatesToCellName returned error: %v", err)
		}
		if err := file.SetCellStr(sheet, cell, value); err != nil {
			t.Fatalf("SetCellStr returned error: %v", err)
		}
	}
}

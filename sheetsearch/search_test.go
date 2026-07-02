package sheetsearch

import (
	"testing"

	"github.com/xuri/excelize/v2"
)

func TestResolveSheetNameByIndex(t *testing.T) {
	workbook := excelize.NewFile()
	defaultSheet := workbook.GetSheetName(workbook.GetActiveSheetIndex())
	if err := workbook.SetSheetName(defaultSheet, "Summary"); err != nil {
		t.Fatalf("SetSheetName returned error: %v", err)
	}
	if _, err := workbook.NewSheet("Report"); err != nil {
		t.Fatalf("NewSheet returned error: %v", err)
	}

	sheetName, err := ResolveSheetName(workbook, "2")
	if err != nil {
		t.Fatalf("ResolveSheetName returned error: %v", err)
	}

	if sheetName != "Report" {
		t.Fatalf("ResolveSheetName = %q, want %q", sheetName, "Report")
	}
}

func TestFindExactText(t *testing.T) {
	workbook := excelize.NewFile()
	defaultSheet := workbook.GetSheetName(workbook.GetActiveSheetIndex())
	if err := workbook.SetSheetName(defaultSheet, "Report"); err != nil {
		t.Fatalf("SetSheetName returned error: %v", err)
	}
	if err := workbook.SetCellStr("Report", "B3", "Actual performance from 10/5/2021 to 06/30/26"); err != nil {
		t.Fatalf("SetCellStr returned error: %v", err)
	}

	match, found, err := FindExactText(
		workbook,
		"Report",
		"Actual performance from 10/5/2021 to 06/30/26",
	)
	if err != nil {
		t.Fatalf("FindExactText returned error: %v", err)
	}
	if !found {
		t.Fatal("expected exact text to be found")
	}
	if match.Cell != "B3" {
		t.Fatalf("match.Cell = %q, want %q", match.Cell, "B3")
	}
}

func TestFindExactTextNotFound(t *testing.T) {
	workbook := excelize.NewFile()
	defaultSheet := workbook.GetSheetName(workbook.GetActiveSheetIndex())
	if err := workbook.SetSheetName(defaultSheet, "Report"); err != nil {
		t.Fatalf("SetSheetName returned error: %v", err)
	}

	_, found, err := FindExactText(workbook, "1", "Missing")
	if err != nil {
		t.Fatalf("FindExactText returned error: %v", err)
	}
	if found {
		t.Fatal("expected exact text to be absent")
	}
}

func TestFindAnchorScanValueLastNonEmptyBeforeBlank(t *testing.T) {
	workbook := excelize.NewFile()
	defaultSheet := workbook.GetSheetName(workbook.GetActiveSheetIndex())
	if err := workbook.SetSheetName(defaultSheet, "Report"); err != nil {
		t.Fatalf("SetSheetName returned error: %v", err)
	}

	values := map[string]string{
		"A1": "Reporting dates",
		"A2": "6/28/2026",
		"A3": "6/29/2026",
		"A4": "6/30/2026",
		"A6": "unrelated section",
	}
	for cell, value := range values {
		if err := workbook.SetCellStr("Report", cell, value); err != nil {
			t.Fatalf("SetCellStr returned error: %v", err)
		}
	}

	match, found, err := FindAnchorScanValue(workbook, AnchorScanOptions{
		Sheet:     "Report",
		Anchor:    "Reporting dates",
		Direction: DirectionDown,
		Select:    "last_non_empty_before_blank",
	})
	if err != nil {
		t.Fatalf("FindAnchorScanValue returned error: %v", err)
	}
	if !found {
		t.Fatal("expected anchor scan match")
	}
	if match.AnchorCell != "A1" {
		t.Fatalf("anchor cell = %q, want A1", match.AnchorCell)
	}
	if match.Cell != "A4" {
		t.Fatalf("match cell = %q, want A4", match.Cell)
	}
	if match.Value != "6/30/2026" {
		t.Fatalf("match value = %q, want 6/30/2026", match.Value)
	}
}

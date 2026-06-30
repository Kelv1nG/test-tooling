package main

import (
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/xuri/excelize/v2"

	"tooling/templates"
)

func TestRunCheckVerificationExactMatch(t *testing.T) {
	referenceDate := time.Date(2026, time.June, 30, 0, 0, 0, 0, time.UTC)
	tempDir := t.TempDir()
	workbookPath := filepath.Join(tempDir, "report_2026_06_30.xlsx")
	workbook := excelize.NewFile()
	defaultSheet := workbook.GetSheetName(workbook.GetActiveSheetIndex())
	if err := workbook.SetSheetName(defaultSheet, "Report"); err != nil {
		t.Fatalf("SetSheetName returned error: %v", err)
	}
	if err := workbook.SetCellStr("Report", "B3", "Actual performance from 10/5/2021 to 06/30/26"); err != nil {
		t.Fatalf("SetCellStr returned error: %v", err)
	}
	if err := workbook.SaveAs(workbookPath); err != nil {
		t.Fatalf("SaveAs returned error: %v", err)
	}
	if err := workbook.Close(); err != nil {
		t.Fatalf("Close returned error: %v", err)
	}

	rows, summary := runCheckVerification([]templates.CheckRowView{
		{
			Index:    1,
			ExcelRow: 2,
			ID:       "CHK-001",
			NewFile:  filepath.Join(tempDir, "report_{yyyy}_{mm}_{dd}.xlsx"),
			Rules: []templates.CheckRuleView{
				{
					Index:        1,
					ID:           "R001",
					CheckID:      "CHK-001",
					Name:         "Performance phrase",
					Type:         "exact_text",
					Enabled:      true,
					Sheet:        "Report",
					ExpectedText: "Actual performance from 10/5/2021 to {mm}/{dd}/{yy}",
				},
			},
		},
	}, referenceDate)

	if summary.Matched != 1 {
		t.Fatalf("summary.Matched = %d, want 1", summary.Matched)
	}
	if len(rows) != 1 {
		t.Fatalf("expected 1 row, got %d", len(rows))
	}
	if rows[0].Status != "Matched" {
		t.Fatalf("row status = %q, want %q", rows[0].Status, "Matched")
	}
	if rows[0].Rules[0].Status != "Matched" {
		t.Fatalf("rule status = %q, want %q", rows[0].Rules[0].Status, "Matched")
	}
	if !strings.Contains(rows[0].Rules[0].Detail, "Exact text found at B3.") {
		t.Fatalf("unexpected rule detail: %q", rows[0].Rules[0].Detail)
	}
}

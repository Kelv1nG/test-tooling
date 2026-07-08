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
			File:     filepath.Join(tempDir, "report_{yyyy}_{mm}_{dd}.xlsx"),
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
	if rows[0].ResolvedFile != workbookPath {
		t.Fatalf("resolved file = %q, want %q", rows[0].ResolvedFile, workbookPath)
	}
	if rows[0].Rules[0].Status != "Matched" {
		t.Fatalf("rule status = %q, want %q", rows[0].Rules[0].Status, "Matched")
	}
	if !strings.Contains(rows[0].Rules[0].Detail, "Exact text found at B3.") {
		t.Fatalf("unexpected rule detail: %q", rows[0].Rules[0].Detail)
	}
}

func TestRunCheckVerificationExactMatchNotFound(t *testing.T) {
	referenceDate := time.Date(2026, time.June, 30, 0, 0, 0, 0, time.UTC)
	tempDir := t.TempDir()
	workbookPath := filepath.Join(tempDir, "report_2026_06_30.xlsx")
	writeExactTextWorkbook(t, workbookPath, "Actual performance from 10/5/2021 to 05/31/26")

	rows, summary := runCheckVerification([]templates.CheckRowView{
		{
			Index:    1,
			ExcelRow: 2,
			ID:       "CHK-001",
			File:     filepath.Join(tempDir, "report_{yyyy}_{mm}_{dd}.xlsx"),
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

	if summary.Changed != 1 {
		t.Fatalf("summary.Changed = %d, want 1", summary.Changed)
	}
	if summary.Errors != 0 {
		t.Fatalf("summary.Errors = %d, want 0", summary.Errors)
	}
	if rows[0].Status != "Changed" {
		t.Fatalf("row status = %q, want Changed", rows[0].Status)
	}
	if rows[0].Rules[0].Status != "Not found" {
		t.Fatalf("rule status = %q, want Not found", rows[0].Rules[0].Status)
	}
	if rows[0].Rules[0].Badge != "amber" {
		t.Fatalf("rule badge = %q, want amber", rows[0].Rules[0].Badge)
	}
	if !strings.Contains(rows[0].Rules[0].Detail, "Exact text not found") {
		t.Fatalf("unexpected rule detail: %q", rows[0].Rules[0].Detail)
	}
	if !strings.Contains(rows[0].Detail, "Exact text not found") {
		t.Fatalf("expected card detail to include not found detail, got %q", rows[0].Detail)
	}
}

func TestRunCheckVerificationWithProgressReportsEachCompletedRow(t *testing.T) {
	referenceDate := time.Date(2026, time.June, 30, 0, 0, 0, 0, time.UTC)
	tempDir := t.TempDir()
	matchingPath := filepath.Join(tempDir, "matching.xlsx")
	changedPath := filepath.Join(tempDir, "changed.xlsx")
	writeExactTextWorkbook(t, matchingPath, "expected")
	writeExactTextWorkbook(t, changedPath, "different")

	inputRows := []templates.CheckRowView{
		{
			Index: 1,
			ID:    "CHK-001",
			File:  matchingPath,
			Rules: []templates.CheckRuleView{
				{
					Index:        1,
					ID:           "R001",
					CheckID:      "CHK-001",
					Type:         "exact_text",
					Enabled:      true,
					Sheet:        "Report",
					ExpectedText: "expected",
				},
			},
		},
		{
			Index: 2,
			ID:    "CHK-002",
			File:  changedPath,
			Rules: []templates.CheckRuleView{
				{
					Index:        1,
					ID:           "R001",
					CheckID:      "CHK-002",
					Type:         "exact_text",
					Enabled:      true,
					Sheet:        "Report",
					ExpectedText: "expected",
				},
			},
		},
	}
	var progressEvents []checkVerificationProgress

	rows, summary := runCheckVerificationWithProgress(
		inputRows,
		referenceDate,
		func(progress checkVerificationProgress) {
			progressEvents = append(progressEvents, progress)
		},
	)

	if len(progressEvents) != len(inputRows) {
		t.Fatalf("got %d progress events, want %d", len(progressEvents), len(inputRows))
	}
	if progressEvents[len(progressEvents)-1].Completed != len(inputRows) {
		t.Fatalf("last completed count = %d, want %d", progressEvents[len(progressEvents)-1].Completed, len(inputRows))
	}
	if progressEvents[len(progressEvents)-1].Total != len(inputRows) {
		t.Fatalf("last total = %d, want %d", progressEvents[len(progressEvents)-1].Total, len(inputRows))
	}
	if summary.Matched != 1 {
		t.Fatalf("summary.Matched = %d, want 1", summary.Matched)
	}
	if summary.Changed != 1 {
		t.Fatalf("summary.Changed = %d, want 1", summary.Changed)
	}
	if rows[0].ID != "CHK-001" || rows[0].Status != "Matched" {
		t.Fatalf("first row = %+v, want CHK-001 matched", rows[0])
	}
	if rows[1].ID != "CHK-002" || rows[1].Status != "Changed" {
		t.Fatalf("second row = %+v, want CHK-002 changed", rows[1])
	}

	summaryRows := buildCheckSummaryRows(rows)
	if len(summaryRows) != 2 {
		t.Fatalf("got %d summary rows, want 2", len(summaryRows))
	}
	if summaryRows[0].CurrentFile != matchingPath {
		t.Fatalf("summary current file = %q, want %q", summaryRows[0].CurrentFile, matchingPath)
	}
	if summaryRows[1].Status != "Not found" {
		t.Fatalf("summary changed status = %q, want Not found", summaryRows[1].Status)
	}
}

func TestRunCheckVerificationRejectsAmbiguousWildcardFile(t *testing.T) {
	referenceDate := time.Date(2026, time.June, 30, 0, 0, 0, 0, time.UTC)
	tempDir := t.TempDir()
	for _, name := range []string{
		"report_2026_06_first_30.xlsx",
		"report_2026_06_second_30.xlsx",
	} {
		writeExactTextWorkbook(
			t,
			filepath.Join(tempDir, name),
			"Actual performance from 10/5/2021 to 06/30/26",
		)
	}

	rows, summary := runCheckVerification([]templates.CheckRowView{
		{
			Index:    1,
			ExcelRow: 2,
			ID:       "CHK-001",
			File:     filepath.Join(tempDir, "report_{yyyy}_{mm}_*_{dd}.xlsx"),
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

	if summary.Errors != 1 {
		t.Fatalf("summary.Errors = %d, want 1", summary.Errors)
	}
	if rows[0].Rules[0].Status != "Error" {
		t.Fatalf("rule status = %q, want Error", rows[0].Rules[0].Status)
	}
	if !strings.Contains(rows[0].Rules[0].Detail, "matched 2 files") {
		t.Fatalf("expected ambiguous wildcard detail, got %q", rows[0].Rules[0].Detail)
	}
}

func TestRunCheckVerificationAnchorScanMatchDate(t *testing.T) {
	referenceDate := time.Date(2026, time.June, 30, 0, 0, 0, 0, time.UTC)
	tempDir := t.TempDir()
	workbookPath := filepath.Join(tempDir, "report_2026_06_30.xlsx")
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
			t.Fatalf("SetCellStr %s returned error: %v", cell, err)
		}
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
			File:     filepath.Join(tempDir, "report_{yyyy}_{mm}_{dd}.xlsx"),
			Rules: []templates.CheckRuleView{
				{
					Index:           1,
					ID:              "R001",
					CheckID:         "CHK-001",
					Name:            "Reporting date",
					Type:            "anchor_scan_match",
					Enabled:         true,
					Sheet:           "Report",
					Anchor:          "Reporting dates",
					ParentDirection: "down",
					ScanSelect:      "last_non_empty_before_blank",
					ExpectedText:    "{mm}/{dd}/{yy}",
					CompareAs:       "date",
				},
			},
		},
	}, referenceDate)

	if summary.Matched != 1 {
		t.Fatalf("summary.Matched = %d, want 1", summary.Matched)
	}
	if summary.Errors != 0 {
		t.Fatalf("summary.Errors = %d, want 0", summary.Errors)
	}
	if rows[0].Rules[0].Status != "Matched" {
		t.Fatalf("rule status = %q, want Matched", rows[0].Rules[0].Status)
	}
	if !strings.Contains(rows[0].Rules[0].Detail, "A4") {
		t.Fatalf("expected rule detail to include selected cell, got %q", rows[0].Rules[0].Detail)
	}
}

func TestRunCheckVerificationReportsAddedHeader(t *testing.T) {
	tempDir := t.TempDir()
	comparePath := filepath.Join(tempDir, "report_2026_04_previous_30.xlsx")
	currentPath := filepath.Join(tempDir, "report_2026_05_current_31.xlsx")

	writeHeaderSampleWorkbook(t, comparePath, false)
	writeHeaderSampleWorkbook(t, currentPath, true)

	rows, summary := runCheckVerification([]templates.CheckRowView{
		{
			Index:               1,
			ID:                  "CHK-001",
			File:                filepath.Join(tempDir, "report_{yyyy}_{mm}_*_{dd}.xlsx"),
			CompareOffsetMonths: -1,
			Rules: []templates.CheckRuleView{
				{
					Index:           1,
					ID:              "R001",
					CheckID:         "CHK-001",
					Name:            "Headers around date anchor",
					Type:            "header_compare",
					Enabled:         true,
					Sheet:           "Report",
					Anchor:          "date",
					ParentDirection: "up",
					MaxHeaderDepth:  "2",
					RequireOrder:    true,
				},
			},
		},
	}, time.Date(2026, time.May, 31, 0, 0, 0, 0, time.UTC))

	if summary.Changed != 1 {
		t.Fatalf("summary.Changed = %d, want 1", summary.Changed)
	}
	if summary.Errors != 0 {
		t.Fatalf("summary.Errors = %d, want 0", summary.Errors)
	}
	if rows[0].Rules[0].Status != "Changed" {
		t.Fatalf("rule status = %q, want Changed", rows[0].Rules[0].Status)
	}
	if rows[0].ResolvedFile != currentPath {
		t.Fatalf("resolved current file = %q, want %q", rows[0].ResolvedFile, currentPath)
	}
	if rows[0].ResolvedCompareFile != comparePath {
		t.Fatalf("resolved compare file = %q, want %q", rows[0].ResolvedCompareFile, comparePath)
	}
	if !strings.Contains(rows[0].Rules[0].Detail, "++ column C") {
		t.Fatalf("unexpected rule detail: %q", rows[0].Rules[0].Detail)
	}
	if strings.Contains(rows[0].Rules[0].Detail, "> value") {
		t.Fatalf("rule detail should not expose the internal header path: %q", rows[0].Rules[0].Detail)
	}
	if !strings.Contains(rows[0].Detail, "++ column C") {
		t.Fatalf("expected card detail to include added header, got %q", rows[0].Detail)
	}
}

func TestRunCheckVerificationReportsRemovedHeader(t *testing.T) {
	tempDir := t.TempDir()
	comparePath := filepath.Join(tempDir, "report_2026_04_30.xlsx")
	currentPath := filepath.Join(tempDir, "report_2026_05_31.xlsx")

	writeHeaderSampleWorkbook(t, comparePath, true)
	writeHeaderSampleWorkbook(t, currentPath, false)

	rows, summary := runCheckVerification([]templates.CheckRowView{
		{
			Index:               1,
			ID:                  "CHK-001",
			File:                filepath.Join(tempDir, "report_{yyyy}_{mm}_{dd}.xlsx"),
			CompareOffsetMonths: -1,
			Rules: []templates.CheckRuleView{
				{
					Index:           1,
					ID:              "R001",
					CheckID:         "CHK-001",
					Name:            "Headers around date anchor",
					Type:            "header_compare",
					Enabled:         true,
					Sheet:           "Report",
					Anchor:          "date",
					ParentDirection: "up",
					MaxHeaderDepth:  "2",
					RequireOrder:    true,
				},
			},
		},
	}, time.Date(2026, time.May, 31, 0, 0, 0, 0, time.UTC))

	if summary.Changed != 1 {
		t.Fatalf("summary.Changed = %d, want 1", summary.Changed)
	}
	if summary.Errors != 0 {
		t.Fatalf("summary.Errors = %d, want 0", summary.Errors)
	}
	if rows[0].Rules[0].Status != "Changed" {
		t.Fatalf("rule status = %q, want Changed", rows[0].Rules[0].Status)
	}
	if !strings.Contains(rows[0].Rules[0].Detail, "-- column C") {
		t.Fatalf("unexpected rule detail: %q", rows[0].Rules[0].Detail)
	}
	if strings.Contains(rows[0].Rules[0].Detail, "> value") {
		t.Fatalf("rule detail should not expose the internal header path: %q", rows[0].Rules[0].Detail)
	}
	if !strings.Contains(rows[0].Detail, "-- column C") {
		t.Fatalf("expected card detail to include removed header, got %q", rows[0].Detail)
	}
}

func TestRunCheckVerificationIgnoresOrderWhenHeaderInsertedBeforeExistingHeader(t *testing.T) {
	tempDir := t.TempDir()
	comparePath := filepath.Join(tempDir, "report_2026_04_30.xlsx")
	currentPath := filepath.Join(tempDir, "report_2026_05_31.xlsx")

	writeReturnHeaderWorkbook(t, comparePath, true)
	writeReturnHeaderWorkbook(t, currentPath, false)

	rows, summary := runCheckVerification([]templates.CheckRowView{
		{
			Index:               1,
			ID:                  "CHK-001",
			File:                filepath.Join(tempDir, "report_{yyyy}_{mm}_{dd}.xlsx"),
			CompareOffsetMonths: -1,
			Rules: []templates.CheckRuleView{
				{
					Index:           1,
					ID:              "R001",
					CheckID:         "CHK-001",
					Name:            "Return headers",
					Type:            "header_compare",
					Enabled:         true,
					Sheet:           "Report",
					Anchor:          "Anchor",
					ParentDirection: "up",
					MaxHeaderDepth:  "1",
					RequireOrder:    false,
				},
			},
		},
	}, time.Date(2026, time.May, 31, 0, 0, 0, 0, time.UTC))

	if summary.Changed != 1 {
		t.Fatalf("summary.Changed = %d, want 1", summary.Changed)
	}
	if summary.Errors != 0 {
		t.Fatalf("summary.Errors = %d, want 0", summary.Errors)
	}

	detail := rows[0].Rules[0].Detail
	if !strings.Contains(detail, "-- Net Return 3") {
		t.Fatalf("expected missing inserted header, got %q", detail)
	}
	for _, unexpected := range []string{
		"-- Net Return 2",
		"++ Net Return 2",
		"column order changed",
	} {
		if strings.Contains(detail, unexpected) {
			t.Fatalf("detail should not contain %q: %q", unexpected, detail)
		}
	}
}

func writeExactTextWorkbook(t *testing.T, path string, value string) {
	t.Helper()

	workbook := excelize.NewFile()
	defaultSheet := workbook.GetSheetName(workbook.GetActiveSheetIndex())
	if err := workbook.SetSheetName(defaultSheet, "Report"); err != nil {
		t.Fatalf("SetSheetName returned error: %v", err)
	}
	if err := workbook.SetCellStr("Report", "B3", value); err != nil {
		t.Fatalf("SetCellStr returned error: %v", err)
	}
	if err := workbook.SaveAs(path); err != nil {
		t.Fatalf("SaveAs returned error: %v", err)
	}
	if err := workbook.Close(); err != nil {
		t.Fatalf("Close returned error: %v", err)
	}
}

func writeReturnHeaderWorkbook(t *testing.T, path string, includeExtraReturn bool) {
	t.Helper()

	workbook := excelize.NewFile()
	defaultSheet := workbook.GetSheetName(workbook.GetActiveSheetIndex())
	if err := workbook.SetSheetName(defaultSheet, "Report"); err != nil {
		t.Fatalf("SetSheetName returned error: %v", err)
	}

	values := map[string]string{
		"A1": "Gross Return",
		"B1": "Net Return 1",
		"C1": "Net Return 2",
		"A3": "Anchor",
		"B3": "Value 1",
		"C3": "Value 2",
	}
	if includeExtraReturn {
		values["D1"] = "Net Return 3"
		values["E1"] = "Some Header"
		values["D3"] = "Value 3"
		values["E3"] = "Some Value"
	} else {
		values["D1"] = "Some Header"
		values["D3"] = "Some Value"
	}

	for cell, value := range values {
		if err := workbook.SetCellStr("Report", cell, value); err != nil {
			t.Fatalf("SetCellStr %s returned error: %v", cell, err)
		}
	}
	if err := workbook.SaveAs(path); err != nil {
		t.Fatalf("SaveAs returned error: %v", err)
	}
	if err := workbook.Close(); err != nil {
		t.Fatalf("Close returned error: %v", err)
	}
}

func writeHeaderSampleWorkbook(t *testing.T, path string, includeAddedColumn bool) {
	t.Helper()

	workbook := excelize.NewFile()
	defaultSheet := workbook.GetSheetName(workbook.GetActiveSheetIndex())
	if err := workbook.SetSheetName(defaultSheet, "Report"); err != nil {
		t.Fatalf("SetSheetName returned error: %v", err)
	}

	values := map[string]string{
		"B3": "column A",
		"C3": "column B",
		"A4": "date",
		"B4": "value",
		"C4": "value",
		"A5": "date2",
		"B5": "value2",
		"C5": "value3",
	}
	if includeAddedColumn {
		values["D3"] = "column C"
		values["D4"] = "value"
		values["D5"] = "value4"
	}

	for cell, value := range values {
		if err := workbook.SetCellStr("Report", cell, value); err != nil {
			t.Fatalf("SetCellStr %s returned error: %v", cell, err)
		}
	}
	if err := workbook.SaveAs(path); err != nil {
		t.Fatalf("SaveAs returned error: %v", err)
	}
	if err := workbook.Close(); err != nil {
		t.Fatalf("Close returned error: %v", err)
	}
}

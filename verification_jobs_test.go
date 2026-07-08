package main

import (
	"path/filepath"
	"strings"
	"testing"
	"time"

	"tooling/templates"
)

func TestVerificationJobPageDataTracksRunningAndCompletedState(t *testing.T) {
	base := templates.PageData{
		ActiveTab:          tabChecking,
		CheckReferenceDate: "2026-06-30",
		HasConfig:          true,
	}
	rows := []templates.CheckRowView{
		{Index: 1, ID: "CHK-001"},
		{Index: 2, ID: "CHK-002"},
	}

	job := newVerificationJob("job-1", base, rows)
	data := job.pageData()
	if !data.CheckRunRunning {
		t.Fatal("expected new job to be running")
	}
	if data.CheckRunCompleted != 0 {
		t.Fatalf("CheckRunCompleted = %d, want 0", data.CheckRunCompleted)
	}
	if data.CheckRunTotal != 2 {
		t.Fatalf("CheckRunTotal = %d, want 2", data.CheckRunTotal)
	}

	summary := templates.CheckSummaryView{
		Attempted: 2,
		Matched:   1,
		Changed:   1,
		HasRun:    true,
	}
	rows[0].Status = "Matched"
	rows[1].Status = "Changed"
	job.finish(rows, summary)

	data = job.pageData()
	if data.CheckRunRunning {
		t.Fatal("expected finished job not to be running")
	}
	if data.CheckRunCompleted != 2 {
		t.Fatalf("CheckRunCompleted = %d, want 2", data.CheckRunCompleted)
	}
	if !data.CheckHasIssues {
		t.Fatal("expected changed summary to mark check issues")
	}
	if !strings.Contains(data.CheckMessage, "matched 1, changed 1") {
		t.Fatalf("unexpected completed message: %q", data.CheckMessage)
	}
}

func TestStartVerificationJobCompletesWithFinalRows(t *testing.T) {
	referenceDate := time.Date(2026, time.June, 30, 0, 0, 0, 0, time.UTC)
	tempDir := t.TempDir()
	workbookPath := filepath.Join(tempDir, "report.xlsx")
	writeExactTextWorkbook(t, workbookPath, "expected")

	app := mustNewApplication(":0", "", "", nil)
	base := templates.PageData{
		ActiveTab:          tabChecking,
		CheckReferenceDate: "2026-06-30",
		HasConfig:          true,
	}
	rows := []templates.CheckRowView{
		{
			Index: 1,
			ID:    "CHK-001",
			File:  workbookPath,
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
	}

	job := app.startVerificationJob(base, rows, referenceDate)
	data := waitForVerificationJobDone(t, job)

	if data.CheckRunCompleted != 1 {
		t.Fatalf("CheckRunCompleted = %d, want 1", data.CheckRunCompleted)
	}
	if data.CheckSummary.Matched != 1 {
		t.Fatalf("CheckSummary.Matched = %d, want 1", data.CheckSummary.Matched)
	}
	if len(data.CheckRows) != 1 || data.CheckRows[0].Status != "Matched" {
		t.Fatalf("unexpected final rows: %+v", data.CheckRows)
	}
}

func waitForVerificationJobDone(
	t *testing.T,
	job *verificationJob,
) templates.PageData {
	t.Helper()

	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		data := job.pageData()
		if !data.CheckRunRunning {
			return data
		}
		time.Sleep(10 * time.Millisecond)
	}

	t.Fatal("verification job did not finish before deadline")
	return templates.PageData{}
}

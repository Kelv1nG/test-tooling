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

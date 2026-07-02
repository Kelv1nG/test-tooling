package main

import (
	"strings"
	"testing"
	"time"
)

func TestConfigurationExampleLoads(t *testing.T) {
	app := mustNewApplication(
		":0",
		"table-definitions.yaml",
		"configuration-example.xlsx",
		nil,
	)

	configuration, err := app.loadConfiguration(
		"table-definitions.yaml",
		"configuration-example.xlsx",
	)
	if err != nil {
		t.Fatalf("loadConfiguration returned error: %v", err)
	}

	if len(configuration.FileCheckConfigs) == 0 {
		t.Fatal("expected configuration example to include file-check configs")
	}

	rows := buildCheckRows(
		configuration.FileCheckConfigs,
		time.Date(2026, time.May, 31, 0, 0, 0, 0, time.UTC),
	)
	rows, summary := runCheckVerification(
		rows,
		time.Date(2026, time.May, 31, 0, 0, 0, 0, time.UTC),
	)
	if summary.Errors > 0 {
		t.Fatalf("expected sample verification to avoid errors, summary: %+v rows: %+v", summary, rows)
	}
	if summary.Matched != 2 {
		t.Fatalf("expected 2 matched sample rules, got %d", summary.Matched)
	}
	if summary.Changed != 1 {
		t.Fatalf("expected 1 changed sample rule, got %d", summary.Changed)
	}
	if len(rows) != 1 || len(rows[0].Rules) != 3 {
		t.Fatalf("expected 1 sample check with 3 rules, got rows: %+v", rows)
	}
	headerRule := rows[0].Rules[1]
	if headerRule.Status != "Changed" {
		t.Fatalf("expected sample header rule to be changed, got %q", headerRule.Status)
	}
	if !strings.Contains(headerRule.Detail, "++ Performance > column C") {
		t.Fatalf("expected sample header rule to report an addition, got %q", headerRule.Detail)
	}
	scanRule := rows[0].Rules[2]
	if scanRule.Status != "Matched" {
		t.Fatalf("expected sample anchor scan rule to match, got %q", scanRule.Status)
	}
	if strings.Contains(headerRule.Detail, "> value") {
		t.Fatalf("sample header rule should not expose the internal header path: %q", headerRule.Detail)
	}
	if strings.Contains(headerRule.Detail, "columnD") {
		t.Fatalf("sample header rule should not report stray worksheet columns: %q", headerRule.Detail)
	}
	if !strings.Contains(rows[0].Detail, "++ Performance > column C") {
		t.Fatalf("expected sample check card to report an addition, got %q", rows[0].Detail)
	}
	if strings.Contains(rows[0].Detail, "columnD") {
		t.Fatalf("sample check card should not report stray worksheet columns: %q", rows[0].Detail)
	}
}

package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestResolvePathTemplate(t *testing.T) {
	referenceDate := time.Date(2026, time.February, 3, 0, 0, 0, 0, time.UTC)

	resolved, err := ResolvePathTemplate(
		`/reports/{yyyy}/{yy}/{ mm }/{ m }/{dd }/{d}/{MMMM}/{mmmm}/{ MMM }/{ mmm }`,
		referenceDate,
	)
	if err != nil {
		t.Fatalf("ResolvePathTemplate returned error: %v", err)
	}

	const expected = "/reports/2026/26/02/2/03/3/February/february/Feb/feb"
	if resolved != expected {
		t.Fatalf("ResolvePathTemplate = %q, want %q", resolved, expected)
	}
}

func TestResolvePathTemplateRejectsUnsupportedPlaceholder(t *testing.T) {
	referenceDate := time.Date(2026, time.June, 29, 0, 0, 0, 0, time.UTC)

	_, err := ResolvePathTemplate(`/reports/{offset}`, referenceDate)
	if err == nil {
		t.Fatal("ResolvePathTemplate returned nil error for unsupported placeholder")
	}
}

func TestResolvePathTemplateSingleMatchResolvesWildcard(t *testing.T) {
	referenceDate := time.Date(2026, time.May, 31, 0, 0, 0, 0, time.UTC)
	tempDir := t.TempDir()
	expectedPath := filepath.Join(tempDir, "file_05_ignore_me_2026.xlsx")
	if err := os.WriteFile(expectedPath, []byte("sample"), 0o644); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}

	resolved, err := ResolvePathTemplateSingleMatch(
		filepath.Join(tempDir, "file_{mm}_*_{yyyy}.xlsx"),
		referenceDate,
	)
	if err != nil {
		t.Fatalf("ResolvePathTemplateSingleMatch returned error: %v", err)
	}
	if resolved != expectedPath {
		t.Fatalf("ResolvePathTemplateSingleMatch = %q, want %q", resolved, expectedPath)
	}
}

func TestResolvePathTemplateSingleMatchReportsNoWildcardMatches(t *testing.T) {
	referenceDate := time.Date(2026, time.May, 31, 0, 0, 0, 0, time.UTC)
	tempDir := t.TempDir()

	_, err := ResolvePathTemplateSingleMatch(
		filepath.Join(tempDir, "file_{mm}_*_{yyyy}.xlsx"),
		referenceDate,
	)
	if err == nil {
		t.Fatal("ResolvePathTemplateSingleMatch returned nil error")
	}
	if !IsPathPatternNoMatch(err) {
		t.Fatalf("expected no-match path pattern error, got %v", err)
	}
}

func TestResolvePathTemplateSingleMatchRejectsAmbiguousWildcard(t *testing.T) {
	referenceDate := time.Date(2026, time.May, 31, 0, 0, 0, 0, time.UTC)
	tempDir := t.TempDir()
	for _, name := range []string{
		"file_05_first_2026.xlsx",
		"file_05_second_2026.xlsx",
	} {
		if err := os.WriteFile(filepath.Join(tempDir, name), []byte("sample"), 0o644); err != nil {
			t.Fatalf("WriteFile %s returned error: %v", name, err)
		}
	}

	_, err := ResolvePathTemplateSingleMatch(
		filepath.Join(tempDir, "file_{mm}_*_{yyyy}.xlsx"),
		referenceDate,
	)
	if err == nil {
		t.Fatal("ResolvePathTemplateSingleMatch returned nil error")
	}
	if !strings.Contains(err.Error(), "matched 2 files") {
		t.Fatalf("expected ambiguous wildcard error, got %v", err)
	}
}

func TestResolveTemplateText(t *testing.T) {
	referenceDate := time.Date(2026, time.June, 30, 0, 0, 0, 0, time.UTC)

	resolved, err := ResolveTemplateText(
		`Actual performance from 10/5/2021 to {mm}/{dd}/{yy}`,
		referenceDate,
	)
	if err != nil {
		t.Fatalf("ResolveTemplateText returned error: %v", err)
	}

	const expected = "Actual performance from 10/5/2021 to 06/30/26"
	if resolved != expected {
		t.Fatalf("ResolveTemplateText = %q, want %q", resolved, expected)
	}
}

package config

import (
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

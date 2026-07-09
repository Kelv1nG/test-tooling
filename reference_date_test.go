package main

import (
	"testing"
	"time"
)

func TestDefaultReferenceDateUsesPreviousMonth(t *testing.T) {
	now := time.Date(2026, time.July, 9, 12, 30, 0, 0, time.Local)

	if got, want := defaultReferenceDateFor(now), "2026-06-09"; got != want {
		t.Fatalf("defaultReferenceDateFor() = %q, want %q", got, want)
	}
}

func TestDefaultReferenceDateClampsToPreviousMonthEnd(t *testing.T) {
	now := time.Date(2026, time.March, 31, 12, 30, 0, 0, time.Local)

	if got, want := defaultReferenceDateFor(now), "2026-02-28"; got != want {
		t.Fatalf("defaultReferenceDateFor() = %q, want %q", got, want)
	}
}

package main

import (
	"testing"
	"time"
)

func TestParseTransferMode(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  transferMode
	}{
		{name: "default", input: "", want: transferModeOverwrite},
		{name: "skip", input: "skip", want: transferModeSkip},
		{name: "skip case insensitive", input: " SKIP ", want: transferModeSkip},
		{name: "unknown falls back", input: "anything", want: transferModeOverwrite},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := parseTransferMode(test.input); got != test.want {
				t.Fatalf("parseTransferMode(%q) = %q, want %q", test.input, got, test.want)
			}
		})
	}
}

func TestNewTransferRunnerUsesSelectedConflictStrategy(t *testing.T) {
	referenceDate := time.Date(2026, time.June, 29, 0, 0, 0, 0, time.UTC)

	runner := newTransferRunner(transferModeSkip, referenceDate)
	if runner.conflictStrategy != conflictStrategySkip {
		t.Fatalf("expected skip conflict strategy, got %v", runner.conflictStrategy)
	}

	runner = newTransferRunner(transferModeOverwrite, referenceDate)
	if runner.conflictStrategy != conflictStrategyOverwrite {
		t.Fatalf("expected overwrite conflict strategy, got %v", runner.conflictStrategy)
	}
}

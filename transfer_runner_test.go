package main

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"tooling/config"
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

func TestTransferRunnerRunCopiesMappingsAndPreservesOrder(t *testing.T) {
	tempDir := t.TempDir()
	mappings := make([]config.FileTransferMap, 8)

	for index := range mappings {
		src := filepath.Join(tempDir, fmt.Sprintf("src-%02d.txt", index+1))
		dest := filepath.Join(tempDir, "out", fmt.Sprintf("dest-%02d.txt", index+1))
		contents := fmt.Sprintf("file %02d", index+1)
		if err := os.WriteFile(src, []byte(contents), 0o644); err != nil {
			t.Fatalf("WriteFile source returned error: %v", err)
		}

		mappings[index] = config.FileTransferMap{
			Src:  src,
			Dest: dest,
		}
	}

	runner := newTransferRunner(
		transferModeOverwrite,
		time.Date(2026, time.June, 29, 0, 0, 0, 0, time.UTC),
	)
	results, summary := runner.run(mappings)

	if summary.Attempted != len(mappings) {
		t.Fatalf("summary.Attempted = %d, want %d", summary.Attempted, len(mappings))
	}
	if summary.Created != len(mappings) {
		t.Fatalf("summary.Created = %d, want %d", summary.Created, len(mappings))
	}
	if summary.Errors != 0 {
		t.Fatalf("summary.Errors = %d, want 0", summary.Errors)
	}
	if len(results) != len(mappings) {
		t.Fatalf("got %d results, want %d", len(results), len(mappings))
	}

	for index, result := range results {
		if result.Index != index+1 {
			t.Fatalf("result %d has Index %d, want %d", index, result.Index, index+1)
		}
		if result.Status != "Created" {
			t.Fatalf("result %d status = %q, want Created", index, result.Status)
		}
		if result.ResolvedSrc != mappings[index].Src {
			t.Fatalf("result %d ResolvedSrc = %q, want %q", index, result.ResolvedSrc, mappings[index].Src)
		}
		if result.ResolvedDest != mappings[index].Dest {
			t.Fatalf("result %d ResolvedDest = %q, want %q", index, result.ResolvedDest, mappings[index].Dest)
		}

		contents, err := os.ReadFile(mappings[index].Dest)
		if err != nil {
			t.Fatalf("ReadFile destination returned error: %v", err)
		}
		want := fmt.Sprintf("file %02d", index+1)
		if string(contents) != want {
			t.Fatalf("destination contents = %q, want %q", string(contents), want)
		}
	}

	summaryRows := buildTransferSummaryRows(results)
	if len(summaryRows) != len(results) {
		t.Fatalf("got %d summary rows, want %d", len(summaryRows), len(results))
	}
	if summaryRows[0].Source != mappings[0].Src {
		t.Fatalf("summary source = %q, want %q", summaryRows[0].Source, mappings[0].Src)
	}
	if summaryRows[0].Destination != mappings[0].Dest {
		t.Fatalf("summary destination = %q, want %q", summaryRows[0].Destination, mappings[0].Dest)
	}
}

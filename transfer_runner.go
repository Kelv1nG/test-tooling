package main

import (
	"strings"
	"sync"
	"time"

	"tooling/config"
	"tooling/templates"
)

const maxConcurrentTransfers = 5

type transferMode string

const (
	transferModeOverwrite transferMode = "overwrite"
	transferModeSkip      transferMode = "skip"

	defaultTransferMode = transferModeOverwrite
)

type transferRunner struct {
	referenceDate    time.Time
	conflictStrategy conflictStrategy
}

func newTransferRunner(
	mode transferMode,
	referenceDate time.Time,
) transferRunner {
	return transferRunner{
		referenceDate:    referenceDate,
		conflictStrategy: mode.conflictStrategy(),
	}
}

func (r transferRunner) run(
	mappings []config.FileTransferMap,
) ([]templates.TransferResultView, templates.TransferSummaryView) {
	runResults := make([]transferRunResult, len(mappings))
	summary := templates.TransferSummaryView{
		Attempted: len(mappings),
		HasRun:    true,
	}
	sem := make(chan struct{}, maxConcurrentTransfers)
	var wg sync.WaitGroup

	for index, mapping := range mappings {
		wg.Add(1)
		sem <- struct{}{}
		go func() {
			defer wg.Done()
			defer func() {
				<-sem
			}()

			runResults[index] = r.runOne(index, mapping)
		}()
	}
	wg.Wait()

	results := make([]templates.TransferResultView, len(runResults))
	for index, result := range runResults {
		results[index] = result.view

		if result.failed {
			summary.Errors++
			continue
		}

		switch result.outcome {
		case copyOutcomeCreated:
			summary.Created++
		case copyOutcomeOverwritten:
			summary.Overwritten++
		case copyOutcomeSkipped:
			summary.Skipped++
		default:
			summary.Errors++
		}
	}

	return results, summary
}

type transferRunResult struct {
	view    templates.TransferResultView
	outcome copyOutcome
	failed  bool
}

func (r transferRunner) runOne(
	index int,
	mapping config.FileTransferMap,
) transferRunResult {
	entry := templates.TransferResultView{
		Index: index + 1,
		Src:   mapping.Src,
		Dest:  mapping.Dest,
	}

	resolved, err := config.ResolveFileTransferMap(mapping, r.referenceDate)
	if err != nil {
		return transferErrorResult(entry, err)
	}
	entry.ResolvedSrc = resolved.Src
	entry.ResolvedDest = resolved.Dest

	outcome, err := copyFile(
		resolved.Src,
		resolved.Dest,
		r.conflictStrategy,
	)
	if err != nil {
		return transferErrorResult(entry, err)
	}

	entry.Status, entry.Badge, entry.Detail = outcome.presentation()
	return transferRunResult{
		view:    entry,
		outcome: outcome,
	}
}

func parseTransferMode(value string) transferMode {
	if strings.EqualFold(strings.TrimSpace(value), string(transferModeSkip)) {
		return transferModeSkip
	}

	return defaultTransferMode
}

func (m transferMode) conflictStrategy() conflictStrategy {
	if m == transferModeSkip {
		return conflictStrategySkip
	}

	return conflictStrategyOverwrite
}

func transferErrorResult(
	entry templates.TransferResultView,
	err error,
) transferRunResult {
	entry.Status = "Error"
	entry.Badge = "rose"
	entry.Detail = err.Error()
	return transferRunResult{
		view:   entry,
		failed: true,
	}
}

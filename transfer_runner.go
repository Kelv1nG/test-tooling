package main

import (
	"strings"
	"time"

	"tooling/config"
	"tooling/templates"
)

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
	results := make([]templates.TransferResultView, 0, len(mappings))
	summary := templates.TransferSummaryView{
		Attempted: len(mappings),
		HasRun:    true,
	}

	for index, mapping := range mappings {
		entry := templates.TransferResultView{
			Index: index + 1,
			Src:   mapping.Src,
			Dest:  mapping.Dest,
		}

		resolved, err := config.ResolveFileTransferMap(mapping, r.referenceDate)
		if err != nil {
			appendTransferError(&summary, &results, entry, err)
			continue
		}

		outcome, err := copyFile(
			resolved.Src,
			resolved.Dest,
			r.conflictStrategy,
		)
		if err != nil {
			appendTransferError(&summary, &results, entry, err)
			continue
		}

		entry.Status, entry.Badge, entry.Detail = outcome.presentation()
		switch outcome {
		case copyOutcomeCreated:
			summary.Created++
		case copyOutcomeOverwritten:
			summary.Overwritten++
		case copyOutcomeSkipped:
			summary.Skipped++
		default:
			summary.Errors++
		}

		results = append(results, entry)
	}

	return results, summary
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

func appendTransferError(
	summary *templates.TransferSummaryView,
	results *[]templates.TransferResultView,
	entry templates.TransferResultView,
	err error,
) {
	entry.Status = "Error"
	entry.Badge = "rose"
	entry.Detail = err.Error()
	summary.Errors++
	*results = append(*results, entry)
}

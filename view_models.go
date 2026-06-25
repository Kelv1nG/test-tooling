package main

import (
	"strings"

	"tooling/config"
	"tooling/templates"
)

func applyTransferResultsToRows(
	rows []templates.TransferRowView,
	results []templates.TransferResultView,
) []templates.TransferRowView {
	return templates.ApplyTransferResultsToRows(rows, results)
}

func runTransfers(
	configuration config.Configuration,
	strategy string,
) ([]templates.TransferResultView, templates.TransferSummaryView) {
	results := make([]templates.TransferResultView, 0, len(configuration.FileTransferMaps))
	var conflictStrategy FileConflictStrategy = OVERWRITE
	summary := templates.TransferSummaryView{
		Attempted: len(configuration.FileTransferMaps),
		HasRun:    true,
	}

	if strategy == "skip" {
		conflictStrategy = SKIP
	}

	for index, mapping := range configuration.FileTransferMaps {
		entry := templates.TransferResultView{
			Index: index + 1,
			Src:   mapping.Src,
			Dest:  mapping.Dest,
		}

		result, err := copyFile(mapping.Src, mapping.Dest, conflictStrategy)
		if err != nil {
			entry.Status = "Error"
			entry.Badge = "rose"
			entry.Detail = err.Error()
			summary.Errors++
			results = append(results, entry)
			continue
		}

		switch result {
		case CopyResultCreated:
			entry.Status = "Created"
			entry.Badge = "emerald"
			entry.Detail = "Destination file created."
			summary.Created++
		case CopyResultOverwritten:
			entry.Status = "Overwritten"
			entry.Badge = "amber"
			entry.Detail = "Existing destination file replaced."
			summary.Overwritten++
		case CopyResultSkipped:
			entry.Status = "Skipped"
			entry.Badge = "slate"
			entry.Detail = "Destination already existed."
			summary.Skipped++
		default:
			entry.Status = "Unknown"
			entry.Badge = "zinc"
			entry.Detail = "Copy result did not map to a known status."
			summary.Errors++
		}

		results = append(results, entry)
	}

	return results, summary
}

func normalizeStrategy(value string) string {
	if strings.EqualFold(strings.TrimSpace(value), "skip") {
		return "skip"
	}

	return "overwrite"
}

func fileExistsOrFalse(path string) bool {
	exists, err := fileExists(path)
	return err == nil && exists
}

func normalizeTab(value string) string {
	switch strings.TrimSpace(value) {
	case "file-transfer":
		return "file-transfer"
	case "checking":
		return "checking"
	default:
		return "configuration"
	}
}

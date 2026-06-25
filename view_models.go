package main

import (
	"strings"

	"tooling/config"
)

type pageData struct {
	ListenAddr        string
	DefinitionsPath   string
	WorkbookPath      string
	LoadedAt          string
	ActiveTab         string
	HasConfig         bool
	LoadError         string
	SaveMessage       string
	SaveHasErrors     bool
	TransferCount     int
	CheckCount        int
	TransferRows      []transferRowView
	CheckRows         []checkRowView
	Strategy          string
	TransferMessage   string
	TransferHasErrors bool
	TransferSummary   transferSummaryView
	TransferResults   []transferResultView
}

type transferRowView struct {
	Index      int
	ExcelRow   int
	Src        string
	SrcExists  bool
	Dest       string
	DestExists bool
	Status     string
	Badge      string
	Detail     string
}

type checkRowView struct {
	Index     int
	ExcelRow  int
	NewFile   string
	NewExists bool
	OldFile   string
	OldExists bool
}

type transferResultView struct {
	Index  int
	Src    string
	Dest   string
	Status string
	Badge  string
	Detail string
}

type transferSummaryView struct {
	Attempted   int
	Created     int
	Overwritten int
	Skipped     int
	Errors      int
	HasRun      bool
}

func buildTransferRows(
	maps []config.FileTransferMap,
) []transferRowView {
	rows := make([]transferRowView, 0, len(maps))

	for index, mapping := range maps {
		rows = append(rows, transferRowView{
			Index:      index + 1,
			ExcelRow:   mapping.ExcelRow,
			Src:        mapping.Src,
			SrcExists:  fileExistsOrFalse(mapping.Src),
			Dest:       mapping.Dest,
			DestExists: fileExistsOrFalse(mapping.Dest),
		})
	}

	return rows
}

func applyTransferResultsToRows(
	rows []transferRowView,
	results []transferResultView,
) []transferRowView {
	if len(rows) == 0 || len(results) == 0 {
		return rows
	}

	resultsByIndex := make(map[int]transferResultView, len(results))
	for _, result := range results {
		resultsByIndex[result.Index] = result
	}

	for index := range rows {
		result, exists := resultsByIndex[rows[index].Index]
		if !exists {
			continue
		}

		rows[index].Status = result.Status
		rows[index].Badge = result.Badge
		rows[index].Detail = result.Detail
	}

	return rows
}

func buildCheckRows(
	rules []config.FileCheckRule,
) []checkRowView {
	rows := make([]checkRowView, 0, len(rules))

	for index, rule := range rules {
		rows = append(rows, checkRowView{
			Index:     index + 1,
			ExcelRow:  rule.ExcelRow,
			NewFile:   rule.NewFile,
			NewExists: fileExistsOrFalse(rule.NewFile),
			OldFile:   rule.OldFile,
			OldExists: fileExistsOrFalse(rule.OldFile),
		})
	}

	return rows
}

func runTransfers(
	configuration config.Configuration,
	strategy string,
) ([]transferResultView, transferSummaryView) {
	results := make([]transferResultView, 0, len(configuration.FileTransferMaps))
	var conflictStrategy FileConflictStrategy = OVERWRITE
	summary := transferSummaryView{
		Attempted: len(configuration.FileTransferMaps),
		HasRun:    true,
	}

	if strategy == "skip" {
		conflictStrategy = SKIP
	}

	for index, mapping := range configuration.FileTransferMaps {
		entry := transferResultView{
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

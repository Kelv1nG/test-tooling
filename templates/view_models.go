package templates

import (
	"os"
	"time"

	"tooling/config"
)

type PageData struct {
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
	TransferRows      []TransferRowView
	CheckRows         []CheckRowView
	Strategy          string
	ReferenceDate     string
	TransferMessage   string
	TransferHasErrors bool
	TransferSummary   TransferSummaryView
	TransferResults   []TransferResultView
}

type TransferRowView struct {
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

type CheckRowView struct {
	Index     int
	ExcelRow  int
	NewFile   string
	NewExists bool
	OldFile   string
	OldExists bool
}

type TransferResultView struct {
	Index  int
	Src    string
	Dest   string
	Status string
	Badge  string
	Detail string
}

type TransferSummaryView struct {
	Attempted   int
	Created     int
	Overwritten int
	Skipped     int
	Errors      int
	HasRun      bool
}

func BuildTransferRows(
	maps []config.FileTransferMap,
	referenceDate time.Time,
) []TransferRowView {
	rows := make([]TransferRowView, 0, len(maps))

	for index, mapping := range maps {
		row := TransferRowView{
			Index:    index + 1,
			ExcelRow: mapping.ExcelRow,
			Src:      mapping.Src,
			Dest:     mapping.Dest,
		}

		resolved, err := config.ResolveFileTransferMap(mapping, referenceDate)
		if err != nil {
			row.Status = "Invalid template"
			row.Badge = "rose"
			row.Detail = err.Error()
			rows = append(rows, row)
			continue
		}

		row.SrcExists = fileExistsOrFalse(resolved.Src)
		row.DestExists = fileExistsOrFalse(resolved.Dest)
		rows = append(rows, row)
	}

	return rows
}

func ApplyTransferResultsToRows(
	rows []TransferRowView,
	results []TransferResultView,
) []TransferRowView {
	if len(rows) == 0 || len(results) == 0 {
		return rows
	}

	resultsByIndex := make(map[int]TransferResultView, len(results))
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

func BuildCheckRows(
	rules []config.FileCheckRule,
) []CheckRowView {
	rows := make([]CheckRowView, 0, len(rules))

	for index, rule := range rules {
		rows = append(rows, CheckRowView{
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

func fileExistsOrFalse(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

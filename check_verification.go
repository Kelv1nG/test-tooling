package main

import (
	"fmt"
	"strings"

	"github.com/xuri/excelize/v2"

	"tooling/headersearch"
	"tooling/templates"
)

func runCheckVerification(
	rows []templates.CheckRowView,
) ([]templates.CheckRowView, templates.CheckSummaryView) {
	summary := templates.CheckSummaryView{HasRun: true}

	for index := range rows {
		row := &rows[index]

		if !rowHasVerificationConfig(*row) {
			row.Status = "Not configured"
			row.Badge = "slate"
			row.Detail = "Add sheet name or index, anchor, direction, and max depth to compare headers."
			summary.Skipped++
			continue
		}

		summary.Attempted++

		options, err := extractOptionsFromRow(*row)
		if err != nil {
			row.Status = "Error"
			row.Badge = "rose"
			row.Detail = err.Error()
			summary.Errors++
			continue
		}

		oldWorkbook, err := excelize.OpenFile(row.OldFile)
		if err != nil {
			row.Status = "Error"
			row.Badge = "rose"
			row.Detail = fmt.Sprintf("open old file: %v", err)
			summary.Errors++
			continue
		}

		oldHeaders, err := headersearch.ExtractHeaders(oldWorkbook, options)
		_ = oldWorkbook.Close()
		if err != nil {
			row.Status = "Error"
			row.Badge = "rose"
			row.Detail = fmt.Sprintf("old file extraction failed: %v", err)
			summary.Errors++
			continue
		}

		newWorkbook, err := excelize.OpenFile(row.NewFile)
		if err != nil {
			row.Status = "Error"
			row.Badge = "rose"
			row.Detail = fmt.Sprintf("open new file: %v", err)
			summary.Errors++
			continue
		}

		newHeaders, err := headersearch.ExtractHeaders(newWorkbook, options)
		_ = newWorkbook.Close()
		if err != nil {
			row.Status = "Error"
			row.Badge = "rose"
			row.Detail = fmt.Sprintf("new file extraction failed: %v", err)
			summary.Errors++
			continue
		}

		result := headersearch.CompareHeaders(
			oldHeaders,
			newHeaders,
			headersearch.CompareOptions{RequireOrder: row.RequireOrder},
		)

		if result.Equal {
			row.Status = "Matched"
			row.Badge = "emerald"
			row.Detail = "Headers match."
			summary.Matched++
			continue
		}

		row.Status = "Changed"
		row.Badge = "amber"
		row.Detail = formatHeaderDifference(result.Difference)
		summary.Changed++
	}

	return rows, summary
}

func extractOptionsFromRow(
	row templates.CheckRowView,
) (headersearch.ExtractOptions, error) {
	depth := parseHeaderMaxDepth(row.HeaderMaxDepth)
	if depth < 1 {
		return headersearch.ExtractOptions{}, fmt.Errorf("max depth must be greater than 0")
	}

	direction := headersearch.Direction(strings.TrimSpace(row.HeaderParentDirection))
	if !direction.Valid() {
		return headersearch.ExtractOptions{}, fmt.Errorf("direction must be one of up, down, left, right")
	}

	return headersearch.ExtractOptions{
		Sheet:           strings.TrimSpace(row.HeaderSheet),
		Anchor:          strings.TrimSpace(row.HeaderAnchor),
		ParentDirection: direction,
		MaxHeaderDepth:  depth,
	}, nil
}

func rowHasVerificationConfig(row templates.CheckRowView) bool {
	return row.HeaderSheet != "" ||
		row.HeaderAnchor != "" ||
		row.HeaderParentDirection != "" ||
		row.HeaderMaxDepth != ""
}

func formatHeaderDifference(
	difference headersearch.HeaderDifference,
) string {
	parts := make([]string, 0, 3)

	if len(difference.Missing) > 0 {
		parts = append(parts, fmt.Sprintf("%d missing from new file", len(difference.Missing)))
	}

	if len(difference.Unexpected) > 0 {
		parts = append(parts, fmt.Sprintf("%d unexpected in new file", len(difference.Unexpected)))
	}

	if difference.Reordered {
		parts = append(parts, "same headers, different order")
	}

	if len(parts) == 0 {
		return "Header comparison found differences."
	}

	return strings.Join(parts, "; ") + "."
}

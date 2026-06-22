package config

import (
	"fmt"
	"strings"
)

func indexHeaders(row []string) map[string]int {
	headers := make(map[string]int, len(row))

	for index, cell := range row {
		header := normalizeHeader(cell)

		if header == "" {
			continue
		}

		headers[header] = index
	}

	return headers
}

func requireColumn(
	headers map[string]int,
	columnName string,
) (int, error) {
	index, exists := headers[normalizeHeader(columnName)]
	if !exists {
		return 0, fmt.Errorf(
			"required column %q was not found",
			columnName,
		)
	}

	return index, nil
}

func getCell(row []string, index int) string {
	if index < 0 || index >= len(row) {
		return ""
	}

	return strings.TrimSpace(row[index])
}

func normalizeHeader(value string) string {
	return strings.ToLower(strings.TrimSpace(value))
}

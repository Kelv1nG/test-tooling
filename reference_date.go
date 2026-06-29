package main

import (
	"fmt"
	"strings"
	"time"
)

const referenceDateLayout = "2006-01-02"

func defaultReferenceDate() string {
	return time.Now().Format(referenceDateLayout)
}

func normalizeReferenceDate(value string) string {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return defaultReferenceDate()
	}

	return trimmed
}

func parseReferenceDate(value string) (time.Time, error) {
	normalized := normalizeReferenceDate(value)

	referenceDate, err := time.ParseInLocation(
		referenceDateLayout,
		normalized,
		time.Local,
	)
	if err != nil {
		return time.Time{}, fmt.Errorf(
			"reference date must use YYYY-MM-DD",
		)
	}

	return referenceDate, nil
}

func referenceDateForDisplay(value string) time.Time {
	referenceDate, err := parseReferenceDate(value)
	if err != nil {
		return time.Now()
	}

	return referenceDate
}

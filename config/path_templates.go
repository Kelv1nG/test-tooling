package config

import (
	"fmt"
	"regexp"
	"strings"
	"time"
)

var templatePlaceholderPattern = regexp.MustCompile(`\{[^{}]+\}`)

type ResolvedFileTransferMap struct {
	Src  string
	Dest string
}

func ResolveFileTransferMap(
	mapping FileTransferMap,
	referenceDate time.Time,
) (ResolvedFileTransferMap, error) {
	resolved := ResolvedFileTransferMap{}

	var errs ValidationErrors

	src, err := ResolvePathTemplate(mapping.Src, referenceDate)
	if err != nil {
		errs = append(errs, fmt.Errorf("source path: %w", err))
	} else {
		resolved.Src = src
	}

	dest, err := ResolvePathTemplate(mapping.Dest, referenceDate)
	if err != nil {
		errs = append(errs, fmt.Errorf("destination path: %w", err))
	} else {
		resolved.Dest = dest
	}

	if len(errs) > 0 {
		return resolved, errs
	}

	return resolved, nil
}

func ResolvePathTemplate(
	value string,
	referenceDate time.Time,
) (string, error) {
	return ResolveTemplateText(value, referenceDate)
}

func ResolveTemplateText(
	value string,
	referenceDate time.Time,
) (string, error) {
	var errs ValidationErrors

	resolved := templatePlaceholderPattern.ReplaceAllStringFunc(
		value,
		func(match string) string {
			token := strings.TrimSpace(
				strings.TrimSuffix(
					strings.TrimPrefix(match, "{"),
					"}",
				),
			)

			replacement, ok := templatePlaceholderValue(token, referenceDate)
			if !ok {
				errs = append(errs, fmt.Errorf("unsupported placeholder %q", match))
				return match
			}

			return replacement
		},
	)

	if len(errs) > 0 {
		return "", errs
	}

	return resolved, nil
}

func ValidatePathTemplate(value string) error {
	return ValidateTemplateText(value)
}

func ValidateTemplateText(value string) error {
	_, err := ResolveTemplateText(
		value,
		time.Date(2026, time.February, 3, 0, 0, 0, 0, time.UTC),
	)
	return err
}

func templatePlaceholderValue(
	token string,
	referenceDate time.Time,
) (string, bool) {
	switch token {
	case "yyyy":
		return referenceDate.Format("2006"), true
	case "yy":
		return referenceDate.Format("06"), true
	case "mm":
		return referenceDate.Format("01"), true
	case "m":
		return referenceDate.Format("1"), true
	case "dd":
		return referenceDate.Format("02"), true
	case "d":
		return referenceDate.Format("2"), true
	case "MMMM":
		return referenceDate.Format("January"), true
	case "mmmm":
		return strings.ToLower(referenceDate.Format("January")), true
	case "MMM":
		return referenceDate.Format("Jan"), true
	case "mmm":
		return strings.ToLower(referenceDate.Format("Jan")), true
	default:
		return "", false
	}
}

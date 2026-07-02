package config

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"
)

var templatePlaceholderPattern = regexp.MustCompile(`\{[^{}]+\}`)

type PathPatternErrorKind string

const (
	// PathPatternErrorInvalid means the wildcard syntax is invalid, or a matched path could not be inspected.
	PathPatternErrorInvalid PathPatternErrorKind = "invalid"

	// PathPatternErrorNoMatch means a valid wildcard pattern did not resolve to any files.
	PathPatternErrorNoMatch PathPatternErrorKind = "no_match"

	// PathPatternErrorAmbiguous means a wildcard pattern resolved to more than one file.
	PathPatternErrorAmbiguous PathPatternErrorKind = "ambiguous"
)

// PathPatternError describes why a wildcard file pattern could not be resolved to one concrete file.
type PathPatternError struct {
	Kind    PathPatternErrorKind
	Pattern string
	Matches []string
	Err     error
}

func (e *PathPatternError) Error() string {
	switch e.Kind {
	case PathPatternErrorInvalid:
		return fmt.Sprintf("invalid path wildcard pattern %q: %v", e.Pattern, e.Err)
	case PathPatternErrorNoMatch:
		return fmt.Sprintf("path wildcard pattern %q matched no files", e.Pattern)
	case PathPatternErrorAmbiguous:
		return fmt.Sprintf(
			"path wildcard pattern %q matched %d files: %s",
			e.Pattern,
			len(e.Matches),
			strings.Join(e.Matches, ", "),
		)
	default:
		if e.Err != nil {
			return e.Err.Error()
		}
		return fmt.Sprintf("path wildcard pattern %q could not be resolved", e.Pattern)
	}
}

func (e *PathPatternError) Unwrap() error {
	return e.Err
}

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

// ResolvePathTemplateSingleMatch resolves date placeholders, then resolves wildcard file patterns to exactly one file.
func ResolvePathTemplateSingleMatch(
	value string,
	referenceDate time.Time,
) (string, error) {
	resolved, err := ResolvePathTemplate(value, referenceDate)
	if err != nil {
		return "", err
	}

	return resolveSinglePathMatch(resolved)
}

// PathTemplateHasWildcard reports whether a resolved path contains filesystem wildcard syntax.
func PathTemplateHasWildcard(value string) bool {
	return strings.ContainsAny(value, "*?")
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

// IsPathPatternNoMatch reports whether err means a wildcard pattern was valid but matched no files.
func IsPathPatternNoMatch(err error) bool {
	var patternErr *PathPatternError
	return errors.As(err, &patternErr) && patternErr.Kind == PathPatternErrorNoMatch
}

func resolveSinglePathMatch(pattern string) (string, error) {
	if !PathTemplateHasWildcard(pattern) {
		return pattern, nil
	}

	matches, err := filepath.Glob(pattern)
	if err != nil {
		return "", &PathPatternError{
			Kind:    PathPatternErrorInvalid,
			Pattern: pattern,
			Err:     err,
		}
	}

	matches, err = fileOnlyMatches(matches)
	if err != nil {
		return "", &PathPatternError{
			Kind:    PathPatternErrorInvalid,
			Pattern: pattern,
			Err:     err,
		}
	}

	switch len(matches) {
	case 0:
		return "", &PathPatternError{
			Kind:    PathPatternErrorNoMatch,
			Pattern: pattern,
		}
	case 1:
		return matches[0], nil
	default:
		return "", &PathPatternError{
			Kind:    PathPatternErrorAmbiguous,
			Pattern: pattern,
			Matches: matches,
		}
	}
}

func fileOnlyMatches(matches []string) ([]string, error) {
	files := make([]string, 0, len(matches))
	for _, match := range matches {
		info, err := os.Stat(match)
		if err != nil {
			return nil, fmt.Errorf("inspect wildcard match %q: %w", match, err)
		}
		if info.IsDir() {
			continue
		}
		files = append(files, match)
	}

	sort.Strings(files)
	return files, nil
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

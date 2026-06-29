package main

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
)

type conflictStrategy int

const (
	conflictStrategyOverwrite conflictStrategy = iota
	conflictStrategySkip
)

type copyOutcome int

const (
	copyOutcomeUnknown copyOutcome = iota
	copyOutcomeCreated
	copyOutcomeSkipped
	copyOutcomeOverwritten
)

func copyFile(
	src string,
	dest string,
	strategy conflictStrategy,
) (copyOutcome, error) {
	srcExists, err := fileExists(src)
	if err != nil {
		return 0, err
	}
	if !srcExists {
		return 0, fmt.Errorf("source file does not exist: %s", src)
	}

	destExists, err := fileExists(dest)
	if err != nil {
		return 0, err
	}

	if destExists {
		switch strategy {
		case conflictStrategyOverwrite:
			if err := writeCopy(src, dest); err != nil {
				return 0, err
			}
			return copyOutcomeOverwritten, nil
		case conflictStrategySkip:
			return copyOutcomeSkipped, nil
		}
	}

	if err := writeCopy(src, dest); err != nil {
		return 0, err
	}
	return copyOutcomeCreated, nil
}

func writeCopy(src, dest string) error {
	fin, err := os.Open(src)
	if err != nil {
		return err
	}
	defer fin.Close()

	if err := os.MkdirAll(filepath.Dir(dest), 0o755); err != nil {
		return err
	}

	fout, err := os.Create(dest)
	if err != nil {
		return err
	}

	if _, err := io.Copy(fout, fin); err != nil {
		_ = fout.Close()
		return err
	}

	if err := fout.Close(); err != nil {
		return err
	}

	return nil
}

func fileExists(path string) (bool, error) {
	_, err := os.Stat(path)

	if err == nil {
		return true, nil
	}

	if errors.Is(err, os.ErrNotExist) {
		return false, nil
	}

	return false, err
}

func fileExistsOrFalse(path string) bool {
	exists, err := fileExists(path)
	return err == nil && exists
}

func (o copyOutcome) presentation() (status string, badge string, detail string) {
	switch o {
	case copyOutcomeCreated:
		return "Created", "emerald", "Destination file created."
	case copyOutcomeOverwritten:
		return "Overwritten", "amber", "Existing destination file replaced."
	case copyOutcomeSkipped:
		return "Skipped", "slate", "Destination already existed."
	default:
		return "Unknown", "zinc", "Copy result did not map to a known status."
	}
}

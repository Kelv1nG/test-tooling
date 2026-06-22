package config

import (
	"fmt"
	"strings"
)

type ValidationErrors []error

func (errs ValidationErrors) Error() string {
	messages := make([]string, 0, len(errs))

	for _, err := range errs {
		if err == nil {
			continue
		}

		messages = append(messages, err.Error())
	}

	return strings.Join(messages, "; ")
}

func (errs ValidationErrors) Unwrap() []error {
	return []error(errs)
}

func (d TableDefinitions) Validate() error {
	var errs ValidationErrors

	errs = append(errs, d.FileTransfer.validationErrors()...)
	errs = append(errs, d.FileCheck.validationErrors()...)

	if len(errs) == 0 {
		return nil
	}

	return errs
}

func (d FileTransferTableDefinition) validationErrors() []error {
	var errs []error

	if strings.TrimSpace(d.Sheet) == "" {
		errs = append(
			errs,
			fmt.Errorf("file_transfer.sheet is required"),
		)
	}

	if strings.TrimSpace(d.SrcCol) == "" {
		errs = append(
			errs,
			fmt.Errorf("file_transfer.src_column is required"),
		)
	}

	if strings.TrimSpace(d.DstCol) == "" {
		errs = append(
			errs,
			fmt.Errorf("file_transfer.dst_column is required"),
		)
	}

	return errs
}

func (d FileCheckTableDefinition) validationErrors() []error {
	var errs []error

	if strings.TrimSpace(d.Sheet) == "" {
		errs = append(
			errs,
			fmt.Errorf("file_check.sheet is required"),
		)
	}

	if strings.TrimSpace(d.NewFileCol) == "" {
		errs = append(
			errs,
			fmt.Errorf("file_check.new_file_column is required"),
		)
	}

	if strings.TrimSpace(d.OldFileCol) == "" {
		errs = append(
			errs,
			fmt.Errorf("file_check.old_file_column is required"),
		)
	}

	return errs
}

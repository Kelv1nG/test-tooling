package config

import (
	"fmt"

	"github.com/xuri/excelize/v2"
)

type Loader struct {
	definitions TableDefinitions
}

func NewLoader(definitionsPath string) (*Loader, error) {
	definitions, err := ReadTableDefinitions(definitionsPath)
	if err != nil {
		return nil, err
	}

	return &Loader{
		definitions: definitions,
	}, nil
}

func (l *Loader) LoadWorkbook(
	path string,
) (Configuration, error) {
	file, err := excelize.OpenFile(path)
	if err != nil {
		return Configuration{}, fmt.Errorf(
			"open workbook %q: %w",
			path,
			err,
		)
	}
	defer file.Close()

	return l.Load(file)
}

func (l *Loader) Load(
	file *excelize.File,
) (Configuration, error) {
	if l == nil {
		return Configuration{}, fmt.Errorf(
			"configuration loader is nil",
		)
	}

	if file == nil {
		return Configuration{}, fmt.Errorf(
			"excel file is nil",
		)
	}

	transferMaps, transferErr := l.definitions.FileTransfer.read(file)

	checkRules, checkErr := l.definitions.FileCheck.read(file)

	var errs ValidationErrors

	if transferErr != nil {
		errs = append(errs, transferErr)
	}

	if checkErr != nil {
		errs = append(errs, checkErr)
	}

	if len(errs) > 0 {
		return Configuration{}, errs
	}

	return Configuration{
		FileTransferMaps: transferMaps,
		FileCheckRules:   checkRules,
	}, nil
}

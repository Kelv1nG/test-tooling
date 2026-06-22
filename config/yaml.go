package config

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

func ReadTableDefinitions(path string) (TableDefinitions, error) {
	file, err := os.Open(path)
	if err != nil {
		return TableDefinitions{}, fmt.Errorf(
			"open table definitions %q: %w",
			path,
			err,
		)
	}
	defer file.Close()

	var definitions TableDefinitions

	decoder := yaml.NewDecoder(file)
	decoder.KnownFields(true)

	if err := decoder.Decode(&definitions); err != nil {
		return TableDefinitions{}, fmt.Errorf(
			"decode table definitions %q: %w",
			path,
			err,
		)
	}

	if err := definitions.Validate(); err != nil {
		return TableDefinitions{}, fmt.Errorf(
			"invalid table definitions: %w",
			err,
		)
	}

	return definitions, nil
}

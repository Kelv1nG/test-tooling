package main

import "testing"

func TestConfigurationExampleLoads(t *testing.T) {
	app := mustNewApplication(
		":0",
		"table-definitions.yaml",
		"configuration-example.xlsx",
		nil,
	)

	configuration, err := app.loadConfiguration(
		"table-definitions.yaml",
		"configuration-example.xlsx",
	)
	if err != nil {
		t.Fatalf("loadConfiguration returned error: %v", err)
	}

	if len(configuration.FileCheckConfigs) == 0 {
		t.Fatal("expected configuration example to include file-check configs")
	}
}

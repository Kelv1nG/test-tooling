package main

import (
	"io"
	"log"
)

func mustNewApplication(
	listenAddr string,
	definitionsPath string,
	workbookPath string,
	logger *log.Logger,
) *application {
	if logger == nil {
		logger = log.New(io.Discard, "", 0)
	}

	return NewApplication(listenAddr, definitionsPath, workbookPath, logger)
}

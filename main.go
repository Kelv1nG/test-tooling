package main

import (
	"errors"
	"flag"
	"log"
	"net/http"
	"os"
	"time"
)

func main() {
	addr := flag.String("addr", ":8082", "HTTP listen address")
	definitionsPath := flag.String("definitions", "table-definitions.yaml", "path to table definitions YAML")
	workbookPath := flag.String("workbook", "configuration-example.xlsx", "path to workbook")
	flag.Parse()

	app := NewApplication(
		*addr,
		*definitionsPath,
		*workbookPath,
		log.New(os.Stdout, "", log.LstdFlags),
	)

	server := &http.Server{
		Addr:              *addr,
		Handler:           app.routes(),
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       15 * time.Second,
		WriteTimeout:      30 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	app.logger.Printf("listening on http://localhost%s\n", *addr)
	if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		app.logger.Fatalf("server failed: %v", err)
	}
}

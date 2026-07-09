package main

import (
	"log"
	"net/http"
	"sync"
	"tooling/templates"
)

type application struct {
	defaultDefinitionsPath string
	defaultWorkbookPath    string
	reportsRoot            string
	listenAddr             string
	logger                 *log.Logger
	verificationJobsMu     sync.Mutex
	verificationJobs       map[string]*verificationJob
}

func NewApplication(
	listenAddr string,
	definitionsPath string,
	workbookPath string,
	reportsRoot string,
	logger *log.Logger,
) *application {
	return &application{
		defaultDefinitionsPath: definitionsPath,
		defaultWorkbookPath:    workbookPath,
		reportsRoot:            reportsRoot,
		listenAddr:             listenAddr,
		logger:                 logger,
		verificationJobs:       map[string]*verificationJob{},
	}
}

func (a *application) routes() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/", a.handleIndex)
	mux.HandleFunc("/configuration", a.handleConfigurationPage)
	mux.HandleFunc("/file-transfer", a.handleFileTransferPage)
	mux.HandleFunc("/checking", a.handleCheckingPage)
	mux.HandleFunc("/load", a.handleLoad)
	mux.HandleFunc("/transfer", a.handleTransfer)
	mux.HandleFunc("/save-transfer", a.handleSaveTransfer)
	mux.HandleFunc("/save-checks", a.handleSaveChecks)
	mux.HandleFunc("/verify-checks", a.handleVerifyChecks)
	mux.HandleFunc("/verify-checks/status", a.handleVerifyChecksStatus)
	mux.HandleFunc("/reports/open", a.handleReportOpen)
	mux.HandleFunc("/reports/download", a.handleReportDownload)
	mux.HandleFunc("/healthz", a.handleHealth)

	return mux
}

func (a *application) renderResponse(
	writer http.ResponseWriter,
	request *http.Request,
	data templates.PageData,
	statusCode int,
) {
	renderStatus := statusCode
	writer.Header().Set("Content-Type", "text/html; charset=utf-8")
	if request != nil && request.Header.Get("HX-Request") == "true" {
		if renderStatus >= http.StatusBadRequest {
			renderStatus = http.StatusOK
		}
	}
	writer.WriteHeader(renderStatus)

	component := templates.Page(data)
	if request != nil && request.Header.Get("HX-Request") == "true" {
		component = templates.AppShell(data)
	}

	if err := component.Render(request.Context(), writer); err != nil {
		a.logger.Printf("render page: %v", err)
	}
}

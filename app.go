package main

import (
	"log"
	"net/http"
	"strings"
	"time"
	"tooling/config"
)

type application struct {
	defaultDefinitionsPath string
	defaultWorkbookPath    string
	listenAddr             string
	logger                 *log.Logger
}

func mustNewApplication(
	listenAddr string,
	definitionsPath string,
	workbookPath string,
	logger *log.Logger,
) *application {
	return &application{
		defaultDefinitionsPath: definitionsPath,
		defaultWorkbookPath:    workbookPath,
		listenAddr:             listenAddr,
		logger:                 logger,
	}
}

func (a *application) routes() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/", a.handleIndex)
	mux.HandleFunc("/load", a.handleLoad)
	mux.HandleFunc("/transfer", a.handleTransfer)
	mux.HandleFunc("/save-transfer", a.handleSaveTransfer)
	mux.HandleFunc("/save-checks", a.handleSaveChecks)
	mux.HandleFunc("/healthz", a.handleHealth)

	return mux
}

func (a *application) newPageData(
	definitionsPath string,
	workbookPath string,
) pageData {
	return pageData{
		ListenAddr:      a.listenAddr,
		DefinitionsPath: definitionsPath,
		WorkbookPath:    workbookPath,
		ActiveTab:       "configuration",
		Strategy:        "overwrite",
	}
}

func (a *application) pageDataFromRequest(
	request *http.Request,
) pageData {
	_ = request.ParseForm()

	definitionsPath := strings.TrimSpace(request.FormValue("definitionsPath"))
	if definitionsPath == "" {
		definitionsPath = a.defaultDefinitionsPath
	}

	workbookPath := strings.TrimSpace(request.FormValue("workbookPath"))
	if workbookPath == "" {
		workbookPath = a.defaultWorkbookPath
	}

	data := a.newPageData(definitionsPath, workbookPath)
	data.Strategy = normalizeStrategy(request.FormValue("strategy"))
	data.ActiveTab = normalizeTab(request.FormValue("activeTab"))
	return data
}

func (a *application) populateConfigData(data *pageData) error {
	configuration, err := a.loadConfiguration(
		data.DefinitionsPath,
		data.WorkbookPath,
	)
	if err != nil {
		return err
	}

	a.applyConfiguration(data, configuration)
	return nil
}

func (a *application) loadConfiguration(
	definitionsPath string,
	workbookPath string,
) (config.Configuration, error) {
	loader, err := config.NewLoader(definitionsPath)
	if err != nil {
		return config.Configuration{}, err
	}

	configuration, err := loader.LoadWorkbook(workbookPath)
	if err != nil {
		return config.Configuration{}, err
	}

	return configuration, nil
}

func (a *application) saveTransferRows(
	definitionsPath string,
	workbookPath string,
	rows []transferRowView,
) error {
	loader, err := config.NewLoader(definitionsPath)
	if err != nil {
		return err
	}

	return loader.SaveTransferWorkbook(
		workbookPath,
		buildTransferMappingsFromRows(rows),
	)
}

func (a *application) saveCheckRows(
	definitionsPath string,
	workbookPath string,
	rows []checkRowView,
) error {
	loader, err := config.NewLoader(definitionsPath)
	if err != nil {
		return err
	}

	return loader.SaveCheckWorkbook(
		workbookPath,
		buildCheckRulesFromRows(rows),
	)
}

func (a *application) applyConfiguration(
	data *pageData,
	configuration config.Configuration,
) {
	data.HasConfig = true
	data.LoadedAt = time.Now().Format(time.RFC1123)
	data.TransferRows = buildTransferRows(configuration.FileTransferMaps)
	data.CheckRows = buildCheckRows(configuration.FileCheckRules)
	data.TransferCount = len(configuration.FileTransferMaps)
	data.CheckCount = len(configuration.FileCheckRules)
}

func (a *application) renderResponse(
	writer http.ResponseWriter,
	request *http.Request,
	data pageData,
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

	component := Page(data)
	if request != nil && request.Header.Get("HX-Request") == "true" {
		component = AppShell(data)
	}

	if err := component.Render(request.Context(), writer); err != nil {
		a.logger.Printf("render page: %v", err)
	}
}

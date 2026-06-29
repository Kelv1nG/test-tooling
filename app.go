package main

import (
	"log"
	"net/http"
	"strings"
	"time"
	"tooling/config"
	"tooling/templates"
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
) templates.PageData {
	return templates.PageData{
		ListenAddr:      a.listenAddr,
		DefinitionsPath: definitionsPath,
		WorkbookPath:    workbookPath,
		ActiveTab:       "configuration",
		Strategy:        "overwrite",
		ReferenceDate:   defaultReferenceDate(),
	}
}

func (a *application) pageDataFromRequest(
	request *http.Request,
) templates.PageData {
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
	data.ReferenceDate = normalizeReferenceDate(request.FormValue("referenceDate"))
	data.ActiveTab = normalizeTab(request.FormValue("activeTab"))
	return data
}

func (a *application) populateConfigData(data *templates.PageData) error {
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
	rows []templates.TransferRowView,
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
	rows []templates.CheckRowView,
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
	data *templates.PageData,
	configuration config.Configuration,
) {
	referenceDate := referenceDateForDisplay(data.ReferenceDate)

	data.HasConfig = true
	data.LoadedAt = time.Now().Format(time.RFC1123)
	data.TransferRows = templates.BuildTransferRows(
		configuration.FileTransferMaps,
		referenceDate,
	)
	data.CheckRows = templates.BuildCheckRows(configuration.FileCheckRules)
	data.TransferCount = len(configuration.FileTransferMaps)
	data.CheckCount = len(configuration.FileCheckRules)
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

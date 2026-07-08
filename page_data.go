package main

import (
	"net/http"
	"strconv"
	"strings"
	"time"

	"tooling/config"
	"tooling/templates"
)

const (
	tabConfiguration = "configuration"
	tabFileTransfer  = "file-transfer"
	tabChecking      = "checking"
)

func (a *application) newPageData(
	definitionsPath string,
	workbookPath string,
) templates.PageData {
	return templates.PageData{
		ListenAddr:         a.listenAddr,
		DefinitionsPath:    definitionsPath,
		WorkbookPath:       workbookPath,
		ActiveTab:          tabConfiguration,
		Strategy:           string(defaultTransferMode),
		ReferenceDate:      defaultReferenceDate(),
		CheckReferenceDate: defaultReferenceDate(),
		CheckPage:          1,
	}
}

func (a *application) pageDataFromRequest(
	request *http.Request,
) templates.PageData {
	_ = request.ParseForm()

	data := a.newPageData(
		requestFormValueOrDefault(
			request,
			"definitionsPath",
			a.defaultDefinitionsPath,
		),
		requestFormValueOrDefault(
			request,
			"workbookPath",
			a.defaultWorkbookPath,
		),
	)

	data.Strategy = string(parseTransferMode(request.FormValue("strategy")))
	data.ReferenceDate = normalizeReferenceDate(request.FormValue("referenceDate"))
	data.CheckReferenceDate = normalizeReferenceDate(request.FormValue("checkReferenceDate"))
	data.CheckPage = normalizeCheckPage(request.FormValue("checkPage"))
	data.ActiveTab = normalizeTab(request.FormValue("activeTab"))
	return data
}

func (a *application) configuredPageDataFromRequest(
	request *http.Request,
	activeTab string,
) (templates.PageData, config.Configuration, error) {
	data := a.pageDataFromRequest(request)
	data.ActiveTab = activeTab

	configuration, err := a.loadConfiguration(
		data.DefinitionsPath,
		data.WorkbookPath,
	)
	if err != nil {
		return data, config.Configuration{}, err
	}

	a.applyConfiguration(&data, configuration)
	return data, configuration, nil
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
	loader, err := a.newLoader(definitionsPath)
	if err != nil {
		return config.Configuration{}, err
	}

	return loader.LoadWorkbook(workbookPath)
}

func (a *application) saveTransferRows(
	definitionsPath string,
	workbookPath string,
	rows []templates.TransferRowView,
) error {
	loader, err := a.newLoader(definitionsPath)
	if err != nil {
		return err
	}

	return loader.SaveTransferWorkbook(
		workbookPath,
		buildTransferMappings(rows),
	)
}

func (a *application) saveCheckRows(
	definitionsPath string,
	workbookPath string,
	rows []templates.CheckRowView,
) error {
	loader, err := a.newLoader(definitionsPath)
	if err != nil {
		return err
	}

	return loader.SaveCheckWorkbook(
		workbookPath,
		buildCheckConfigs(rows),
	)
}

func (a *application) newLoader(
	definitionsPath string,
) (*config.Loader, error) {
	return config.NewLoader(definitionsPath)
}

func (a *application) applyConfiguration(
	data *templates.PageData,
	configuration config.Configuration,
) {
	transferReferenceDate := referenceDateForDisplay(data.ReferenceDate)
	checkReferenceDate := referenceDateForDisplay(data.CheckReferenceDate)

	data.HasConfig = true
	data.LoadedAt = time.Now().Format(time.RFC1123)
	data.TransferRows = buildTransferRows(
		configuration.FileTransferMaps,
		transferReferenceDate,
	)
	data.CheckRows = buildCheckRows(configuration.FileCheckConfigs, checkReferenceDate)
	data.TransferCount = len(configuration.FileTransferMaps)
	data.CheckCount = len(configuration.FileCheckConfigs)
}

func requestFormValueOrDefault(
	request *http.Request,
	field string,
	fallback string,
) string {
	value := strings.TrimSpace(request.FormValue(field))
	if value == "" {
		return fallback
	}

	return value
}

func normalizeTab(value string) string {
	switch strings.TrimSpace(value) {
	case tabFileTransfer:
		return tabFileTransfer
	case tabChecking:
		return tabChecking
	default:
		return tabConfiguration
	}
}

func tabPath(tab string) string {
	switch normalizeTab(tab) {
	case tabFileTransfer:
		return "/file-transfer"
	case tabChecking:
		return "/checking"
	default:
		return "/configuration"
	}
}

func normalizeCheckPage(value string) int {
	page, err := strconv.Atoi(strings.TrimSpace(value))
	if err != nil || page < 1 {
		return 1
	}

	return page
}

package main

import (
	"fmt"
	"net/http"
)

func (a *application) handleIndex(
	writer http.ResponseWriter,
	request *http.Request,
) {
	if request.Method != http.MethodGet {
		http.Error(writer, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	data := a.newPageData(
		a.defaultDefinitionsPath,
		a.defaultWorkbookPath,
	)
	data.ActiveTab = normalizeTab(request.URL.Query().Get("tab"))

	if fileExistsOrFalse(a.defaultDefinitionsPath) && fileExistsOrFalse(a.defaultWorkbookPath) {
		if err := a.populateConfigData(&data); err != nil {
			data.LoadError = err.Error()
		}
	}

	a.renderResponse(writer, request, data, http.StatusOK)
}

func (a *application) handleLoad(
	writer http.ResponseWriter,
	request *http.Request,
) {
	if request.Method != http.MethodPost {
		http.Error(writer, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	data := a.pageDataFromRequest(request)
	if err := a.populateConfigData(&data); err != nil {
		data.LoadError = err.Error()
		data.ActiveTab = "configuration"
		a.renderResponse(writer, request, data, http.StatusBadRequest)
		return
	}

	data.ActiveTab = "configuration"
	a.renderResponse(writer, request, data, http.StatusOK)
}

func (a *application) handleTransfer(
	writer http.ResponseWriter,
	request *http.Request,
) {
	if request.Method != http.MethodPost {
		http.Error(writer, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	data := a.pageDataFromRequest(request)
	data.ActiveTab = "file-transfer"
	configuration, err := a.loadConfiguration(
		data.DefinitionsPath,
		data.WorkbookPath,
	)
	if err != nil {
		data.LoadError = err.Error()
		a.renderResponse(writer, request, data, http.StatusBadRequest)
		return
	}

	a.applyConfiguration(&data, configuration)

	results, summary := runTransfers(configuration, data.Strategy)
	data.TransferResults = results
	data.TransferSummary = summary
	data.TransferRows = applyTransferResultsToRows(
		data.TransferRows,
		results,
	)

	if summary.Errors > 0 {
		data.TransferHasErrors = true
		data.TransferMessage = fmt.Sprintf(
			"Transfer run finished with %d failure(s).",
			summary.Errors,
		)
		a.renderResponse(writer, request, data, http.StatusInternalServerError)
		return
	}

	data.TransferMessage = fmt.Sprintf(
		"Transfer run completed for %d mapping(s) using %s mode.",
		len(configuration.FileTransferMaps),
		data.Strategy,
	)

	a.renderResponse(writer, request, data, http.StatusOK)
}

func (a *application) handleSaveTransfer(
	writer http.ResponseWriter,
	request *http.Request,
) {
	if request.Method != http.MethodPost {
		http.Error(writer, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	data := a.pageDataFromRequest(request)
	data.ActiveTab = "file-transfer"

	if err := a.populateConfigData(&data); err != nil {
		data.LoadError = err.Error()
		a.renderResponse(writer, request, data, http.StatusBadRequest)
		return
	}

	rows, err := parseTransferRowsForm(request.Form)
	if len(rows) > 0 {
		data.TransferRows = rows
		data.TransferCount = len(rows)
	}

	if err != nil {
		data.SaveHasErrors = true
		data.SaveMessage = err.Error()
		a.renderResponse(writer, request, data, http.StatusBadRequest)
		return
	}

	if err := a.saveTransferRows(data.DefinitionsPath, data.WorkbookPath, rows); err != nil {
		data.SaveHasErrors = true
		data.SaveMessage = err.Error()
		a.renderResponse(writer, request, data, http.StatusInternalServerError)
		return
	}

	if err := a.populateConfigData(&data); err != nil {
		data.SaveHasErrors = true
		data.SaveMessage = fmt.Sprintf("Transfer rows were saved, but reload failed: %v", err)
		a.renderResponse(writer, request, data, http.StatusInternalServerError)
		return
	}

	data.SaveMessage = fmt.Sprintf("Saved %d transfer row(s) to the workbook.", len(rows))
	a.renderResponse(writer, request, data, http.StatusOK)
}

func (a *application) handleSaveChecks(
	writer http.ResponseWriter,
	request *http.Request,
) {
	if request.Method != http.MethodPost {
		http.Error(writer, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	data := a.pageDataFromRequest(request)
	data.ActiveTab = "checking"

	if err := a.populateConfigData(&data); err != nil {
		data.LoadError = err.Error()
		a.renderResponse(writer, request, data, http.StatusBadRequest)
		return
	}

	rows, err := parseCheckRowsForm(request.Form)
	if len(rows) > 0 {
		data.CheckRows = rows
		data.CheckCount = len(rows)
	}

	if err != nil {
		data.SaveHasErrors = true
		data.SaveMessage = err.Error()
		a.renderResponse(writer, request, data, http.StatusBadRequest)
		return
	}

	if err := a.saveCheckRows(data.DefinitionsPath, data.WorkbookPath, rows); err != nil {
		data.SaveHasErrors = true
		data.SaveMessage = err.Error()
		a.renderResponse(writer, request, data, http.StatusInternalServerError)
		return
	}

	if err := a.populateConfigData(&data); err != nil {
		data.SaveHasErrors = true
		data.SaveMessage = fmt.Sprintf("Check rows were saved, but reload failed: %v", err)
		a.renderResponse(writer, request, data, http.StatusInternalServerError)
		return
	}

	data.SaveMessage = fmt.Sprintf("Saved %d check row(s) to the workbook.", len(rows))
	a.renderResponse(writer, request, data, http.StatusOK)
}

func (a *application) handleHealth(
	writer http.ResponseWriter,
	_ *http.Request,
) {
	writer.Header().Set("Content-Type", "text/plain; charset=utf-8")
	writer.WriteHeader(http.StatusOK)
	_, _ = writer.Write([]byte("ok"))
}

package main

import (
	"fmt"
	"net/http"
)

func (a *application) handleIndex(
	writer http.ResponseWriter,
	request *http.Request,
) {
	if !allowMethod(writer, request, http.MethodGet) {
		return
	}

	data := a.newPageData(
		a.defaultDefinitionsPath,
		a.defaultWorkbookPath,
	)
	data.ActiveTab = normalizeTab(request.URL.Query().Get("tab"))

	// Default files are optional at startup; preload workbook-backed data only
	// when both paths are present so the first page can still render cleanly.
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
	if !allowMethod(writer, request, http.MethodPost) {
		return
	}

	// Loading is intentionally read-only: use the submitted paths to preview the
	// workbook configuration without saving anything back to disk.
	data, _, err := a.configuredPageDataFromRequest(request, tabConfiguration)
	if err != nil {
		data = a.pageDataFromRequest(request)
		data.LoadError = err.Error()
		data.ActiveTab = tabConfiguration
		a.renderResponse(writer, request, data, http.StatusBadRequest)
		return
	}

	a.renderResponse(writer, request, data, http.StatusOK)
}

func (a *application) handleTransfer(
	writer http.ResponseWriter,
	request *http.Request,
) {
	if !allowMethod(writer, request, http.MethodPost) {
		return
	}

	data, configuration, err := a.configuredPageDataFromRequest(
		request,
		tabFileTransfer,
	)
	if err != nil {
		data.LoadError = err.Error()
		a.renderResponse(writer, request, data, http.StatusBadRequest)
		return
	}

	referenceDate, err := parseReferenceDate(data.ReferenceDate)
	if err != nil {
		data.TransferHasErrors = true
		data.TransferMessage = err.Error()
		a.renderResponse(writer, request, data, http.StatusBadRequest)
		return
	}

	runner := newTransferRunner(
		parseTransferMode(data.Strategy),
		referenceDate,
	)
	results, summary := runner.run(configuration.FileTransferMaps)
	data.TransferResults = results
	data.TransferSummary = summary
	data.TransferRows = applyTransferResultsToRows(
		data.TransferRows,
		results,
	)

	// A completed run can still be a failed user operation when any mapping
	// failed, so report the row-level results with an error status.
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
	if !allowMethod(writer, request, http.MethodPost) {
		return
	}

	data := a.pageDataFromRequest(request)
	data.ActiveTab = tabFileTransfer

	if err := a.populateConfigData(&data); err != nil {
		data.LoadError = err.Error()
		a.renderResponse(writer, request, data, http.StatusBadRequest)
		return
	}

	referenceDate, err := parseReferenceDate(data.ReferenceDate)
	if err != nil {
		data.SaveHasErrors = true
		data.SaveMessage = err.Error()
		a.renderResponse(writer, request, data, http.StatusBadRequest)
		return
	}

	// Parse the posted rows before persisting so validation errors can be shown
	// against the exact values the user submitted.
	rows, err := parseTransferRowsForm(request.Form, referenceDate)
	data.TransferRows = rows
	data.TransferCount = len(rows)

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

	// Reload after save so generated workbook values and normalized paths are
	// reflected back into the editable view.
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
	if !allowMethod(writer, request, http.MethodPost) {
		return
	}

	data := a.pageDataFromRequest(request)
	data.ActiveTab = tabChecking

	if err := a.populateConfigData(&data); err != nil {
		data.LoadError = err.Error()
		a.renderResponse(writer, request, data, http.StatusBadRequest)
		return
	}

	referenceDate, err := parseReferenceDate(data.CheckReferenceDate)
	if err != nil {
		data.SaveHasErrors = true
		data.SaveMessage = err.Error()
		a.renderResponse(writer, request, data, http.StatusBadRequest)
		return
	}

	// Parse the posted rows before persisting so validation errors can be shown
	// against the exact values the user submitted.
	rows, err := parseCheckRowsForm(request.Form, referenceDate)
	data.CheckRows = rows
	data.CheckCount = len(rows)

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

	// Reload after save so generated IDs and workbook-backed defaults become
	// the source of truth for the next edit.
	if err := a.populateConfigData(&data); err != nil {
		data.SaveHasErrors = true
		data.SaveMessage = fmt.Sprintf("Check rows were saved, but reload failed: %v", err)
		a.renderResponse(writer, request, data, http.StatusInternalServerError)
		return
	}

	data.SaveMessage = fmt.Sprintf("Saved %d check config(s) to the workbook.", len(rows))
	a.renderResponse(writer, request, data, http.StatusOK)
}

func (a *application) handleVerifyChecks(
	writer http.ResponseWriter,
	request *http.Request,
) {
	if !allowMethod(writer, request, http.MethodPost) {
		return
	}

	data := a.pageDataFromRequest(request)
	data.ActiveTab = tabChecking

	if err := a.populateConfigData(&data); err != nil {
		data.LoadError = err.Error()
		a.renderResponse(writer, request, data, http.StatusBadRequest)
		return
	}

	referenceDate, err := parseReferenceDate(data.CheckReferenceDate)
	if err != nil {
		data.CheckHasIssues = true
		data.CheckMessage = err.Error()
		a.renderResponse(writer, request, data, http.StatusBadRequest)
		return
	}

	// Verification runs against the current form payload rather than requiring
	// a save first, which lets users test edits before committing them.
	rows, err := parseCheckRowsForm(request.Form, referenceDate)
	data.CheckRows = rows
	data.CheckCount = len(rows)

	if err != nil {
		data.CheckHasIssues = true
		data.CheckMessage = err.Error()
		a.renderResponse(writer, request, data, http.StatusBadRequest)
		return
	}

	data.CheckRows, data.CheckSummary = runCheckVerification(data.CheckRows, referenceDate)
	data.CheckHasIssues = data.CheckSummary.Changed > 0 || data.CheckSummary.Errors > 0
	data.CheckMessage = fmt.Sprintf(
		"Verification checked %d rule(s): matched %d, changed %d, errors %d, skipped %d.",
		data.CheckSummary.Attempted,
		data.CheckSummary.Matched,
		data.CheckSummary.Changed,
		data.CheckSummary.Errors,
		data.CheckSummary.Skipped,
	)

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

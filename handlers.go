package main

import (
	"fmt"
	"net/http"
	"strconv"
)

func (a *application) handleIndex(
	writer http.ResponseWriter,
	request *http.Request,
) {
	if request.URL.Path != "/" {
		http.NotFound(writer, request)
		return
	}
	if !allowMethod(writer, request, http.MethodGet) {
		return
	}

	http.Redirect(
		writer,
		request,
		tabPath(normalizeTab(request.URL.Query().Get("tab"))),
		http.StatusSeeOther,
	)
}

func (a *application) handleConfigurationPage(
	writer http.ResponseWriter,
	request *http.Request,
) {
	a.handleTabPage(writer, request, tabConfiguration)
}

func (a *application) handleFileTransferPage(
	writer http.ResponseWriter,
	request *http.Request,
) {
	a.handleTabPage(writer, request, tabFileTransfer)
}

func (a *application) handleCheckingPage(
	writer http.ResponseWriter,
	request *http.Request,
) {
	a.handleTabPage(writer, request, tabChecking)
}

func (a *application) handleTabPage(
	writer http.ResponseWriter,
	request *http.Request,
	activeTab string,
) {
	if !allowMethod(writer, request, http.MethodGet) {
		return
	}

	data := a.newPageData(
		a.defaultDefinitionsPath,
		a.defaultWorkbookPath,
	)
	data.ActiveTab = normalizeTab(activeTab)

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
	data.TransferSummaryRows = buildTransferSummaryRows(results)
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

func (a *application) handleTransferPathCheck(
	writer http.ResponseWriter,
	request *http.Request,
) {
	if !allowMethod(writer, request, http.MethodPost) {
		return
	}

	data := a.pageDataFromRequest(request)
	data.ActiveTab = tabFileTransfer
	data.HasConfig = true

	referenceDate, err := parseReferenceDate(data.ReferenceDate)
	if err != nil {
		data.SaveHasErrors = true
		data.SaveMessage = err.Error()
		a.renderResponse(writer, request, data, http.StatusBadRequest)
		return
	}

	rows, err := parseTransferRowsForm(request.Form, referenceDate)
	data.TransferRows = rows
	data.TransferCount = len(rows)

	if err != nil {
		data.SaveHasErrors = true
		data.SaveMessage = err.Error()
		a.renderResponse(writer, request, data, http.StatusBadRequest)
		return
	}

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
	existingRows := data.CheckRows

	referenceDate, err := parseReferenceDate(data.CheckReferenceDate)
	if err != nil {
		data.SaveHasErrors = true
		data.SaveMessage = err.Error()
		a.renderResponse(writer, request, data, http.StatusBadRequest)
		return
	}

	saveCheckIndex, saveSingleCheck, err := parseSaveCheckIndex(request.FormValue("saveCheckIndex"))
	if err != nil {
		data.SaveHasErrors = true
		data.SaveMessage = err.Error()
		a.renderResponse(writer, request, data, http.StatusBadRequest)
		return
	}

	checkFormValues := request.Form
	if saveSingleCheck {
		checkFormValues, err = filterSingleCheckRowsForm(request.Form, saveCheckIndex)
		if err != nil {
			data.SaveHasErrors = true
			data.SaveMessage = err.Error()
			a.renderResponse(writer, request, data, http.StatusBadRequest)
			return
		}
	}

	// Parse the posted rows before persisting so validation errors can be shown
	// against the exact values the user submitted.
	rows, err := parseCheckRowsForm(checkFormValues, referenceDate)
	data.CheckRows = rows
	data.CheckCount = len(rows)
	if saveSingleCheck && len(rows) == 1 {
		data.CheckRows = mergeSingleCheckRow(existingRows, rows[0])
		data.CheckCount = len(data.CheckRows)
	}

	if err != nil {
		data.SaveHasErrors = true
		data.SaveMessage = err.Error()
		a.renderResponse(writer, request, data, http.StatusBadRequest)
		return
	}

	rowsToSave := rows
	if saveSingleCheck {
		if len(rows) != 1 {
			data.SaveHasErrors = true
			data.SaveMessage = "single-card save did not include exactly one check config"
			a.renderResponse(writer, request, data, http.StatusBadRequest)
			return
		}
		rowsToSave = mergeSingleCheckRow(existingRows, rows[0])
	}

	if err := a.saveCheckRows(data.DefinitionsPath, data.WorkbookPath, rowsToSave); err != nil {
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

	if saveSingleCheck && len(rows) == 1 {
		data.SaveMessage = fmt.Sprintf("Saved check config %s to the workbook.", rows[0].ID)
	} else {
		data.SaveMessage = fmt.Sprintf("Saved %d check config(s) to the workbook.", len(rows))
	}
	a.renderResponse(writer, request, data, http.StatusOK)
}

func parseSaveCheckIndex(value string) (int, bool, error) {
	if value == "" {
		return 0, false, nil
	}

	index, err := strconv.Atoi(value)
	if err != nil || index < 1 {
		return 0, false, fmt.Errorf("save target check config is invalid")
	}

	return index, true, nil
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

	job := a.startVerificationJob(data, rows, referenceDate)
	a.renderResponse(writer, request, job.pageData(), http.StatusOK)
}

func (a *application) handleVerifyChecksStatus(
	writer http.ResponseWriter,
	request *http.Request,
) {
	if !allowMethod(writer, request, http.MethodGet) {
		return
	}
	writer.Header().Set("Cache-Control", "no-store")

	jobID := request.URL.Query().Get("id")
	if jobID == "" {
		data := a.newPageData(
			a.defaultDefinitionsPath,
			a.defaultWorkbookPath,
		)
		data.ActiveTab = tabChecking
		data.CheckHasIssues = true
		data.CheckMessage = "Verification run is missing a job ID."
		retargetVerificationStatus(writer, request)
		a.renderResponse(writer, request, data, http.StatusBadRequest)
		return
	}

	job, ok := a.verificationJob(jobID)
	if !ok {
		data := a.newPageData(
			a.defaultDefinitionsPath,
			a.defaultWorkbookPath,
		)
		data.ActiveTab = tabChecking
		data.CheckHasIssues = true
		data.CheckMessage = "Verification run was not found. Start verification again."
		retargetVerificationStatus(writer, request)
		a.renderResponse(writer, request, data, http.StatusNotFound)
		return
	}

	data := job.progressData()
	if !data.CheckRunRunning {
		retargetVerificationStatus(writer, request)
		data = job.pageData()
	}
	a.renderResponse(writer, request, data, http.StatusOK)
}

func retargetVerificationStatus(
	writer http.ResponseWriter,
	request *http.Request,
) {
	if request != nil && request.Header.Get("HX-Request") == "true" {
		writer.Header().Set("HX-Retarget", "#checking-panel")
	}
}

func (a *application) handleHealth(
	writer http.ResponseWriter,
	_ *http.Request,
) {
	writer.Header().Set("Content-Type", "text/plain; charset=utf-8")
	writer.WriteHeader(http.StatusOK)
	_, _ = writer.Write([]byte("ok"))
}

package main

import (
	"fmt"
	"mime"
	"net/http"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"strings"
)

func (a *application) handleReportOpen(
	writer http.ResponseWriter,
	request *http.Request,
) {
	if !allowMethod(writer, request, http.MethodGet) {
		return
	}

	relativePath, err := validateReportDownloadRequest(
		a.reportsRoot,
		request.URL.Query().Get("path"),
	)
	if err != nil {
		status := reportDownloadErrorStatus(err)
		http.Error(writer, http.StatusText(status), status)
		return
	}
	fullPath, err := resolveReportDownloadPath(
		a.reportsRoot,
		request.URL.Query().Get("path"),
	)
	if err != nil {
		status := reportDownloadErrorStatus(err)
		http.Error(writer, http.StatusText(status), status)
		return
	}
	info, err := os.Stat(fullPath)
	if err != nil {
		if os.IsNotExist(err) {
			http.Error(writer, "file not found", http.StatusNotFound)
			return
		}

		a.logger.Printf("access report file %q: %v", fullPath, err)
		http.Error(
			writer,
			http.StatusText(http.StatusInternalServerError),
			http.StatusInternalServerError,
		)
		return
	}
	if info.IsDir() {
		http.Error(writer, "cannot download directory", http.StatusBadRequest)
		return
	}

	downloadURL := reportDownloadAbsoluteURL(request, relativePath)
	http.Redirect(
		writer,
		request,
		"ms-excel:ofv|u|"+downloadURL,
		http.StatusFound,
	)
}

func (a *application) handleReportDownload(
	writer http.ResponseWriter,
	request *http.Request,
) {
	if !allowMethod(writer, request, http.MethodGet) {
		return
	}

	fullPath, err := resolveReportDownloadPath(
		a.reportsRoot,
		request.URL.Query().Get("path"),
	)
	if err != nil {
		status := reportDownloadErrorStatus(err)
		http.Error(writer, http.StatusText(status), status)
		return
	}

	info, err := os.Stat(fullPath)
	if err != nil {
		if os.IsNotExist(err) {
			http.Error(writer, "file not found", http.StatusNotFound)
			return
		}

		a.logger.Printf("access report file %q: %v", fullPath, err)
		http.Error(
			writer,
			http.StatusText(http.StatusInternalServerError),
			http.StatusInternalServerError,
		)
		return
	}
	if info.IsDir() {
		http.Error(writer, "cannot download directory", http.StatusBadRequest)
		return
	}

	writer.Header().Set(
		"Content-Disposition",
		mime.FormatMediaType("inline", map[string]string{
			"filename": filepath.Base(fullPath),
		}),
	)
	http.ServeFile(writer, request, fullPath)
}

func reportDownloadAbsoluteURL(
	request *http.Request,
	relativePath string,
) string {
	scheme := "http"
	if request.TLS != nil {
		scheme = "https"
	}

	values := url.Values{}
	values.Set("path", relativePath)

	return (&url.URL{
		Scheme:   scheme,
		Host:     request.Host,
		Path:     "/reports/download",
		RawQuery: values.Encode(),
	}).String()
}

var (
	errReportsRootMissing = fmt.Errorf("reports root is not configured")
	errReportsRootInvalid = fmt.Errorf("reports root must be absolute")
	errReportPathMissing  = fmt.Errorf("missing report path")
	errReportPathInvalid  = fmt.Errorf("invalid report path")
)

func reportDownloadErrorStatus(err error) int {
	switch err {
	case errReportPathMissing, errReportPathInvalid:
		return http.StatusBadRequest
	default:
		return http.StatusInternalServerError
	}
}

func resolveReportDownloadPath(
	reportsRoot string,
	rawPath string,
) (string, error) {
	cleanPath, err := validateReportDownloadRequest(reportsRoot, rawPath)
	if err != nil {
		return "", err
	}

	fullPath := filepath.Join(reportsRoot, filepath.FromSlash(cleanPath))
	absRoot, err := filepath.Abs(reportsRoot)
	if err != nil {
		return "", errReportsRootInvalid
	}
	absFullPath, err := filepath.Abs(fullPath)
	if err != nil {
		return "", errReportPathInvalid
	}

	relToRoot, err := filepath.Rel(absRoot, absFullPath)
	if err != nil || relToRoot == ".." || strings.HasPrefix(relToRoot, ".."+string(os.PathSeparator)) {
		return "", errReportPathInvalid
	}

	return absFullPath, nil
}

func validateReportDownloadRequest(
	reportsRoot string,
	rawPath string,
) (string, error) {
	reportsRoot = strings.TrimSpace(reportsRoot)
	if reportsRoot == "" {
		return "", errReportsRootMissing
	}
	if !filepath.IsAbs(reportsRoot) {
		return "", errReportsRootInvalid
	}

	cleanPath, err := cleanReportRelativePath(rawPath)
	if err != nil {
		return "", err
	}

	return cleanPath, nil
}

func cleanReportRelativePath(rawPath string) (string, error) {
	if strings.TrimSpace(rawPath) == "" {
		return "", errReportPathMissing
	}
	if strings.ContainsRune(rawPath, 0) {
		return "", errReportPathInvalid
	}

	normalized := strings.ReplaceAll(rawPath, `\`, "/")
	if isAbsoluteReportPath(normalized) {
		return "", errReportPathInvalid
	}

	for _, segment := range strings.Split(normalized, "/") {
		if segment == ".." {
			return "", errReportPathInvalid
		}
	}

	cleanPath := path.Clean(normalized)
	if cleanPath == "." || cleanPath == ".." || strings.HasPrefix(cleanPath, "../") {
		return "", errReportPathInvalid
	}

	return cleanPath, nil
}

func isAbsoluteReportPath(value string) bool {
	value = strings.TrimSpace(value)
	if value == "" {
		return false
	}
	normalized := strings.ReplaceAll(value, `\`, "/")
	if strings.HasPrefix(normalized, "/") {
		return true
	}
	if len(normalized) >= 2 && normalized[1] == ':' && isASCIIAlpha(normalized[0]) {
		return true
	}

	return filepath.IsAbs(value) || path.IsAbs(normalized)
}

func isASCIIAlpha(value byte) bool {
	return (value >= 'A' && value <= 'Z') || (value >= 'a' && value <= 'z')
}

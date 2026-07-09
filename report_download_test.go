package main

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestReportOpenRedirectsToExcelURI(t *testing.T) {
	root := t.TempDir()
	filePath := filepath.Join(root, "monthly", "abc.xlsx")
	if err := os.MkdirAll(filepath.Dir(filePath), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filePath, []byte("workbook"), 0o644); err != nil {
		t.Fatal(err)
	}

	app := mustNewApplicationWithReportsRoot(root)
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(
		http.MethodGet,
		"http://reports.local/reports/open?path=monthly/abc.xlsx",
		nil,
	)
	app.routes().ServeHTTP(recorder, request)

	if recorder.Code != http.StatusFound {
		t.Fatalf("status = %d, want %d; body=%q", recorder.Code, http.StatusFound, recorder.Body.String())
	}

	location := recorder.Header().Get("Location")
	if !strings.HasPrefix(location, "ms-excel:ofv|u|") {
		t.Fatalf("redirect location = %q, want ms-excel prefix", location)
	}
	if !strings.Contains(location, "http://reports.local/reports/download?path=monthly%2Fabc.xlsx") {
		t.Fatalf("redirect location = %q, want absolute download URL", location)
	}
}

func TestReportDownloadServesFile(t *testing.T) {
	root := t.TempDir()
	filePath := filepath.Join(root, "monthly", "abc.xlsx")
	if err := os.MkdirAll(filepath.Dir(filePath), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filePath, []byte("workbook"), 0o644); err != nil {
		t.Fatal(err)
	}

	app := mustNewApplicationWithReportsRoot(root)
	recorder := httptest.NewRecorder()
	app.routes().ServeHTTP(
		recorder,
		httptest.NewRequest(http.MethodGet, "/reports/download?path=monthly/abc.xlsx", nil),
	)

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body=%q", recorder.Code, http.StatusOK, recorder.Body.String())
	}
	if got := recorder.Body.String(); got != "workbook" {
		t.Fatalf("body = %q, want workbook", got)
	}
	if got := recorder.Header().Get("Content-Disposition"); got != `inline; filename=abc.xlsx` {
		t.Fatalf("Content-Disposition = %q, want inline filename", got)
	}
}

func TestReportDownloadRejectsInvalidPaths(t *testing.T) {
	root := t.TempDir()
	app := mustNewApplicationWithReportsRoot(root)

	tests := []struct {
		name       string
		queryPath  string
		wantStatus int
	}{
		{
			name:       "missing path",
			queryPath:  "",
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "parent traversal",
			queryPath:  "../secret.xlsx",
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "nested traversal",
			queryPath:  "monthly/../../../secret.xlsx",
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "slash absolute path",
			queryPath:  "/etc/passwd",
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "windows drive absolute path",
			queryPath:  `C:\Users\Kelvin\secret.txt`,
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "unc absolute path",
			queryPath:  `\\other-server\share\secret.xlsx`,
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "missing file",
			queryPath:  "monthly/missing.xlsx",
			wantStatus: http.StatusNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			target := "/reports/download"
			if tt.queryPath != "" {
				target += "?path=" + url.QueryEscape(tt.queryPath)
			}

			recorder := httptest.NewRecorder()
			app.routes().ServeHTTP(
				recorder,
				httptest.NewRequest(http.MethodGet, target, nil),
			)

			if recorder.Code != tt.wantStatus {
				t.Fatalf("status = %d, want %d; body=%q", recorder.Code, tt.wantStatus, recorder.Body.String())
			}
		})
	}
}

func TestReportDownloadRejectsDirectory(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "monthly"), 0o755); err != nil {
		t.Fatal(err)
	}

	app := mustNewApplicationWithReportsRoot(root)
	recorder := httptest.NewRecorder()
	app.routes().ServeHTTP(
		recorder,
		httptest.NewRequest(http.MethodGet, "/reports/download?path=monthly", nil),
	)

	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d; body=%q", recorder.Code, http.StatusBadRequest, recorder.Body.String())
	}
}

func TestReportDownloadRequiresConfiguredRoot(t *testing.T) {
	tests := []struct {
		name        string
		reportsRoot string
	}{
		{name: "missing root", reportsRoot: ""},
		{name: "relative root", reportsRoot: "reports"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			app := mustNewApplicationWithReportsRoot(tt.reportsRoot)
			recorder := httptest.NewRecorder()
			app.routes().ServeHTTP(
				recorder,
				httptest.NewRequest(http.MethodGet, "/reports/download?path=monthly/abc.xlsx", nil),
			)

			if recorder.Code != http.StatusInternalServerError {
				t.Fatalf("status = %d, want %d; body=%q", recorder.Code, http.StatusInternalServerError, recorder.Body.String())
			}
		})
	}
}

func mustNewApplicationWithReportsRoot(reportsRoot string) *application {
	app := mustNewApplication(":0", "", "", nil)
	app.reportsRoot = reportsRoot
	return app
}

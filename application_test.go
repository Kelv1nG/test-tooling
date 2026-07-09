package main

import (
	"io"
	"log"
	"net/http"
	"net/http/httptest"
	"testing"
)

func mustNewApplication(
	listenAddr string,
	definitionsPath string,
	workbookPath string,
	logger *log.Logger,
) *application {
	if logger == nil {
		logger = log.New(io.Discard, "", 0)
	}

	return NewApplication(listenAddr, definitionsPath, workbookPath, "", logger)
}

func TestWorkflowTabRoutes(t *testing.T) {
	app := mustNewApplication(":0", "", "", nil)
	handler := app.routes()

	redirectRecorder := httptest.NewRecorder()
	handler.ServeHTTP(
		redirectRecorder,
		httptest.NewRequest(http.MethodGet, "/?tab=checking", nil),
	)
	if redirectRecorder.Code != http.StatusSeeOther {
		t.Fatalf("root status = %d, want %d", redirectRecorder.Code, http.StatusSeeOther)
	}
	if location := redirectRecorder.Header().Get("Location"); location != "/checking" {
		t.Fatalf("root redirect location = %q, want /checking", location)
	}

	for _, path := range []string{"/configuration", "/file-transfer", "/checking"} {
		t.Run(path, func(t *testing.T) {
			recorder := httptest.NewRecorder()
			handler.ServeHTTP(
				recorder,
				httptest.NewRequest(http.MethodGet, path, nil),
			)
			if recorder.Code != http.StatusOK {
				t.Fatalf("%s status = %d, want %d", path, recorder.Code, http.StatusOK)
			}
		})
	}

	notFoundRecorder := httptest.NewRecorder()
	handler.ServeHTTP(
		notFoundRecorder,
		httptest.NewRequest(http.MethodGet, "/not-a-tab", nil),
	)
	if notFoundRecorder.Code != http.StatusNotFound {
		t.Fatalf("unknown route status = %d, want %d", notFoundRecorder.Code, http.StatusNotFound)
	}
}

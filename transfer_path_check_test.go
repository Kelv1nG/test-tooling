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

func TestHandleTransferPathCheckRefreshesPostedRowsForReferenceDate(t *testing.T) {
	root := t.TempDir()
	sourcePath := filepath.Join(root, "source_2026_06.txt")
	if err := os.WriteFile(sourcePath, []byte("source"), 0o644); err != nil {
		t.Fatal(err)
	}

	form := url.Values{
		"definitionsPath":  []string{"unused-definitions.yaml"},
		"workbookPath":     []string{"unused-workbook.xlsx"},
		"activeTab":        []string{tabFileTransfer},
		"strategy":         []string{"overwrite"},
		"referenceDate":    []string{"2026-06-09"},
		"transferExcelRow": []string{"0"},
		"transferSrc":      []string{filepath.ToSlash(filepath.Join(root, "source_{yyyy}_{mm}.txt"))},
		"transferDest":     []string{filepath.ToSlash(filepath.Join(root, "destination_{yyyy}_{mm}.txt"))},
	}
	request := httptest.NewRequest(
		http.MethodPost,
		"/transfer/check-paths",
		strings.NewReader(form.Encode()),
	)
	request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	request.Header.Set("HX-Request", "true")
	request.Header.Set("HX-Target", "file-transfer-panel")

	recorder := httptest.NewRecorder()
	app := mustNewApplication(
		":0",
		"table-definitions.yaml",
		"configuration-example.xlsx",
		nil,
	)
	app.routes().ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body=%q", recorder.Code, http.StatusOK, recorder.Body.String())
	}

	body := recorder.Body.String()
	if strings.Contains(body, `id="app-shell"`) {
		t.Fatal("path refresh unexpectedly rendered the full app shell")
	}
	if !strings.Contains(body, `id="file-transfer-panel"`) {
		t.Fatal("path refresh did not render the file transfer panel")
	}
	if !strings.Contains(body, `bg-emerald-100 px-3 py-1 text-xs font-semibold text-emerald-700">Yes</span>`) {
		t.Fatalf("expected source existence badge to be Yes; body=%q", body)
	}
	if !strings.Contains(body, `bg-slate-100 px-3 py-1 text-xs font-semibold text-slate-600">No</span>`) {
		t.Fatalf("expected destination existence badge to be No; body=%q", body)
	}
}

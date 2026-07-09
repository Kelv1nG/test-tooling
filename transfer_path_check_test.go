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
		"definitionsPath":  []string{"table-definitions.yaml"},
		"workbookPath":     []string{"configuration-example.xlsx"},
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
	if !strings.Contains(body, `bg-emerald-100 px-3 py-1 text-xs font-semibold text-emerald-700">Yes</span>`) {
		t.Fatalf("expected source existence badge to be Yes; body=%q", body)
	}
	if !strings.Contains(body, `bg-slate-100 px-3 py-1 text-xs font-semibold text-slate-600">No</span>`) {
		t.Fatalf("expected destination existence badge to be No; body=%q", body)
	}
}

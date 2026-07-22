package main

import (
	"compress/gzip"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestGzipMiddlewareCompressesHTMLAndPreservesResponse(t *testing.T) {
	t.Parallel()

	body := strings.Repeat("<p>responsive HTML</p>", 40)
	handler := gzipMiddleware(http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
		writer.Header().Set("Content-Type", "text/html; charset=utf-8")
		writer.Header().Set("Content-Length", "9999")
		writer.Header().Set("X-Response", "preserved")
		writer.WriteHeader(http.StatusCreated)
		_, _ = io.WriteString(writer, body)
	}))

	request := httptest.NewRequest(http.MethodGet, "/", nil)
	request.Header.Set("Accept-Encoding", "br, gzip;q=0.8")
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, request)

	response := recorder.Result()
	defer response.Body.Close()
	if response.StatusCode != http.StatusCreated {
		t.Fatalf("status = %d, want %d", response.StatusCode, http.StatusCreated)
	}
	if got := response.Header.Get("Content-Encoding"); got != "gzip" {
		t.Fatalf("Content-Encoding = %q, want gzip", got)
	}
	if got := response.Header.Get("Content-Length"); got != "" {
		t.Fatalf("Content-Length = %q, want empty", got)
	}
	if got := response.Header.Get("X-Response"); got != "preserved" {
		t.Fatalf("X-Response = %q, want preserved", got)
	}
	if got := response.Header.Get("Vary"); got != "Accept-Encoding" {
		t.Fatalf("Vary = %q, want Accept-Encoding", got)
	}

	compressed, err := gzip.NewReader(response.Body)
	if err != nil {
		t.Fatalf("open gzip body: %v", err)
	}
	defer compressed.Close()
	decoded, err := io.ReadAll(compressed)
	if err != nil {
		t.Fatalf("read gzip body: %v", err)
	}
	if got := string(decoded); got != body {
		t.Fatalf("decoded body = %q, want %q", got, body)
	}
}

func TestGzipMiddlewareUsesDetectedTextContentType(t *testing.T) {
	t.Parallel()

	handler := gzipMiddleware(http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
		_, _ = io.WriteString(writer, "plain text without an explicit content type")
	}))
	request := httptest.NewRequest(http.MethodGet, "/", nil)
	request.Header.Set("Accept-Encoding", "gzip")
	recorder := httptest.NewRecorder()

	handler.ServeHTTP(recorder, request)

	if got := recorder.Header().Get("Content-Encoding"); got != "gzip" {
		t.Fatalf("Content-Encoding = %q, want gzip", got)
	}
	if got := recorder.Header().Get("Content-Type"); !strings.HasPrefix(got, "text/plain") {
		t.Fatalf("Content-Type = %q, want text/plain", got)
	}
}

func TestGzipMiddlewareHonorsDisabledGzip(t *testing.T) {
	t.Parallel()

	for _, acceptEncoding := range []string{"", "br", "gzip;q=0", "gzip;q=0, *;q=1"} {
		acceptEncoding := acceptEncoding
		t.Run(acceptEncoding, func(t *testing.T) {
			t.Parallel()
			const body = "an uncompressed HTML response"
			handler := gzipMiddleware(http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
				writer.Header().Set("Content-Type", "text/html")
				_, _ = io.WriteString(writer, body)
			}))
			request := httptest.NewRequest(http.MethodGet, "/", nil)
			if acceptEncoding != "" {
				request.Header.Set("Accept-Encoding", acceptEncoding)
			}
			recorder := httptest.NewRecorder()

			handler.ServeHTTP(recorder, request)

			if got := recorder.Header().Get("Content-Encoding"); got != "" {
				t.Fatalf("Content-Encoding = %q, want empty", got)
			}
			if got := recorder.Body.String(); got != body {
				t.Fatalf("body = %q, want %q", got, body)
			}
			if got := recorder.Header().Get("Vary"); got != "Accept-Encoding" {
				t.Fatalf("Vary = %q, want Accept-Encoding", got)
			}
		})
	}
}

func TestGzipMiddlewareLeavesDownloadsUncompressed(t *testing.T) {
	t.Parallel()

	for _, test := range []struct {
		name               string
		contentType        string
		contentDisposition string
	}{
		{name: "xlsx", contentType: "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet"},
		{name: "text attachment", contentType: "text/csv", contentDisposition: `attachment; filename="report.csv"`},
	} {
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			const body = "download bytes"
			handler := gzipMiddleware(http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
				writer.Header().Set("Content-Type", test.contentType)
				if test.contentDisposition != "" {
					writer.Header().Set("Content-Disposition", test.contentDisposition)
				}
				_, _ = io.WriteString(writer, body)
			}))
			request := httptest.NewRequest(http.MethodGet, "/reports/download", nil)
			request.Header.Set("Accept-Encoding", "gzip")
			recorder := httptest.NewRecorder()

			handler.ServeHTTP(recorder, request)

			if got := recorder.Header().Get("Content-Encoding"); got != "" {
				t.Fatalf("Content-Encoding = %q, want empty", got)
			}
			if got := recorder.Body.String(); got != body {
				t.Fatalf("body = %q, want %q", got, body)
			}
		})
	}
}

func TestGzipMiddlewareAppendsVaryOnce(t *testing.T) {
	t.Parallel()

	handler := gzipMiddleware(http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
		writer.Header().Add("Vary", "HX-Request")
		writer.Header().Add("Vary", "accept-encoding")
		writer.Header().Set("Content-Type", "text/html")
		_, _ = io.WriteString(writer, "hello")
	}))
	request := httptest.NewRequest(http.MethodGet, "/", nil)
	request.Header.Set("Accept-Encoding", "gzip")
	recorder := httptest.NewRecorder()

	handler.ServeHTTP(recorder, request)

	values := strings.ToLower(strings.Join(recorder.Header().Values("Vary"), ","))
	if strings.Count(values, "accept-encoding") != 1 {
		t.Fatalf("Vary = %q, want Accept-Encoding exactly once", values)
	}
	if !strings.Contains(values, "hx-request") {
		t.Fatalf("Vary = %q, want existing HX-Request value", values)
	}
}

func TestGzipMiddlewareDoesNotCreateBodyForBodylessStatus(t *testing.T) {
	t.Parallel()

	handler := gzipMiddleware(http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
		writer.Header().Set("Content-Type", "text/html")
		writer.WriteHeader(http.StatusNoContent)
	}))
	request := httptest.NewRequest(http.MethodGet, "/", nil)
	request.Header.Set("Accept-Encoding", "gzip")
	recorder := httptest.NewRecorder()

	handler.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusNoContent {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusNoContent)
	}
	if recorder.Body.Len() != 0 {
		t.Fatalf("body length = %d, want 0", recorder.Body.Len())
	}
	if got := recorder.Header().Get("Content-Encoding"); got != "" {
		t.Fatalf("Content-Encoding = %q, want empty", got)
	}
}

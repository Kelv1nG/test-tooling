package main

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"tooling/templates"
)

func TestRenderResponseScopesHTMXTargets(t *testing.T) {
	app := mustNewApplication(":0", "", "", nil)
	data := templates.PageData{
		ActiveTab:          tabChecking,
		HasConfig:          true,
		CheckReferenceDate: "2026-06-30",
	}

	tests := []struct {
		name      string
		target    string
		want      string
		doNotWant string
	}{
		{
			name:      "configuration panel",
			target:    "configuration-panel",
			want:      `id="configuration-panel"`,
			doNotWant: `id="app-shell"`,
		},
		{
			name:      "file transfer panel",
			target:    "file-transfer-panel",
			want:      `id="file-transfer-panel"`,
			doNotWant: `id="app-shell"`,
		},
		{
			name:      "checking panel",
			target:    "checking-panel",
			want:      `id="checking-panel"`,
			doNotWant: `id="app-shell"`,
		},
		{
			name:      "verification progress",
			target:    "check-run-progress",
			want:      `id="check-run-progress"`,
			doNotWant: `id="checking-panel"`,
		},
		{
			name:      "unknown target falls back to app shell",
			target:    "app-shell",
			want:      `id="app-shell"`,
			doNotWant: "",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			request := httptest.NewRequest(http.MethodGet, "/checking", nil)
			request.Header.Set("HX-Request", "true")
			request.Header.Set("HX-Target", test.target)
			recorder := httptest.NewRecorder()

			app.renderResponse(recorder, request, data, http.StatusOK)

			body := recorder.Body.String()
			if !strings.Contains(body, test.want) {
				t.Fatalf("response does not contain %q", test.want)
			}
			if test.doNotWant != "" && strings.Contains(body, test.doNotWant) {
				t.Fatalf("response unexpectedly contains %q", test.doNotWant)
			}
		})
	}
}

func TestVerificationStatusPollScopesRunningAndCompletedResponses(t *testing.T) {
	app := mustNewApplication(":0", "", "", nil)
	base := templates.PageData{
		ActiveTab:          tabChecking,
		HasConfig:          true,
		CheckReferenceDate: "2026-06-30",
	}
	rows := []templates.CheckRowView{{Index: 1, ID: "CHK-001"}}
	job := newVerificationJob("job-responsive", base, rows)

	app.verificationJobsMu.Lock()
	app.verificationJobs[job.id] = job
	app.verificationJobsMu.Unlock()

	runningRecorder := performVerificationStatusPoll(app, job.id)
	if cacheControl := runningRecorder.Header().Get("Cache-Control"); cacheControl != "no-store" {
		t.Fatalf("running poll Cache-Control = %q, want no-store", cacheControl)
	}
	if retarget := runningRecorder.Header().Get("HX-Retarget"); retarget != "" {
		t.Fatalf("running poll retarget = %q, want empty", retarget)
	}
	runningBody := runningRecorder.Body.String()
	if !strings.Contains(runningBody, `id="check-run-progress"`) {
		t.Fatal("running poll did not render the progress panel")
	}
	if !strings.Contains(runningBody, `hx-trigger="load delay:1s"`) {
		t.Fatal("running poll did not schedule the next status request")
	}
	if !strings.Contains(runningBody, "/verify-checks/status?id=job-responsive") {
		t.Fatal("running poll did not retain its status URL")
	}
	if strings.Contains(runningBody, `id="checking-panel"`) {
		t.Fatal("running poll unexpectedly rendered the full checking panel")
	}

	job.finish(rows, templates.CheckSummaryView{Attempted: 1, Matched: 1, HasRun: true})

	completedRecorder := performVerificationStatusPoll(app, job.id)
	if retarget := completedRecorder.Header().Get("HX-Retarget"); retarget != "#checking-panel" {
		t.Fatalf("completed poll retarget = %q, want %q", retarget, "#checking-panel")
	}
	completedBody := completedRecorder.Body.String()
	if !strings.Contains(completedBody, `id="checking-panel"`) {
		t.Fatal("completed poll did not render the checking panel")
	}
	if strings.Contains(completedBody, `id="app-shell"`) {
		t.Fatal("completed poll unexpectedly rendered the full app shell")
	}
	if strings.Contains(completedBody, `hx-trigger="load delay:1s"`) {
		t.Fatal("completed poll unexpectedly scheduled another status request")
	}
}

func TestVerificationStatusErrorsAreVisibleAndTerminal(t *testing.T) {
	app := mustNewApplication(":0", "", "", nil)
	tests := []struct {
		name          string
		path          string
		wantMessage   string
		nonHTMXStatus int
	}{
		{
			name:          "missing job ID",
			path:          "/verify-checks/status",
			wantMessage:   "Verification run is missing a job ID.",
			nonHTMXStatus: http.StatusBadRequest,
		},
		{
			name:          "unknown job ID",
			path:          "/verify-checks/status?id=unknown",
			wantMessage:   "Verification run was not found. Start verification again.",
			nonHTMXStatus: http.StatusNotFound,
		},
	}

	for _, test := range tests {
		for _, htmx := range []bool{false, true} {
			name := "full page"
			if htmx {
				name = "HTMX"
			}
			t.Run(test.name+"/"+name, func(t *testing.T) {
				request := httptest.NewRequest(http.MethodGet, test.path, nil)
				if htmx {
					request.Header.Set("HX-Request", "true")
					request.Header.Set("HX-Target", "check-run-progress")
				}
				recorder := httptest.NewRecorder()

				app.handleVerifyChecksStatus(recorder, request)

				wantStatus := test.nonHTMXStatus
				if htmx {
					wantStatus = http.StatusOK
				}
				if recorder.Code != wantStatus {
					t.Fatalf("status = %d, want %d", recorder.Code, wantStatus)
				}
				if got := recorder.Header().Get("Cache-Control"); got != "no-store" {
					t.Fatalf("Cache-Control = %q, want no-store", got)
				}
				if htmx && recorder.Header().Get("HX-Retarget") != "#checking-panel" {
					t.Fatalf("HX-Retarget = %q, want #checking-panel", recorder.Header().Get("HX-Retarget"))
				}

				body := recorder.Body.String()
				if !strings.Contains(body, test.wantMessage) {
					t.Fatalf("response does not contain %q", test.wantMessage)
				}
				if strings.Contains(body, `hx-trigger="load delay:1s"`) {
					t.Fatal("terminal error response unexpectedly scheduled another poll")
				}
				if strings.Contains(body, `action="/save-checks"`) {
					t.Fatal("terminal error response exposed an empty save form")
				}
				if !htmx && !strings.Contains(body, "<!doctype html>") {
					t.Fatal("non-HTMX error did not render a full page")
				}
			})
		}
	}
}

func TestWorkflowTabsNavigateLocallyWithLinkFallbacks(t *testing.T) {
	var output strings.Builder
	data := templates.PageData{ActiveTab: tabConfiguration}
	if err := templates.Page(data).Render(context.Background(), &output); err != nil {
		t.Fatalf("render page: %v", err)
	}

	body := output.String()
	for _, path := range []string{"/configuration", "/file-transfer", "/checking"} {
		if !strings.Contains(body, `href="`+path+`"`) {
			t.Fatalf("page does not contain link fallback for %s", path)
		}
		if strings.Contains(body, `hx-get="`+path+`"`) {
			t.Fatalf("tab %s still performs an HTMX navigation", path)
		}
	}
	if strings.Contains(body, "hx-push-url") {
		t.Fatal("tab markup still contains HTMX history handling")
	}
	if !strings.Contains(body, "navigateTab('checking', $event)") {
		t.Fatal("tab markup does not use local navigation")
	}
	if !strings.Contains(body, `action="/load"`) || !strings.Contains(body, `hx-target="#app-shell"`) {
		t.Fatal("workbook load no longer refreshes the cross-panel app shell")
	}
}

func performVerificationStatusPoll(
	app *application,
	jobID string,
) *httptest.ResponseRecorder {
	request := httptest.NewRequest(
		http.MethodGet,
		"/verify-checks/status?id="+jobID,
		nil,
	)
	request.Header.Set("HX-Request", "true")
	request.Header.Set("HX-Target", "check-run-progress")
	recorder := httptest.NewRecorder()
	app.handleVerifyChecksStatus(recorder, request)
	return recorder
}

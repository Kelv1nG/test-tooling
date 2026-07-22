package main

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"strconv"
	"sync"
	"time"

	"tooling/templates"
)

const verificationJobTTL = 30 * time.Minute

type verificationJob struct {
	mu        sync.Mutex
	id        string
	base      templates.PageData
	rows      []templates.CheckRowView
	summary   templates.CheckSummaryView
	total     int
	completed int
	done      bool
	createdAt time.Time
	updatedAt time.Time
}

func newVerificationJob(
	id string,
	base templates.PageData,
	rows []templates.CheckRowView,
) *verificationJob {
	now := time.Now()
	rows = cloneCheckRows(rows)
	base.CheckRows = cloneCheckRows(rows)
	base.CheckCount = len(rows)
	base.CheckSummary = templates.CheckSummaryView{HasRun: true}
	base.CheckSummaryRows = buildCheckSummaryRows(rows)
	base.CheckRunID = id
	base.CheckRunCompleted = 0
	base.CheckRunTotal = len(rows)
	base.CheckRunRunning = true
	base.CheckHasIssues = false
	base.CheckMessage = formatVerificationRunningMessage(0, len(rows))

	return &verificationJob{
		id:        id,
		base:      base,
		rows:      rows,
		summary:   templates.CheckSummaryView{HasRun: true},
		total:     len(rows),
		createdAt: now,
		updatedAt: now,
	}
}

func (a *application) startVerificationJob(
	base templates.PageData,
	rows []templates.CheckRowView,
	referenceDate time.Time,
) *verificationJob {
	a.pruneVerificationJobs(time.Now().Add(-verificationJobTTL))

	job := newVerificationJob(newVerificationJobID(), base, rows)
	a.verificationJobsMu.Lock()
	a.verificationJobs[job.id] = job
	a.verificationJobsMu.Unlock()

	go job.run(referenceDate)
	return job
}

func (a *application) verificationJob(
	id string,
) (*verificationJob, bool) {
	a.verificationJobsMu.Lock()
	defer a.verificationJobsMu.Unlock()

	job, ok := a.verificationJobs[id]
	return job, ok
}

func (a *application) pruneVerificationJobs(cutoff time.Time) {
	a.verificationJobsMu.Lock()
	defer a.verificationJobsMu.Unlock()

	for id, job := range a.verificationJobs {
		if job.isOlderThan(cutoff) {
			delete(a.verificationJobs, id)
		}
	}
}

func (j *verificationJob) run(referenceDate time.Time) {
	rows, summary := runCheckVerificationWithProgress(
		cloneCheckRows(j.rows),
		referenceDate,
		func(progress checkVerificationProgress) {
			j.applyProgress(progress)
		},
	)
	j.finish(rows, summary)
}

func (j *verificationJob) applyProgress(
	progress checkVerificationProgress,
) {
	j.mu.Lock()
	defer j.mu.Unlock()

	if progress.Index >= 0 && progress.Index < len(j.rows) {
		j.rows[progress.Index] = cloneCheckRow(progress.Row)
	}
	j.summary = progress.Summary
	j.summary.HasRun = true
	j.completed = progress.Completed
	j.updatedAt = time.Now()
}

func (j *verificationJob) finish(
	rows []templates.CheckRowView,
	summary templates.CheckSummaryView,
) {
	j.mu.Lock()
	defer j.mu.Unlock()

	j.rows = cloneCheckRows(rows)
	j.summary = summary
	j.summary.HasRun = true
	j.completed = len(rows)
	j.done = true
	j.updatedAt = time.Now()
}

func (j *verificationJob) pageData() templates.PageData {
	j.mu.Lock()
	defer j.mu.Unlock()

	data := j.base
	data.CheckRows = cloneCheckRows(j.rows)
	data.CheckCount = len(j.rows)
	data.CheckSummary = j.summary
	data.CheckSummary.HasRun = true
	data.CheckSummaryRows = buildCheckSummaryRows(data.CheckRows)
	data.CheckRunID = j.id
	data.CheckRunCompleted = j.completed
	data.CheckRunTotal = j.total
	data.CheckRunRunning = !j.done
	data.CheckHasIssues = checkSummaryHasIssues(j.summary)

	if j.done {
		data.CheckMessage = formatVerificationCompleteMessage(j.summary)
	} else {
		data.CheckMessage = formatVerificationRunningMessage(j.completed, j.total)
	}

	return data
}

func (j *verificationJob) progressData() templates.PageData {
	j.mu.Lock()
	defer j.mu.Unlock()

	return templates.PageData{
		CheckRunID:        j.id,
		CheckRunCompleted: j.completed,
		CheckRunTotal:     j.total,
		CheckRunRunning:   !j.done,
		CheckSummary:      j.summary,
	}
}

func (j *verificationJob) isOlderThan(cutoff time.Time) bool {
	j.mu.Lock()
	defer j.mu.Unlock()

	return j.done && j.updatedAt.Before(cutoff)
}

func newVerificationJobID() string {
	bytes := make([]byte, 16)
	if _, err := rand.Read(bytes); err == nil {
		return hex.EncodeToString(bytes)
	}

	return strconv.FormatInt(time.Now().UnixNano(), 36)
}

func formatVerificationRunningMessage(
	completed int,
	total int,
) string {
	return fmt.Sprintf(
		"Verification is running: %d of %d file check(s) complete.",
		completed,
		total,
	)
}

func formatVerificationCompleteMessage(
	summary templates.CheckSummaryView,
) string {
	return fmt.Sprintf(
		"Verification checked %d rule(s): matched %d, changed %d, errors %d, skipped %d.",
		summary.Attempted,
		summary.Matched,
		summary.Changed,
		summary.Errors,
		summary.Skipped,
	)
}

func checkSummaryHasIssues(
	summary templates.CheckSummaryView,
) bool {
	return summary.Changed > 0 || summary.Errors > 0
}

func cloneCheckRows(
	rows []templates.CheckRowView,
) []templates.CheckRowView {
	if rows == nil {
		return nil
	}

	cloned := make([]templates.CheckRowView, len(rows))
	for index := range rows {
		cloned[index] = cloneCheckRow(rows[index])
	}

	return cloned
}

func cloneCheckRow(
	row templates.CheckRowView,
) templates.CheckRowView {
	row.Rules = append([]templates.CheckRuleView(nil), row.Rules...)
	return row
}

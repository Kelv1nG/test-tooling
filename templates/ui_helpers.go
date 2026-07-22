package templates

import (
	"fmt"
	"net/url"
	"path"
	"strings"

	"github.com/a-h/templ"
)

const (
	transferRowsPerPage = 10
	checkRowsPerPage    = 10
	summaryRowsPerPage  = 10
)

func appShellExpression(activeTab string) string {
	return fmt.Sprintf("appShell(%q)", activeTab)
}

func strategyStateExpression(
	strategy string,
	referenceDate string,
	transferPage int,
	summaryHasRun bool,
) string {
	if transferPage < 1 {
		transferPage = 1
	}
	tableView := "configured"
	if summaryHasRun {
		tableView = "summary"
	}

	return fmt.Sprintf(
		`{
			strategy: %q,
			referenceDate: %q,
			transferTableView: %q,
			transferSearchQuery: "",
			transferDestExistsFilter: "all",
			transferPage: %d,
			transferPageSize: %d,
			transferSummaryFilter: "all",
			transferSummaryPage: 1,
			transferSummaryPageSize: %d,
			transferRowsVersion: 0,
			init() {
				this.goToTransferPage(this.transferPage);
				this.goToTransferSummaryPage(this.transferSummaryPage);
			},
			get transferFilteredRows() { this.transferRowsVersion; return window.transferFilteredRows(this.$root, this.transferDestExistsFilter, this.transferSearchQuery); },
			get transferCount() { return this.transferFilteredRows.length; },
			get transferPageCount() { return Math.max(1, Math.ceil(this.transferCount / this.transferPageSize)); },
			get transferPageStart() { return this.transferCount === 0 ? 0 : ((this.transferPage - 1) * this.transferPageSize) + 1; },
			get transferPageEnd() { return Math.min(this.transferPage * this.transferPageSize, this.transferCount); },
			get transferSummaryFilteredRows() { return window.summaryFilteredRows(this.$root, "[data-transfer-summary-row]", this.transferSummaryFilter); },
			get transferSummaryCount() { return this.transferSummaryFilteredRows.length; },
			get transferSummaryPageCount() { return Math.max(1, Math.ceil(this.transferSummaryCount / this.transferSummaryPageSize)); },
			get transferSummaryPageStart() { return this.transferSummaryCount === 0 ? 0 : ((this.transferSummaryPage - 1) * this.transferSummaryPageSize) + 1; },
			get transferSummaryPageEnd() { return Math.min(this.transferSummaryPage * this.transferSummaryPageSize, this.transferSummaryCount); },
			setTransferDestExistsFilter(filter) {
				this.transferDestExistsFilter = filter;
				this.goToTransferPage(1);
			},
			goToTransferPage(page) {
				const nextPage = Number(page) || 1;
				this.transferPage = Math.min(Math.max(nextPage, 1), this.transferPageCount);
			},
			refreshTransferPagination() {
				this.transferRowsVersion += 1;
				this.goToTransferPage(this.transferPage);
			},
			transferRowVisible(row) {
				this.transferRowsVersion;
				return window.summaryRowVisible(row, this.transferFilteredRows, this.transferPage, this.transferPageSize);
			},
			setTransferSummaryFilter(filter) {
				this.transferSummaryFilter = filter;
				this.goToTransferSummaryPage(1);
			},
			goToTransferSummaryPage(page) {
				const nextPage = Number(page) || 1;
				this.transferSummaryPage = Math.min(Math.max(nextPage, 1), this.transferSummaryPageCount);
			},
			transferSummaryRowVisible(row) {
				return window.summaryRowVisible(row, this.transferSummaryFilteredRows, this.transferSummaryPage, this.transferSummaryPageSize);
			}
		}`,
		strategy,
		referenceDate,
		tableView,
		transferPage,
		transferRowsPerPage,
		summaryRowsPerPage,
	)
}

func transferPathFieldExpression(value string) string {
	return fmt.Sprintf(`{ value: %q }`, value)
}

func transferDestExistsValue(exists bool) string {
	if exists {
		return "yes"
	}

	return "no"
}

func transferRowSearchText(row TransferRowView) string {
	parts := []string{
		row.Src,
		row.ResolvedSrc,
		row.Dest,
		row.ResolvedDest,
	}

	return strings.Join(parts, " ")
}

func checkConfigSearchText(row CheckRowView) string {
	parts := []string{
		row.File,
		row.ResolvedFile,
		row.ResolvedCompareFile,
	}

	return strings.Join(parts, " ")
}

func referenceDateStateExpression(referenceDate string) string {
	return fmt.Sprintf(`{ referenceDate: %q }`, referenceDate)
}

func checkingTabStateExpression(
	referenceDate string,
	checkCount int,
	checkPage int,
	showSummary bool,
) string {
	if checkPage < 1 {
		checkPage = 1
	}
	checkView := "checks"
	if showSummary {
		checkView = "summary"
	}

	return fmt.Sprintf(
		`{
			referenceDate: %q,
			checkView: %q,
			checkSearchQuery: "",
			checkSummaryFilter: "all",
			checkSummaryPage: 1,
			checkSummaryPageSize: %d,
			checkPage: %d,
			checkPageSize: %d,
			checkRowsVersion: 0,
			init() {
				this.goToCheckPage(this.checkPage);
				this.goToCheckSummaryPage(this.checkSummaryPage);
			},
			get checkFilteredRows() { this.checkRowsVersion; return window.checkFilteredRows(this.$root, this.checkSearchQuery); },
			get checkCount() { return this.checkFilteredRows.length; },
			get checkPageCount() { return Math.max(1, Math.ceil(this.checkCount / this.checkPageSize)); },
			get checkPageStart() { return this.checkCount === 0 ? 0 : ((this.checkPage - 1) * this.checkPageSize) + 1; },
			get checkPageEnd() { return Math.min(this.checkPage * this.checkPageSize, this.checkCount); },
			get checkSummaryFilteredRows() { return window.summaryFilteredRows(this.$root, "[data-check-summary-row]", this.checkSummaryFilter); },
			get checkSummaryCount() { return this.checkSummaryFilteredRows.length; },
			get checkSummaryPageCount() { return Math.max(1, Math.ceil(this.checkSummaryCount / this.checkSummaryPageSize)); },
			get checkSummaryPageStart() { return this.checkSummaryCount === 0 ? 0 : ((this.checkSummaryPage - 1) * this.checkSummaryPageSize) + 1; },
			get checkSummaryPageEnd() { return Math.min(this.checkSummaryPage * this.checkSummaryPageSize, this.checkSummaryCount); },
			goToCheckPage(page) {
				const nextPage = Number(page) || 1;
				this.checkPage = Math.min(Math.max(nextPage, 1), this.checkPageCount);
			},
			refreshCheckPagination() {
				this.checkRowsVersion += 1;
				this.goToCheckPage(this.checkPage);
			},
			checkConfigVisible(config) {
				this.checkRowsVersion;
				return window.summaryRowVisible(config, this.checkFilteredRows, this.checkPage, this.checkPageSize);
			},
			setCheckSummaryFilter(filter) {
				this.checkSummaryFilter = filter;
				this.goToCheckSummaryPage(1);
			},
			goToCheckSummaryPage(page) {
				const nextPage = Number(page) || 1;
				this.checkSummaryPage = Math.min(Math.max(nextPage, 1), this.checkSummaryPageCount);
			},
			checkSummaryRowVisible(row) {
				return window.summaryRowVisible(row, this.checkSummaryFilteredRows, this.checkSummaryPage, this.checkSummaryPageSize);
			}
		}`,
		referenceDate,
		checkView,
		summaryRowsPerPage,
		checkPage,
		checkRowsPerPage,
	)
}

func checkConfigStateExpression(row CheckRowView, expandOnIssues bool) string {
	return fmt.Sprintf(
		`{ editing: false, expanded: %t, file: %q, offset: %d }`,
		checkConfigStartsExpanded(row, expandOnIssues),
		row.File,
		row.CompareOffsetMonths,
	)
}

func checkConfigStartsExpanded(row CheckRowView, expandOnIssues bool) bool {
	if !expandOnIssues {
		return false
	}
	if row.Badge == "amber" || row.Badge == "rose" {
		return true
	}
	for _, rule := range row.Rules {
		if rule.Badge == "amber" || rule.Badge == "rose" {
			return true
		}
	}

	// Form-level validation errors can leave rows without a per-row badge.
	return row.Badge == "" && row.Status == ""
}

func ruleTypeStateExpression(ruleType string) string {
	if ruleType == "" {
		ruleType = "exact_text"
	}

	return fmt.Sprintf(`{ ruleType: %q }`, ruleType)
}

func compareOffsetLabel(months int) string {
	switch months {
	case 0:
		return "No comparison"
	case -1:
		return "Previous month"
	default:
		return fmt.Sprintf("%d months", months)
	}
}

func compareOffsetIsStandard(months int) bool {
	switch months {
	case 0, -1, -2, -3, -6, -12:
		return true
	default:
		return false
	}
}

func saveMessageClasses(hasErrors bool) string {
	if hasErrors {
		return "rounded-[1.5rem] border border-rose-200 bg-rose-50 px-4 py-4 text-sm text-rose-700"
	}

	return "rounded-[1.5rem] border border-emerald-200 bg-emerald-50 px-4 py-4 text-sm text-emerald-800"
}

func transferMessageClasses(hasErrors bool) string {
	if hasErrors {
		return "mb-4 rounded-[1.5rem] border border-amber-200 bg-amber-50 px-4 py-4 text-sm text-amber-800"
	}

	return "mb-4 rounded-[1.5rem] border border-emerald-200 bg-emerald-50 px-4 py-4 text-sm text-emerald-800"
}

func transferStatusClasses(badge string) string {
	switch badge {
	case "emerald":
		return "inline-flex rounded-full bg-emerald-100 px-3 py-1 text-xs font-semibold text-emerald-700"
	case "amber":
		return "inline-flex rounded-full bg-amber-100 px-3 py-1 text-xs font-semibold text-amber-700"
	case "slate":
		return "inline-flex rounded-full bg-slate-100 px-3 py-1 text-xs font-semibold text-slate-700"
	case "rose":
		return "inline-flex rounded-full bg-rose-100 px-3 py-1 text-xs font-semibold text-rose-700"
	case "zinc":
		return "inline-flex rounded-full bg-zinc-100 px-3 py-1 text-xs font-semibold text-zinc-700"
	default:
		return "inline-flex rounded-full bg-slate-100 px-3 py-1 text-xs font-semibold text-slate-700"
	}
}

func checkMessageClasses(hasIssues bool) string {
	if hasIssues {
		return "mb-4 rounded-[1.5rem] border border-amber-200 bg-amber-50 px-4 py-4 text-sm text-amber-800"
	}

	return "mb-4 rounded-[1.5rem] border border-emerald-200 bg-emerald-50 px-4 py-4 text-sm text-emerald-800"
}

func checkRunStatusPath(id string) string {
	return "/verify-checks/status?id=" + url.QueryEscape(id)
}

func checkRunProgressPercent(
	completed int,
	total int,
) int {
	if total <= 0 {
		if completed > 0 {
			return 100
		}
		return 0
	}

	percent := completed * 100 / total
	if percent < 0 {
		return 0
	}
	if percent > 100 {
		return 100
	}

	return percent
}

func checkRunProgressStyle(
	completed int,
	total int,
) string {
	return fmt.Sprintf("width: %d%%", checkRunProgressPercent(completed, total))
}

func reportOpenHref(
	filePath string,
	reportsRoot string,
) string {
	relativePath, ok := reportRelativePath(filePath, reportsRoot)
	if !ok {
		return ""
	}

	return "/reports/open?path=" + url.QueryEscape(relativePath)
}

func safeReportOpenHref(
	filePath string,
	reportsRoot string,
) templ.SafeURL {
	return templ.SafeURL(reportOpenHref(filePath, reportsRoot))
}

func reportRelativePath(
	filePath string,
	reportsRoot string,
) (string, bool) {
	root := cleanReportComparablePath(reportsRoot)
	target := cleanReportComparablePath(filePath)
	if root == "" || target == "" {
		return "", false
	}

	compareRoot := root
	compareTarget := target
	if reportPathUsesWindowsRules(root) || reportPathUsesWindowsRules(target) {
		compareRoot = strings.ToLower(compareRoot)
		compareTarget = strings.ToLower(compareTarget)
	}

	if compareTarget == compareRoot {
		return "", false
	}
	if !strings.HasPrefix(compareTarget, compareRoot+"/") {
		return "", false
	}

	return target[len(root)+1:], true
}

func cleanReportComparablePath(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return ""
	}

	normalized := strings.ReplaceAll(value, `\`, "/")
	isUNC := strings.HasPrefix(normalized, "//")
	cleaned := path.Clean(normalized)
	if cleaned == "." {
		return ""
	}
	if isUNC && !strings.HasPrefix(cleaned, "//") {
		cleaned = "/" + cleaned
	}

	return strings.TrimRight(cleaned, "/")
}

func reportPathUsesWindowsRules(value string) bool {
	if strings.HasPrefix(value, "//") {
		return true
	}
	return len(value) >= 2 && value[1] == ':' && isASCIIAlpha(value[0])
}

func isASCIIAlpha(value byte) bool {
	return (value >= 'A' && value <= 'Z') || (value >= 'a' && value <= 'z')
}

func resizeColumnLabel(label string) string {
	if label == "#" {
		return "Resize number column"
	}

	return "Resize " + label + " column"
}

func summaryColumnMinWidth(value int) string {
	if value <= 0 {
		value = 64
	}

	return fmt.Sprintf("%d", value)
}

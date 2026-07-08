package templates

import (
	"fmt"
	"net/url"
)

const checkRowsPerPage = 10

func appShellExpression(activeTab string) string {
	return fmt.Sprintf("appShell(%q)", activeTab)
}

func strategyStateExpression(
	strategy string,
	referenceDate string,
) string {
	return fmt.Sprintf(
		`{ strategy: %q, referenceDate: %q }`,
		strategy,
		referenceDate,
	)
}

func transferPathFieldExpression(value string) string {
	return fmt.Sprintf(`{ value: %q }`, value)
}

func referenceDateStateExpression(referenceDate string) string {
	return fmt.Sprintf(`{ referenceDate: %q }`, referenceDate)
}

func checkingTabStateExpression(
	referenceDate string,
	checkCount int,
	checkPage int,
) string {
	if checkPage < 1 {
		checkPage = 1
	}

	return fmt.Sprintf(
		`{
			referenceDate: %q,
			checkPage: %d,
			checkPageSize: %d,
			checkCount: %d,
			init() { this.goToCheckPage(this.checkPage); },
			get checkPageCount() { return Math.max(1, Math.ceil(this.checkCount / this.checkPageSize)); },
			get checkPageStart() { return this.checkCount === 0 ? 0 : ((this.checkPage - 1) * this.checkPageSize) + 1; },
			get checkPageEnd() { return Math.min(this.checkPage * this.checkPageSize, this.checkCount); },
			goToCheckPage(page) {
				const nextPage = Number(page) || 1;
				this.checkPage = Math.min(Math.max(nextPage, 1), this.checkPageCount);
			},
			refreshCheckPagination(editor) {
				this.checkCount = editor?.querySelectorAll("[data-check-config]").length ?? this.checkCount;
				this.goToCheckPage(this.checkPage);
			}
		}`,
		referenceDate,
		checkPage,
		checkRowsPerPage,
		checkCount,
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

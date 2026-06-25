package templates

import "fmt"

func appShellExpression(activeTab string) string {
	return fmt.Sprintf("appShell(%q)", activeTab)
}

func strategyStateExpression(strategy string) string {
	return fmt.Sprintf(`{ strategy: %q }`, strategy)
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

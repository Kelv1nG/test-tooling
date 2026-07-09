package templates

import (
	"strings"
	"testing"
)

func TestCheckConfigStartsExpanded(t *testing.T) {
	tests := []struct {
		name           string
		row            CheckRowView
		expandOnIssues bool
		want           bool
	}{
		{
			name: "initial rows stay collapsed",
			row: CheckRowView{
				Badge: "rose",
			},
			expandOnIssues: false,
			want:           false,
		},
		{
			name: "changed row opens after an issue run",
			row: CheckRowView{
				Badge: "amber",
			},
			expandOnIssues: true,
			want:           true,
		},
		{
			name: "errored rule opens its parent row",
			row: CheckRowView{
				Badge: "emerald",
				Rules: []CheckRuleView{
					{Badge: "rose"},
				},
			},
			expandOnIssues: true,
			want:           true,
		},
		{
			name: "matched row stays collapsed after another row has issues",
			row: CheckRowView{
				Badge: "emerald",
			},
			expandOnIssues: true,
			want:           false,
		},
		{
			name:           "form level errors open unbadged rows",
			row:            CheckRowView{},
			expandOnIssues: true,
			want:           true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := checkConfigStartsExpanded(tt.row, tt.expandOnIssues); got != tt.want {
				t.Fatalf("checkConfigStartsExpanded() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestTransferRowHelpers(t *testing.T) {
	if got := transferDestExistsValue(true); got != "yes" {
		t.Fatalf("transferDestExistsValue(true) = %q, want yes", got)
	}
	if got := transferDestExistsValue(false); got != "no" {
		t.Fatalf("transferDestExistsValue(false) = %q, want no", got)
	}

	text := transferRowSearchText(TransferRowView{
		Src:          `\\server\share\May Fargo Something File.xlsx`,
		ResolvedSrc:  `\\server\share\May Fargo Something File.xlsx`,
		Dest:         `D:\reports\May Fargo Something File.xlsx`,
		ResolvedDest: `D:\reports\May Fargo Something File.xlsx`,
	})
	for _, want := range []string{"May Fargo Something File", `D:\reports`} {
		if !strings.Contains(text, want) {
			t.Fatalf("transferRowSearchText() = %q, want it to contain %q", text, want)
		}
	}
}

func TestReportOpenHref(t *testing.T) {
	tests := []struct {
		name        string
		path        string
		reportsRoot string
		want        string
	}{
		{
			name:        "unc path under reports root",
			path:        `\\server\share\reports\monthly\abc.xlsx`,
			reportsRoot: `\\server\share\reports`,
			want:        "/reports/open?path=monthly%2Fabc.xlsx",
		},
		{
			name:        "windows drive path under reports root",
			path:        `Z:\reports\monthly\abc.xlsx`,
			reportsRoot: `z:\reports`,
			want:        "/reports/open?path=monthly%2Fabc.xlsx",
		},
		{
			name:        "path outside reports root is not linked",
			path:        `Z:\other\abc.xlsx`,
			reportsRoot: `Z:\reports`,
			want:        "",
		},
		{
			name:        "empty reports root is not linked",
			path:        `Z:\reports\monthly\abc.xlsx`,
			reportsRoot: "",
			want:        "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := reportOpenHref(tt.path, tt.reportsRoot); got != tt.want {
				t.Fatalf("reportOpenHref(%q, %q) = %q, want %q", tt.path, tt.reportsRoot, got, tt.want)
			}
			if got := string(safeReportOpenHref(tt.path, tt.reportsRoot)); got != tt.want {
				t.Fatalf("safeReportOpenHref(%q, %q) = %q, want %q", tt.path, tt.reportsRoot, got, tt.want)
			}
		})
	}
}

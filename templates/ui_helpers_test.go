package templates

import "testing"

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

func TestFileLinkHref(t *testing.T) {
	tests := []struct {
		name string
		path string
		want string
	}{
		{
			name: "unc network path",
			path: `\\server\share\monthly report.xlsx`,
			want: "file://server/share/monthly%20report.xlsx",
		},
		{
			name: "windows drive path",
			path: `Z:\reports\file.xlsx`,
			want: "file:///Z:/reports/file.xlsx",
		},
		{
			name: "posix absolute path",
			path: "/mnt/share/monthly report.xlsx",
			want: "file:///mnt/share/monthly%20report.xlsx",
		},
		{
			name: "relative path is not linked",
			path: "sample/report.xlsx",
			want: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := fileLinkHref(tt.path); got != tt.want {
				t.Fatalf("fileLinkHref(%q) = %q, want %q", tt.path, got, tt.want)
			}
			if got := string(safeFileLinkHref(tt.path)); got != tt.want {
				t.Fatalf("safeFileLinkHref(%q) = %q, want %q", tt.path, got, tt.want)
			}
		})
	}
}

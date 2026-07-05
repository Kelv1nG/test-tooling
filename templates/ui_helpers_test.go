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

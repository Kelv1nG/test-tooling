package main

import "testing"

func TestTabPath(t *testing.T) {
	tests := []struct {
		tab  string
		want string
	}{
		{tab: tabConfiguration, want: "/configuration"},
		{tab: tabFileTransfer, want: "/file-transfer"},
		{tab: tabChecking, want: "/checking"},
		{tab: "", want: "/configuration"},
		{tab: "unknown", want: "/configuration"},
	}

	for _, tt := range tests {
		t.Run(tt.tab, func(t *testing.T) {
			if got := tabPath(tt.tab); got != tt.want {
				t.Fatalf("tabPath(%q) = %q, want %q", tt.tab, got, tt.want)
			}
		})
	}
}

func TestNormalizeCheckPage(t *testing.T) {
	tests := []struct {
		name  string
		value string
		want  int
	}{
		{name: "empty defaults to first page", value: "", want: 1},
		{name: "invalid defaults to first page", value: "abc", want: 1},
		{name: "zero defaults to first page", value: "0", want: 1},
		{name: "negative defaults to first page", value: "-2", want: 1},
		{name: "positive page is preserved", value: "3", want: 3},
		{name: "whitespace is trimmed", value: " 4 ", want: 4},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := normalizeCheckPage(tt.value); got != tt.want {
				t.Fatalf("normalizeCheckPage(%q) = %d, want %d", tt.value, got, tt.want)
			}
		})
	}
}

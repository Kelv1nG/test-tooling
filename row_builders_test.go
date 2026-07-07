package main

import (
	"testing"

	"tooling/templates"
)

func TestMergeSingleCheckRowReplacesMatchingID(t *testing.T) {
	rows := mergeSingleCheckRow(
		[]templates.CheckRowView{
			{Index: 1, ID: "CHK-001", File: "first.xlsx"},
			{Index: 2, ID: "CHK-002", File: "old.xlsx"},
		},
		templates.CheckRowView{
			ID:   "CHK-002",
			File: "new.xlsx",
			Rules: []templates.CheckRuleView{
				{ID: "R001"},
			},
		},
	)

	if len(rows) != 2 {
		t.Fatalf("expected 2 rows, got %d", len(rows))
	}
	if rows[1].File != "new.xlsx" {
		t.Fatalf("expected matching row to be replaced, got %q", rows[1].File)
	}
	if rows[1].Index != 2 || rows[1].Rules[0].Index != 1 {
		t.Fatalf("expected rows and rules to be reindexed, got %+v", rows[1])
	}
}

func TestMergeSingleCheckRowAppendsNewID(t *testing.T) {
	rows := mergeSingleCheckRow(
		[]templates.CheckRowView{
			{Index: 1, ID: "CHK-001", File: "first.xlsx"},
		},
		templates.CheckRowView{
			ID:   "CHK-002",
			File: "second.xlsx",
		},
	)

	if len(rows) != 2 {
		t.Fatalf("expected 2 rows, got %d", len(rows))
	}
	if rows[1].ID != "CHK-002" || rows[1].Index != 2 {
		t.Fatalf("expected target row to be appended and reindexed, got %+v", rows[1])
	}
}

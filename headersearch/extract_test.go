package headersearch

import (
	"errors"
	"testing"

	"github.com/xuri/excelize/v2"
)

func TestExtractHeadersValidation(t *testing.T) {
	workbook := newWorkbook(t, "Report", nil, nil)

	tests := []struct {
		name     string
		workbook *excelize.File
		options  ExtractOptions
		wantErr  error
	}{
		{
			name:     "nil workbook",
			workbook: nil,
			options:  ExtractOptions{Sheet: "Report", Anchor: "A", ParentDirection: DirectionUp, MaxHeaderDepth: 1},
		},
		{
			name:     "empty sheet",
			workbook: workbook,
			options:  ExtractOptions{Anchor: "A", ParentDirection: DirectionUp, MaxHeaderDepth: 1},
		},
		{
			name:     "missing sheet",
			workbook: workbook,
			options:  ExtractOptions{Sheet: "Missing", Anchor: "A", ParentDirection: DirectionUp, MaxHeaderDepth: 1},
		},
		{
			name:     "empty anchor",
			workbook: workbook,
			options:  ExtractOptions{Sheet: "Report", ParentDirection: DirectionUp, MaxHeaderDepth: 1},
		},
		{
			name:     "invalid direction",
			workbook: workbook,
			options:  ExtractOptions{Sheet: "Report", Anchor: "A", ParentDirection: Direction("sideways"), MaxHeaderDepth: 1},
			wantErr:  ErrInvalidDirection,
		},
		{
			name:     "zero max depth",
			workbook: workbook,
			options:  ExtractOptions{Sheet: "Report", Anchor: "A", ParentDirection: DirectionUp, MaxHeaderDepth: 0},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			_, err := ExtractHeaders(test.workbook, test.options)
			if err == nil {
				t.Fatal("expected error, got nil")
			}
			if test.wantErr != nil && !errors.Is(err, test.wantErr) {
				t.Fatalf("expected %v, got %v", test.wantErr, err)
			}
		})
	}
}

func TestExtractHeadersResolvesSheetByIndex(t *testing.T) {
	workbook := excelize.NewFile()
	defaultSheet := workbook.GetSheetName(workbook.GetActiveSheetIndex())
	if err := workbook.SetSheetName(defaultSheet, "Summary"); err != nil {
		t.Fatalf("SetSheetName returned error: %v", err)
	}
	if _, err := workbook.NewSheet("Report"); err != nil {
		t.Fatalf("NewSheet returned error: %v", err)
	}
	if err := workbook.SetCellStr("Report", "A1", "Fund Name"); err != nil {
		t.Fatalf("SetCellStr returned error: %v", err)
	}
	if err := workbook.SetCellStr("Report", "B1", "Fund Inception Date"); err != nil {
		t.Fatalf("SetCellStr returned error: %v", err)
	}

	table, err := ExtractHeaders(workbook, ExtractOptions{
		Sheet:           "2",
		Anchor:          "Fund Inception Date",
		ParentDirection: DirectionUp,
		MaxHeaderDepth:  1,
	})
	if err != nil {
		t.Fatalf("ExtractHeaders returned error: %v", err)
	}

	if table.Sheet != "Report" {
		t.Fatalf("expected resolved sheet name Report, got %q", table.Sheet)
	}
}

func TestExtractHeadersInvalidSheetIndex(t *testing.T) {
	workbook := newWorkbook(t, "Report", map[string]string{
		"A1": "Fund Inception Date",
	}, nil)

	_, err := ExtractHeaders(workbook, ExtractOptions{
		Sheet:           "3",
		Anchor:          "Fund Inception Date",
		ParentDirection: DirectionUp,
		MaxHeaderDepth:  1,
	})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestExtractHeadersExactAnchorMatching(t *testing.T) {
	workbook := newWorkbook(t, "Report", map[string]string{
		"A1": "Fund Inception Date",
		"B1": "fund inception date",
		"C1": "Fund Inception Date ",
		"D1": " Fund Inception Date",
	}, nil)

	table, err := ExtractHeaders(workbook, ExtractOptions{
		Sheet:           "Report",
		Anchor:          "Fund Inception Date",
		ParentDirection: DirectionUp,
		MaxHeaderDepth:  1,
	})
	if err != nil {
		t.Fatalf("ExtractHeaders returned error: %v", err)
	}

	if table.AnchorPosition.Axis != "A1" {
		t.Fatalf("expected anchor at A1, got %s", table.AnchorPosition.Axis)
	}
}

func TestExtractHeadersAnchorErrors(t *testing.T) {
	t.Run("not found", func(t *testing.T) {
		workbook := newWorkbook(t, "Report", map[string]string{"A1": "Other"}, nil)
		_, err := ExtractHeaders(workbook, ExtractOptions{
			Sheet:           "Report",
			Anchor:          "Fund Inception Date",
			ParentDirection: DirectionUp,
			MaxHeaderDepth:  1,
		})
		if !errors.Is(err, ErrAnchorNotFound) {
			t.Fatalf("expected ErrAnchorNotFound, got %v", err)
		}
	})

	t.Run("multiple exact anchors", func(t *testing.T) {
		workbook := newWorkbook(t, "Report", map[string]string{
			"A1": "Fund Inception Date",
			"B1": "Fund Inception Date",
		}, nil)
		_, err := ExtractHeaders(workbook, ExtractOptions{
			Sheet:           "Report",
			Anchor:          "Fund Inception Date",
			ParentDirection: DirectionUp,
			MaxHeaderDepth:  1,
		})
		if !errors.Is(err, ErrMultipleAnchors) {
			t.Fatalf("expected ErrMultipleAnchors, got %v", err)
		}
	})
}

func TestExtractHeadersHierarchiesAndMerges(t *testing.T) {
	tests := []struct {
		name           string
		cells          map[string]string
		merges         [][2]string
		options        ExtractOptions
		wantPaths      [][]string
		wantAnchorAxis string
	}{
		{
			name: "single row header",
			cells: map[string]string{
				"A1": "Fund Name",
				"B1": "Fund Inception Date",
				"C1": "Currency",
			},
			options: ExtractOptions{
				Sheet:           "Report",
				Anchor:          "Fund Inception Date",
				ParentDirection: DirectionUp,
				MaxHeaderDepth:  2,
			},
			wantAnchorAxis: "B1",
			wantPaths: [][]string{
				{"Fund Name"},
				{"Fund Inception Date"},
				{"Currency"},
			},
		},
		{
			name: "two row hierarchy with merged parent",
			cells: map[string]string{
				"A1": "Fund Information",
				"C1": "Performance",
				"A2": "Fund Name",
				"B2": "Fund Inception Date",
				"C2": "1 Year",
			},
			merges: [][2]string{{"A1", "B1"}},
			options: ExtractOptions{
				Sheet:           "Report",
				Anchor:          "Fund Inception Date",
				ParentDirection: DirectionUp,
				MaxHeaderDepth:  1,
			},
			wantAnchorAxis: "B2",
			wantPaths: [][]string{
				{"Fund Information", "Fund Name"},
				{"Fund Information", "Fund Inception Date"},
				{"Performance", "1 Year"},
			},
		},
		{
			name: "skips blank spacer row before header",
			cells: map[string]string{
				"A1": "Header A",
				"B1": "Header B",
				"C1": "Header C",
				"A3": "Anchor",
				"B3": "Value B",
				"C3": "Value C",
			},
			options: ExtractOptions{
				Sheet:           "Report",
				Anchor:          "Anchor",
				ParentDirection: DirectionUp,
				MaxHeaderDepth:  1,
			},
			wantAnchorAxis: "A3",
			wantPaths: [][]string{
				{"Header A", "Anchor"},
				{"Header B", "Value B"},
				{"Header C", "Value C"},
			},
		},
		{
			name: "skips blank spacer row before multiple header levels",
			cells: map[string]string{
				"A1": "Group 1",
				"C1": "Group 2",
				"A2": "Header A",
				"B2": "Header B",
				"C2": "Header C",
				"A4": "Anchor",
				"B4": "Value B",
				"C4": "Value C",
			},
			merges: [][2]string{{"A1", "B1"}},
			options: ExtractOptions{
				Sheet:           "Report",
				Anchor:          "Anchor",
				ParentDirection: DirectionUp,
				MaxHeaderDepth:  2,
			},
			wantAnchorAxis: "A4",
			wantPaths: [][]string{
				{"Group 1", "Header A", "Anchor"},
				{"Group 1", "Header B", "Value B"},
				{"Group 2", "Header C", "Value C"},
			},
		},
		{
			name: "stops at blank boundary",
			cells: map[string]string{
				"A2": "Fund Name",
				"B2": "Fund Inception Date",
				"C2": "Currency",
				"E2": "Other Table",
			},
			options: ExtractOptions{
				Sheet:           "Report",
				Anchor:          "Fund Inception Date",
				ParentDirection: DirectionUp,
				MaxHeaderDepth:  2,
			},
			wantAnchorAxis: "B2",
			wantPaths: [][]string{
				{"Fund Name"},
				{"Fund Inception Date"},
				{"Currency"},
			},
		},
		{
			name: "vertical sibling scanning for right parent direction",
			cells: map[string]string{
				"A1": "Fund Inception Date",
				"A2": "1 Year",
				"B1": "Fund Information",
				"B2": "Performance",
			},
			options: ExtractOptions{
				Sheet:           "Report",
				Anchor:          "Fund Inception Date",
				ParentDirection: DirectionRight,
				MaxHeaderDepth:  2,
			},
			wantAnchorAxis: "A1",
			wantPaths: [][]string{
				{"Fund Information", "Fund Inception Date"},
				{"Performance", "1 Year"},
			},
		},
		{
			name: "collapses consecutive duplicates",
			cells: map[string]string{
				"A1": "Fund",
				"A2": "Fund",
				"A3": "Fund Inception Date",
			},
			merges: nil,
			options: ExtractOptions{
				Sheet:           "Report",
				Anchor:          "Fund Inception Date",
				ParentDirection: DirectionUp,
				MaxHeaderDepth:  3,
			},
			wantAnchorAxis: "A3",
			wantPaths: [][]string{
				{"Fund", "Fund Inception Date"},
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			workbook := newWorkbook(t, "Report", test.cells, test.merges)
			table, err := ExtractHeaders(workbook, test.options)
			if err != nil {
				t.Fatalf("ExtractHeaders returned error: %v", err)
			}

			if table.AnchorPosition.Axis != test.wantAnchorAxis {
				t.Fatalf("expected anchor axis %s, got %s", test.wantAnchorAxis, table.AnchorPosition.Axis)
			}

			if len(table.Headers) != len(test.wantPaths) {
				t.Fatalf("expected %d headers, got %d", len(test.wantPaths), len(table.Headers))
			}

			for index, wantPath := range test.wantPaths {
				if !pathEqual(table.Headers[index].Path, wantPath) {
					t.Fatalf("header %d path = %v, want %v", index, table.Headers[index].Path, wantPath)
				}
			}
		})
	}
}

func TestExtractHeadersStopsAtBlankParentLayerAndMaxDepth(t *testing.T) {
	workbook := newWorkbook(t, "Report", map[string]string{
		"A1": "Outer",
		"A2": "Inner",
		"A3": "Fund Inception Date",
	}, nil)

	table, err := ExtractHeaders(workbook, ExtractOptions{
		Sheet:           "Report",
		Anchor:          "Fund Inception Date",
		ParentDirection: DirectionUp,
		MaxHeaderDepth:  1,
	})
	if err != nil {
		t.Fatalf("ExtractHeaders returned error: %v", err)
	}

	if !pathEqual(table.Headers[0].Path, []string{"Inner", "Fund Inception Date"}) {
		t.Fatalf("unexpected path: %v", table.Headers[0].Path)
	}
}

func TestExtractHeadersCanIgnoreAnchorLayer(t *testing.T) {
	workbook := newWorkbook(t, "Report", map[string]string{
		"A1": "Gross Return",
		"B1": "Net Return 1",
		"C1": "Net Return 2",
		"A3": "Anchor",
		"B3": "Value 1",
		"C3": "Value 2",
	}, nil)

	table, err := ExtractHeaders(workbook, ExtractOptions{
		Sheet:             "Report",
		Anchor:            "Anchor",
		ParentDirection:   DirectionUp,
		MaxHeaderDepth:    1,
		IgnoreAnchorLayer: true,
	})
	if err != nil {
		t.Fatalf("ExtractHeaders returned error: %v", err)
	}

	wantPaths := [][]string{
		{"Gross Return"},
		{"Net Return 1"},
		{"Net Return 2"},
	}
	if len(table.Headers) != len(wantPaths) {
		t.Fatalf("expected %d headers, got %d", len(wantPaths), len(table.Headers))
	}
	for index, wantPath := range wantPaths {
		if !pathEqual(table.Headers[index].Path, wantPath) {
			t.Fatalf("header %d path = %v, want %v", index, table.Headers[index].Path, wantPath)
		}
	}
}

func newWorkbook(
	t *testing.T,
	sheet string,
	cells map[string]string,
	merges [][2]string,
) *excelize.File {
	t.Helper()

	workbook := excelize.NewFile()
	currentSheet := workbook.GetSheetName(workbook.GetActiveSheetIndex())
	if currentSheet != sheet {
		if err := workbook.SetSheetName(currentSheet, sheet); err != nil {
			t.Fatalf("SetSheetName: %v", err)
		}
	}

	for axis, value := range cells {
		if err := workbook.SetCellStr(sheet, axis, value); err != nil {
			t.Fatalf("SetCellStr(%s): %v", axis, err)
		}
	}

	for _, merge := range merges {
		if err := workbook.MergeCell(sheet, merge[0], merge[1]); err != nil {
			t.Fatalf("MergeCell(%s:%s): %v", merge[0], merge[1], err)
		}
	}

	return workbook
}

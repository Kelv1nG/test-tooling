package headersearch

import "testing"

func TestCompareHeaders(t *testing.T) {
	left := HeaderTable{
		Headers: []ColumnHeader{
			{Path: []string{"Fund Information", "Fund Name"}},
			{Path: []string{"Fund Information", "Fund Inception Date"}},
			{Path: []string{"Performance", "1 Year"}},
		},
	}

	t.Run("identical ordered headers are equal", func(t *testing.T) {
		result := CompareHeaders(left, left, CompareOptions{RequireOrder: true})
		if !result.Equal {
			t.Fatalf("expected equal result, got %#v", result)
		}
	})

	t.Run("reordered headers fail ordered comparison", func(t *testing.T) {
		right := HeaderTable{
			Headers: []ColumnHeader{
				{Path: []string{"Fund Information", "Fund Inception Date"}},
				{Path: []string{"Fund Information", "Fund Name"}},
				{Path: []string{"Performance", "1 Year"}},
			},
		}

		result := CompareHeaders(left, right, CompareOptions{RequireOrder: true})
		if result.Equal {
			t.Fatalf("expected unequal result")
		}
		if !result.Difference.Reordered {
			t.Fatalf("expected reordered difference, got %#v", result.Difference)
		}
		if len(result.Difference.Missing) != 0 || len(result.Difference.Unexpected) != 0 {
			t.Fatalf("expected pure reorder, got %#v", result.Difference)
		}
	})

	t.Run("reordered headers pass unordered comparison", func(t *testing.T) {
		right := HeaderTable{
			Headers: []ColumnHeader{
				{Path: []string{"Fund Information", "Fund Inception Date"}},
				{Path: []string{"Fund Information", "Fund Name"}},
				{Path: []string{"Performance", "1 Year"}},
			},
		}

		result := CompareHeaders(left, right, CompareOptions{RequireOrder: false})
		if !result.Equal {
			t.Fatalf("expected unordered equal result, got %#v", result)
		}
	})

	t.Run("missing and unexpected headers are reported", func(t *testing.T) {
		right := HeaderTable{
			Headers: []ColumnHeader{
				{Path: []string{"Fund Information", "Fund Name"}},
				{Path: []string{"Performance", "3 Year"}},
			},
		}

		result := CompareHeaders(left, right, CompareOptions{RequireOrder: false})
		if result.Equal {
			t.Fatalf("expected unequal result")
		}
		if len(result.Difference.Missing) != 2 {
			t.Fatalf("expected 2 missing paths, got %#v", result.Difference.Missing)
		}
		if len(result.Difference.Unexpected) != 1 {
			t.Fatalf("expected 1 unexpected path, got %#v", result.Difference.Unexpected)
		}
	})

	t.Run("duplicate counts are respected", func(t *testing.T) {
		right := HeaderTable{
			Headers: []ColumnHeader{
				{Path: []string{"Fund Information", "Fund Name"}},
				{Path: []string{"Fund Information", "Fund Name"}},
				{Path: []string{"Fund Information", "Fund Inception Date"}},
				{Path: []string{"Performance", "1 Year"}},
			},
		}

		result := CompareHeaders(left, right, CompareOptions{RequireOrder: false})
		if result.Equal {
			t.Fatalf("expected unequal result")
		}
		if len(result.Difference.Unexpected) != 1 {
			t.Fatalf("expected duplicate to be reported as unexpected, got %#v", result.Difference.Unexpected)
		}
	})
}

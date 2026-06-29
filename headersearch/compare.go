package headersearch

import "encoding/json"

func CompareHeaders(
	left HeaderTable,
	right HeaderTable,
	options CompareOptions,
) ComparisonResult {
	if options.RequireOrder && orderedPathsEqual(left.Headers, right.Headers) {
		return ComparisonResult{Equal: true}
	}

	difference := HeaderDifference{
		Missing:    missingPaths(left.Headers, right.Headers),
		Unexpected: missingPaths(right.Headers, left.Headers),
	}

	if options.RequireOrder && len(difference.Missing) == 0 && len(difference.Unexpected) == 0 {
		difference.Reordered = true
	}

	return ComparisonResult{
		Equal:      len(difference.Missing) == 0 && len(difference.Unexpected) == 0 && !difference.Reordered,
		Difference: difference,
	}
}

func orderedPathsEqual(
	left []ColumnHeader,
	right []ColumnHeader,
) bool {
	if len(left) != len(right) {
		return false
	}

	for index := range left {
		if !pathEqual(left[index].Path, right[index].Path) {
			return false
		}
	}

	return true
}

func missingPaths(
	left []ColumnHeader,
	right []ColumnHeader,
) [][]string {
	counts := make(map[string]int, len(right))
	for _, header := range right {
		counts[pathKey(header.Path)]++
	}

	missing := make([][]string, 0)
	for _, header := range left {
		key := pathKey(header.Path)
		if counts[key] > 0 {
			counts[key]--
			continue
		}

		missing = append(missing, append([]string(nil), header.Path...))
	}

	return missing
}

func pathEqual(left []string, right []string) bool {
	if len(left) != len(right) {
		return false
	}

	for index := range left {
		if left[index] != right[index] {
			return false
		}
	}

	return true
}

func pathKey(path []string) string {
	encoded, _ := json.Marshal(path)
	return string(encoded)
}

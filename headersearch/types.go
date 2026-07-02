package headersearch

import "tooling/sheetsearch"

const (
	DirectionUp    = sheetsearch.DirectionUp
	DirectionDown  = sheetsearch.DirectionDown
	DirectionLeft  = sheetsearch.DirectionLeft
	DirectionRight = sheetsearch.DirectionRight
)

type Direction = sheetsearch.Direction

type ExtractOptions struct {
	Sheet           string
	Anchor          string
	ParentDirection Direction
	MaxHeaderDepth  int
	// IgnoreAnchorLayer excludes the row or column containing the anchor from
	// header paths. This is useful when the anchor layer contains data values
	// under the headers rather than leaf header labels.
	IgnoreAnchorLayer bool
}

type CellPosition struct {
	Row    int
	Column int
	Axis   string
}

type ColumnHeader struct {
	LeafPosition CellPosition
	Path         []string
}

type HeaderTable struct {
	Sheet           string
	Anchor          string
	AnchorPosition  CellPosition
	ParentDirection Direction
	Headers         []ColumnHeader
}

type CompareOptions struct {
	RequireOrder bool
}

type HeaderDifference struct {
	Missing    [][]string
	Unexpected [][]string
	Reordered  bool
}

type ComparisonResult struct {
	Equal      bool
	Difference HeaderDifference
}

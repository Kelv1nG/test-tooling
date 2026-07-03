package headersearch

import "tooling/sheetsearch"

// Direction constants are re-exported so callers can configure header
// extraction without importing the lower-level sheetsearch package.
const (
	DirectionUp    = sheetsearch.DirectionUp
	DirectionDown  = sheetsearch.DirectionDown
	DirectionLeft  = sheetsearch.DirectionLeft
	DirectionRight = sheetsearch.DirectionRight
)

// Direction describes the direction from a leaf header toward its parent
// header layers.
type Direction = sheetsearch.Direction

// ExtractOptions defines the search contract for a single header table.
type ExtractOptions struct {
	// Sheet accepts the same name-or-index input as sheetsearch.ResolveSheetName.
	Sheet string
	// Anchor must match one cell exactly; it becomes the known leaf header that
	// defines which contiguous table span to extract.
	Anchor string
	// ParentDirection points from the leaf layer toward parent headers, so the
	// perpendicular axis is used to collect sibling leaf headers.
	ParentDirection Direction
	// MaxHeaderDepth limits non-empty parent layers, not blank spacer rows or
	// columns between a table's leaf and parent headers.
	MaxHeaderDepth int
	// IgnoreAnchorLayer excludes the row or column containing the anchor from
	// header paths. This is useful when the anchor layer contains data values
	// under the headers rather than leaf header labels.
	IgnoreAnchorLayer bool
}

// CellPosition stores one-based spreadsheet coordinates plus the A1-style axis
// used in messages and UI output.
type CellPosition struct {
	Row    int
	Column int
	Axis   string
}

// ColumnHeader is one extracted leaf header and the full semantic path that
// reaches it from the outermost parent layer.
type ColumnHeader struct {
	LeafPosition CellPosition
	Path         []string
}

// HeaderTable is the normalized header model returned by ExtractHeaders.
type HeaderTable struct {
	Sheet           string
	Anchor          string
	AnchorPosition  CellPosition
	ParentDirection Direction
	Headers         []ColumnHeader
}

// CompareOptions controls whether matching header paths must also appear in
// the same order.
type CompareOptions struct {
	RequireOrder bool
}

// HeaderDifference reports path differences relative to the left table passed
// to CompareHeaders.
type HeaderDifference struct {
	// Missing are paths present in the left table but absent from the right.
	Missing [][]string
	// Unexpected are paths present in the right table but absent from the left.
	Unexpected [][]string
	// Reordered means both sides contain the same paths, but RequireOrder found
	// them in a different sequence.
	Reordered  bool
}

// ComparisonResult summarizes whether two extracted header tables match and,
// when they do not, why.
type ComparisonResult struct {
	Equal      bool
	Difference HeaderDifference
}

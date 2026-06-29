package headersearch

type Direction string

const (
	DirectionUp    Direction = "up"
	DirectionDown  Direction = "down"
	DirectionLeft  Direction = "left"
	DirectionRight Direction = "right"
)

type ExtractOptions struct {
	Sheet           string
	Anchor          string
	ParentDirection Direction
	MaxHeaderDepth  int
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

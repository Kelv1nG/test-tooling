package sheetsearch

// Direction is one of the four cardinal directions used to traverse worksheet
// cells from an anchor position.
type Direction string

const (
	DirectionUp    Direction = "up"
	DirectionDown  Direction = "down"
	DirectionLeft  Direction = "left"
	DirectionRight Direction = "right"
)

func (d Direction) Valid() bool {
	switch d {
	case DirectionUp, DirectionDown, DirectionLeft, DirectionRight:
		return true
	default:
		return false
	}
}

// Move returns the 1-based spreadsheet coordinates reached by moving distance
// cells from row/column. It rejects invalid directions and top/left underflow.
func (d Direction) Move(row int, column int, distance int) (int, int, bool) {
	switch d {
	case DirectionUp:
		row -= distance
	case DirectionDown:
		row += distance
	case DirectionLeft:
		column -= distance
	case DirectionRight:
		column += distance
	default:
		return 0, 0, false
	}

	if row < 1 || column < 1 {
		return 0, 0, false
	}

	return row, column, true
}

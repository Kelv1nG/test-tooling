package headersearch

import (
	"fmt"

	"github.com/xuri/excelize/v2"
)

func (d Direction) Valid() bool {
	switch d {
	case DirectionUp, DirectionDown, DirectionLeft, DirectionRight:
		return true
	default:
		return false
	}
}

func siblingDirections(
	parent Direction,
) (negative Direction, positive Direction, err error) {
	switch parent {
	case DirectionUp, DirectionDown:
		return DirectionLeft, DirectionRight, nil
	case DirectionLeft, DirectionRight:
		return DirectionUp, DirectionDown, nil
	default:
		return "", "", ErrInvalidDirection
	}
}

func move(
	position CellPosition,
	direction Direction,
	distance int,
) (CellPosition, bool) {
	row := position.Row
	column := position.Column

	switch direction {
	case DirectionUp:
		row -= distance
	case DirectionDown:
		row += distance
	case DirectionLeft:
		column -= distance
	case DirectionRight:
		column += distance
	default:
		return CellPosition{}, false
	}

	if row < 1 || column < 1 {
		return CellPosition{}, false
	}

	next, err := newCellPosition(row, column)
	if err != nil {
		return CellPosition{}, false
	}

	return next, true
}

func newCellPosition(
	row int,
	column int,
) (CellPosition, error) {
	axis, err := excelize.CoordinatesToCellName(column, row)
	if err != nil {
		return CellPosition{}, fmt.Errorf(
			"resolve cell axis row=%d column=%d: %w",
			row,
			column,
			err,
		)
	}

	return CellPosition{
		Row:    row,
		Column: column,
		Axis:   axis,
	}, nil
}

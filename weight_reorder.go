package main

// MoveWeightRow moves a row to the indicated position. The order is also
// retained in project serialization and in Word/Excel report generation.
func MoveWeightRow(rows []WeightRow, from, to int) bool {
	if from < 0 || from >= len(rows) || to < 0 || to >= len(rows) || from == to {
		return false
	}
	moving := rows[from]
	if from < to {
		copy(rows[from:to], rows[from+1:to+1])
	} else {
		copy(rows[to+1:from+1], rows[to:from])
	}
	rows[to] = moving
	return true
}

// DuplicateWeightRow inserts a separate, editable copy immediately after the original.
func DuplicateWeightRow(rows []WeightRow, row int) []WeightRow {
	if row < 0 || row >= len(rows) {
		return rows
	}
	rows = append(rows, WeightRow{})
	copy(rows[row+2:], rows[row+1:len(rows)-1])
	rows[row+1] = rows[row]
	return rows
}

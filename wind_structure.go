package main

func windStructureDamping(code string) float64 {
	// СП 20.13330.2016, таблица 11.5. Normative values are taken only
	// from the immutable external mathematical JSON, never hard-coded.
	index := 2 // reinforced concrete, masonry or enclosed steel frames
	switch code {
	case "steel":
		index = 0
	case "mixed":
		index = 1
	}
	db, err := getWindDB()
	if err != nil {
		return 0
	}
	table, ok := db.Tables["11.5"]
	if !ok || len(table.Rows) <= index || len(table.Rows[index]) == 0 {
		return 0
	}
	value, err := numberCell(table.Rows[index][0])
	if err != nil {
		return 0
	}
	return value
}

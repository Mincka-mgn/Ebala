package main

func weightReportBlocks(r TempReport) []ProjectWeightBlock {
	if len(r.WeightBlocks) > 0 {
		return r.WeightBlocks
	}
	if len(r.WeightRows) > 0 {
		return []ProjectWeightBlock{{Title: "Собственный вес", Rows: r.WeightRows}}
	}
	return nil
}

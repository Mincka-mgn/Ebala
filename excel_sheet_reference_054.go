package main

import "strings"

// excelRenameOwnSheetRefs rewrites references in a single generated calculation
// block, never the completed workbook. tempExcelCell has already XML-escaped
// apostrophes (&#39;), while some older XML builders emit literal ones.
// Match the COMPLETE quoted sheet token plus !, so similarly named sheets,
// references from other blocks and quoted user names are not re-indexed.
func excelRenameOwnSheetRefs(body, suffix string, names ...string) string {
	if suffix == "" {
		return body
	}
	changes := make([]string, 0, len(names)*4)
	for _, name := range names {
		for _, quote := range []string{"&#39;", "'"} {
			changes = append(changes,
				quote+name+quote+"!", quote+name+suffix+quote+"!")
		}
	}
	return strings.NewReplacer(changes...).Replace(body)
}

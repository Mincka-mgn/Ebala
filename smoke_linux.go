//go:build !windows

package main

func main() {
	p := initialProject()
	for _, v := range climateRecords() {
		if v.Place == "Казань" {
			p.SetSite(climateFromRecord(v))
			break
		}
	}
	if err := ExportExcel(p, "/mnt/data/poverka_v0_4_source/example_v03.xlsx"); err != nil {
		panic(err)
	}
	if err := ExportWord(p, "/mnt/data/poverka_v0_4_source/example_v03.docx"); err != nil {
		panic(err)
	}
}

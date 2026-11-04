from pathlib import Path
p=Path('/mnt/data/ebala_022_work')
f=p/'crane_placement_windows.go';s=f.read_text(encoding='utf-8')
s=s.replace('"strconv"\n','')
s=s.replace('\tcranePlaceExcel\n)', '\tcranePlaceExcel\n\tcranePlaceScrollUp\n\tcranePlaceScrollDown\n)')
s=s.replace('label("Число пролётов", 25, 103, 170, 24)\n\tedit(cranePlaceSpanCount, "1", 202, 101, 72, 30)\n\tlabel("Длины пролётов, м (через пробел или ;) ", 295, 103, 340, 24)\n\tedit(cranePlaceLengths, "6", 643, 101, 300, 30)', 'label("Длины пролётов, м (через пробел или ;) ", 25, 103, 350, 24)\n\tedit(cranePlaceLengths, "6", 384, 101, 558, 30)')
s=s.replace('create("EDIT", "", WS_CHILD|WS_VISIBLE|WS_BORDER|WS_VSCROLL|ES_MULTILINE|0x40|ES_READONLY, 25, 193, 1085, 45, cranePlaceTrainInfo)', 'create("EDIT", "", WS_CHILD|WS_VISIBLE|WS_BORDER|WS_VSCROLL|ES_MULTILINE|0x40|ES_READONLY, 25, 193, 950, 45, cranePlaceTrainInfo)\n\tbutton(cranePlaceScrollUp, "▲", 984, 193, 57, 23)\n\tbutton(cranePlaceScrollDown, "▼", 1046, 193, 64, 23)')
s=s.replace('case cranePlaceOpen, cranePlaceOpen + 20:', 'case cranePlaceScrollUp, cranePlaceScrollDown:\n\t\tif controls[cranePlaceTrainInfo] != 0 {\n\t\t\tdelta := int32(2)\n\t\t\tif id == cranePlaceScrollUp { delta = -2 }\n\t\t\tpSend.Call(controls[cranePlaceTrainInfo], 0xB6, 0, uintptr(delta)) // EM_LINESCROLL\n\t\t}\n\t\treturn true\n\tcase cranePlaceOpen, cranePlaceOpen + 20:')
start=s.index('\t\tcount, err := strconv.Atoi(strings.TrimSpace(get(cranePlaceSpanCount)))')
end=s.index('\n\t\tif suspicious {',start)
s=s[:start]+'''\t\trawLengths := strings.TrimSpace(get(cranePlaceLengths))
		pieces := strings.FieldsFunc(rawLengths, func(r rune) bool {
			return r == ' ' || r == ';' || r == '\\n' || r == '\\t'
		})
		if len(pieces) < 1 || len(pieces) > 20 {
			message("Укажите от 1 до 20 длин пролётов в метрах, через пробел или ;", true)
			return true
		}
		spans, suspicious, err := parseCraneSpanMeters(rawLengths, len(pieces))
		if err != nil {
			message(err.Error(), true)
			return true
		}'''+s[end:]
# reset obsolete count preserved id but no control
f.write_text(s,encoding='utf-8')

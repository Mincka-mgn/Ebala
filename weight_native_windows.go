//go:build windows

package main

import (
	"encoding/json"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"syscall"
	"unsafe"
)

// Standard Windows controls. No embedded browser or HTTP listener.
var comctl = syscall.NewLazyDLL("comctl32.dll")
var initCommon = comctl.NewProc("InitCommonControls")
var pSetFocus = u32.NewProc("SetFocus")

const (
	LVS_REPORT                   = 0x0001
	LVS_SINGLESEL                = 0x0004
	LVS_SHOWSELALWAYS            = 0x0008
	LVM_FIRST                    = 0x1000
	LVM_INSERTCOLUMNW            = LVM_FIRST + 97
	LVM_INSERTITEMW              = LVM_FIRST + 77
	LVM_SETITEMTEXTW             = LVM_FIRST + 116
	LVM_DELETEALLITEMS           = LVM_FIRST + 9
	LVM_GETNEXTITEM              = LVM_FIRST + 12
	LVM_SETITEMSTATE             = LVM_FIRST + 43
	LVM_SETEXTENDEDLISTVIEWSTYLE = LVM_FIRST + 54
	LVNI_SELECTED                = 0x0002
	LVIS_SELECTED                = 0x0002
	WM_NOTIFY                    = 0x004E
	EN_KILLFOCUS_WEIGHT          = 0x0200
)

type listColumn struct {
	Mask    uint32
	Fmt     int32
	Cx      int32
	Text    *uint16
	TextLen int32
	Sub     int32
	Image   int32
	Order   int32
}
type listItem struct {
	Mask      uint32
	Index     int32
	Sub       int32
	State     uint32
	StateMask uint32
	Text      *uint16
	Max       int32
	Image     int32
	Param     uintptr
	Indent    int32
	Group     int32
	Cols      uint32
	PtrCols   uintptr
	PtrFmt    uintptr
	GroupID   int32
}
type notificationHeader struct {
	Hwnd uintptr
	ID   uintptr
	Code int32
}
type material struct {
	ID          string   `json:"id"`
	CategoryID  string   `json:"category_id"`
	CategoryIDs []string `json:"category_ids,omitempty"`
	Name        string   `json:"name"`
	Kind        string   `json:"kind"`
	Min         *float64 `json:"density_min,omitempty"`
	Max         *float64 `json:"density_max,omitempty"`
	Thickness   float64  `json:"default_thickness_mm,omitempty"`
	Gamma       float64  `json:"default_gamma,omitempty"`
	GNorm       float64  `json:"g_norm,omitempty"`
}
type weightCatalog struct {
	SchemaVersion int              `json:"schema_version"`
	Categories    []weightCategory `json:"categories,omitempty"`
	Items         []material       `json:"items"`
}
type weightCategory struct {
	ID    string `json:"id"`
	Group string `json:"group"`
	Name  string `json:"name"`
}
type catRow struct {
	Group  string
	ID     string
	Header bool
}

var catalogBase, catalogUser weightCatalog
var shownCats []catRow
var expandedWeightGroups = map[string]bool{}
var shownMaterials []material
var chosenCategory string
var currentWeightIndex = -1
var weightUIBusy bool
var weightCatLoaded bool
var builtinTemplates []WeightTemplate
var userTemplates []WeightTemplate
var visibleTemplates []WeightTemplate
var visibleTemplateIsUser []bool
var weightTemplatesLoaded bool
var lastWeightMessage string
var weightMonoFont uintptr
var weightTotalFont uintptr

func weightLoadCatalog() error {
	dir := filepath.Join(storageRoot(), "Catalog")
	b, err := os.ReadFile(filepath.Join(dir, "materials.json"))
	if err != nil {
		return fmt.Errorf("Основной справочник не найден: %w", err)
	}
	if err = json.Unmarshal(b, &catalogBase); err != nil {
		return err
	}
	b, err = os.ReadFile(filepath.Join(dir, "custom.json"))
	if os.IsNotExist(err) {
		catalogUser = weightCatalog{SchemaVersion: 1, Items: []material{}}
	} else if err != nil {
		return err
	} else if err = json.Unmarshal(b, &catalogUser); err != nil {
		return err
	}
	if catalogUser.SchemaVersion == 0 {
		catalogUser.SchemaVersion = 1
	}
	// Migrate old 0.06 personal materials from the removed generic category.
	for i := range catalogUser.Items {
		if catalogUser.Items[i].CategoryID == "c25" {
			catalogUser.Items[i].CategoryID = "c3"
		}
	}
	weightCatLoaded = true
	return nil
}
func weightSaveCustom() error {
	b, e := json.MarshalIndent(catalogUser, "", "  ")
	if e != nil {
		return e
	}
	return writeAtomic(filepath.Join(storageRoot(), "Catalog", "custom.json"), append(b, '\n'))
}
func allWeightMaterials() []material {
	return append(append([]material{}, catalogBase.Items...), catalogUser.Items...)
}
func weightHasCategory(m material, category string) bool {
	if category == "" || m.CategoryID == category {
		return true
	}
	for _, cat := range m.CategoryIDs {
		if cat == category {
			return true
		}
	}
	return false
}
func weightCategoryName(id string) string {
	for _, v := range catalogBase.Categories {
		if v.ID == id {
			return v.Name
		}
	}
	return "Другое"
}
func weightCategoryGroup(id string) string {
	for _, v := range catalogBase.Categories {
		if v.ID == id {
			return v.Group
		}
	}
	return "Другое"
}
func weightCategories() {
	weightUIBusy = true
	defer func() { weightUIBusy = false }()
	send(901, LB_RESETCONTENT, 0, 0)
	shownCats = nil
	seen := map[string]bool{}
	for _, c := range catalogBase.Categories {
		if !seen[c.Group] {
			shownCats = append(shownCats, catRow{Group: c.Group, Header: true})
			seen[c.Group] = true
			v := "▾  "
			if !expandedWeightGroups[c.Group] {
				v = "▸  "
			}
			weightAddList(901, v+c.Group)
		}
		if expandedWeightGroups[c.Group] {
			shownCats = append(shownCats, catRow{Group: c.Group, ID: c.ID})
			weightAddList(901, "      "+c.Name)
		}
	}
	for i, v := range shownCats {
		if v.ID != "" && v.ID == chosenCategory {
			send(901, LB_SETCURSEL, uintptr(i), 0)
			break
		}
	}
}
func weightAddList(id int, s string) {
	pSend.Call(controls[id], LB_ADDSTRING, 0, uintptr(unsafe.Pointer(ptr(s))))
}
func weightChosenCat() {
	ix := int(int32(send(901, LB_GETCURSEL, 0, 0)))
	if ix < 0 || ix >= len(shownCats) {
		return
	}
	row := shownCats[ix]
	if row.Header {
		expandedWeightGroups[row.Group] = !expandedWeightGroups[row.Group]
		weightCategories()
		return
	}
	chosenCategory = row.ID
	// A category switch is a new browsing action, not a continuation of the search.
	set(930, "")
	weightFillMaterials()
}
func weightFillMaterials() {
	send(902, LB_RESETCONTENT, 0, 0)
	shownMaterials = nil
	filter := strings.ToLower(strings.TrimSpace(get(930)))
	for _, m := range allWeightMaterials() {
		if !weightHasCategory(m, chosenCategory) {
			continue
		}
		if filter != "" && !strings.Contains(strings.ToLower(m.Name), filter) {
			continue
		}
		shownMaterials = append(shownMaterials, m)
	}
	sort.SliceStable(shownMaterials, func(i, j int) bool {
		return strings.ToLower(shownMaterials[i].Name) < strings.ToLower(shownMaterials[j].Name)
	})
	for _, m := range shownMaterials {
		value := ""
		if m.Kind == "direct" {
			value = fmt.Sprintf("  •  %.1f кгс/м²", m.GNorm)
		} else if m.Max != nil {
			if m.Min != nil {
				if *m.Min == *m.Max {
					value = fmt.Sprintf("  •  %.0f кг/м³", *m.Max)
				} else {
					value = fmt.Sprintf("  •  %.0f–%.0f кг/м³", *m.Min, *m.Max)
				}
			} else {
				value = fmt.Sprintf("  •  до %.0f кг/м³", *m.Max)
			}
		}
		weightAddList(902, m.Name+value)
	}
}

// The third panel is a library of complete assemblies, independent of the material catalog.
func weightLoadTemplates() error {
	b, err := readTemplates(templateBuiltInPath(), false)
	if err != nil {
		return err
	}
	u, err := readTemplates(templateUserPath(), true)
	if err != nil {
		return err
	}
	builtinTemplates, userTemplates = b, u
	weightTemplatesLoaded = true
	return nil
}
func weightRefreshTemplates() {
	if controls[950] == 0 {
		return
	}
	send(950, LB_RESETCONTENT, 0, 0)
	visibleTemplates, visibleTemplateIsUser = nil, nil
	for _, t := range builtinTemplates {
		visibleTemplates = append(visibleTemplates, t)
		visibleTemplateIsUser = append(visibleTemplateIsUser, false)
		weightAddList(950, "  "+t.Name)
	}
	for _, t := range userTemplates {
		visibleTemplates = append(visibleTemplates, t)
		visibleTemplateIsUser = append(visibleTemplateIsUser, true)
		weightAddList(950, "★ "+t.Name)
	}
}
func weightSelectedTemplate() (WeightTemplate, bool, bool) {
	ix := int(int32(send(950, LB_GETCURSEL, 0, 0)))
	if ix < 0 || ix >= len(visibleTemplates) {
		return WeightTemplate{}, false, false
	}
	return visibleTemplates[ix], visibleTemplateIsUser[ix], true
}
func weightApplyTemplate() {
	t, _, ok := weightSelectedTemplate()
	if !ok {
		message("Выберите готовую конструкцию справа.", true)
		return
	}
	if len(weightCurrent().Rows) > 0 {
		q := "Заменить текущий состав конструкцией «" + t.Name + "»?\nНесохранённый состав будет потерян."
		v, _, _ := pMsgBox.Call(hwnd, uintptr(unsafe.Pointer(ptr(q))), uintptr(unsafe.Pointer(ptr("Загрузить конструкцию"))), MB_ICONWARNING|MB_YESNO)
		if v != IDYES {
			return
		}
	}
	w := weightCurrent()
	w.Rows = cloneWeightRows(t.Rows)
	w.Name = t.Name
	set(900, w.Name)
	weightRenderRows(-1)
}
func weightSaveTemplate() {
	w := weightCurrent()
	w.Name = strings.TrimSpace(get(900))
	if w.Name == "" {
		message("Сначала впишите название конструкции в поле над справочником.\nОно будет названием вашего сохранённого варианта.", true)
		pSetFocus.Call(controls[900])
		return
	}
	if len(w.Rows) == 0 {
		message("Нечего сохранять: состав конструкции пуст.", true)
		return
	}
	if x, ok := findTemplateByName(userTemplates, w.Name); ok {
		q := "Ваш вариант «" + x.Name + "» уже существует. Заменить его текущим составом?"
		v, _, _ := pMsgBox.Call(hwnd, uintptr(unsafe.Pointer(ptr(q))), uintptr(unsafe.Pointer(ptr("Обновить конструкцию"))), MB_ICONWARNING|MB_YESNO)
		if v != IDYES {
			return
		}
	}
	next, _, err := saveNamedConstruction(userTemplates, w.Name, w.Rows)
	if err != nil {
		message(err.Error(), true)
		return
	}
	if err = saveUserTemplates(next); err != nil {
		message(err.Error(), true)
		return
	}
	userTemplates = next
	weightRefreshTemplates()
	message("Конструкция сохранена в разделе «Мои конструкции». Она останется при следующем запуске.", false)
}
func weightDeleteTemplate() {
	t, mine, ok := weightSelectedTemplate()
	if !ok {
		message("Выберите конструкцию для удаления.", true)
		return
	}
	if !mine {
		message("Предустановленные конструкции удалять нельзя. Можно редактировать их состав в расчёте и сохранять под новым именем.", false)
		return
	}
	q := "Удалить сохранённую конструкцию «" + t.Name + "»?"
	v, _, _ := pMsgBox.Call(hwnd, uintptr(unsafe.Pointer(ptr(q))), uintptr(unsafe.Pointer(ptr("Удаление конструкции"))), MB_ICONWARNING|MB_YESNO)
	if v != IDYES {
		return
	}
	next := make([]WeightTemplate, 0, len(userTemplates))
	for _, x := range userTemplates {
		if x.ID != t.ID {
			next = append(next, x)
		}
	}
	if err := saveUserTemplates(next); err != nil {
		message(err.Error(), true)
		return
	}
	userTemplates = next
	weightRefreshTemplates()
}

func weightSelectedMaterial() (material, bool) {
	ix := int(int32(send(902, LB_GETCURSEL, 0, 0)))
	if ix < 0 || ix >= len(shownMaterials) {
		return material{}, false
	}
	return shownMaterials[ix], true
}
func weightMakeRow(m material) WeightRow {
	r := WeightRow{Name: m.Name, Kind: m.Kind, Gamma: m.Gamma, Thickness: m.Thickness, GNorm: m.GNorm}
	if r.Gamma <= 0 {
		r.Gamma = 1.1
	}
	if m.Max != nil {
		r.Density = *m.Max
	}
	if r.Kind != "layer" {
		r.Kind = "direct"
	}
	return r
}

var weightCurrentBlockID int

func weightCurrent() *WeightCalc {
	if project.Weight == nil {
		v := project.CurrentWeight()
		project.Weight = &v
	}
	return project.Weight
}
func weightAddMaterial() {
	m, ok := weightSelectedMaterial()
	if !ok {
		message("Выберите материал в справочнике.", true)
		return
	}
	w := weightCurrent()
	w.Rows = append(w.Rows, weightMakeRow(m))
	weightRenderRows(len(w.Rows) - 1)
}
func weightAddFree() {
	w := weightCurrent()
	w.Rows = append(w.Rows, WeightRow{Name: "Произвольный элемент", Kind: "direct", Gamma: 1.1})
	weightRenderRows(len(w.Rows) - 1)
	weightStartCellEdit(len(w.Rows)-1, 0)
}
func weightSelectedRow() int {
	v, _, _ := pSend.Call(controls[903], LVM_GETNEXTITEM, ^uintptr(0), uintptr(LVNI_SELECTED))
	if int32(v) < 0 {
		return -1
	}
	return int(v)
}
func weightSelectRow(idx int) {
	if idx < 0 {
		return
	}
	state := listItem{State: LVIS_SELECTED, StateMask: LVIS_SELECTED}
	pSend.Call(controls[903], LVM_SETITEMSTATE, uintptr(idx), uintptr(unsafe.Pointer(&state)))
}
func weightListColumns() {
	h := controls[903]
	columns := []struct {
		txt   string
		width int
	}{{"Наименование", 355}, {"Способ расчёта", 135}, {"t, мм", 74}, {"ρ, кг/м³", 99}, {"gн, кгс/м²", 109}, {"γf", 70}, {"gр, кгс/м²", 120}, {"Удалить", 77}}
	for i, c := range columns {
		p := ptr(c.txt)
		col := listColumn{Mask: 0x0001 | 0x0002 | 0x0004, Fmt: 0, Cx: int32(c.width), Text: p, Sub: int32(i)}
		pSend.Call(h, LVM_INSERTCOLUMNW, uintptr(i), uintptr(unsafe.Pointer(&col)))
	}
	// Full-row selection and grid lines. 0x4 was LVS_EX_CHECKBOXES: remove it.
	pSend.Call(h, LVM_SETEXTENDEDLISTVIEWSTYLE, 0, 0x20|0x1)
}
func weightDisplayValues(r WeightRow) []string {
	thick, den := "", ""
	if r.Kind == "layer" {
		thick = decimalInput(r.Thickness, 1)
		den = decimalInput(r.Density, 1)
	}
	mode := "По толщине"
	if r.Kind != "layer" {
		mode = "Готовая нагрузка"
	}
	return []string{r.Name, mode, thick, den, decimalInput(r.NormKgf(), 1), decimalInput(r.Gamma, 4), decimalInput(r.NormKgf()*r.Gamma, 1), "Удалить"}
}

func weightRememberSnapshot() {
	w := weightCurrent()
	weightTotals()
	if activePage != "weight" || len(w.Rows) == 0 || w.Validate() != nil {
		return
	}
	if !project.ReportManaged {
		project.initReportBlocks(tempSections.Sections)
		for _, b := range project.SavedBlocks {
			if b.Kind == "weight" && b.WeightCalc != nil {
				weightCurrentBlockID = b.ID
				break
			}
		}
	}
	weightCurrentBlockID = project.putWeightCalc(weightCurrentBlockID, *w)
	projectHomeOverview()
}

// Update one edited row in-place. ListView scroll offset remains unchanged.
func weightRefreshSingleRow(ix int) {
	if ix < 0 || ix >= len(weightCurrent().Rows) {
		return
	}
	values := weightDisplayValues(weightCurrent().Rows[ix])
	if ix < len(tempRowsCache[903]) {
		tempRowsCache[903][ix] = values
	}
	for i, value := range values {
		sub := listItem{Sub: int32(i), Text: ptr(value)}
		pSend.Call(controls[903], LVM_SETITEMTEXTW, uintptr(ix), uintptr(unsafe.Pointer(&sub)))
	}
	currentWeightIndex = ix
	weightSelectRow(ix)
	weightRememberSnapshot()
}

func weightRenderRows(want int) {
	weightHideCellEdit()
	w := weightCurrent()
	if activePage == "weight" && controls[900] != 0 {
		w.Name = strings.TrimSpace(get(900))
	}
	weightUIBusy = true
	top, _, _ := pSend.Call(controls[903], LVM_FIRST+39, 0, 0) // LVM_GETTOPINDEX
	tempRowsCache[903] = make([][]string, 0, len(w.Rows))
	pSend.Call(controls[903], LVM_DELETEALLITEMS, 0, 0)
	for i, r := range w.Rows {
		values := weightDisplayValues(r)
		tempRowsCache[903] = append(tempRowsCache[903], values)
		it := listItem{Mask: 1, Index: int32(i), Text: ptr(values[0])}
		pSend.Call(controls[903], LVM_INSERTITEMW, 0, uintptr(unsafe.Pointer(&it)))
		for j, val := range values[1:] {
			sub := listItem{Sub: int32(j + 1), Text: ptr(val)}
			pSend.Call(controls[903], LVM_SETITEMTEXTW, uintptr(i), uintptr(unsafe.Pointer(&sub)))
		}
	}
	if want >= len(w.Rows) {
		want = len(w.Rows) - 1
	}
	currentWeightIndex = want
	if want >= 0 {
		weightSelectRow(want)
	}
	// Do not jump back to row zero after a structural change.
	if int(top) < len(w.Rows) && top > 0 {
		pSend.Call(controls[903], LVM_FIRST+19, top, 0) // LVM_ENSUREVISIBLE
	}
	weightUIBusy = false
	weightRememberSnapshot()
}
func weightShowEditor(ix int) {
	weightUIBusy = true
	defer func() { weightUIBusy = false }()
	fields := []int{904, 905, 906, 907, 908, 909}
	if ix < 0 || ix >= len(weightCurrent().Rows) {
		for _, i := range fields {
			set(i, "")
		}
		set(931, "Выберите строку расчёта для редактирования.")
		return
	}
	r := weightCurrent().Rows[ix]
	set(904, r.Name)
	if r.Kind == "layer" {
		set(905, decimalInput(r.Thickness, 1))
		set(906, decimalInput(r.Density, 1))
		set(931, "Слой по толщине и плотности. Изменения сохраняются автоматически.")
	} else {
		set(905, "")
		set(906, "")
		set(931, "Готовая нагрузка на 1 м². Название и веса можно изменить.")
	}
	set(907, decimalInput(r.NormKgf(), 1))
	set(908, decimalInput(r.Gamma, 4))
	set(909, decimalInput(r.NormKgf()*r.Gamma, 1))
	for _, id := range []int{905, 906} {
		pEnable.Call(controls[id], boolToUintptr(r.Kind == "layer"))
	}
}
func boolToUintptr(v bool) uintptr {
	if v {
		return 1
	}
	return 0
}
func weightTotals() {
	w := weightCurrent()
	n, d, g := w.Totals()
	set(928, fmt.Sprintf("ИТОГО:   gн = %.1f кгс/м²  (%.3f кПа)       gр = %.1f кгс/м²  (%.3f кПа)", n, kgfM2ToKpa(n), d, kgfM2ToKpa(d)))
	if n > 0 {
		set(929, fmt.Sprintf("Эквивалентный коэффициент надёжности γf = %.3f", g))
	} else {
		set(929, "Итоговый коэффициент надёжности: не определён (нулевой вес)")
	}
}
func weightEditChanged(id int) {
	if weightUIBusy {
		return
	}
	ix := currentWeightIndex
	w := weightCurrent()
	if ix < 0 || ix >= len(w.Rows) {
		return
	}
	r := w.Rows[ix]
	if id == 904 {
		name := strings.TrimSpace(get(904))
		if name == "" {
			weightShowEditor(ix)
			return
		}
		r.Name = name
		w.Rows[ix] = r
		weightRefreshSingleRow(ix)
		return
	}
	v, e := parseNumber(get(id))
	if e != nil {
		weightShowEditor(ix)
		return
	}
	fields := map[int]string{905: "thickness", 906: "density", 907: "norm", 908: "gamma", 909: "design"}
	next, e := EditWeightField(r, fields[id], v)
	if e != nil {
		message(e.Error(), true)
		weightShowEditor(ix)
		return
	}
	w.Rows[ix] = next
	weightRefreshSingleRow(ix)
}
func weightMove(dir int) {
	w := weightCurrent()
	i := currentWeightIndex
	j := i + dir
	if i < 0 || j < 0 || j >= len(w.Rows) {
		return
	}
	if MoveWeightRow(w.Rows, i, j) {
		weightRenderRows(j)
	}
}
func weightRemove() {
	w := weightCurrent()
	ix := currentWeightIndex
	if ix < 0 || ix >= len(w.Rows) {
		return
	}
	w.Rows = append(w.Rows[:ix], w.Rows[ix+1:]...)
	weightRenderRows(ix)
}
func weightDuplicate() {
	w := weightCurrent()
	ix := currentWeightIndex
	if ix < 0 || ix >= len(w.Rows) {
		return
	}
	w.Rows = DuplicateWeightRow(w.Rows, ix)
	weightRenderRows(ix + 1)
}
func weightDropRow(from, to int) {
	w := weightCurrent()
	if MoveWeightRow(w.Rows, from, to) {
		weightRenderRows(to)
	} else if from >= 0 && from < len(w.Rows) {
		weightRenderRows(from)
	}
}
func weightAddToCatalog() {
	if chosenCategory == "" {
		message("Сначала выберите категорию слева.", true)
		return
	}
	w := weightCurrent()
	i := currentWeightIndex
	if i < 0 || i >= len(w.Rows) {
		message("Выберите строку расчёта.", true)
		return
	}
	r := w.Rows[i]
	m := material{ID: fmt.Sprintf("user-%d", len(catalogUser.Items)+1), CategoryID: chosenCategory, Name: r.Name, Kind: r.Kind, GNorm: r.NormKgf(), Thickness: r.Thickness, Gamma: r.Gamma}
	if r.Kind == "layer" {
		v := r.Density
		m.Max = &v
		m.Min = &v
	}
	used := map[string]bool{}
	for _, v := range catalogUser.Items {
		used[v.ID] = true
	}
	for used[m.ID] {
		m.ID = fmt.Sprintf("user-%d", len(catalogUser.Items)+len(used)+1)
	}
	catalogUser.Items = append(catalogUser.Items, m)
	if err := weightSaveCustom(); err != nil {
		message(err.Error(), true)
		return
	}
	weightFillMaterials()
	message("Запись добавлена в выбранную категорию личного справочника.", false)
}
func weightRemoveCatalogItem() {
	m, ok := weightSelectedMaterial()
	if !ok {
		return
	}
	for i, entry := range catalogUser.Items {
		if entry.ID == m.ID && entry.CategoryID == m.CategoryID {
			catalogUser.Items = append(catalogUser.Items[:i], catalogUser.Items[i+1:]...)
			if e := weightSaveCustom(); e != nil {
				message(e.Error(), true)
				return
			}
			weightFillMaterials()
			return
		}
	}
	message("Встроенные материалы не удаляются. Можно удалить только свои дополнения.", true)
}
func weightExportCustom() {
	path := chooseSave("ebcat", "Сохранение моих материалов", "Мои материалы.ebcat", false)
	if path == "" {
		return
	}
	b, e := json.MarshalIndent(catalogUser, "", "  ")
	if e == nil {
		e = os.WriteFile(path, append(b, '\n'), 0644)
	}
	if e != nil {
		message(e.Error(), true)
	} else {
		message("Ваши материалы сохранены в файл. При обновлении программы его можно загрузить обратно.", false)
	}
}
func weightImportCustom() {
	path := chooseSave("ebcat", "Загрузка моих материалов", "", true)
	if path == "" {
		return
	}
	b, e := os.ReadFile(path)
	if e != nil {
		message(e.Error(), true)
		return
	}
	var inc weightCatalog
	if e = json.Unmarshal(b, &inc); e != nil {
		message("Не удалось прочитать файл с материалами.", true)
		return
	}
	if inc.SchemaVersion != 1 || len(inc.Items) > 3000 {
		message("Этот файл с материалами не поддерживается.", true)
		return
	}
	ids := map[string]bool{}
	for _, m := range catalogUser.Items {
		ids[m.ID] = true
	}
	validCats := map[string]bool{}
	for _, c := range catalogBase.Categories {
		validCats[c.ID] = true
	}
	count := 0
	for _, m := range inc.Items {
		if m.CategoryID == "c25" {
			m.CategoryID = "c3"
		}
		if strings.TrimSpace(m.Name) == "" || !validCats[m.CategoryID] || (m.Kind != "layer" && m.Kind != "direct") || math.IsNaN(m.Gamma) || m.Gamma <= 0 {
			continue
		}
		if ids[m.ID] {
			continue
		}
		validExtras := []string{}
		for _, cat := range m.CategoryIDs {
			if validCats[cat] && cat != m.CategoryID {
				validExtras = append(validExtras, cat)
			}
		}
		m.CategoryIDs = validExtras
		catalogUser.Items = append(catalogUser.Items, m)
		ids[m.ID] = true
		count++
	}
	if e = weightSaveCustom(); e != nil {
		message(e.Error(), true)
		return
	}
	weightFillMaterials()
	message(fmt.Sprintf("Добавлено записей: %d. Существующие дополнения сохранены.", count), false)
}
func weightSaveProject() {
	path := chooseSave("ebala", "Проект Ебалы", "Расчёт.ebala", false)
	if path == "" {
		return
	}
	if e := project.Save(path); e != nil {
		message(e.Error(), true)
	} else {
		message("Расчёт сохранён.", false)
	}
}
func weightOpenProject() {
	path := chooseSave("ebala", "Проект Ебалы", "", true)
	if path == "" {
		return
	}
	p, e := LoadProject(path)
	if e != nil {
		message(e.Error(), true)
		return
	}
	project = p
	// Projects from 0.03 are converted only when opened, not in a new session.
	weightRenderRows(-1)
	refreshClimateUI()
	message("Проект открыт.", false)
}
func weightExport(kind string) {
	w := *weightCurrent()
	w.Name = strings.TrimSpace(get(900))

	if e := w.Validate(); e != nil {
		message(e.Error(), true)
		return
	}
	path := chooseSave(kind, "Отчёт "+strings.ToUpper(kind), "Собственный вес."+kind, false)
	if path == "" {
		return
	}
	saveAndNotify(path, func(file string) error {
		if kind == "xlsx" {
			return ExportWeightExcel(w, file)
		}
		return ExportWeightWord(w, file)
	})
}
func weightNameSave() {
	if weightUIBusy {
		return
	}
	name := strings.TrimSpace(get(900))
	weightCurrent().Name = name
	if weightCurrentBlockID > 0 {
		project.renameBlock(weightCurrentBlockID, name)
	}
}
func weightAction(id int) bool {
	switch id {
	case 910:
		weightAddMaterial()
	case 911:
		weightAddFree()
	case 912:
		weightRemove()
	case 932:
		weightDuplicate()
	case 913:
		weightMove(-1)
	case 914:
		weightMove(1)
	case 915:
		weightAddToCatalog()
	case 916:
		weightRemoveCatalogItem()
	case 917:
		weightExportCustom()
	case 918:
		weightImportCustom()
	case 919:
		weightNameSave()
		weightSaveProject()
	case 920:
		weightOpenProject()
	case 921:
		weightNameSave()
		weightExport("xlsx")
	case 922:
		weightNameSave()
		weightExport("docx")
	case 923:
		weightTotals()
	case 955:
		// Reset only the editable draft; stored blocks are deleted in the overview.
		weightCurrentBlockID = 0
		editingReportBlockID = 0
		project.Weight = &WeightCalc{Rows: []WeightRow{}}
		set(900, "")
		weightRenderRows(-1)
		projectHomeOverview()
	case 956:
		weightNameSave()
		w := *weightCurrent()
		if len(w.Rows) == 0 {
			message("Добавьте хотя бы один слой или нагрузку", true)
			return true
		}
		if e := w.Validate(); e != nil {
			message(e.Error(), true)
			return true
		}
		if !project.ReportManaged {
			project.initReportBlocks(tempSections.Sections)
			for _, b := range project.SavedBlocks {
				if b.Kind == "weight" && b.WeightCalc != nil {
					weightCurrentBlockID = b.ID
					break
				}
			}
		}
		weightCurrentBlockID = project.putWeightCalc(weightCurrentBlockID, w)
		name, ok := askReportBlockName(project.suggestedCurrentName("weight", weightCurrentBlockID))
		if !ok {
			return true
		}
		project.renameBlock(weightCurrentBlockID, name)
		weightCurrentBlockID = 0
		editingReportBlockID = 0
		project.Weight = &WeightCalc{Rows: []WeightRow{}}
		set(900, "")
		weightRenderRows(-1)
		projectHomeOverview()
	case 951:
		weightApplyTemplate()
	case 952:
		weightSaveTemplate()
	case 953:
		weightDeleteTemplate()
	case 954:
		weightShowGammaReference()
	default:
		return false
	}
	return true
}
func weightNotification(id int, code int) bool {
	if activePage != "weight" {
		return false
	}
	if id == 901 && code == LBN_SELCHANGE {
		weightChosenCat()
		return true
	}
	if id == 950 && code == 2 { // double click a ready-made construction
		weightApplyTemplate()
		return true
	}
	if id == 950 && code == LBN_SELCHANGE {
		return true
	}
	if id == 902 && code == LBN_SELCHANGE {
		return true
	}
	if id == 902 && code == 2 {
		weightAddMaterial()
		return true
	}
	if code == EN_CHANGE && id == 930 {
		if weightCatLoaded {
			weightFillMaterials()
		}
		return true
	}
	if id == 941 && code == 1 /* CBN_SELCHANGE */ {
		weightCommitKindSelection()
		return true
	}
	if id == 940 && code == EN_KILLFOCUS_WEIGHT {
		weightCommitCellEdit(false)
		return true
	}
	if code == EN_KILLFOCUS_WEIGHT && id == 900 {
		weightNameSave()
		return true
	}
	return false
}
func weightNotify(lp uintptr) bool {
	if activePage != "weight" {
		return false
	}
	n := (*notificationHeader)(unsafe.Pointer(lp))
	if n.ID == 903 {
		if n.Code == -2 {
			act := (*weightItemActivate)(unsafe.Pointer(lp))
			if act.Item >= 0 && act.Sub == 7 {
				w := weightCurrent()
				ix := int(act.Item)
				if ix < len(w.Rows) {
					w.Rows = append(w.Rows[:ix], w.Rows[ix+1:]...)
					weightRenderRows(ix)
				}
				return true
			}
		}
		if n.Code == -109 { // LVN_BEGINDRAG
			act := (*weightItemActivate)(unsafe.Pointer(lp))
			weightBeginDrag(int(act.Item))
			return true
		}
		if n.Code == -3 { // NM_DBLCLK, including subitem information
			act := (*weightItemActivate)(unsafe.Pointer(lp))
			if act.Item >= 0 {
				weightStartCellEdit(int(act.Item), int(act.Sub))
				return true
			}
		}
		if n.Code == -2 || n.Code == -101 || n.Code == -3 {
			ix := weightSelectedRow()
			if ix >= 0 {
				currentWeightIndex = ix
			}
			return true
		}
	}
	return false
}
func weightPageEntered() {
	if !weightCatLoaded {
		if e := weightLoadCatalog(); e != nil {
			message(e.Error(), true)
			return
		}
	}
	if project.Weight == nil {
		w := project.CurrentWeight()
		project.Weight = &w
	}
	if !weightTemplatesLoaded {
		if e := weightLoadTemplates(); e != nil {
			message(e.Error(), true)
			return
		}
	}
	weightCategories()
	weightFillMaterials()
	weightRefreshTemplates()
	weightHideCatalogEdit()
	weightHideCellEdit()
	set(900, weightCurrent().Name)
	tempResize()
	weightRenderRows(-1)
}
func weightFooter() {
	button(955, "Сбросить нагрузки", 20, 810, 218, 36)
	button(956, "Сохранить в ведомость и новый расчёт", 249, 810, 398, 36)
	button(921, "Расчёт Excel", 658, 810, 219, 36)
	button(922, "Отчёт Word", 888, 810, 220, 36)
}
func buildWeightNative() {
	buildingPage = "weight"
	navigation("Собственный вес", "Справочник материалов и конструкций · расчёт нагрузки на 1 м²")
	label("Название конструкции (необязательно)", 20, 98, 300, 24)
	edit(900, "", 324, 95, 784, 27)
	label("КАТЕГОРИИ · раскрывайте группы щелчком", 20, 131, 255, 22)
	label("МАТЕРИАЛЫ · свои записи синим", 266, 131, 287, 22)
	label("Поиск", 551, 133, 54, 22)
	edit(930, "", 602, 128, 167, 27)
	label("ГОТОВЫЕ КОНСТРУКЦИИ", 785, 131, 323, 22)
	create("LISTBOX", "", WS_CHILD|WS_VISIBLE|WS_BORDER|WS_VSCROLL|LBS_NOTIFY, 20, 158, 230, 268, 901)
	create("LISTBOX", "", WS_CHILD|WS_VISIBLE|WS_BORDER|WS_VSCROLL|LBS_NOTIFY|LBS_OWNERDRAWFIXED|LBS_HASSTRINGS, 265, 158, 504, 268, 902)
	create("LISTBOX", "", WS_CHILD|WS_VISIBLE|WS_BORDER|WS_VSCROLL|LBS_NOTIFY, 785, 158, 323, 268, 950)
	button(910, "Добавить", 265, 432, 115, 32)
	button(917, "Сохранить материалы", 388, 432, 186, 32)
	button(918, "Загрузить материалы", 582, 432, 187, 32)
	button(951, "Применить", 785, 432, 97, 32)
	button(952, "Сохранить состав", 890, 432, 143, 32)
	button(953, "Удалить", 1041, 432, 67, 32)
	label("СОСТАВ КОНСТРУКЦИИ · двойной щелчок по ячейке для редактирования", 20, 473, 800, 24)
	button(954, "Подсказка по γf", 864, 470, 244, 27)
	initCommon.Call()
	create("SysListView32", "", WS_CHILD|WS_VISIBLE|WS_BORDER|WS_VSCROLL|LVS_REPORT|LVS_SINGLESEL|LVS_SHOWSELALWAYS|0x0400, 20, 500, 1088, 220, 903)
	weightListColumns()
	im, _, _ := tempImageListCreate.Call(1, 40, 0x20, 1, 1)
	pSend.Call(controls[903], 0x1003, 1, im)
	edit(940, "", 20, 500, 200, 22)
	create("COMBOBOX", "", WS_CHILD|WS_VSCROLL|0x0003 /* CBS_DROPDOWNLIST */, 20, 500, 170, 240, 941)
	pSend.Call(controls[941], 0x0143 /* CB_ADDSTRING */, 0, uintptr(unsafe.Pointer(ptr("По толщине"))))
	pSend.Call(controls[941], 0x0143 /* CB_ADDSTRING */, 0, uintptr(unsafe.Pointer(ptr("Готовая нагрузка"))))
	weightInitCellSubclass()
	weightInitListSubclass()
	weightHideCellEdit()
	button(911, "Добавить строку", 20, 728, 186, 31)
	button(912, "Удалить строку", 216, 728, 185, 31)
	button(913, "↑", 415, 728, 50, 31)
	button(914, "↓", 471, 728, 50, 31)
	button(915, "Сохранить строку в справочник", 535, 728, 343, 31)
	create("STATIC", "", WS_CHILD|WS_VISIBLE, 20, 763, 1088, 25, 928)
	weightTotalFont, _, _ = pCreateFont.Call(21, 0, 0, 0, 700, 0, 0, 0, 1, 0, 0, 0, 0, uintptr(unsafe.Pointer(ptr("Segoe UI"))))
	if weightTotalFont != 0 {
		pSend.Call(controls[928], WM_SETFONT, weightTotalFont, 1)
	}
	create("STATIC", "", WS_CHILD|WS_VISIBLE, 20, 786, 1088, 23, 929)
	weightFooter()
}

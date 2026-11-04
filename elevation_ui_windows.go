//go:build windows

package main

import (
	"unsafe"
)

const CB_RESETCONTENT = 0x014B

var mountainChoices []MountainArea

func refreshMountainOptions(region, selectedID string) {
	if controls[643] == 0 {
		return
	}
	mountainChoices = append([]MountainArea(nil), mountainAreas...)
	changingUnits = true
	send(643, CB_RESETCONTENT, 0, 0)
	pSend.Call(controls[643], CB_ADDSTRING, 0, uintptr(unsafe.Pointer(ptr("Не применять"))))
	index := 0
	for i, m := range mountainChoices {
		pSend.Call(controls[643], CB_ADDSTRING, 0, uintptr(unsafe.Pointer(ptr(m.Caption))))
		if m.ID == selectedID {
			index = i + 1
		}
	}
	send(643, CB_SETCURSEL, uintptr(index), 0)
	changingUnits = false
}
func selectedMountainID() string {
	i := int(int32(send(643, CB_GETCURSEL, 0, 0)))
	if i > 0 && i <= len(mountainChoices) {
		return mountainChoices[i-1].ID
	}
	return ""
}
func fillMountainControls(s Site) {
	refreshMountainOptions(s.Region, s.MountainID)
	set(644, s.ElevationText())
	set(645, mountainStatus(s))
}
func mountainStatus(s Site) string {
	if warning := s.ElevationWarning(); warning != "" {
		return "ПРЕДУПРЕЖДЕНИЕ: " + warning
	}
	if s.MountainID == "" {
		if s.ValidElevationForCalc() && s.ElevationM < 500 {
			return "Поправка высоты не применяется — высота менее 500 м (" + decimalInput(s.ElevationM, 2) + " м)."
		}
		if s.ValidElevationForCalc() {
			return "Высота " + decimalInput(s.ElevationM, 2) + " м. Поправка не включена; проверьте район по карте 1 СП."
		}
		return "Поправка высоты не включена. Выберите горный район и укажите отметку площадки."
	}
	_, _, m, on, err := s.ElevationDetails()
	if err != nil {
		return "ПРЕДУПРЕЖДЕНИЕ: " + err.Error() + ". Высотная поправка не применяется."
	}
	if !on {
		return "ПРЕДУПРЕЖДЕНИЕ: Данные Е.1 не позволяют применить поправку. Ручной выбор сохранён."
	}
	if s.ElevationM <= m.H0 {
		return "Горная территория назначена вручную. h ≤ h₀=" + decimalInput(m.H0, 0) + " м; снег не увеличивается."
	}
	return s.ElevationSummary() + " (только снег)"
}
func readMountainFields(s *Site) {
	s.MountainID = selectedMountainID()
	raw := get(644)
	s.SetElevationText(raw)
	s.ElevationSource = "Введено пользователем"
	same := s.Region == draftSite.Region && s.Place == draftSite.Place && s.District == draftSite.District
	if same && raw == draftSite.ElevationText() {
		s.ElevationSource = draftSite.ElevationSource
	}
}

//go:build windows

package main

import (
	"errors"
	"strings"
	"sync"
)

type elevationReply struct {
	token                            uint64
	place, region, district, oldEdit string
	elevation                        WikiElevation
	err                              error
}

var elevationLock sync.Mutex
var elevationPending elevationReply
var elevationSequence uint64

func lookupCurrentElevation() {
	place, region, district := strings.TrimSpace(get(633)), strings.TrimSpace(get(631)), strings.TrimSpace(get(632))
	if place == "" {
		set(645, "Для проверки высоты введите населённый пункт.")
		return
	}
	if region == "" {
		set(645, "Для проверки высоты сначала укажите субъект РФ (защита от одноимённых населённых пунктов).")
		return
	}
	elevationLock.Lock()
	elevationSequence++
	token := elevationSequence
	elevationLock.Unlock()
	old := get(644)
	set(645, "Проверяю высоту населённого пункта по Википедии / Wikidata...")
	go func() {
		v, err := lookupWikiElevation(place, region, district)
		elevationLock.Lock()
		elevationPending = elevationReply{token: token, place: place, region: region, district: district, oldEdit: old, elevation: v, err: err}
		elevationLock.Unlock()
		pPostMessage.Call(hwnd, WM_APP_ELEVATION, 0, 0)
	}()
}
func finishElevationLookup() {
	elevationLock.Lock()
	r := elevationPending
	latest := elevationSequence
	elevationLock.Unlock()
	if r.token != latest || r.place != strings.TrimSpace(get(633)) || r.region != strings.TrimSpace(get(631)) || r.district != strings.TrimSpace(get(632)) {
		return
	}
	if r.err != nil {
		if errors.Is(r.err, errWikiNotFound) {
			set(645, "В Википедии не нашлось. Можно указать высоту вручную.")
		} else if errors.Is(r.err, errWikiRegion) {
			set(645, "Найдено несколько одноимённых пунктов. Уточните район или задайте высоту вручную.")
		} else {
			set(645, "Ошибка при получении высоты из Википедии. Подробности показаны отдельно.")
			message("Проверка высоты не завершена.\n\n"+r.err.Error()+"\n\nСделайте снимок этого сообщения для исправления поиска.", false)
		}
		return
	}
	if strings.TrimSpace(get(644)) != strings.TrimSpace(r.oldEdit) {
		return
	} // user already edited height
	set(644, decimalInput(r.elevation.Height, 2))
	draftSite.SetElevationText(get(644))
	draftSite.ElevationSource = r.elevation.Source
	s, err := readSiteManual()
	if err == nil {
		set(645, mountainStatus(s))
	} else if r.elevation.Height < 500 {
		set(645, "Поправка высоты не применяется — высота менее 500 м. Источник: Википедия / Wikidata.")
	} else {
		set(645, "Высота из Википедии: "+decimalInput(r.elevation.Height, 2)+" м. Применение по карте 1 СП проверяется отдельно.")
	}
}

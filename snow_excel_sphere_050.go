package main

import (
	"fmt"
	"regexp"
	"sort"
	"strings"
)

// These cells implement the spherical/conical geometry independently of the
// Go snow arrays. Values from Go are retained only in labelled audit columns.
// The Excel formulas are intentionally expressed in ordinary worksheet syntax.
func snowExcelSphereGeometry(s SnowCalc, input string) string {
	var b strings.Builder
	add := func(i int, vals ...string) { b.WriteString(tempExcelRow(i, vals...)) }
	src := "'" + input + "'!"
	add(1, "На этом листе рассчитана геометрия Б.15а и промежуточные коэффициенты. В столбце B указаны формулы Excel. Ниже приведены 121 контрольное сечение сферической части.")
	add(2, "H, м", "="+src+"B10", "Высота конька")
	add(3, "D, м", "="+src+"B11", "Диаметр покрытия")
	add(4, "f, м", "="+src+"B15", "Подъём от стенки до конька")
	add(5, "r, м", "="+src+"B16", "Радиус конического участка")
	add(6, "α₂, град", "="+src+"B13", "Постоянный угол конуса")
	add(7, "α₁ у стенки, град", "="+src+"B34", "Тангенциальная геометрия сферы")
	add(8, "dx, м", "=B3/2-B5", "Горизонтальная ширина сферического пояса")
	add(9, "dh, м", "=B4-B5*TAN(RADIANS(B6))", "Перепад высот по сферическому поясу")
	denom := "(2*(B9*COS(RADIANS(B6))-B8*SIN(RADIANS(B6))))"
	add(10, "R, м (со знаком)", "=IF(ABS("+denom+")<1E-9,1E12,(B8^2+B9^2)/"+denom+")", "Радиус окружности касательного сопряжения")
	add(11, "Xc, м", "=B5-B10*SIN(RADIANS(B6))", "Координата центра по горизонтали")
	add(12, "Yc, м", "=B2-B5*TAN(RADIANS(B6))-B10*COS(RADIANS(B6))", "Координата центра по высоте")
	add(13, "Cr1 по (Б.10)", "=2.55-EXP(0.8-14*B4/B3)", "Б.10: Cr1 = 2,55 − exp(0,8 − 14f/D)")
	add(14, "r30, м", "=IF(OR(B11+ABS(B10)*0.5<=0,ABS(B10)>1E11),B5,B11+ABS(B10)*0.5)", "Радиус сечения α=30°")
	add(15, "μ₂ на сопряжении", "=B30", "Значение сферического коэффициента у конуса")
	add(16, "μ₂,max", "=MAX(B30:B150)", "Максимум коэффициента Б.10 из 121 сечения")
	add(17, "Угол для зоны 5, град", "=IF("+src+"B32=\"Да\",B7,B6)", "С галочкой α₁; по умолчанию α₂")
	add(18, "μ₂, зона 5", "=IF(AND(B17>7,B17<=15),0.8*IF(B6<=30,1,IF(B6>=60,0,(60-B6)/30)),IF(AND(B17>15,B17<=30),0.5*B15,0))", "Ветвление зоны 5; выбранный угол в B17")
	add(19, "Cr2 по (Б.12), (Б.13)", "=IF(OR(B6<=7,B6>=60),0,IF(B6<=30,2.15*POWER(RADIANS(B6),1/3),1.7*30/B6))", "Коэффициент конической части; угол α₂")
	add(20, "Отметка стенки, м", "=B2-B4", "H − f")
	add(24, "Формулы ниже вычисляются в Excel по радиусу и углу местной касательной, без подстановки готового μ из Go.")
	add(25, "Радиус от центра, м", "μ₂ сферы без sin β", "|α₁(z)|, град", "Основание")
	for i := 0; i <= 120; i++ {
		row := 30 + i
		rr := fmt.Sprintf("A%d", row)
		aa := fmt.Sprintf("C%d", row)
		add(row, fmt.Sprintf("=$B$5+($B$3/2-$B$5)*%d/120", i),
			fmt.Sprintf("=IF(%s<=30,MAX(0,$B$13*(%s/$B$14)^2),IF(%s<=45,MAX(0,$B$13+(1.5-$B$13)*(%s-30)/15),IF(%s<60,1.5*(60-%s)/15,0)))", aa, rr, aa, aa, aa, aa),
			fmt.Sprintf("=IF(%s<=$B$5,$B$6,ABS(DEGREES(ASIN(MAX(-1,MIN(1,(%s-$B$11)/ABS($B$10)))))))", rr, rr),
			"Б.10, затем линейная интерполяция 30°–60°")
	}
	return b.String()
}

func snowExcelSpherePolar(s SnowCalc, c SnowCase, index int, geometry, input, coef string) string {
	var b strings.Builder
	add := func(i int, v ...string) { b.WriteString(tempExcelRow(i, v...)) }
	g := "'" + geometry + "'!"
	src := "'" + input + "'!"
	k := "'" + coef + "'!"
	add(1, fmt.Sprintf("На этом листе вычисляется коэффициент μ по каждому сектору Б.15а (вариант %d). Столбцы E и H независимы: E — формула Excel, H — контроль из ядра, I — разность.", index+1))
	add(2, "Sg, кПа", "="+src+"B3", "ce", "="+k+"B19", "ct", "="+k+"B21", "γf", "="+src+"B4")
	add(3, "β, град", "От края, м", "Радиус z, м", "Зона", "μ по формулам Excel", "Sн=μ×ce×ct×Sg", "Sр=γf×Sн", "μ из расчётного ядра", "Δ μ Excel − Go", "|α(z)|, град", "Отметка кровли, м", "μ₂ сферы / sinβ", "μ₂ конуса / sinβ", "Нормативная формула")
	for i, p := range c.Polar {
		r := 4 + i
		z := fmt.Sprintf("C%d", r)
		beta := fmt.Sprintf("A%d", r)
		zz := fmt.Sprintf("D%d", r)
		angle := fmt.Sprintf("J%d", r)
		h := fmt.Sprintf("K%d", r)
		zone := ""
		mu := ""
		meaning := ""
		if lower, upper, partial := snowPartialSectorBounds(c); partial {
			// The sector limits are determined from the load case, not from Go's μ.
			zone = ""
			base := fmt.Sprintf("IF(%s<0,0,IF(%s<=30,1,IF(%s>=60,0,(60-%s)/30)))", h, angle, angle, angle)
			mu = fmt.Sprintf("=IF(AND(MOD(%s,360)>=%g,MOD(%s,360)<%g),%s,0)", beta, lower, beta, upper, base)
			meaning = "П. 10.4: частичное загружение сектора на основе μ₁"
		} else if index == 0 {
			zone = "" // Variant 1 has no spatial-zone partition.
			mu = fmt.Sprintf("=IF(%s<0,0,IF(%s<=30,1,IF(%s>=60,0,(60-%s)/30)))", h, angle, angle, angle)
			meaning = "Таблица Б.3: μ₁ по местной касательной"
		} else {
			betasym := fmt.Sprintf("IF(%s>180,360-%s,%s)", beta, beta, beta)
			zone = fmt.Sprintf("=IF(%s>=%s$B$5,IF(%s<60,1,IF(%s>120,6,2)),IF(%s<60,3,IF(%s>120,5,4)))", z, g, betasym, betasym, betasym, betasym)
			sine := fmt.Sprintf("SIN(RADIANS(%s))", betasym)
			mu = fmt.Sprintf("=IF(%s<0,0,IF(%s=1,L%d*%s,IF(%s=2,0.5*%s$B$16,IF(OR(%s=3,%s=4),M%d*%s,IF(%s=5,%s$B$18,IF(%s=6,0.5*L%d*%s,0))))))", h, zz, r, sine, zz, g, zz, zz, r, sine, zz, g, zz, r, sine)
			meaning = "Б.12(б), рис. Б.15а: зоны 1–6; (Б.10), (Б.12), (Б.13)"
		}
		local := fmt.Sprintf("=IF(%s<=%s$B$5,%s$B$6,ABS(DEGREES(ASIN(MAX(-1,MIN(1,(%s-%s$B$11)/ABS(%s$B$10)))))))", z, g, g, z, g, g)
		height := fmt.Sprintf("=IF(%s<=%s$B$5,%s$B$2-%s*TAN(RADIANS(%s$B$6)),IF(%s$B$10>=0,%s$B$12+SQRT(MAX(0,%s$B$10^2-(%s-%s$B$11)^2)),%s$B$12-SQRT(MAX(0,%s$B$10^2-(%s-%s$B$11)^2))))", z, g, g, z, g, g, g, g, z, g, g, g, z, g)
		dome := fmt.Sprintf("=IF(%s<=30,MAX(0,%s$B$13*(%s/%s$B$14)^2),IF(%s<=45,MAX(0,%s$B$13+(1.5-%s$B$13)*(%s-30)/15),IF(%s<60,1.5*(60-%s)/15,0)))", angle, g, z, g, angle, g, g, angle, angle, angle)
		cone := fmt.Sprintf("=MAX(0,%s$B$19*%s/%s$B$5)", g, z, g)
		add(r, xnum(p.Beta), fmt.Sprintf("=%s$B$3/2-%s", g, z), xnum(p.Radius), zone, mu,
			fmt.Sprintf("=E%d*$B$2*$D$2*$F$2", r), fmt.Sprintf("=F%d*$H$2", r), xnum(p.Mu), fmt.Sprintf("=E%d-H%d", r, r), local, height, dome, cone, meaning)
	}
	return b.String()
}

func snowExcelFormulaGuide(s SnowCalc) string {
	var b strings.Builder
	add := func(i int, v ...string) { b.WriteString(tempExcelRow(i, v...)) }
	add(1, "На этом листе приведены математические формулы определения коэффициентов μ в человекочитаемом виде, расшифровка обозначений и нормативные основания. Рабочие формулы находятся в числовых листах.")
	add(2, "Обозначение", "Математическая формула / условие", "Расшифровка", "Основание")
	add(3, "Нормативная нагрузка", "Sн = μ · ce · ct · Sg", "μ — коэффициент формы; ce — снос; ct — температура; Sg — вес снега на земле", "СП 20.13330.2016, п. 10.1")
	add(4, "Расчётная нагрузка", "Sр = γf · Sн", "γf — надёжность по снеговой нагрузке", "СП 20.13330.2016, п. 10.12")
	add(5, "Характерный размер", "lc = min(100; 2b − b²/Lmax)", "b=min(L,B), Lmax=max(L,B); у многопролётного покрытия L — суммарная ширина", "СП 20.13330.2016, (10.3)")
	add(6, "Коэффициент сноса", "ce = max(0,5; min(1; (kv − 0,4√k)·(0,8 + 0,002lc)))", "При выполнении условий п. 10.7; k по табл. 11.2, kv по табл. 10.2", "СП 20.13330.2016, п. 10.7")
	if s.Shape == "sphere_cone" {
		add(8, "Сопряжение", "R = (dx²+dh²)/(2·(dh cos α₂ − dx sin α₂))", "dx=D/2−r; dh=f−r·tg α₂; R — подписанный радиус сферы", "Геометрия принятой модели")
		add(9, "Зона 1", "μ₂ = Cr1·(z/r30)²·sin β (до α₁=30°)", "За 30°: линейная интерполяция до 45° и 60°", "Б.10, Б.12(б)")
		add(10, "Зона 2", "μ = 0,5·μ₂,max (при β=90°)", "μ₂,max — максимум по сферической части", "Б.12(б)")
		add(11, "Зоны 3 и 4", "μ₂ = Cr2·(z/r)·sin β", "Cr2=2,15·(α₂π/180)^(1/3) при 7°<α₂≤30°; Cr2=1,7·30/α₂ при 30°<α₂<60°", "Б.12, Б.13")
		add(12, "Зона 5 при 7°<a≤15°", "μ₂ = 0,8·μ₁(α₂)", "a=α₂ по умолчанию; при галочке a=α₁ у стенки", "Б.12(б), интерпретация")
		add(13, "Зона 5 при 15°<a≤30°", "μ₂ = 0,5·μ₂(z=r)", "a=α₂ или α₁; остальные ветви μ₂=0", "Б.12(б), интерпретация")
		add(14, "Зона 6", "μ₂ = 0,5·Cr1·(z/r30)²·sin β (до α₁=30°)", "За 30° используется Б.10 с интерполяцией; множитель 0,5 сохраняется", "Б.10, Б.12(б)")
		add(15, "Угловые границы", "β = 60° и β = 120°", "β отсчитывается от правого радиуса в принятой сетке", "Рисунок Б.15а")
	} else {
		add(8, "Общий коэффициент для скатов", "μ=1 при α≤30°; μ=(60°−α)/30° при 30°<α<60°; μ=0 при α≥60°", "Для иных схем см. лист каждого варианта; при частичной нагрузке учитывается собственная форма", "Приложение Б")
	}
	return b.String()
}

var snowWorkbookSheet = regexp.MustCompile(`<sheet name="[^"]+" sheetId="[0-9]+" r:id="rId[0-9]+"/>`)

func snowExcelReorderSheets(entries map[string]string) {
	wb := entries["xl/workbook.xml"]
	start := strings.Index(wb, "<sheets>")
	end := strings.Index(wb, "</sheets>")
	if start < 0 || end < start {
		return
	}
	content := wb[start+len("<sheets>") : end]
	items := snowWorkbookSheet.FindAllString(content, -1)
	if len(items) == 0 || strings.TrimSpace(snowWorkbookSheet.ReplaceAllString(content, "")) != "" {
		return
	}
	rank := func(sheet string) int {
		switch {
		case strings.Contains(sheet, `name="Исходные данные`):
			return 0
		case strings.Contains(sheet, `name="Коэффициенты`):
			return 1
		case strings.Contains(sheet, `name="Геометрия Б.15а`):
			return 2
		case strings.Contains(sheet, `name="Формулы снега`):
			return 3
		case strings.Contains(sheet, `name="Вариант `):
			return 4
		case strings.Contains(sheet, `name="Сетка Б.15а`):
			return 5
		case strings.Contains(sheet, `name="Нагрузки`):
			return 6
		case strings.Contains(sheet, `name="Схема`):
			return 7
		case strings.Contains(sheet, `name="Эпюра`), strings.Contains(sheet, `name="Сечение`):
			return 8
		default:
			return 7
		}
	}
	sort.SliceStable(items, func(i, j int) bool { return rank(items[i]) < rank(items[j]) })
	entries["xl/workbook.xml"] = wb[:start+len("<sheets>")] + strings.Join(items, "") + wb[end:]
}

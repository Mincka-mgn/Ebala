package main

import (
	"fmt"
	"math"
	"strings"
)

func xCol(n int) string {
	if n < 1 {
		panic("invalid spreadsheet column")
	}
	var a string
	for n > 0 {
		n--
		a = string(rune('A'+n%26)) + a
		n /= 26
	}
	return a
}

func auditNum(col int, row int, v float64) string {
	return cellNum(fmt.Sprintf("%s%d", xCol(col), row), v, 3)
}
func auditText(col int, row int, v string) string {
	return cellText(fmt.Sprintf("%s%d", xCol(col), row), v, 0)
}
func auditFormula(col, row int, f string, cache float64) string {
	return fmt.Sprintf(`<c r="%s%d" s="3"><f>%s</f><v></v></c>`, xCol(col), row, esc(f))
}
func auditAddr(col, row int) string { return fmt.Sprintf("%s%d", xCol(col), row) }

// Independent workbook reconstruction of every unit-load influence ordinate.
// Fixed crane positions are audited, not optimized a second time in Excel.
// cranePlacementSheetParts is shared by standalone and general XLSX exports.
// It contains only accepted spans/wheel positions and independent spreadsheet
// calculations. Go-derived force values are never embedded in formula cells.
type cranePlacementSheetParts struct {
	Summary      string
	Detail       string
	Beam         string
	Inputs       string
	DetailWidths []float64
}

func cranePlacementAuditSheets(r CranePlacementResult) (cranePlacementSheetParts, error) {
	if len(r.Cases) == 0 {
		return cranePlacementSheetParts{}, fmt.Errorf("нет рассчитанных положений кранов")
	}
	n := len(r.SpansMM)
	if n < 1 {
		return cranePlacementSheetParts{}, fmt.Errorf("не заданы пролёты")
	}
	m := n - 1
	beamFirstSpan := 5
	beamLastSpan := beamFirstSpan + n - 1
	beamCoeffs := n + 9
	var beam, detail, summary, source strings.Builder
	// Fixed optimizer positions are a numerical input exception. All loads,
	// span lengths, ordinates and final forces then recalculate in Excel.
	source.WriteString(xrow(1, cellText("A1", "На этом листе приведены принятые длины пролётов и колесные нагрузки. Положения колес и расчётных сечений найдены программой численно; при их изменении расчётный лист пересчитывает усилия без Go.", 1)))
	source.WriteString(xrow(2, cellText("A2", "Единицы: метры, кН. Изменение пролётов или нагрузки изменяет ординаты линий влияния и результирующее усилие.", 0)))
	source.WriteString(xrow(4, cellText("A4", "№ пролёта", 2), cellText("B4", "L, м", 2)))
	for i, length := range r.SpansMM {
		source.WriteString(xrow(5+i, auditNum(1, 5+i, float64(i+1)), auditNum(2, 5+i, length/1000)))
	}
	sourceWheelHeader := 5 + n + 3
	source.WriteString(xrow(sourceWheelHeader, cellText(auditAddr(1, sourceWheelHeader), "№ случая", 2), cellText(auditAddr(2, sourceWheelHeader), "Колесо", 2), cellText(auditAddr(3, sourceWheelHeader), "x, м", 2), cellText(auditAddr(4, sourceWheelHeader), "Pн, кН", 2), cellText(auditAddr(5, sourceWheelHeader), "Сечение, м", 2), cellText(auditAddr(6, sourceWheelHeader), "Вид усилия", 2)))
	sourcePointRow := sourceWheelHeader + 1
	// SHEET 'Балка': lengths, support coordinates, and the tridiagonal system.
	beam.WriteString(xrow(1, cellText("A1", "На этом листе заданы исходные длины пролётов и формулы расчёта коэффициентов схемы балки.", 1)))
	beam.WriteString(xrow(2, cellText("A2", "Единичные нагрузки P=1 кН, длины в метрах, EI постоянно. Метод трёх моментов (Клапейрон).", 0)))
	beam.WriteString(xrow(3, cellText("A3", "Lлев·M(j−1) + 2(Lлев+Lправ)·M(j) + Lправ·M(j+1) = B(j); крайние M=0.", 0)))
	beam.WriteString(xrow(4, cellText("A4", "Пролёт", 2), cellText("B4", "L, м", 2), cellText("C4", "От x, м", 2), cellText("D4", "До x, м", 2)))
	position := 0.0
	for i, lmm := range r.SpansMM {
		row := beamFirstSpan + i
		L := lmm / 1000
		var start string
		if i == 0 {
			start = auditNum(3, row, 0)
		} else {
			start = auditFormula(3, row, fmt.Sprintf("D%d", row-1), position)
		}
		beam.WriteString(xrow(row, auditNum(1, row, float64(i+1)), auditFormula(2, row, fmt.Sprintf("'Исходные данные'!B%d", row), L), start, auditFormula(4, row, fmt.Sprintf("C%d+B%d", row, row), position+L)))
		position += L
	}
	heading := n + 8
	beam.WriteString(xrow(heading, cellText(auditAddr(1, heading), "Внутр. опора j", 2), cellText(auditAddr(2, heading), "Lлев, м", 2), cellText(auditAddr(3, heading), "Lправ, м", 2), cellText(auditAddr(4, heading), "a_j", 2), cellText(auditAddr(5, heading), "b_j", 2), cellText(auditAddr(6, heading), "c_j", 2), cellText(auditAddr(7, heading), "c′_j", 2), cellText(auditAddr(8, heading), "Знаменатель", 2)))
	cPrime := make([]float64, m)
	for j := 1; j <= m; j++ {
		row := beamCoeffs + j - 1
		left, right := r.SpansMM[j-1]/1000, r.SpansMM[j]/1000
		lower := left
		if j == 1 {
			lower = 0
		}
		upper := right
		if j == m {
			upper = 0
		}
		diag := 2 * (left + right)
		denom := diag
		if j > 1 {
			denom -= lower * cPrime[j-2]
		}
		cPrime[j-1] = upper / denom
		leftRef := fmt.Sprintf("B%d", beamFirstSpan+j-1)
		rightRef := fmt.Sprintf("B%d", beamFirstSpan+j)
		lowRef := leftRef
		if j == 1 {
			lowRef = "0"
		}
		upRef := rightRef
		if j == m {
			upRef = "0"
		}
		cpFormula := fmt.Sprintf("F%d/E%d", row, row)
		denomFormula := fmt.Sprintf("E%d", row)
		if j > 1 {
			cpFormula = fmt.Sprintf("F%d/(E%d-D%d*G%d)", row, row, row, row-1)
			denomFormula = fmt.Sprintf("E%d-D%d*G%d", row, row, row-1)
		}
		beam.WriteString(xrow(row, auditNum(1, row, float64(j)), auditFormula(2, row, leftRef, left), auditFormula(3, row, rightRef, right), auditFormula(4, row, lowRef, lower), auditFormula(5, row, fmt.Sprintf("2*(B%d+C%d)", row, row), diag), auditFormula(6, row, upRef, upper), auditFormula(7, row, cpFormula, cPrime[j-1]), auditFormula(8, row, denomFormula, denom)))
	}
	after := beamCoeffs + m + 2
	if m == 0 {
		beam.WriteString(xrow(after, cellText(auditAddr(1, after), "Один пролёт: M опор = 0. Ординаты η вычисляются по треугольной линии влияния.", 0)))
	} else {
		beam.WriteString(xrow(after, cellText(auditAddr(1, after), "Для единичного колеса в пролёте j: B(j)=-a·b·(L+a)/L; B(j−1)=-a·b·(2L−a)/L.", 0)))
		beam.WriteString(xrow(after+1, cellText(auditAddr(1, after+1), "Прямой ход: d′j=(B(j)-a_j·d′(j−1))/(b_j-a_j·c′(j−1)).", 0)))
		beam.WriteString(xrow(after+2, cellText(auditAddr(1, after+2), "Обратный ход: M(j)=d′j-c′j·M(j+1). Затем реакции и η по равновесию каждого пролёта.", 0)))
	}
	// SHEET 'Колёса': one line per unit wheel, all intermediates and formulas.
	detail.WriteString(xrow(1, cellText("A1", "На этом листе вычисляются ординаты линий влияния η и вклады колесных нагрузок по формулам Excel.", 1)))
	detail.WriteString(xrow(2, cellText("A2", "Каждая строка: P=1 кН в координате x. По трём моментам определены опорные M, реакции и η. F=Pфакт·η.", 0)))
	detail.WriteString(xrow(3, cellText("A3", "Фиксированные положения найдены программой. Таблица предназначена для независимой проверки, а не повторной оптимизации.", 0)))
	head := 5
	headers := []string{"№ случая", "Колесо · кран", "x, м", "Pн, кН", "η, м или 1", "Pн·η", "Усилие", "Сеч. x, м", "Пролёт сеч.", "Пролёт колеса", "a, м", "b, м", "L колеса, м", "z сечения, м", "M0, кН·м", "M лев., кН·м", "M прав., кН·м", "R лев., кН", "R прав., кН"}
	for j := 1; j <= m; j++ {
		headers = append(headers, fmt.Sprintf("B%d", j))
	}
	for j := 1; j <= m; j++ {
		headers = append(headers, fmt.Sprintf("d′%d", j))
	}
	for j := 1; j <= m; j++ {
		headers = append(headers, fmt.Sprintf("M%d оп.", j))
	}
	headerCells := make([]string, len(headers))
	for i, h := range headers {
		headerCells[i] = cellText(auditAddr(i+1, head), h, 2)
	}
	detail.WriteString(xrow(head, headerCells...))
	summary.WriteString(xrow(1, cellText("A1", "На этом листе приведены итоговые воздействия мостовых кранов: найденные численным способом положения колес и нагрузки, вычисленные по формулам Excel.", 1)))
	summary.WriteString(xrow(2, cellText("A2", "Оптимальные координаты рассчитаны программой; Excel независимо восстанавливает η по методу трёх моментов.", 0)))
	summary.WriteString(xrow(3, cellText("A3", "Расчётное усилие определяется по зависимости Σ(Pн·η); исходные пролёты и коэффициенты приведены на листе «Балка».", 0)))
	summary.WriteString(xrow(4, cellText("A4", "№", 2), cellText("B4", "Опасное положение", 2), cellText("C4", "Σ(P·η), проверка", 2), cellText("D4", "Ед.", 2), cellText("E4", "Сечение, мм", 2), cellText("F4", "Результат формулы", 2)))
	row := 6
	totalWheelRows := 0
	for _, c := range r.Cases {
		totalWheelRows += len(c.Points)
	}
	endDataRow := 5 + totalWheelRows
	for ci, c := range r.Cases {
		firstSourceRow := sourcePointRow
		target, kind, e := craneTargetSpan(r.SpansMM, c)
		if e != nil {
			return cranePlacementSheetParts{}, e
		}
		subtotal := 0.0
		for wi, p := range c.Points {
			audit, e := craneInfluenceUnitAudit(r.SpansMM, c, p.XMM)
			if e != nil {
				return cranePlacementSheetParts{}, e
			}
			etaOriginal, e := craneCaseInfluence(r.SpansMM, c, p.XMM)
			if e != nil {
				return cranePlacementSheetParts{}, e
			}
			if math.Abs(audit.Eta-etaOriginal) > 1e-7*math.Max(1, math.Abs(etaOriginal)) {
				return cranePlacementSheetParts{}, fmt.Errorf("η не совпадает в положении %d колесо %d: по трём моментам %.9g, из расчётного ядра %.9g", ci+1, wi+1, audit.Eta, etaOriginal)
			}
			subtotal += p.PKN * audit.Eta
			// Shared references to spans and precomputed sweep coefficients on sheet 'Балка'.
			rc := func(col int) string { return auditAddr(col, row) }
			referenceLen := fmt.Sprintf("INDEX('Балка'!$B$%d:$B$%d,%s)", beamFirstSpan, beamLastSpan, rc(9))
			if n == 1 {
				referenceLen = fmt.Sprintf("'Балка'!$B$%d", beamFirstSpan)
			}
			var cells []string
			source.WriteString(xrow(sourcePointRow, auditNum(1, sourcePointRow, float64(ci+1)),
				auditText(2, sourcePointRow, fmt.Sprintf("%d · %s", wi+1, p.Crane)),
				auditNum(3, sourcePointRow, p.XMM/1000), auditNum(4, sourcePointRow, p.PKN),
				auditNum(5, sourcePointRow, c.SectionMM/1000), auditText(6, sourcePointRow, kind)))
			cells = append(cells, auditNum(1, row, float64(ci+1)), auditText(2, row, fmt.Sprintf("%d · %s", wi+1, p.Crane)),
				auditFormula(3, row, fmt.Sprintf("'Исходные данные'!C%d", sourcePointRow), p.XMM/1000),
				auditFormula(4, row, fmt.Sprintf("'Исходные данные'!D%d", sourcePointRow), p.PKN))
			// E=η and F=Pη. Explicit formulas, not imported numeric ordinates.
			etaFormula := fmt.Sprintf("IF(%s=\"M\",%s*(1-%s/%s)+%s*%s/%s+%s,IF(%s=\"QR\",%s,-%s))", rc(7), rc(16), rc(14), referenceLen, rc(17), rc(14), referenceLen, rc(15), rc(7), rc(18), rc(19))
			cells = append(cells, auditFormula(5, row, etaFormula, audit.Eta), auditFormula(6, row, fmt.Sprintf("D%d*E%d", row, row), p.PKN*audit.Eta), auditText(7, row, kind), auditFormula(8, row, fmt.Sprintf("'Исходные данные'!E%d", sourcePointRow), c.SectionMM/1000), auditNum(9, row, float64(target)),
				auditFormula(10, row, func() string {
					if n == 1 {
						return fmt.Sprintf("IF(OR(C%d<0,C%d>'Балка'!$D$%d),0,1)", row, row, beamFirstSpan)
					}
					return fmt.Sprintf("IF(OR(C%d<0,C%d>MAX('Балка'!$D$%d:$D$%d)),0,MATCH(MAX(0,C%d-0.000000001),'Балка'!$C$%d:$C$%d,1))", row, row, beamFirstSpan, beamLastSpan, row, beamFirstSpan, beamLastSpan)
				}(), float64(audit.Span)))
			// A single-span range contains only one cell. Use direct
			// references rather than INDEX(one-cell-range,1), which some
			// compatible spreadsheet engines mis-evaluate as #VALUE!.
			leftX := fmt.Sprintf("INDEX('Балка'!$C$%d:$C$%d,J%%d)", beamFirstSpan, beamLastSpan)
			leftL := fmt.Sprintf("INDEX('Балка'!$B$%d:$B$%d,J%%d)", beamFirstSpan, beamLastSpan)
			if n == 1 {
				leftX = fmt.Sprintf("'Балка'!$C$%d", beamFirstSpan)
				leftL = fmt.Sprintf("'Балка'!$B$%d", beamFirstSpan)
			}
			refForRow := func(pattern string) string {
				if n == 1 {
					return pattern
				}
				return fmt.Sprintf(pattern, row)
			}
			if audit.Span == 0 {
				cells = append(cells, auditFormula(11, row, fmt.Sprintf("IF(J%d=0,0,C%d-"+refForRow(leftX)+")", row, row), 0))
				cells = append(cells, auditFormula(12, row, fmt.Sprintf("IF(J%d=0,0,"+refForRow(leftL)+"-K%d)", row, row), 0))
				cells = append(cells, auditFormula(13, row, fmt.Sprintf("IF(J%d=0,0,"+refForRow(leftL)+")", row), 0))
			} else {
				cells = append(cells, auditFormula(11, row, fmt.Sprintf("IF(J%d=0,0,C%d-"+refForRow(leftX)+")", row, row), audit.A))
				cells = append(cells, auditFormula(12, row, fmt.Sprintf("IF(J%d=0,0,"+refForRow(leftL)+"-K%d)", row, row), audit.B))
				cells = append(cells, auditFormula(13, row, fmt.Sprintf("IF(J%d=0,0,"+refForRow(leftL)+")", row), audit.L))
			}
			zFormula := fmt.Sprintf("H%d-INDEX('Балка'!$C$%d:$C$%d,I%d)", row, beamFirstSpan, beamLastSpan, row)
			if n == 1 {
				zFormula = fmt.Sprintf("H%d-'Балка'!$C$%d", row, beamFirstSpan)
			}
			cells = append(cells, auditFormula(14, row, zFormula, audit.Z))
			m0Formula := fmt.Sprintf("IF(AND(J%d=I%d,G%d=\"M\"),IF(N%d<=K%d,L%d*N%d/M%d,K%d*(M%d-N%d)/M%d),0)", row, row, row, row, row, row, row, row, row, row, row, row)
			cells = append(cells, auditFormula(15, row, m0Formula, audit.M0))
			if m > 0 {
				mLeft := fmt.Sprintf("IF(I%d=1,0,INDEX(%s:%s,1,I%d-1))", row, rc(20+2*m), rc(19+3*m), row)
				mRight := fmt.Sprintf("IF(I%d=%d,0,INDEX(%s:%s,1,I%d))", row, n, rc(20+2*m), rc(19+3*m), row)
				if m == 1 {
					mLeft = fmt.Sprintf("IF(I%d=1,0,%s)", row, rc(19+2*m+1))
					mRight = fmt.Sprintf("IF(I%d=%d,0,%s)", row, n, rc(19+2*m+1))
				}
				cells = append(cells, auditFormula(16, row, mLeft, audit.ML), auditFormula(17, row, mRight, audit.MR))
			} else {
				cells = append(cells, auditFormula(16, row, "0", 0), auditFormula(17, row, "0", 0))
			}
			rLeft := fmt.Sprintf("(Q%d-P%d)/%s+IF(J%d=I%d,L%d/M%d,0)", row, row, referenceLen, row, row, row, row)
			rRight := fmt.Sprintf("(P%d-Q%d)/%s+IF(J%d=I%d,K%d/M%d,0)", row, row, referenceLen, row, row, row, row)
			cells = append(cells, auditFormula(18, row, rLeft, audit.RLeft), auditFormula(19, row, rRight, audit.RRight))
			for j := 1; j <= m; j++ {
				formula := fmt.Sprintf("IF(J%d=%d,-K%d*L%d*(M%d+K%d)/M%d,IF(J%d=%d,-K%d*L%d*(2*M%d-K%d)/M%d,0))", row, j, row, row, row, row, row, row, j+1, row, row, row, row, row)
				cells = append(cells, auditFormula(19+j, row, formula, audit.RHS[j-1]))
			}
			for j := 1; j <= m; j++ {
				beamRow := beamCoeffs + j - 1
				rhs := rc(19 + j)
				formula := fmt.Sprintf("%s/'Балка'!E%d", rhs, beamRow)
				if j > 1 {
					prevForward := rc(19 + m + j - 1)
					formula = fmt.Sprintf("(%s-'Балка'!D%d*%s)/('Балка'!E%d-'Балка'!D%d*'Балка'!G%d)", rhs, beamRow, prevForward, beamRow, beamRow, beamRow-1)
				}
				cells = append(cells, auditFormula(19+m+j, row, formula, audit.Forward[j-1]))
			}
			for j := 1; j <= m; j++ {
				beamRow := beamCoeffs + j - 1
				forward := rc(19 + m + j)
				formula := forward
				if j < m {
					nextM := rc(19 + 2*m + j + 1)
					formula = fmt.Sprintf("%s-'Балка'!G%d*%s", forward, beamRow, nextM)
				}
				cells = append(cells, auditFormula(19+2*m+j, row, formula, audit.Supports[j-1]))
			}
			detail.WriteString(xrow(row, cells...))
			row++
			sourcePointRow++
		}
		if math.Abs(subtotal-c.Value) > 1e-6*math.Max(1, math.Abs(c.Value)) {
			return cranePlacementSheetParts{}, fmt.Errorf("положение %d: сумма P·η %.6f не равна расчёту программы %.6f", ci+1, subtotal, c.Value)
		}
		sr := ci + 5
		summary.WriteString(xrow(sr, auditNum(1, sr, float64(ci+1)), auditText(2, sr, c.Name), auditFormula(3, sr, fmt.Sprintf("SUMIF('Колёса'!$A$6:$A$%d,A%d,'Колёса'!$F$6:$F$%d)", endDataRow, sr, endDataRow), subtotal), auditText(4, sr, c.Units), auditFormula(5, sr, fmt.Sprintf("'Исходные данные'!E%d*1000", firstSourceRow), c.SectionMM), auditFormula(6, sr, fmt.Sprintf("C%d", sr), subtotal)))
	}
	widths := []float64{12, 35, 17, 16, 17, 19, 16, 19, 18, 19, 14, 14, 18, 17, 16, 17, 17, 17, 17}
	for i := 0; i < 3*m; i++ {
		widths = append(widths, 16)
	}
	return cranePlacementSheetParts{
		Summary: summary.String(), Detail: detail.String(), Beam: beam.String(),
		Inputs: source.String(), DetailWidths: widths,
	}, nil
}

func exportCranePlacementAuditExcel(r CranePlacementResult, path string) error {
	sheets, err := cranePlacementAuditSheets(r)
	if err != nil {
		return err
	}
	return craneExcelPackage(path, sheets.Summary, sheets.Detail,
		sheets.Beam, sheets.Inputs, sheets.DetailWidths)
}

from pathlib import Path
p=Path('/mnt/data/ebala060_stage2/wind_reports.go')
s=p.read_text()
s=s.replace('''geom += fmt.Sprintf(", подъём двускатной крыши h = %.3f м", i.RoofRise)''','''geom += fmt.Sprintf(", подъём двускатной крыши h = %.3f м", i.RoofRise)''')
s=s.replace('''if i.Geometry == "free_wall" {
		geom += ". Для отдельно стоящей стены zₑ = h (п. В.1.1)."
	} else {
		geom += fmt.Sprintf("; по п. 11.1.5 zₑ = %.3f м.", r.Ze)
	}''','''if i.Geometry == "free_wall" {
        geom += ". Для отдельно стоящей стены zₑ = h (п. В.1.1)."
    } else if strings.HasPrefix(i.AeroTable, "В.3") {
        geom += fmt.Sprintf("; двускатная крыша, L = %.3f м, B = %.3f м, β = %.3f°, расчёт на уровне конька H, zₑ = %.3f м.", i.RoofSpan, i.RoofLength, i.RoofPitchDeg, r.Ze)
        if i.RoofBranch == "1" || i.RoofBranch == "2" {geom += " Знаковый вариант В.3а: " + i.RoofBranch + "."}
    } else {
        geom += fmt.Sprintf("; по п. 11.1.5 zₑ = %.3f м.", r.Ze)
    }''')
s=s.replace('''b.WriteString(snowWordParagraph(fmt.Sprintf("Для вертикальной расчётной поверхности (табл. 11.7) ρ = b = %.3f м, χ = H = %.3f м; по таблице 11.4 ζ = %.6f, по таблице 11.6 ν = %.6f.", i.Rho, i.Chi, r.Zeta, r.Nu)))''','''if strings.HasPrefix(i.AeroTable, "В.3") {
            b.WriteString(snowWordParagraph(fmt.Sprintf("Для прямоугольной проекции выбранного участка крыши (плоскость xoy, табл. 11.7): ρ = %.3f м, χ = %.3f м. По таблице 11.4 ζ = %.6f, по таблице 11.6 ν = %.6f.", i.Rho, i.Chi, r.Zeta, r.Nu)))
        } else {
            b.WriteString(snowWordParagraph(fmt.Sprintf("Для вертикальной расчётной поверхности (табл. 11.7) ρ = b = %.3f м, χ = H = %.3f м; по таблице 11.4 ζ = %.6f, по таблице 11.6 ν = %.6f.", i.Rho, i.Chi, r.Zeta, r.Nu)))
        }''')
s=s.replace('''b.WriteString(snowWordParagraph("Эпюра нагрузки по высоте сооружения. Для воспроизводимости приведены контрольные ординаты; полный массив точек хранится в расчётном блоке и выводится в независимом XLSX."))''','''profileNote := "Эпюра нагрузки по высоте сооружения. Для воспроизводимости приведены контрольные ординаты; полный массив точек хранится в расчётном блоке и выводится в независимом XLSX."
        if strings.HasPrefix(i.AeroTable, "В.3") {
            profileNote = "Ветровое давление на выбранный участок ската определено на высоте конька H. Значения на отметках карниза и конька показаны как геометрические границы зоны, а не как независимая расчётная эпюра по стене."
        }
        b.WriteString(snowWordParagraph(profileNote))''')
insert='''
// Export the exact normative source data for roof tables, including sign
// alternatives. Every cell comes from the checked JSON, not Go constants.
func windExcelRoofTables(entries map[string]string, db *windDB) error {
    for _, spec := range []struct {
        name, source string
        branch int
        angles []float64
        zones []string
    }{
        {"Ветер В.3а 1", "В.3а", 1, []float64{0, 15, 30}, []string{"F", "G", "H", "I", "J"}},
        {"Ветер В.3а 2", "В.3а", 2, []float64{15, 30, 45, 60, 75}, []string{"F", "G", "H", "I", "J"}},
        {"Ветер В.3б", "В.3б", 1, []float64{0, 15, 30, 45, 60, 75}, []string{"F", "G", "H", "I"}},
    } {
        var out strings.Builder
        out.WriteString(tempExcelRow(1, "На этом листе приведены коэффициенты СП 20.13330.2016, таблица "+spec.source+", знаковая ветвь "+strconv.Itoa(spec.branch)))
        out.WriteString(tempExcelRow(2,"Источник",spec.source))
        out.WriteString(tempExcelRow(3,append([]string{"β, град"},spec.zones...)...))
        for idx, beta := range spec.angles {
            row:=[]string{xnum(beta)}
            for _, zone:=range spec.zones {
                c,e:=db.roofCoefficient(spec.source,zone,beta,spec.branch)
                if e!=nil{return fmt.Errorf("Таблица Excel %s: %w",spec.name,e)}
                row=append(row,xnum(c))
            }
            out.WriteString(tempExcelRow(idx+4,row...))
        }
        if _,e:=snowExcelSheet(entries,spec.name,out.String(),false);e!=nil{return e}
    }
    return nil
}

// Linear interpolation INSIDE one normative table/alternative. The angle and
// both c endpoints remain visible and editable in the normative XLSX sheet.
func windExcelRoofInterp(sheet string, last int, beta, zone string) string {
    axis:="'"+sheet+"'!$A$4:$A$"+strconv.Itoa(last)
    headers:="'"+sheet+"'!$B$3:$F$3"
    vals:="'"+sheet+"'!$B$4:$F$"+strconv.Itoa(last)
    if sheet=="Ветер В.3б" {headers="'"+sheet+"'!$B$3:$E$3"; vals="'"+sheet+"'!$B$4:$E$"+strconv.Itoa(last)}
    col:="MATCH("+zone+","+headers+",0)"
    i:="IF("+beta+"<=INDEX("+axis+",1),1,IF("+beta+">=INDEX("+axis+",ROWS("+axis+")),ROWS("+axis+")-1,MATCH("+beta+","+axis+",1)))"
    low:="INDEX("+axis+","+i+")"
    high:="INDEX("+axis+",("+i+")+1)"
    loVal:="INDEX("+vals+","+i+","+col+")"
    hiVal:="INDEX("+vals+",("+i+")+1,"+col+")"
    return "("+loVal+"+("+beta+"-"+low+")/("+high+"-"+low+")*("+hiVal+"-"+loVal+"))"
}

// Derive ρ/χ from editable overall geometry and the normative V.4 geometry.
// These are Excel calculations: do NOT forward calculated Go zone dimensions.
func windExcelRoofCorrelation(in WindInput, ref func(int) string) (string,string,string) {
    B,L,H:=ref(25),ref(24),ref(8)
    e:="MIN("+L+",2*"+H+")"
    if in.RoofFlow=="CROSS" {e="MIN("+B+",2*"+H+")"}
    var rho,chi string
    if in.RoofFlow=="CROSS" {
        a:="MIN(0.1*B45,"+L+"/2)"
        chi=a
        if in.AeroZone=="H" || in.AeroZone=="I" {chi="MAX(0,"+L+"/2-"+a+")"}
        switch in.AeroZone {
        case "F":rho="MIN(B45/4,"+B+"/2)"
        case "G":rho="MAX(0,"+B+"-2*MIN(B45/4,"+B+"/2))"
        default:rho=B
        }
    } else {
        a:="MIN(B45/10,"+B+")"
        b:="MIN(B45/2,"+B+")"
        switch in.AeroZone {
        case "F","G":chi=a
        case "H":chi="MAX(0,"+b+"-"+a+")"
        default:chi="MAX(0,"+B+"-"+b+")"
        }
        switch in.AeroZone {
        case "F":rho="MIN(B45/4,"+L+"/2)"
        case "G":rho="MAX(0,"+L+"/2-MIN(B45/4,"+L+"/2))"
        default:rho=L+"/2"
        }
    }
    return "="+e,"="+rho,"="+chi
}
'''
pos=s.index('func windExcelSingle(')
s=s[:pos]+insert+s[pos:]
# Add normative excel tables in windExcelTables end before return: exact location
at='''	return nil
}

// Export the exact normative source data'''
# function immediately prior to our inserted block is windExcelTables
s=s.replace(at,'''	return windExcelRoofTables(entries, db)
}

// Export the exact normative source data''',1)
# rows 24..26
s=s.replace('''{"Шаг эпюры, м (0 = исторический режим 0.59)", xnum(i.ProfileStep)},''','''{"Шаг эпюры, м (0 = исторический режим 0.59)", xnum(i.ProfileStep)},
        {"Общая ширина здания L, м", xnum(i.RoofSpan)},
        {"Общая длина здания B, м", xnum(i.RoofLength)},
        {"Знаковый вариант покрытия (1 или 2)", i.RoofBranch},''')
# Replace coefficient formula. Note sheet names have 10 char no issues
old='''add(16, "Аэродинамический коэффициент c", fmt.Sprintf("=IF(%s=\\"manual\\",%s,IF(%s=\\"В.1\\",VLOOKUP(%s,'Ветер В.1'!$A$3:$B$6,2,FALSE),VLOOKUP(%s,'Ветер В.2'!$A$3:$B$7,2,FALSE)))", A(11), A(13), A(11), A(12), A(12)))'''
new='''roof1:=windExcelRoofInterp("Ветер В.3а 1",6,"B44",A(12))
    roof2:=windExcelRoofInterp("Ветер В.3а 2",8,"B44",A(12))
    roofB:=windExcelRoofInterp("Ветер В.3б",9,"B44",A(12))
    t15:="'Ветер В.3а 1'!$A$5"
    t30:="'Ветер В.3а 1'!$A$6"
    roofA:=fmt.Sprintf("IF(B44<%s,%s,IF(B44<=%s,IF(%s=\\"2\\",%s,%s),%s))",t15,roof1,t30,A(26),roof2,roof1,roof2)
    add(16, "Аэродинамический коэффициент c", fmt.Sprintf("=IF(%s=\\"manual\\",%s,IF(%s=\\"В.1\\",VLOOKUP(%s,'Ветер В.1'!$A$3:$B$6,2,FALSE),IF(%s=\\"В.2\\",VLOOKUP(%s,'Ветер В.2'!$A$3:$B$7,2,FALSE),IF(%s=\\"В.3а\\",%s,%s))))", A(11),A(13),A(11),A(12),A(11),A(12),A(11),roofA,roofB))'''
assert old in s,'old coefficient not found'; s=s.replace(old,new)
# set refs for correlation
s=s.replace('''xAxis := "'Ветер 11.6'!$A$4:$A$10"''','''rhoRef, chiRef := A(16), A(17)
    if i.AeroTable == "В.3а" || i.AeroTable == "В.3б" {rhoRef, chiRef = "B46", "B47"}
    xAxis := "'Ветер 11.6'!$A$4:$A$10"''')
s=s.replace('A(16), xAxis','rhoRef, xAxis').replace('A(17), yAxis','chiRef, yAxis')
s=s.replace('''A(16), A(17), A(16)))''','''rhoRef, chiRef, rhoRef))''')
# spot check all after
end='''	add(43, "Предельная частота f_lim, Гц", fmt.Sprintf("=IF(B42=\\"\\",\\"\\",%s)", strings.TrimPrefix(fl, "=")))'''
newend=end+'''
    add(44,"Уклон β двускатной крыши, град", fmt.Sprintf("=IF(%s>0,DEGREES(ATAN(2*%s/%s)),0)",A(24),A(22),A(24)))
    eExpr,rhoExpr,chiExpr:=windExcelRoofCorrelation(i,A)
    if i.AeroTable=="В.3а" || i.AeroTable=="В.3б" {
        add(45,"Параметр e=min(b,2H) для рисунка В.4, м",eExpr)
        add(46,"ρ зоны крыши, м (расчёт по геометрии)",rhoExpr)
        add(47,"χ зоны крыши, м (расчёт по геометрии)",chiExpr)
    } else {
        add(45,"Параметр e — только для крыши", "")
        add(46,"ρ зоны крыши — только для крыши", "")
        add(47,"χ зоны крыши — только для крыши", "")
    }'''
assert end in s; s=s.replace(end,newend)
# roof profile height fix
s=s.replace('''H, d, step := ref(8), ref(9), ref(23)''','''H, d, step := ref(8), ref(9), ref(23)
    if in.AeroTable=="В.3а" || in.AeroTable=="В.3б" {
        if z==in.Height {return "="+H}
        return "=MAX(0,"+H+"-"+ref(22)+")"
    }''')
s=s.replace('''ze := windExcelZeExpr(srcRef(7), srcRef(8), srcRef(9), a)''','''ze := windExcelZeExpr(srcRef(7), srcRef(8), srcRef(9), a)
        if c.Input.AeroTable=="В.3а" || c.Input.AeroTable=="В.3б" {ze = "="+calcRef(4)}''')
p.write_text(s)

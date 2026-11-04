"""Static source invariants and independent threshold arithmetic for stage 8.
Does NOT compile Go, create an XLSX, execute Excel formulas, or launch Windows.
"""
from pathlib import Path
import math
import hashlib
import sys

base = Path(__file__).resolve().parents[2]
source = {n: (base/n).read_text(encoding='utf-8') for n in (
    'snow_excel_variant_plan_phase8.go','snow_excel_v031.go','snow_report.go','snow_excel_sphere_050.go')}
checks = []
def check(label, assertion):
    checks.append((label, bool(assertion)))
plan = source['snow_excel_variant_plan_phase8.go']
v031 = source['snow_excel_v031.go']
rep = source['snow_report.go']
order = source['snow_excel_sphere_050.go']
check('A1 on the variant catalogue', 'На этом листе перечислены предусмотренные варианты' in plan)
check('Original Go cases preserved, reserve appended', 'for _, c := range s.Cases' in plan and 'Reserved: true' in plan)
check('Gable uniform and uneven option catalogue', 'reserve("2. Неравномерная"' in plan and 'reserve("3. Неравномерная (зеркальная)"' in plan)
check('Gable optional single-slope load', 'reserve("Частичное загружение левого ската"' in plan)
check('Gable ridge accumulation', 'reserve("Вариант 3 Б.1. Накопление у конька"' in plan)
check('Vault partial alternatives', 'reserve("Частичное загружение половины свода"' in plan)
check('Skylight-wide alternatives', 'reserve("Дополнительное местное загружение у левого ската фонаря"' in plan)
check('Parapet extra local variant', 'reserve("2. Местная нагрузка у парапета"' in plan)
check('Paired roof variants', 'addPart("Левый пролёт: "' in plan and 'addPart("Правый пролёт: "' in plan)
check('Step B.8 reserved from equal elevations', 'reserve("Занос у перепада высот:' in plan)
check('B.8 step applicability reads calculated table', "'Перепад Б.8'!$B$25" in plan)
check('Non-applicable mu stays empty', 'f = `=IF($B$3="Да",`' in v031)
check('Non-applicable normative load stays empty', '=IF(C%d="","",C%d*' in v031)
check('Non-applicable design load stays empty', '=IF(D%d="","",D%d*' in v031)
check('Non-applicable reduced load stays empty', 'AND($B$3="Да"' in v031)
check('Go mu not transferred through catalogue', 'v.Case.Intervals' not in plan and 'v.Case.Polar' not in plan)
check('Real formula checked before IF wrapper', 'muFormula, _ := snowExcelMuFormula(' in v031 and 'strings.Contains(muFormula, "NA()")' in v031)
check('Ridge equipment in source worksheet', 'add(35, "Оборудование у конька' in v031)
check('Catalogue generated with suffix for multiple blocks', '"Перечень загружений"+suffix' in v031)
check('Charts use complete variant plan', 'for i := range plan {' in v031)
check('Main report references catalogue', 'catalog := "\'Перечень загружений" + suffix + "\'!"' in rep)
check('Main report does not use Go interval count', 'for j := range c.Intervals' not in rep[rep.index('func snowExcelAppend('):rep.index('func ExportSnowWord(')])
check('Catalogue is placed before main load summary', 'name="Перечень загружений' in order)
check('No release executable produced in source folder', not any(base.glob('*.exe')))

# Independently verify the closed-form arch support tangent over both shallow
# and deep caps against the original circular geometry formula.
for L, f in ((18,0.5),(18,2),(18,9),(18,12),(24,1),(24,25)):
    R=L*L/(8*f)+f/2
    expected=math.degrees(math.atan2(L/2, math.sqrt(max(0,R*R-(L/2)**2))))
    actual=math.degrees(2*math.atan(min(2*f/L,L/(2*f))))
    check(f'Vault edge tangent L={L}, f={f}', abs(expected-actual)<1e-8)

# Independent applicability thresholds (not a substitute for Excel runtime).
def gable_nonuniform(a): return 15 <= a <= 40
def gable_partial(a): return abs(a)>20
for angle, a, b in ((10, False, False),(15,True,False),(20,True,False),(21,True,True),(40,True,True),(41,False,True)):
    check(f'Gable angle conditions: {angle}', gable_nonuniform(angle)==a and gable_partial(angle)==b)

ok=sum(v for _,v in checks)
report=[f'# Статическая проверка этапа 8', '',f'Пройдено: {ok}/{len(checks)}.', '', 'Go-тесты, генерация XLSX, фактический перерасчёт Microsoft Excel и Windows GUI не выполнялись.', '', '| Проверка | Результат |','|---|---|']
report += [f'| {name} | {"ОК" if value else "ОШИБКА"} |' for name,value in checks]
(base/'EXCEL_PHASE8_STATIC_CHECK.md').write_text('\n'.join(report)+'\n',encoding='utf-8')
print(f'Phase8 static conditions: {ok}/{len(checks)}')
if ok != len(checks):
    for name,value in checks:
        if not value: print('FAIL:',name)
    sys.exit(1)

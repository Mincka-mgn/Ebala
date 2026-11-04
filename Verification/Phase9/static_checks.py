"""Stage 9 source-only checks. Does not compile Go or execute Excel."""
from pathlib import Path
import hashlib
import math
import sys

root = Path(__file__).resolve().parents[2]
# SHA256 of the approved PROJECT.md in the stage-8 source checkpoint.
project_sha256 = 'fb36a9876d09230b38a90f51841ab93e5d1c7a274db87c57013054ffc78a7676'
files = {name:(root/name).read_text(encoding='utf-8') for name in (
 'snow_excel_variant_plan_phase8.go','snow_excel_grid_phase6.go',
 'snow_excel_v031.go','snow_excel_sector_phase9.go','snow_excel_sphere_050.go',
 'excel_phase9_dynamic_test.go')}
p=files['snow_excel_variant_plan_phase8.go']; g=files['snow_excel_grid_phase6.go']
v=files['snow_excel_v031.go']; s=files['snow_excel_sector_phase9.go']
tests=files['excel_phase9_dynamic_test.go']; order=files['snow_excel_sphere_050.go']
checks=[]
def check(n,ok): checks.append((n,bool(ok)))
check('Previous source checkpoint preserved in this directory', (root/'EXCEL_PHASE8_CHANGES.md').exists())
check('PROJECT.md is identical to stage 8', hashlib.sha256((root/'PROJECT.md').read_bytes()).hexdigest()==project_sha256)
check('Explicit 12 bay source limit', 'snowExcelGridMultiSamples = 12 * 32' in g)
check('Reserve individual left and right bay-side alternatives', 'bay <= 12' in p and 'side <= 2' in p)
check('Reserve both two-adjacent-slope arrangements', 'scheme <= 2' in p)
check('Reserve all four alternating one-slope arrangements', 'scheme <= 4' in p)
check('Reserve 11 adjacent multi-vault alternatives', 'bay < 12' in p)
check('Recalculate active bay condition from Excel B17', "'Исходные данные'!$B$17" in p)
check('Live count >= 2 and <= 12', '%s>=2,%s<=12,%s=INT(%s)' in p)
check('Individual bay activation conditioned on edited count', 'AND(ABS(%s)>20,%s>=%d)' in p)
check('Per-bay live coordinate formulas', 'index / 32' in g and 'index % 32' in g and 'IF(AND(%s$B$17>=%d' in g)
check('Empty inactive-bay coordinates', 'return coord(sub), coord(sub + 1)' in g and 'B$11,"")' in g)
check('Prevent mu formula evaluating on empty grid rows', 'A%d<>"",B%d<>""' in v)
check('Reserve six spatial round sectors without Go Polar dependency', 'for _, sector := range [][2]int{{0, 180}' in p)
check('One shared spatial sheet rather than six duplicated grids', 'snowExcelSheet(entries, "Секторная сетка"+suffix' in v)
check('Shared sector spatial grid has independent inputs', "src := \"'Исходные данные'!\"" in s and "cf := \"'Коэффициенты'!\"" in s)
check('Shared sector grid contains all six independent mu columns', 'for _, p := range sectors' in s and 'μ %d–%d°' in s)
check('Shared sector grid has independent normative load columns', 'Sн %d–%d°' in s)
check('Shared sector grid has independent design load columns', 'Sр %d–%d°' in s)
check('No source Go Polar or Go mu copied into shared sector grid', '.Polar' not in s and '.Mu' not in s)
check('Angular grid one degree 0..360', 'az := 0; az <= 360; az++' in s)
check('Radial grid 12 intervals', 'ir := 0; ir <= 12; ir++' in s)
check('Sector active condition reevaluated from geometry', 'eligibility := ' in s and 'snowIsPartialSector' in s)
check('Sector sheet references renamed for combined output', 'rename(snowExcelSectorMatrix(s))' in v)
check('Original sector Polar sheets no longer duplicated', 'len(c.Polar) > 0 && !snowIsPartialSector(c)' in v)
check('Sheet order recognizes sector sheet', 'name="Секторная сетка' in order)
check('Sector grid columns A to V are readable', 'max=\"22\" width=\"23\"' in v and 'ref=\"A1:V1\"' in v)
check('New unit tests prepared but not executed', 'TestExcelPhase9CompleteMultiGableTopology' in tests and 'TestExcelPhase9MultiVaultAllAdjacentPairs' in tests and 'TestExcelPhase9RoundPolarSectors' in tests)

# Independent geometric logic checks. These do not evaluate the Go or Excel formulas.
L = 18.
for n in (2,3,8,12):
    points=[]
    for index in range(12*32):
        bay, j=divmod(index,32)
        if bay<n:
            points.append(((bay+j/32)*L,(bay+(j+1)/32)*L))
    check(f'Independent bay geometry n={n}: continuous and covers nL', len(points)==n*32 and abs(points[-1][1]-n*L)<1e-10 and all(abs(points[i][1]-points[i+1][0])<1e-10 for i in range(len(points)-1)))
sectors=((0,180),(180,360),(0,90),(90,180),(180,270),(270,360))
for az in (0,45,89,90,135,179,180,225,269,270,315,359,360):
    k=az%360
    active=[i for i,(a,b) in enumerate(sectors) if a<=k<b]
    check(f'Independent angular sectors at beta={az}°: one half + one quarter',len(active)==2 and sum(i<2 for i in active)==1 and sum(i>=2 for i in active)==1)

passed=sum(v for _,v in checks)
report=['# Ebala — этап 9: статическая проверка','',f'Пройдено: **{passed}/{len(checks)}**.','',
'Проверяются исходники, состав резерва и независимые арифметические примеры. Это НЕ исполнение Go, НЕ генерация XLSX и НЕ проверка Excel.', '', '| Проверка | Результат |','|---|---|']
report += [f'| {name} | {"ОК" if result else "НЕ ПРОШЛО"} |' for name,result in checks]
(root/'EXCEL_PHASE9_STATIC_CHECK.md').write_text('\n'.join(report)+'\n',encoding='utf-8')
print(f'Phase 9 static checks: {passed}/{len(checks)}')
for name,ok in checks:
    if not ok: print('FAILED:',name)
if passed!=len(checks): sys.exit(1)

"""Phase 6: source-only validation; does NOT execute Go or recalculate XLSX."""
from pathlib import Path
import re
root=Path(__file__).resolve().parents[2]
read=lambda f:(root/f).read_text(encoding='utf-8')
v=read('snow_excel_v031.go')
g=read('snow_excel_grid_phase6.go')
c=read('snow_excel_chart_052.go')
r=read('snow_excel_round_phase2.go')
s=read('snow_excel_sphere_050.go')
checks={
 'fixed sample capacity independent of c.Intervals': 'samples := snowExcelSampleCount(s)' in v and 'for j := 0; j < samples; j++' in v,
 'sample endpoints use Excel formulas': 'snowExcelSampleInterval(s, j, samples)' in v,
 'legacy interval rescaling not used for primary case sheet': 'snowExcelIntervalX(s, c, v.X1)' not in v,
 'dynamic sample code does not read Go interval coordinates': 'c.Intervals' not in g,
 'multi-bay grid uses editable span and number of bays': '$B$11*' in g and '$B$17' in g,
 'step grid uses both editable roof widths': '$B$18' in g and '$B$19' in g,
 'dome uses independently calculated projection': "cf + \"$B$47\"" in g,
 'paired roof grid uses two editable widths': '$B$11+' in g and '$B$41' in g,
 'live chart gets independent sample count': 'snowExcelSampleCount(s)); err != nil' in v,
 'live chart uses midpoint x, not old left endpoints': "!$H$5:$H$" in c,
 'midpoint has own Excel formula': 'fmt.Sprintf("=(A%d+B%d)/2"' in v,
 'reduced snow is conditional on current January temperature': 'src+"$B$23' not in v and "'Исходные данные'!$B$23" in v and 'B$8<=-5' in v,
 'report note does not copy Go-computed μ': 'add(3, "Описание", c.Note)' not in v[v.index('func snowExcelVariantData('):v.index('func snowExcelGridStyle(')],
 'angle input has an editable mode selector': 'Способ задания уклона' in v and 'if s.SlopeMode == "height"' in v,
 'rise-derived angle calculated in Excel': 'DEGREES(ATAN(' in v and '$B$37' in v,
 'cone polar angle refers to computed slope': 'ang = "=IF(" + beta + "=\\"\\",\\"\\"," + cf + "$B$4)"' in r,
 'cone polar coefficient uses computed slope': 'cf, cf, cf, cf, cf, z, src' in r,
 'sphere-cone geometry uses coefficient angle and block suffix': 'coef := "Коэффициенты" + strings.TrimPrefix(input, "Исходные данные")' in s and '"=\'"+coef+"\'!B4"' in s,
 'prepared unit tests for variable interval count': 'TestPhase6SnowVariantGridUsesEditableDimensions' in read('excel_phase6_contract_test.go'),
}
for label, ok in checks.items():
    print(('PASS' if ok else 'FAIL')+': '+label)
if not all(checks.values()):
    raise SystemExit(1)
print(f'PASS: {len(checks)} static checks. No Go compilation and no XLSX execution.')

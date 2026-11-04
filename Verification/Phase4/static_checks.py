"""Non-compiling source-level checks for phase 4, not a substitute for Go/Excel tests."""
from pathlib import Path
import re
root=Path(__file__).resolve().parents[2]
get=lambda name:(root/name).read_text(encoding='utf8')
checks={
    'single numerical source for standalone+common crane placement': 'cranePlacementAuditSheets(r CranePlacementResult)' in get('crane_placement_audit_excel.go'),
    'common export includes placement calculation sheets': 'excelPhase4CranePlacementSheets(entries, r)' in get('temporary_excel.go'),
    'four sheets generated for each placement': all((f'{{"{part}", sheets.' in get('excel_phase4_crane_placements.go')) for part in ('Вход','Балка','Колёса','Итоги')),
    'all standalone cross-sheet references renamed': all((f'"\'{part}\'!"' in get('excel_phase4_crane_placements.go')) for part in ('Исходные данные','Балка','Колёса')),
    'main ledger reads placement effort from formula sheet': 'excelPhase4PlacementSheetName(placementIndex, "Итоги"), caseRow' in get('temporary_excel.go'),
    'no Go numerical effort in common placement': 'tempRU(c.Value' not in get('temporary_excel.go'),
    'numeric source positions remain permitted': 'excelPhase4PlacementSheetName(placementIndex, "Вход"), pointRow' in get('temporary_excel.go'),
    'formula cells have no Go numeric cache': '<v></v>' in get('crane_placement_audit_excel.go'),
    'legacy K=0 is not overwritten when explicitly provided': 'if _, supplied := sel.Params["K"]; !supplied {' in get('excel_051_ledger.go'),
    'AB zero input is not overwritten with catalogue default': 'if kind == "AB" && !frontKnown && !rearKnown {' in get('excel_051_ledger.go'),
    'two saved placements have disjoint worksheet names': all((f'"Р0{i} Вход"' in get('excel_phase4_crane_contract_test.go') or f'"Р0{i} Вход"' in get('excel_phase4_crane_contract_test.go')) for i in (1,2)),
    'styles stripped before common XLSX assignment': 'phase4CellStyleRX.ReplaceAllString(body, "$1")' in get('excel_phase4_crane_placements.go'),
    'old stale-PNG expectation removed for Excel': 'ext == "docx" && countPNG < 2' in get('v043_regression_test.go'),
    'linear sphere/cone limit is a calculated mode': 'add(22, "Линейное сопряжение"' in get('snow_excel_sphere_050.go'),
    'linear sphere/cone does not inject radius 1/0': '1E-9,1/0' not in get('snow_excel_sphere_050.go'),
    'linear sphere/cone uses line inclination and height': get('snow_excel_sphere_050.go').count('$B$22=\\"Да\\"') >= 2,
}
for name,good in checks.items():
    print(('OK ' if good else 'FAIL ')+ name)
if not all(checks.values()):raise SystemExit(1)
print(f'PASS: {len(checks)} source-level checks (without Go compilation)')

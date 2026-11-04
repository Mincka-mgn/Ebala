"""Source-only checks for phase 5. Does not invoke Go build or generate a workbook.

Passing does NOT mean Excel formulas were evaluated by Microsoft Excel.
"""
from pathlib import Path
import ast
import re

root=Path(__file__).resolve().parents[1]
read=lambda name:(root/name).read_text(encoding='utf-8')
ui=read('main_windows.go')
norm=read('ice_excel_norm_051.go')
grid=read('ice_excel_grid_phase2.go')
ice=read('ice_excel.go')
snow=read('snow_excel_v031.go')

checks={
 'GUI no longer exposes legacy demonstration ExportExcel': 'return ExportExcel(project, file)' not in ui and 'return exportTempExcel(project.selectedReport(), file)' in ui,
 'missing research thickness is blank, not NA': 'NA()' not in norm and 'ISNUMBER(%sB%d)' in norm,
 'table 12.2 lookup checks observed numeric values': 'studyNode := func(index string)' in norm and 'ISNUMBER(INDEX($D$4:$D$6' in norm,
 'ice source failures do not deliberately inject DIV/0': '1/0' not in grid,
 'ice height grid still uses dynamic Excel SMALL': 'SMALL($A$4:$B$305' in grid,
 'ice input values feed tabular source': "'Гололёд табл.12.2'!$G$" in ice,
 'missing ice coefficients leave result blank': 'B%d=\\"\\",C%d=\\"\\",D%d=\\"\\",E%d=\\"\\",F%d=\\"\\"' in ice,
 'new ice formula observes manually accepted zero ice drag': '*%s$B$46*%s)' in ice and 'IF(%s$B$46>0,' not in ice,
 'wind norm missing lookup leaves blank': 'IFERROR(INDEX($B$4:$B$11,MATCH' in norm,
 'common ice summary points to source input': '"=\'Гололёд - исходные\'!B10"' in ice,
 'missing independent snow mu stops export': 'невозможно сформировать проверочный Excel' in snow and 'strings.Contains(variant, "<f>NA()</f>")' in snow,
}

# Also check only the modified Excel formula templates in ice_excel.go.
formulas=['kf','bf','thick','norm','design','dIced','mean','leading']
for name in formulas:
 lines=[l for l in ice.splitlines() if re.search(r'\b'+name+r' := fmt\.Sprintf',l)]
 assert len(lines)==1, f'missing formula for {name}'
 line=lines[0]
 m=re.search(r'fmt\.Sprintf\("((?:\\.|[^"\\])*)", (.*)\)$',line)
 assert m, f'unparseable Go formatting call for {name}'
 pattern=ast.literal_eval('"'+m.group(1)+'"')
 count=len(re.findall(r'%(?:s|d|f|v)',pattern))
 provided=len(m.group(2).split(', '))
 checks[f'{name}: matching number of Go format arguments']=(count==provided)
 # Check parenthesis shape, replacing every Go substitution with one cell ref.
 formula=re.sub(r'%(?:s|d|f|v)','X5',pattern)
 active=False;depth=0;balanced=True
 for ch in formula:
  if ch=='"':active=not active
  if not active:
   if ch=='(':depth+=1
   if ch==')':
    depth-=1
    if depth<0:balanced=False
 checks[f'{name}: Excel formula brackets balanced']=balanced and depth==0 and not active

for title,success in checks.items():
 print(('PASS' if success else 'FAIL')+': '+title)
if not all(checks.values()):raise SystemExit(1)
print(f'PASS: {len(checks)} non-compiling checks. Actual XLSX/Go/Office execution NOT verified.')

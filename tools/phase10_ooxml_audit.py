import zipfile, re, xml.etree.ElementTree as ET, pathlib, json, sys
from collections import Counter
N={'m':'http://schemas.openxmlformats.org/spreadsheetml/2006/main', 'r':'http://schemas.openxmlformats.org/officeDocument/2006/relationships'}
results=[]
for path in sorted(pathlib.Path(sys.argv[1] if len(sys.argv) > 1 else '.').glob('*.xlsx')):
 issues=[]; refs=[]; formulas=0; cached=0; missing_a1=[]; missing_sheets=[]; chart_refs=[]
 with zipfile.ZipFile(path) as z:
  for item in z.namelist():
   if item.endswith('.xml'):
    try:ET.fromstring(z.read(item))
    except Exception as e:issues.append(f'invalid XML {item}: {e}')
  wb=ET.fromstring(z.read('xl/workbook.xml'))
  all_sheet_names={x.attrib.get('name','') for x in wb.findall('m:sheets/m:sheet',N)}
  rel=ET.fromstring(z.read('xl/_rels/workbook.xml.rels'))
  rels={x.attrib['Id']:x.attrib['Target'] for x in rel}
  first_sheet=wb.find('m:sheets/m:sheet',N).attrib.get('name')
  for sh in wb.findall('m:sheets/m:sheet',N):
   name=sh.attrib['name'];rid=sh.attrib['{'+N['r']+'}id'];relpath=rels[rid];f=('xl/'+relpath.lstrip('/')) if not relpath.startswith('/') else relpath.lstrip('/')
   f=f.replace('xl/xl/','xl/')
   if f not in z.namelist():issues.append(f'missing sheet file {name}: {f}');continue
   root=ET.fromstring(z.read(f));c=root.find('.//m:sheetData/m:row/m:c',N)
   txt=''.join(c.itertext()) if c is not None else ''
   if c is None or c.attrib.get('r')!='A1' or not txt.startswith('На этом листе'):missing_a1.append((name,txt[:75]))
   for c in root.findall('.//m:c',N):
    ff=c.find('m:f',N)
    if ff is None:continue
    formulas+=1
    val=c.find('m:v',N)
    if val is not None and val.text not in ['',None]:cached+=1
    text=ff.text or ''
    if 'NA()' in text or '#REF!' in text:issues.append(f'{name}:{c.attrib.get("r")} formula error token')
    for target in re.findall(r"'((?:[^']|'')+)'!",text):
     target=target.replace("''", "'")
     if target not in all_sheet_names:missing_sheets.append((name,c.attrib.get('r'),target))
  for key in z.namelist():
   if key.startswith('xl/charts/') and key.endswith('.xml'):
    chart=ET.fromstring(z.read(key))
    for f in chart.iter():
     if f.tag.endswith('}f') and f.text:
      chart_refs.append(f.text)
      for target in re.findall(r"'((?:[^']|'')+)'!",f.text):
       if target.replace("''", "'") not in all_sheet_names:missing_sheets.append((key,'chart',target))
  calc=wb.find('m:calcPr',N)
  if first_sheet!='Исходные данные':issues.append('first sheet is not input: '+first_sheet)
  if missing_a1:issues.append(f'{len(missing_a1)} missing A1 descriptions')
  if missing_sheets:issues.append(f'{len(missing_sheets)} unresolved quoted sheet references')
  results.append(dict(file=path.name, sheets=len(all_sheet_names),formulas=formulas,cached_nonempty=cached,chart_refs=len(chart_refs),calc=calc.attrib if calc is not None else None,issues=issues,missing_a1=missing_a1[:10],missing_ref=missing_sheets[:15]))
print(json.dumps(results,ensure_ascii=False,indent=2))


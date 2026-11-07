"""Read-only OOXML validation. Does not compute or replace Excel formulas."""
from pathlib import Path
from zipfile import ZipFile
import xml.etree.ElementTree as ET
from collections import Counter
import sys
fdoc=Path(sys.argv[1]); fxlsx=Path(sys.argv[2])
for p in (fdoc,fxlsx):
    with ZipFile(p) as z:
        bad=z.testzip()
        if bad:raise RuntimeError(f'bad ZIP member {bad}')
        for name in z.namelist():
            if name.endswith(('.xml','.rels')):ET.fromstring(z.read(name))
        print(p.name,'XML parts OK',len(z.namelist()))
with ZipFile(fxlsx) as z:
    ns={'x':'http://schemas.openxmlformats.org/spreadsheetml/2006/main'}
    book=ET.fromstring(z.read('xl/workbook.xml'))
    sheets=[s.attrib['name'] for s in book.findall('.//x:sheet',ns)]
    formulas=0; epures=[]; bad=[]
    for name in z.namelist():
        if not(name.startswith('xl/worksheets/') and name.endswith('.xml')):continue
        root=ET.fromstring(z.read(name))
        fs=root.findall('.//x:f',ns);formulas+=len(fs)
        for f in fs:
            if '#REF!' in (f.text or ''):bad.append(name)
        a2=root.find('.//x:c[@r="A2"]',ns)
        if a2 is not None and 'z' in ET.tostring(a2,encoding='unicode'):
            vals=[(c.attrib.get('r'),(c.find('x:f',ns).text if c.find('x:f',ns)is not None else None)) for c in root.findall('.//x:c',ns) if c.attrib.get('r','').startswith('A')]
            if len(vals)>7:epures.append((name,vals[:9]))
    print('Sheets',len(sheets),'formulas',formulas,'epure candidates',len(epures))
    print('Example epure coordinates:',epures[-1] if epures else 'none')
    if bad:raise RuntimeError('Workbook has #REF!: '+','.join(bad))
    assert len(sheets)>50 and formulas>2000
with ZipFile(fdoc) as z:
    xml=z.read('word/document.xml').decode()
    assert 'Крыша' in xml and 'СП 20.13330.2016' in xml
print('PASS')

#!/usr/bin/env python3
"""Independent OOXML integrity check; never evaluates engineering formulas.
Only standard Python library is used; the workbook is not modified.
"""
import json
import re
import sys
from pathlib import Path, PurePosixPath
from zipfile import ZipFile
from xml.etree import ElementTree as ET

X = '{http://schemas.openxmlformats.org/spreadsheetml/2006/main}'
R = '{http://schemas.openxmlformats.org/officeDocument/2006/relationships}'
P = '{http://schemas.openxmlformats.org/package/2006/relationships}'
C = '{http://schemas.openxmlformats.org/drawingml/2006/chart}'
REF = re.compile(r"'((?:[^']|'')+)'!")
CELL = re.compile(r'^([A-Z]+)([1-9][0-9]*)$')


def check(path):
    errors=[]; stats={'file':path.name, 'sheets':0,'formulas':0,'charts':0,'chart_refs':0,'rows':0}
    with ZipFile(path) as z:
        if z.testzip() is not None:
            errors.append('архив ZIP повреждён')
        names=set(z.namelist())
        workbook=ET.fromstring(z.read('xl/workbook.xml'))
        relroot=ET.fromstring(z.read('xl/_rels/workbook.xml.rels'))
        rels={el.attrib['Id']:el.attrib.get('Target','') for el in relroot.findall(P+'Relationship')}
        sheets=workbook.find(X+'sheets')
        sheetlist=[(el.attrib['name'], el.attrib[R+'id']) for el in sheets.findall(X+'sheet')]
        sheetnames={name for name,_ in sheetlist}
        stats['sheets']=len(sheetlist)
        if len(sheetnames) != len(sheetlist): errors.append('повторяющиеся имена листов')
        if not sheetlist or sheetlist[0][0] != 'Исходные данные': errors.append('первый лист не «Исходные данные»')
        for sheetname,rid in sheetlist:
            target=rels.get(rid)
            if not target: errors.append(f'{sheetname}: нет связи workbook.xml.rels');continue
            part='xl/'+target.lstrip('/') if not target.startswith('/xl/') else target[1:]
            if part not in names: errors.append(f'{sheetname}: отсутствует файл {part}');continue
            a1=None; seen=set(); lastrow=0
            with z.open(part) as stream:
                for event,el in ET.iterparse(stream,events=('end',)):
                    if el.tag==X+'c':
                        pos=el.get('r','')
                        if pos in seen: errors.append(f'{sheetname}: дубль {pos}')
                        seen.add(pos)
                        match=CELL.match(pos)
                        if match and int(match.group(2)) < 1: errors.append(f'{sheetname}: неправильный номер строки {pos}')
                        if pos=='A1':
                            val=el.find(X+'v')
                            a1=val.text if val is not None else ''
                            if a1=='':
                                t=el.find(X+'is')
                                a1=''.join(t.itertext()) if t is not None else a1
                        f=el.find(X+'f')
                        if f is not None:
                            stats['formulas']+=1
                            formula=f.text or ''
                            if '#REF!' in formula or '%!' in formula or formula.strip().upper()=='NA()':
                                errors.append(f'{sheetname}!{pos}: недопустимая формула {formula[:80]}')
                            for ref in REF.findall(formula):
                                clean=ref.replace("''", "'")
                                if clean not in sheetnames:
                                    errors.append(f'{sheetname}!{pos}: неизвестный лист «{clean}»')
                        el.clear()
                    if el.tag==X+'row':
                        stats['rows']+=1;el.clear()
            if not (a1 or '').startswith('На этом листе'):
                errors.append(f'{sheetname}: A1 начинается не с «На этом листе»')
        for part in sorted(n for n in names if n.startswith('xl/charts/') and n.endswith('.xml')):
            stats['charts']+=1
            with z.open(part) as fp:
                for event, el in ET.iterparse(fp, events=('end',)):
                    if el.tag == C+'f':
                        stats['chart_refs']+=1
                        value=el.text or ''
                        for ref in REF.findall(value):
                            clean=ref.replace("''", "'")
                            if clean not in sheetnames:
                                errors.append(f'{part}: ссылка на отсутствующий лист «{clean}»')
                    el.clear()
    stats['errors']=errors[:50]
    stats['ok']=not errors
    return stats

if __name__=='__main__':
    files=[]
    for p in map(Path,sys.argv[1:]):
        files+=sorted(p.glob('*.xlsx')) if p.is_dir() else [p]
    result=[check(p) for p in files]
    report={'books':len(result),'sheets':sum(r['sheets'] for r in result),'formulas':sum(r['formulas'] for r in result),'charts':sum(r['charts'] for r in result),'chart_refs':sum(r['chart_refs'] for r in result),'passed':sum(r['ok'] for r in result),'details':result}
    print(json.dumps(report,ensure_ascii=False,indent=2))
    sys.exit(0 if report['passed']==report['books'] else 1)

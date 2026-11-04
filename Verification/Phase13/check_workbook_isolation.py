#!/usr/bin/env python3
"""Independent OOXML QA for 0.54, verifying every formula in every sheet.
No external Excel library. No modification of user workbooks.

Usage: python check_workbook_isolation.py path.xlsx [other.xlsx ...]
"""
import json
import re
import sys
from pathlib import Path
from xml.etree import ElementTree as ET
from zipfile import ZipFile

X = '{http://schemas.openxmlformats.org/spreadsheetml/2006/main}'
R = '{http://schemas.openxmlformats.org/officeDocument/2006/relationships}'
P = '{http://schemas.openxmlformats.org/package/2006/relationships}'
C = '{http://schemas.openxmlformats.org/drawingml/2006/chart}'
REF = re.compile(r"'((?:[^']|'')+)'!")
SNOW = re.compile(r'^(Исходные данные|Коэффициенты|Таблица 10\.2|Таблица 11\.2|Границы зон|Перепад Б\.8|Формулы снега|Геометрия Б\.15а|Секторная сетка|Перечень загружений|Вариант \d+|Сетка Б\.15а вар\.\d+|Сетка круглой кровли \d+|Схема покрытия|График \d+)(?: (\d+))?$')
ICE = re.compile(r'^(Гололёд - исходные|Гололёд - расчёт|Гололёд - сетка|Гололёд - параметры|Гололёд табл\.12\.[234]|Ветер табл\.11\.[12]|Гололёд - график)(?: (\d+))?$')


def inspect(file):
    data = {'filename': file.name, 'sheets': 0, 'formulas': 0,
            'chart_formulas': 0, 'missing_references': 0, 'wrong_block_references': 0,
            'navigation_references': 0, 'invalid_formulas': 0, 'samples': []}
    def bad(key, message):
        data[key] += 1
        if len(data['samples']) < 25:
            data['samples'].append(message)

    with ZipFile(file) as z:
        if z.testzip():
            bad('invalid_formulas','Повреждённый ZIP')
        names = set(z.namelist())
        book = ET.fromstring(z.read('xl/workbook.xml'))
        relationships = ET.fromstring(z.read('xl/_rels/workbook.xml.rels'))
        targets = {e.get('Id'): e.get('Target', '') for e in relationships.findall(P+'Relationship')}
        sheets = [(e.get('name'), e.get(R+'id')) for e in book.find(X+'sheets').findall(X+'sheet')]
        data['sheets'] = len(sheets)
        sheetnames = {name for name,_ in sheets}
        if len(sheets) != len(sheetnames):
            bad('invalid_formulas', 'Повторяющиеся названия листов')
        if sheets[0][0] != 'Исходные данные':
            bad('invalid_formulas','Первый лист не «Исходные данные»')

        def check_refs(formula, sheet, cell):
            for refmatch in REF.finditer(formula):
                ref = refmatch.group(1).replace("''", "'")
                if ref not in sheetnames:
                    bad('missing_references',f'{sheet}!{cell}: {ref}')
                sm = SNOW.match(sheet)
                if sm and sm.group(2):
                    index = sm.group(2)
                    tm = SNOW.match(ref)
                    if ref == 'Исходные данные':
                        bad('navigation_references',f'{sheet}!{cell}: ссылка на навигационный лист')
                    if tm and tm.group(2) != index:
                        bad('wrong_block_references',f'{sheet}!{cell}: снег {index} -> {ref}')
                im = ICE.match(sheet)
                if im and im.group(2):
                    index = im.group(2)
                    tm = ICE.match(ref)
                    if tm and tm.group(2) != index:
                        bad('wrong_block_references',f'{sheet}!{cell}: гололёд {index} -> {ref}')

        for name, rid in sheets:
            target=targets.get(rid, '')
            part=('xl/'+target.lstrip('/')) if not target.startswith('/xl/') else target[1:]
            if part not in names:
                bad('invalid_formulas',f'{name}: нет part {part}')
                continue
            with z.open(part) as stream:
                a1 = None
                for event, element in ET.iterparse(stream, events=('end',)):
                    if element.tag == X+'c':
                        position = element.get('r', '')
                        if position == 'A1':
                            textnode = element.find(X+'is')
                            a1 = ''.join(textnode.itertext()) if textnode is not None else (element.findtext(X+'v') or '')
                        f = element.find(X+'f')
                        if f is not None:
                            data['formulas'] += 1
                            formula=f.text or ''
                            if '#REF!' in formula or formula.strip().upper() == 'NA()':
                                bad('invalid_formulas',f'{name}!{position}: {formula[:80]}')
                            check_refs(formula, name, position)
                        element.clear()
                    elif element.tag == X+'row':
                        element.clear()
                if not (a1 or '').startswith('На этом листе'):
                    bad('invalid_formulas',f'{name}: неверный A1')
        for part in sorted(n for n in names if n.startswith('xl/charts/') and n.endswith('.xml')):
            with z.open(part) as stream:
                for _, element in ET.iterparse(stream, events=('end',)):
                    if element.tag == C+'f':
                        data['chart_formulas'] += 1
                        check_refs(element.text or '', part, 'chart')
                    element.clear()
    data['ok'] = not any(data[key] for key in ('missing_references','wrong_block_references','navigation_references','invalid_formulas'))
    return data


if __name__ == '__main__':
    paths = [Path(f) for f in sys.argv[1:]]
    if not paths:
        sys.exit('Передайте один или несколько XLSX.')
    results = [inspect(f) for f in paths]
    print(json.dumps(results, ensure_ascii=False, indent=2))
    sys.exit(0 if all(r['ok'] for r in results) else 1)

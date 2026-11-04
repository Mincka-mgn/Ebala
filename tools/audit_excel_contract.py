#!/usr/bin/env python3
"""Static, read-only XLSX contract audit for Ebala.

No external libraries, Microsoft Office or Go compiler required.
This scanner NEVER recalculates formulas; it detects structural violations and
potential provenance leaks to review in the independent numeric test phase.

Usage: python tools/audit_excel_contract.py a.xlsx b.xlsx [--json]
Exit 0 if structural invariants pass; exit 1 for critical findings.
Warnings (numeric constants in computed sheets, immutable grids) do not
trigger a failure until the corresponding module is converted and certified.
"""
from __future__ import annotations

import argparse
import json
import posixpath
import re
import sys
import zipfile
from dataclasses import dataclass, asdict
from xml.etree import ElementTree as ET

NS = {'x': 'http://schemas.openxmlformats.org/spreadsheetml/2006/main',
      'r': 'http://schemas.openxmlformats.org/officeDocument/2006/relationships',
      'p': 'http://schemas.openxmlformats.org/package/2006/relationships'}
REF_RE = re.compile(r"'((?:[^']|'')+)'!")
NUMERIC_FORMULA = re.compile(r'\s*=?(?:[-+]?\d+(?:\.\d+)?(?:[eE][-+]?\d+)?)\s*$')
COMPUTED_NAME = re.compile(r'(?i)расч[её]т|вариант|нагрузки|коэффициент|сетка')
SOURCE_NAME = re.compile(r'(?i)исходн|табл\.|таблица|параметры')

@dataclass
class Issue:
    level: str
    sheet: str
    cell: str
    code: str
    message: str


def _ns_attr(local):
    return '{' + NS['r'] + '}' + local


def _read_string(cell):
    texts = cell.findall('.//x:is/x:t', NS)
    if texts:
        return ''.join((t.text or '') for t in texts)
    v = cell.find('x:v', NS)
    return v.text if v is not None else ''


def inspect(path):
    issues = []
    def add(level, sheet, cell, code, message):
        issues.append(Issue(level, sheet, cell, code, message))
    try:
        with zipfile.ZipFile(path) as z:
            names = set(z.namelist())
            workbook = ET.fromstring(z.read('xl/workbook.xml'))
            relxml = ET.fromstring(z.read('xl/_rels/workbook.xml.rels'))
            rels = {r.attrib['Id']: r.attrib['Target'] for r in relxml}
            sheets = []
            for item in workbook.findall('x:sheets/x:sheet', NS):
                name = item.attrib.get('name', '')
                rid = item.attrib.get(_ns_attr('id'))
                target = rels.get(rid)
                if target is None:
                    add('critical', name, '', 'SHEET_REL', 'Нет отношения листа workbook.xml.rels')
                    continue
                part = target.lstrip('/') if target.startswith('/') else posixpath.normpath(posixpath.join('xl', target))
                sheets.append((name, part))
            known = {name for name, _ in sheets}
            if len(sheets) != len(known):
                add('critical', '', '', 'DUPLICATE_SHEET', 'Повторяющиеся имена листов')
            if sheets and not (sheets[0][0].startswith('Исходные данные') or 'исходные' in sheets[0][0].lower()):
                add('warning', sheets[0][0], 'A1', 'SHEET_ORDER', 'Первый лист не является листом исходных данных')
            for name, part in sheets:
                if part not in names:
                    add('critical', name, '', 'MISSING_SHEET', 'Отсутствует ' + part)
                    continue
                root = ET.fromstring(z.read(part))
                cells = {c.attrib.get('r', ''): c for c in root.findall('.//x:sheetData/x:row/x:c', NS)}
                a1 = _read_string(cells['A1']) if 'A1' in cells else ''
                if not a1.startswith('На этом листе'):
                    add('critical', name, 'A1', 'A1_DESCRIPTION', 'Нет описания листа со словами «На этом листе…»')
                previous_row = 0
                for row in root.findall('x:sheetData/x:row', NS):
                    number = int(row.attrib.get('r', '0'))
                    if number <= previous_row:
                        add('critical', name, 'A'+str(number), 'ROW_ORDER', 'Строки листа не упорядочены')
                    previous_row = number
                if SOURCE_NAME.search(name) and not COMPUTED_NAME.search(name):
                    for address, c in cells.items():
                        if c.find('x:f', NS) is not None and 'исходн' in name.lower():
                            add('warning', name, address, 'SOURCE_HAS_FORMULA', 'Формула на листе исходных данных: проверить, не перенести ли в расчётный лист')
                for address, c in cells.items():
                    f = c.find('x:f', NS)
                    v = c.find('x:v', NS)
                    if f is not None:
                        formula = f.text or ''
                        if re.search(r'(?i)(?<![A-Z])NA\s*\(', formula) or '#REF!' in formula:
                            add('critical', name, address, 'FORMULA_ERROR', formula[:160])
                        if NUMERIC_FORMULA.fullmatch(formula):
                            add('warning', name, address, 'CONSTANT_FORMULA', 'Псевдоформула без зависимости: ' + formula[:80])
                        for ref in REF_RE.findall(formula):
                            ref = ref.replace("''", "'")
                            if ref not in known:
                                add('critical', name, address, 'BAD_SHEET_REF', ref)
                    elif v is not None and c.attrib.get('t') not in ('str','inlineStr','e'):
                        if COMPUTED_NAME.search(name) and not SOURCE_NAME.search(name):
                            text = (v.text or '').strip()
                            if re.fullmatch(r'[+-]?\d+(?:\.\d+)?(?:[Ee][+-]?\d+)?', text):
                                add('warning', name, address, 'RAW_RESULT_CANDIDATE',
                                    'Числовая константа на расчётном листе, проверить происхождение: ' + text)
            return issues
    except (OSError, zipfile.BadZipFile, KeyError, ET.ParseError) as e:
        return [Issue('critical', '', '', 'XLSX_UNREADABLE', str(e))]


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('files', nargs='+')
    parser.add_argument('--json', action='store_true')
    args = parser.parse_args()
    failed = False
    for filename in args.files:
        issues = inspect(filename)
        failed |= any(i.level == 'critical' for i in issues)
        if args.json:
            print(json.dumps({'file': filename, 'issues': [asdict(i) for i in issues]}, ensure_ascii=False))
        else:
            severity = {k: sum(i.level == k for i in issues) for k in ['critical','warning']}
            print(f'{filename}: critical={severity["critical"]}, warnings={severity["warning"]}')
            for issue in issues[:30]:
                print(f'  {issue.level:8} {issue.code:24} {issue.sheet}!{issue.cell}: {issue.message}')
            if len(issues) > 30:
                print(f'  ... +{len(issues)-30} замечаний (полный вывод: --json)')
    return 1 if failed else 0


if __name__ == '__main__':
    sys.exit(main())

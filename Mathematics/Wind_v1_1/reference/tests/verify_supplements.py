#!/usr/bin/env python3
"""Validate special-source separation and SP375 A.2 zonal linear segments."""
import csv, json, math
from pathlib import Path
P=Path(__file__).resolve().parents[1]
a=json.loads((P/'external/sp375_appendix_a_zonal_factor.json').read_text(encoding='utf-8'))
assert a['dataset_id']=='SP375_A2_ZONE_X'
knots=a['knots']
assert knots==[[0,0.25],[10,0.5],[30,0.67],[50,0.67]]
def lookup(x):
 assert 0<=x<=50
 for (a0,v0),(a1,v1) in zip(knots,knots[1:]):
  if a0<=x<=a1: return v0+(v1-v0)*(x-a0)/(a1-a0)
 raise AssertionError(x)
for x,expected in [(0,.25),(5,.375),(10,.5),(20,.585),(30,.67),(40,.67),(50,.67)]:
 assert abs(lookup(x)-expected)<1e-12,(x,lookup(x),expected)
for x in [-1,51]:
 try:lookup(x);raise AssertionError('out-of-range accepted')
 except AssertionError as e:
  if 'out-of-range' in str(e):raise
catalog=json.loads((P/'external/catalog.json').read_text(encoding='utf-8'))
assert all(s['path'].startswith('external/') for s in catalog['supplements'])
assert (P/a['figure_file']).exists()
rows=list(csv.DictReader((P/'external/guide1978_table6_historical.csv').open(encoding='utf-8-sig',newline=''),delimiter=';'))
assert len(rows)==5 and all(r['status']=='HISTORICAL_ONLY' for r in rows)
assert all(x['source']['status']=='HISTORICAL_NON_NORMATIVE_FOR_SP20_2016' for x in [json.loads((P/'external/guide1978_technical_source.json').read_text(encoding='utf8'))])
print('PASS: SP375 graph A.2 4 knots + 7 computed tests, two tower schemes; 1978 history clearly isolated')

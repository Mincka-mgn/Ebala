#!/usr/bin/env python3
"""Tests of the adopted 11.1 approximation and its transparent limitations."""
from pathlib import Path
import json,sys,math,csv
R=Path(__file__).resolve().parents[1]
sys.path.insert(0,str(R/'tools'))
from demo_arithmetic_reader import formula_value
FID='11.1.xi_davenport_approx'
g=json.loads((R/'data/graph_11_1_dynamic_coefficient.json').read_text(encoding='utf8'))
assert g['active_method']['formula_id']==FID
assert g['printed_digitized_points_usage']=='COMPARISON_AND_TESTS_ONLY_NOT_CALCULATION_SOURCE'
rows=[]
for c in g['curves']:
 d=c['delta']
 for t,v in c['points']:
  calc=formula_value(FID,{'Tg1':t,'delta_damp':d})
  rows.append((d,t,v,calc))
assert len(rows)==24
m=max(abs(a-b) for d,t,a,b in rows)
assert m<0.1,m
assert abs(formula_value(FID,{'Tg1':0,'delta_damp':0.18})-1)<1e-12
assert abs(formula_value(FID,{'Tg1':0.2,'delta_damp':0.18})-2.5)<.15
# No interpolation between printed curves; δ can be intermediate.
for t in [0.02,0.1,0.2,0.3]:
 vals=[formula_value(FID,{'Tg1':t,'delta_damp':d}) for d in [0.15,0.18,0.22,0.25,0.30]]
 assert all(vals[i]>vals[i+1] for i in range(len(vals)-1)),(t,vals)
for d in [0.15,0.18,0.22,0.3]:
 vals=[formula_value(FID,{'Tg1':t,'delta_damp':d}) for t in [0,.02,.05,.1,.2,.3]]
 assert all(vals[i]<vals[i+1] for i in range(len(vals)-1)),(d,vals)
try:
 formula_value(FID,{'Tg1':.2,'delta_damp':0})
 raise AssertionError('δ=0 must not evaluate')
except ZeroDivisionError:pass
print(f'PASS: figure 11.1: {len(rows)} printed checkpoints, max |Δξ|={m:.4f}; continuous intermediate δ; boundary tests')

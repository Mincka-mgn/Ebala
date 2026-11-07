#!/usr/bin/env python3
import json,hashlib,math,sys,csv,zipfile
from pathlib import Path
P=Path(__file__).resolve().parents[1]
get=lambda s:json.loads((P/s).read_text(encoding='utf-8'))
F=get('data/formulas.json')['formulas']; FID={f['id'] for f in F}
CS=get('data/cases.json')['cases'];Sy=get('data/symbols.json')['symbols']
T=get('data/tables_section11.json')['tables'];G=get('data/graph_11_1_dynamic_coefficient.json')
assert len(FID)==len(F) and len({c['case_id'] for c in CS})==len(CS)
assert {t['id'] for t in T}=={'11.'+str(i) for i in range(1,9)}
assert {x['id'] for x in Sy}>= {'w0','z_e','k','zeta','v_corr','c_e','delta_rough','Re'}
assert {f'B1_{i}' for i in range(1,18)}<=set(c['case_id'].split('_')[0]+'_'+c['case_id'].split('_')[1] for c in CS if c['case_id'].startswith('B1_'))
for c in CS:
 for f in c['formula_ids']:assert f in FID,(c['case_id'],f)
assert G['status']=='ENGINEERING_APPROXIMATION_APPROVED_WITH_EXPLICIT_VERIFICATION_LIMITS'
assert all(x['source_ref'] for x in F)
SYM_IDS={s['id'] for s in Sy}
for c in CS:
 if c['case_id'].startswith('SP375_'):
  assert all(field in SYM_IDS for field in c['input_ids']),(c['case_id'],c['input_ids'])
ex=get('samples/control_examples.json')['control_examples'];
for e in ex:assert all(x in FID for x in e['formulas'])
near=lambda a,b:assert_close(a,b)
def assert_close(a,b):assert abs(a-b)<1e-9,(a,b)
e=ex[0]['expected']; assert_close(e['w_m'],258.4); assert_close(e['w_g'],258.4*.92*.81)
assert_close(e['w'],e['w_m']+e['w_g']);assert_close(e['w_design'],e['w']*1.4)
assert_close(ex[1]['expected']['w_peak'],380*.85*(1+.92)*1.2*.9)
assert_close(ex[2]['expected']['z_e'],35);assert_close(ex[3]['expected']['V_cr'],12)
assert get('data/reference_linkage.json')['reference_version']=='0.6.0'
assert len(ex)==10
assert_close(ex[5]['expected']['Delta2_k'],0.3)
assert_close(ex[6]['expected']['Delta2_k'],0.6375)
assert ex[7]['expected']['mode_frequency_convergence'] is True
assert ex[8]['expected']['mode_frequency_convergence'] is False
assert ex[9]['formulas']==['11.1.xi_davenport_approx']
ref=Path(__file__).resolve().parents[2]/'reference'
if ref.exists():
 import csv
 tabids={r['table_id'] for r in csv.DictReader((ref/'data/tables_appendix_v.csv').open(encoding='utf-8',newline=''))}
 data=json.loads((ref/'data/SP20_wind_all_in_one.json').read_text(encoding='utf-8'))
 graphids={x['figure'] for x in data['graphs_straight']['figures']}|{'В.16','В.17'}
 for c in CS:
  for t in c['reference_table_ids']:assert t in tabids,(c['case_id'],t)
  for g in c['reference_graph_ids']:assert g in graphids,(c['case_id'],g)
 for f in F:
  lookup=f.get('lookup') or {}
  if lookup.get('dataset')=='reference':
   for t in lookup.get('tables',[]):assert t in tabids,(f['id'],t)
   if lookup.get('table'):assert lookup['table'] in tabids,(f['id'],lookup['table'])
   if lookup.get('graph'):assert lookup['graph'] in graphids,(f['id'],lookup['graph'])

if ref.exists():
 b=json.loads((ref/'formulas/curved_graphs.json').read_text(encoding='utf-8'))
 assert b['models']['B16']['applicability_SP']['delta_over_d']=='< 1e-4'
 old=(ref/'data/tables_appendix_v.csv').read_text(encoding='utf-8');assert old.count('В.1')>0
 print('external reference folder found, B16 applicability checked')
else:print('external reference not extracted; checked linkage metadata only')

# Проверки не только предвычисленных чисел: читаем формулы из JSON и выполняем белосписочное арифметическое дерево.
sys.path.insert(0,str(P/'tools'))
from demo_arithmetic_reader import formula_value
assert_close(formula_value('11.2',{'w0':380,'k':.85,'c_aerodynamic':.8}),258.4)
assert_close(formula_value('11.5',{'w_m':258.4,'zeta':.92,'v_corr':.81}),258.4*.92*.81)
assert_close(formula_value('11.1',{'w_m':258.4,'w_g':258.4*.92*.81}),258.4*(1+.92*.81))
assert_close(formula_value('11.10',{'w0':380,'k':.85,'zeta':.92,'c_peak':1.2,'nu_peak':.9}),380*.85*1.92*1.2*.9)
assert_close(formula_value('11.11',{'k_v':1,'f_i':1.2,'d':2,'St':.2}),12)
assert_close(formula_value('11.4',{'k10':.65,'z_e':20,'alpha_terrain':.2}),.65*2**.4)
assert_close(formula_value('11.6',{'zeta10':1.06,'z_e':20,'alpha_terrain':.2}),1.06*2**(-.2))
assert_close(formula_value('B.1.12.inclined_cx',{'c_x':.8,'beta_wind':60}),.6)
assert_close(formula_value('11.1.xi_davenport_approx',ex[9]['inputs']), ex[9]['expected']['xi'])

# Отдельный независимый регрессионный контроль ошибок, обнаруженных протоколом 06.11.2026.
import runpy
runpy.run_path(str(P/'tests/verify_appendix_m_regressions.py'),run_name='__main__')
runpy.run_path(str(P/'tests/verify_dynamic_xi_approximation.py'),run_name='__main__')
runpy.run_path(str(P/'tests/verify_symbol_dimensions.py'),run_name='__main__')
runpy.run_path(str(P/'tests/verify_sp375_rules.py'),run_name='__main__')
print('PASS:',len(F),'formulas,',len(CS),'cases,',len(Sy),'symbols,',len(T),'section-11 tables;',len(ex),'control examples')

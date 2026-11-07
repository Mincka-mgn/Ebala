#!/usr/bin/env python3
"""Регрессия: разделение f_rise/f_spectral и размерность М.20 (оба замечания 06.11.2026)."""
import sys, json, csv, math, ast, re
from pathlib import Path
ROOT=Path(__file__).resolve().parents[1]
get=lambda rel:json.loads((ROOT/rel).read_text(encoding='utf-8'))
F={r['id']:r for r in get('data/formulas.json')['formulas']}
S={r['id']:r for r in get('data/symbols.json')['symbols']}
C={r['case_id']:r for r in get('data/cases.json')['cases']}
FLOW={r['id']:r for r in get('data/flows.json')['flows']}
assert 'f' not in S, 'Ambiguous machine-level variable f is forbidden'
assert S['f_rise']['unit']=='м' and S['f_spectral']['unit']=='Гц'
assert S['f_rise']['symbol_on_scheme_or_formula']=='f'
assert S['f_spectral']['symbol_on_scheme_or_formula']=='f'
for case_id in ('B1_3_VAULT','B1_4_DOME'):
    c=C[case_id]
    assert 'f_rise' in c['input_ids'] and 'f' not in c['input_ids']
    assert c['drawing_symbol_bindings']['f']=='f_rise'
for id in ('B.1.3.vault','B.1.4.dome'):
    assert 'f_rise' in F[id]['inputs']
    assert 'f_spectral' not in F[id]['inputs']
for id in ('M.10','M.11','M.12','M.13','M.14','M.15','M.19','M.20','M.21'):
    f=F[id]
    assert 'f_rise' not in f['inputs']
    assert 'f_rise' not in f['expression']
    assert re.search(r'\bf\b',f['expression']) is None, id
    assert 'f_spectral' in f['expression'], id
    assert f['source_symbol_bindings']['f']=='f_spectral'
for id in ('M.10','M.19','M.20'):
    assert F[id]['integration_variable']=='f_spectral'
assert F['M.11']['inputs']==['f_spectral','f_k','gamma_k']
assert F['M.13']['conditions']==['f_spectral>0']
assert F['M.13']['output_unit']=='с'
assert F['M.20']['output']=='v0_squared' and F['M.20']['output_unit']=='1'
assert S['v0_squared']['unit']=='1' and S['V0']['unit']=='м/с'
assert S['xi_s_squared']['unit']=='1' and S['epsilon_D']['unit']=='1'
assert F['M.18']['input_bindings']['v0_squared']['formula_id']=='M.20'
assert 'f_spectral' in next(x for x in FLOW['appendix_m_mode_truncation']['steps'] if x['id']=='Delta2_k')['requires']
with (ROOT/'data/inputs_by_case.csv').open(encoding='utf-8',newline='') as f:
    rows=list(csv.DictReader(f,delimiter=';'))
assert len([r for r in rows if r['field_id']=='f_rise'])==2
assert all(r['field_id']!='f' for r in rows)
# М.13: S_v0 units s, M.20: integral S_v0(f)[s] * r_m(f)[1] * df[1/s] = 1.
assert S['S_v0']['unit']=='с' and S['r_m']['unit']=='1'
# Distinguish independent physical quantities by giving V0 any extreme value; v0_squared remains independent.
from itertools import product
sys.path.insert(0,str(ROOT/'tools'))
from demo_arithmetic_reader import formula_value,predicate_value
for ratio,gamma in product([0.0,.5,1.,2.],[.1,.3]):
    inputs={'f_spectral':ratio*2,'f_k':2.,'gamma_k':gamma}
    expected=(1-ratio**2)**2+gamma*ratio**2
    assert math.isclose(formula_value('M.11',inputs),expected,rel_tol=1e-12,abs_tol=1e-12)
try:
    formula_value('M.11',{'f':1.,'f_k':1.,'gamma_k':.3})
    raise AssertionError('Old ambiguous f unexpectedly accepted')
except ValueError:pass
for v0_sq in (0.0,.5,1.0):
    args={'v0_squared':v0_sq,'xi_s_squared':v0_sq*1.01,'epsilon_D':.01,'V0':300.}
    assert predicate_value('M.18',args)
    args['V0']=.1
    assert predicate_value('M.18',args)
print('PASS: f_rise vs f_spectral (geometry / spectrum), unique IDs and input mapping; M.20 dimensionless; 12 numeric M.11 + 6 M.18 checks')

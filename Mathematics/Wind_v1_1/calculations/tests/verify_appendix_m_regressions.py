#!/usr/bin/env python3
"""Контроль нормативных исправлений М.11 и М.18 (СП 20.13330.2016, печ. стр. 125–126).
Не проверяет численный интегратор приложения М, который в Ebala ещё не реализован.
"""
import json,math,sys,ast
from pathlib import Path
P=Path(__file__).resolve().parents[1]
sys.path.insert(0,str(P/'tools'))
from demo_arithmetic_reader import formula_value,predicate_value
F={x['id']:x for x in json.loads((P/'data/formulas.json').read_text(encoding='utf-8'))['formulas']}
S={x['id']:x for x in json.loads((P/'data/symbols.json').read_text(encoding='utf-8'))['symbols']}
assert F['M.11']['expression']=='(1-(f_spectral/f_k)**2)**2 + gamma_k*(f_spectral/f_k)**2'
assert F['M.18']['expression']=='xi_s_squared <= (1+epsilon_D)*v0_squared'
assert F['M.18']['inputs']==['xi_s_squared','epsilon_D','v0_squared']
assert F['M.20']['output']=='v0_squared'
assert 'V0' not in F['M.18']['inputs']
assert 'V0' not in {n.id for n in ast.walk(ast.parse(F['M.18']['expression'],mode='eval')) if isinstance(n,ast.Name)}
assert F['M.18']['input_bindings']['v0_squared']['formula_id']=='M.20'
assert F['M.18']['input_bindings']['xi_s_squared']['formula_id']=='M.19'
assert S['V0']['symbol_on_scheme_or_formula']=='V₀'
assert S['v0_squared']['symbol_on_scheme_or_formula']=='v₀²'
assert S['v0_squared']['unit']=='1'
assert S['f_rise']['unit']=='м' and S['f_spectral']['unit']=='Гц'

# Независимо рассчитываем эталон по перепроверенной формуле СП.
for ratio,gamma in [(0.0,.3),(.5,.3),(1.0,.3),(2.0,.3),(1.0,.1),(1.0,.6)]:
    expected=(1-ratio**2)**2+gamma*ratio**2
    actual=formula_value('M.11',{'f_spectral':ratio*2.0,'f_k':2.0,'gamma_k':gamma})
    assert math.isclose(actual,expected,rel_tol=1e-12,abs_tol=1e-12),(ratio,gamma,actual,expected)
assert formula_value('M.11',{'f_spectral':1.,'f_k':1.,'gamma_k':.3})==.3

# На границе должно быть <=, выше границы false. V0 не влияет на результат.
for xi,v0_sq,eps,expected in [(1.00,1.,.01,True),(1.01,1.,.01,True),(1.02,1.,.01,False),(.5,0.,.01,False),(0.,0.,.01,True),(.33,.4,.01,True)]:
    args={'xi_s_squared':xi,'v0_squared':v0_sq,'epsilon_D':eps,'V0':1000.0}
    actual=predicate_value('M.18',args)
    assert actual is expected,(args,actual,expected)
    args['V0']=0.1
    assert predicate_value('M.18',args) is expected,'V0 must not affect M.18'
try:
    predicate_value('M.18',{'xi_s_squared':1.,'epsilon_D':.01,'V0':100.})
    raise AssertionError('M.18 must reject absent v0_squared')
except ValueError:
    pass
print('PASS: M.11 (6 численных проверок), M.18 (6 критериев + независимость V0 + отсутствие v0_squared)')

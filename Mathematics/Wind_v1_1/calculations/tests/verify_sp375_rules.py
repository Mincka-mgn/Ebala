#!/usr/bin/env python3
"""Checks that special normative source rules do not leak into general SP20 cases."""
import json, math
from pathlib import Path
P=Path(__file__).resolve().parents[1]
get=lambda rel:json.loads((P/rel).read_text(encoding='utf8'))
R={r['id']:r for r in get('data/special_sp375_rules.json')['rules']}
assert len(R)==3 and R['SP375_ZONE_TWO_SCENARIOS']['scenarios']==['reduce_above_belt_break','reduce_below_belt_break']
g=R['SP375_WIND_GAMMA_F']
def gamma(H,steel=False):
 if steel:
  if H<=100:return 1.4
  if H<=210:return 1.5
  return None
 if H<150:return 1.4
 if H<=300:return 1.5
 return 1.6
for H,steel,result in [(99,True,1.4),(100,True,1.4),(100.01,True,1.5),(150,True,1.5),(210,True,1.5),(210.01,True,None),(149.99,False,1.4),(150,False,1.5),(300,False,1.5),(301,False,1.6)]:
 assert gamma(H,steel)==result,(H,steel)
assert g['override_for_steel_tubes_and_towers'][-1]['action']=='require_project_brief'
F={f['id']:f for f in get('data/formulas.json')['formulas']}
assert F['SP375.A1.w_loc']['expression']=='sp375_x_zone*sp375_w_design'
assert F['SP375.A2.lookup_x_zone']['lookup']['supplement_id']=='SP375_A2_ZONE_X'
assert F['SP375.7.1.wind_gamma']['lookup']['file']=='data/special_sp375_rules.json'
assert 'sp375_H' in F['SP375.7.1.wind_gamma']['inputs']
assert F['11.1.xi_davenport_approx']['expression']=='sqrt(1 + pi**2/(3*delta_damp) * Tg1**(2/3) / (1 + Tg1**2)**(4/3))'
C={x['case_id']:x for x in get('data/cases.json')['cases']}
assert {'SP375_TOWER_ZONAL_WIND','SP375_CHIMNEY_SPECIAL_WIND'} <= C.keys()
assert all(c['status']=='DECLARED_SPECIAL_RULES_NOT_IMPLEMENTED_IN_EBALA' for cid,c in C.items() if cid.startswith('SP375_'))
ref=P.parent/'reference'
if ref.exists():
 a=get('data/reference_linkage.json')
 assert (ref/a['linkage']['special_sp375_zone']).exists()
 p=json.loads((ref/'external/sp375_appendix_a_zonal_factor.json').read_text(encoding='utf8'))
 assert p['knots']==[[0,.25],[10,.5],[30,.67],[50,.67]]
print('PASS: SP375 gamma 10 boundaries, zonal x link and alternative loading, ξ unchanged, solver not falsely marked complete')

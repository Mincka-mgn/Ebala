#!/usr/bin/env python3
"""Check integrity and engineering spot-checks of the SP20 wind reference bundle.
No external modules and no modification of original source code.
"""
import csv, json, math
from pathlib import Path
P = Path(__file__).resolve().parent.parent

def near(a,b,tol=1e-8):
    assert abs(a-b)<tol, (a,b)

def curve(f,x):
    pts = f['points']
    log = f['x_scale']=='log10' if 'x_scale' in f else f['log']
    assert x>=pts[0][0] and x<=pts[-1][0]
    for (x0,y0),(x1,y1) in zip(pts,pts[1:]):
        if x0<=x<=x1:
            r = math.log10(x/x0)/math.log10(x1/x0) if log else (x-x0)/(x1-x0)
            return y0+(y1-y0)*r
    raise AssertionError(x)

def cp_b16(beta, v):
    bmin, cm, bb, cb = (v[k] for k in ('beta_min_deg','cp_min','beta_b_deg','cp_b'))
    assert 0<=beta<=180
    if beta<=bmin: return 1-(1-cm)*math.sin(math.pi*beta/(2*bmin))**2
    if beta<=bb: return cb-(cb-cm)*math.cos(math.pi*(beta-bmin)/(2*(bb-bmin)))
    return cb

def cx_b17_base(re,delta_ref):
    assert re>0 and delta_ref>0
    if delta_ref>=.05: return 1.2
    denom=1+.4*math.log10(re/1e6)
    assert denom!=0, 'Denominator undefined at this Reynolds number'
    rough = 1.2+.18*math.log10(10*delta_ref)/denom
    down=.11/(1.6*re/1e6)**1.4
    return max(.4,min(1.2,down),rough)

def cx_b17(re,delta,mode='CONTINUOUS_LOG_ROUGHNESS'):
    assert re>0 and delta>=0 and math.isfinite(re) and math.isfinite(delta)
    assert mode in ('CONTINUOUS_LOG_ROUGHNESS','DISCRETE_CONSERVATIVE_UPPER_COEFFICIENT')
    if delta>=.05: return 1.2
    if delta<=1e-5: return cx_b17_base(re,1e-5)
    if mode=='CONTINUOUS_LOG_ROUGHNESS': return cx_b17_base(re,delta)
    refs=[1e-5,1e-4,1e-3,1e-2,.05]
    for a,b in zip(refs,refs[1:]):
        if delta==a: return cx_b17_base(re,a)
        if a<delta<b: return max(cx_b17_base(re,a),cx_b17_base(re,b))
    raise AssertionError((re,delta))

graph_csv=list(csv.DictReader((P/'data/straight_graph_points.csv').open(encoding='utf-8',newline=''),delimiter=';'))
assert len(graph_csv)==67 and all(len(r)==13 and r['x'] and r['y'] for r in graph_csv)
DATA=json.loads((P/'data/SP20_wind_all_in_one.json').read_text(encoding='utf-8'))
rows=list(csv.DictReader((P/'data/tables_appendix_v.csv').open(encoding='utf-8',newline='')))
assert len(rows)==189 and len(DATA['tables']['rows'])==189
assert set(r['table_id'] for r in rows)==set(DATA['tables']['table_ids']) and len(set(r['table_id'] for r in rows))==13
assert rows==DATA['tables']['rows']
assert json.loads((P/'data/straight_graphs.json').read_text(encoding='utf-8'))==DATA['graphs_straight']
assert json.loads((P/'formulas/curved_graphs.json').read_text(encoding='utf-8'))==DATA['graphs_nonlinear']
assert json.loads((P/'data/branch_selection_rules.json').read_text(encoding='utf-8'))==DATA['selection_rules']
figs=DATA['graphs_straight']['figures']; assert {f['figure'] for f in figs}=={'В.5','В.6а','В.14','В.19','В.23'}
for fig in figs:
    sc=fig['x']['scale']; assert sc in ('linear','log10')
    for grp in fig['curve_groups']:
        for c in grp['curves']:
            ps=c['points']; assert len(ps)>=2 and all(math.isfinite(x) and math.isfinite(y) for x,y in ps)
            assert all(a[0]<b[0] for a,b in zip(ps,ps[1:]))
            if sc=='log10': assert all(x>0 for x,y in ps)
            f={'points':ps,'x_scale':sc}
            for x,y in ps: near(curve(f,x),y,2e-8)
find=lambda f: next(z for z in figs if z['figure']==f)
get=lambda fig,code: next(c for g in find(fig)['curve_groups'] for c in g['curves'] if c['id']==code)
near(curve({'points':get('В.23','B23_PHI_100')['points'],'x_scale':'log10'},10),.7)
near(curve({'points':get('В.5','B5_CE1_HGE05_NEG')['points'],'x_scale':'linear'},.25),-.75)
near(curve({'points':get('В.5','B5_CE1_HGE05_POS')['points'],'x_scale':'linear'},.25),.13333333333333333)
for v in DATA['graphs_nonlinear']['models']['B16']['branches']:
    near(cp_b16(0,v),1); near(cp_b16(v['beta_min_deg'],v),v['cp_min']); near(cp_b16(v['beta_b_deg'],v),v['cp_b']); near(cp_b16(180,v),v['cp_b'])
v=next(v for v in DATA['graphs_nonlinear']['models']['B16']['branches'] if v['reference_Re']==2e6)
near(cp_b16(100,v),-0.7-1.2/math.sqrt(2),1e-10)
near(cx_b17(1e6,1e-3),.84,1e-10)
near(cx_b17(1e6,.05),1.2)
near(cx_b17(5e7,1e-5),.7713,0.003)
for r in csv.DictReader((P/'verification/V17_control_points.csv').open(encoding='utf-8-sig',newline='')):
    re=float(r['Re']); delta=float(r['Delta_over_d']); actual=cx_b17(re,delta); documented=float(r['c_formula_Eurocode'])
    assert abs(actual-documented)<0.025, (re,delta,actual,documented)
assert DATA['dataset_version']=='0.6.0'
assert DATA['graphs_nonlinear']['models']['B16']['model_id']=='B16_BULGARIAN_TRIGONOMETRIC_2026_11'
segs=DATA['graphs_nonlinear']['models']['B16']['segments']
assert 'cos(' in segs[1]['expression'] and '**4' not in segs[1]['expression']
verification=list(csv.DictReader((P/'verification/V16_bulgarian_7_angles.csv').open(encoding='utf-8-sig',newline=''),delimiter=';'))
assert len(verification)==21
for line in verification:
    b=next(z for z in DATA['graphs_nonlinear']['models']['B16']['branches'] if z['reference_Re']==int(line['Re']))
    val=cp_b16(float(line['beta_deg']),b)
    near(val,float(line['cp_Bulgarian_formula']),6e-5)
    near(val-float(line['cp_SP_approx_read']),float(line['deviation_formula_minus_SP']),6e-5)
assert max(abs(float(line['deviation_formula_minus_SP'])) for line in verification)<0.08
# Two roughness modes, exact nodes, bounds, the main requested intermediate roughness example.
b17=DATA['graphs_nonlinear']['models']['B17']
assert b17['roughness_selection']['default_mode']=='CONTINUOUS_LOG_ROUGHNESS'
assert DATA['selection_rules']['cases']['B17']['default_mode']=='CONTINUOUS_LOG_ROUGHNESS'
near(cx_b17(1e6,3e-4),0.7458818258495392,1e-12)
near(cx_b17(1e6,3e-4,'DISCRETE_CONSERVATIVE_UPPER_COEFFICIENT'),0.84,1e-12)
for d,expected in [(0,.48),(1e-5,.48),(1e-4,.66),(1e-3,.84),(1e-2,1.02),(.03,1.1058818258495392),(.05,1.2)]:
    near(cx_b17(1e6,d),expected,1e-11)
assert cx_b17(1e6,.04999)<cx_b17(1e6,.05) # explicit plateau boundary may jump
for re in [1e5,2.5e5,1e6,1e7,5e7]:
    for d in [0,1e-6,1e-5,3e-5,1e-4,3e-4,1e-3,3e-3,1e-2,.03,.0499,.05,.075]:
        a=cx_b17(re,d)
        b=cx_b17(re,d,'DISCRETE_CONSERVATIVE_UPPER_COEFFICIENT')
        assert b+1e-12>=a,(re,d,a,b)
        if d in (1e-5,1e-4,1e-3,1e-2,.05): near(a,b,1e-12)
mode_rows=list(csv.DictReader((P/'verification/V17_roughness_modes.csv').open(encoding='utf-8-sig',newline=''),delimiter=';'))
assert len(mode_rows)==65
for row in mode_rows:
    re=float(row['Re']); d=float(row['delta_over_d'])
    near(cx_b17(re,d),float(row['c_continuous']),1e-9)
    near(cx_b17(re,d,'DISCRETE_CONSERVATIVE_UPPER_COEFFICIENT'),float(row['c_discrete_conservative']),1e-9)
assert 'разных' not in b17['roughness_selection']['default_mode']
print('OK | Tables: 13/189 | Straight graphs: 5/67 nodes | Curved graphs: 2 | Bulgarian B16 21 angle checks + B17 continuous/discrete 65 samples: PASSED')

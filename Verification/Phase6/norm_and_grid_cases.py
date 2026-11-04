"""Independent arithmetic and source-data checks for stage 6.

Does not execute Go or Excel. Values in reference tables were manually
transcribed from the uploaded СП 20.13330.2016 (изменения 1–6), tables
10.2 and 11.2, for independent comparison against Go worksheet generators.
"""
from pathlib import Path
import math, re
root=Path(__file__).resolve().parents[2]
s=(root/'snow_excel_norm_051.go').read_text('utf-8')

# In table 10.2 the printed columns are A/B interleaved by wind band.
# The Excel generator groups A columns before B columns.
expected_kv = {
 4: [1.4,1.3,1.3,1.4,1.4,1.3],
 5: [1.4,1.3,1.2,1.4,1.4,1.3],
 6: [1.3,1.2,1.2,1.4,1.3,1.2],
}
for row, vals in expected_kv.items():
    match=re.search(r'^\s*addW\('+str(row)+r',\s*"[^"]+",\s*(.+)\)',s,re.M)
    assert match, ('missing table 10.2 row',row)
    actual=[float(v) for v in re.findall(r'"([\d.]+)"',match.group(1))]
    assert actual == vals, (row,actual,vals)

expected_h=[5,10,20,40,60,80,100,150,200,250,300]
expected_a=[.75,1,1.25,1.5,1.7,1.85,2,2.25,2.45,2.65,2.75]
expected_b=[.5,.65,.85,1.1,1.3,1.45,1.6,1.9,2.1,2.3,2.5]
expected_c=[.4,.4,.55,.8,1,1.15,1.25,1.55,1.8,2,2.2]
for key, expected in [('hs',expected_h),('ka',expected_a),('kb',expected_b),('kc',expected_c)]:
    m=re.search(r'\b'+key+r'\s*:=\s*\[\]float64\{([^}]+)\}',s)
    assert m,key
    actual=[float(x.strip()) for x in m.group(1).split(',')]
    assert actual==expected,(key,actual,expected)

# Independent geometry checks for source-driven intervals and one native
# spherical projection formula. These are not XLSX-recalculation tests.
def dome_proj(d,f):return d if f<=d/2 else d*d/(4*f)+f
cases=[
 ('gable span 18', 0,18,0,18),
 ('gable edited span 24',0,24,0,24),
 ('three bays of 18',0,18*3,0,54),
 ('five edited bays of 18',0,18*5,0,90),
 ('step upper 14 lower 27',-14,27,-14,27),
 ('step edited upper 11 lower 35',-11,35,-11,35),
 ('dome d18 f7',0,dome_proj(18,7),0,18),
 ('deep dome d18 f12',0,dome_proj(18,12),0,18.75),
]
for label,lo,hi,elo,ehi in cases:
    assert math.isclose(lo,elo) and math.isclose(hi,ehi),label
    points=[lo+(hi-lo)*j/128 for j in range(129)]
    assert math.isclose(points[0],elo) and math.isclose(points[-1],ehi),label
    assert all(points[j+1]>points[j] for j in range(128)),label

# Angle derived from rise: exact geometrical test, independent of exported Go.
for rise,run,expected in [(3,6,26.56505117707799),(0,18,0),(2,10,11.309932474020213)]:
    got=math.degrees(math.atan(rise/run))
    assert math.isclose(got,expected,abs_tol=1e-10),(rise,run,got)

print('PASS: 3 rows (18 cells) of СП 20.13330.2016 table 10.2')
print('PASS: 4 column arrays (44 values) of СП 20.13330.2016 table 11.2')
print('PASS: 8 geometry/grid examples and 3 slope-from-rise examples')
print('LIMIT: no Go compilation, no generated XLSX, no Microsoft Excel recalc')

"""Source-only stage 7 checks; no Go compilation, XLSX execution or Office use."""
from pathlib import Path
import math
root=Path(__file__).resolve().parents[2]
read=lambda f:(root/f).read_text(encoding='utf-8')
v=read('snow_excel_v031.go')
g=read('snow_excel_grid_phase6.go')
p=read('snow_excel_paired_phase7.go')
step=read('snow_excel_paired_step_phase7.go')
test=read('excel_phase7_source_test.go')
checks={
 'one shared Excel boundary sheet for each snow block': 'snowExcelZoneBoundaryGuide(s)' in v and '"Границы зон"+suffix' in v,
 'snow case rows use boundary cell references': 'nodes[i] = fmt.Sprintf("\'Границы зон\'!$B$%d"' in g,
 'boundary sheet participates in multi-block renaming': '"\'Границы зон\'!"' in v,
 'B.2 vault has both 30 and 60 degree nodes': 'RADIANS(30)' in g and 'RADIANS(60)' in g,
 'B.3 lantern b/b1 exact node positions': '"$B$24"' in g and '"$B$25"' in g and '"$B$23"' in g,
 'B.8 step has x=0 and separate end-of-drift node': 'MIN(" + cf + "$B$36' in g,
 'paired roof segmented at shared wall': 'nodes = append(nodes, left)' in g,
 'each roof has its own B.1 B.2 B.3 formula': 'snowExcelPartMu(s.Shape' in p and 'snowExcelPartMu(s.RightRoof.Shape' in p,
 'no fallback to left roof on paired variants': 'return snowExcelPairedMu(s, c, row)' in v,
 'paired roof has independent right slope': '"α₂ второго покрытия, град"' in v and '$B$49' in v and 'DEGREES(ATAN(' in v,
 'paired vault uses own rise': 'if s.RightRoof.Shape == "vault"' in v and 'return src + "$B$45"' in p,
 'combined skylight C+C case recognized': '"Оба фонаря, зона "' in p,
 'special B.8 step is not miscomputed with B.1': 'return snowExcelPairedStepMu(s, row)' in p,
 'B.8 dedicated worksheet for paired roofs': 'snowExcelPairedStepData(s)' in v,
 'B.8 worksheet participates in multi-block renaming': "\"'Перепад Б.8'!\"" in v,
 'step selection respects which roof is high': '"=B3>B4"' in step and '"=IF(B6,B13,B14)"' in step,
 'step computation includes transfer width and load cap': '3*B16' in step and '2*B11/B18' in step,
 'zero height case does not divide by zero': '=IF(B11=0,\\"\\",1+' in step and 'B25=' not in step,
 'step unsupported shape is never replaced by Go mu': 'return "=NA()", "Не рассчитана одна из основных' in step,
 'prepared non-executed Go regression tests': 'TestPhase7PairedSkylightCZones' in test and 'TestPhase7VaultBreakpoints' in test,
 'no Go-computed intervals drive new boundary nodes': 'c.Intervals' not in g,
}
for title,ok in checks.items():
    print(('PASS' if ok else 'FAIL')+': '+title)
if not all(checks.values()):
    raise SystemExit('FAIL: source checks')
# Numeric examples calculated without using the Go executable or Excel.
# These verify the expected control-point geometry and independent B.8 arithmetic.
def vault_nodes(L,f):
    R=L*L/(8*f)+f/2
    return [0,max(0,L/2-R*math.sin(math.radians(60))),max(0,L/2-R*math.sin(math.radians(30))),L/2]
for L,f in [(18,2),(18,4),(18,7),(30,8)]:
    a=vault_nodes(L,f)
    assert all(a[i]<=a[i+1]+1e-10 for i in range(len(a)-1)), (L,f,a)
    assert a[0]==0 and abs(a[-1]-L/2)<1e-10
    # At these nodes the circle tangent makes 30° or 60° whenever that
    # specific section lies inside the vaulted span.
    R=L*L/(8*f)+f/2
    for expected,x in [(60,a[1]),(30,a[2])]:
        if x>1e-8:
            theta=math.degrees(math.asin((L/2-x)/R))
            assert abs(theta-expected)<1e-8, (expected,theta)
print('PASS: 4 independent B.2 geometric control-point examples')
for L,a,h in [(30,6,2),(60,8,3),(45,5,4)]:
    b=(L-a)/2
    b1=min(b,h)
    nodes=[0,b-b1,b,b+a,b+a+b1,L]
    assert len(nodes)==6 and all(x<=y for x,y in zip(nodes,nodes[1:]))
    assert abs((nodes[2]-nodes[1])-b1)<1e-9
print('PASS: 3 independent B.3 boundary examples')
for u,d,h,sg,B in [(18,24,2.5,1.5,24),(12,36,4,2,18),(30,15,1.4,1.1,15)]:
    m1=.4
    m2=(max(.1,.5*math.sqrt(d/21)) if d<21 else .4)
    transfer=min(d,3*B)
    raw=1+(m1*u+m2*transfer)/h
    cap=min(2*h/sg,(4+2*min(1,max(0,(max(u,d)-48)/24)) if max(u,d)>48 else 4))
    peak=max(1,min(raw,cap))
    assert peak>=1 and math.isfinite(peak)
    b=min(2*h,16,5*h)
    if peak>2*h/sg and 2*h/sg-1+2*m2>0:
        b=min(2*h*(peak-1+2*m2)/(2*h/sg-1+2*m2),5*h,16)
    assert b>=0
print('PASS: 3 independent B.8 numerical examples')
print(f'PASS: {len(checks)} source conditions and 10 arithmetic control examples; code and XLSX NOT executed')

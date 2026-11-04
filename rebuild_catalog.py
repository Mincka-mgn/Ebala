"""One-time migration of built-in Ebala v0.06 material catalog.
Does not touch Catalog/custom.json (user additions). Run once for new release.
"""
from __future__ import annotations
import json
from pathlib import Path
root=Path(__file__).resolve().parent
src=root/'catalog_v06_snapshot.json'
data=json.loads(src.read_text(encoding='utf-8'))
items={v['id']:v for v in data['items']}
cats=[
 ('Теплоизоляция и засыпки', [('c1','Пенополистиролы и полимерные'),('c2','Минеральная вата'),('c3','Другие утеплители и теплоизоляционные плиты'),('c4','Керамзит, шлак и лёгкие засыпки'),('c5','Песок, гравий и минеральные засыпки')]),
 ('Бетоны и растворы', [('c6','Лёгкие бетоны'),('c7','Ячеистые и поризованные бетоны'),('c9','Тяжёлые бетоны, растворы и стяжки')]),
 ('Кладка, стены и перегородки', [('c8','Кирпичная и блочная кладка'),('c18','Стеновые панели'),('c31','Перегородки')]),
 ('Древесина и отделка', [('c10','Древесина и пиломатериалы'),('c11','Фанера, ДВП, ДСП и древесные плиты'),('c14','Полы и напольные покрытия'),('c28','Штукатурки и отделочные растворы'),('c29','Потолки, облицовка и фасады')]),
 ('Паро- и гидроизоляция', [('c13','Рулонная гидроизоляция и мембраны'),('c26','Пароизоляция'),('c22','Обмазки, мастики и асфальтобетон')]),
 ('Кровли и покрытия', [('c24','Кровельные материалы'),('c17','Плиты перекрытий и покрытий'),('c19','Профилированные стальные настилы'),('c20','Стропила, обрешётка и дощатый настил'),('c21','Фермы и прогоны'),('c23','Кровельные системы и элементы')]),
 ('Каркасы и несущие системы', [('c30','Ригели, балки, колонны и каркасы')]),
 ('Другое', [('c12','Природный и бутовый камень'),('c15','Металлы'),('c16','Стекло и пеностекло'),('c33','Минеральное сырьё'),('c27','Прочие материалы')]),
]
data['categories']=[dict(id=i,group=g,name=n) for g,chs in cats for i,n in chs]
valid={c['id'] for c in data['categories']}
# Primary category, then optional extra search locations. Existing identities retained when useful.
changes={
 'm_38':'c11','m_38_repeat':'c11','m_51':'c8', 'm_95':'c4', 'm_250':'c3',
 'scad-012':'c14','scad-025':'c13','scad-035':'c22',
 'scad-043':'c2','scad-044':'c2','scad-045':'c2','scad-046':'c1','scad-047':'c1','scad-048':'c1',
 'scad-049':'c2','scad-050':'c1','scad-051':'c2','scad-052':'c2',
 'scad-062':'c28','scad-071':'c28',
 'scad-103':'c17','scad-104':'c14','scad-106':'c18','scad-107':'c24',
 'scad-110':'c20','scad-113':'c21','scad-115':'c30','scad-117':'c30','scad-118':'c31',
 'scad-119':'c29','scad-120':'c11','scad-121':'c14','scad-122':'c29','scad-123':'c14',
 'scad-124':'c29','scad-125':'c29',
 'scad-150':'c33','scad-151':'c12','scad-152':'c33','scad-153':'c5','scad-154':'c33',
 'scad-155':'c12','scad-156':'c12','scad-157':'c9','scad-158':'c12',
 'scad-159':'c10','scad-160':'c10','scad-161':'c10','scad-162':'c10',
 'scad-163':'c33','scad-164':'c27','scad-165':'c10','scad-166':'c5','scad-167':'c5','scad-168':'c22',
 'scad-169':'c28','scad-170':'c28','scad-171':'c16','scad-172':'c11','scad-173':'c3',
}
for id,v in items.items():
 if id in changes: v['category_id']=changes[id]
 # SCAD bulk-mineral per 100 mm is a convention, not a measured default for generic materials
 if id.startswith('scad-') and v.get('kind')=='layer' and v.get('default_thickness_mm') == 100 and int(id[5:])>=150:
  v['default_thickness_mm']=0
 if v.get('kind')=='layer' and 'density_original' in v:
  v.pop('density_original')
# Dedup only true equivalences, preserve material grades and distinct profile dimensions.
drops={
 'm_38_repeat', 'm_27', # merge mineral wool density bands below
 'scad-044','scad-046', # 2 bitumen-wool thicknesses -> one; EPP 120mm/36 kgf is inconsistent
 'scad-054','scad-056','scad-058','scad-060', # different density for same bulk material -> one range
 'scad-061','scad-062','scad-064', # screed and gypsum plaster duplicates; cement-sand mortar
 'scad-166','scad-171','scad-172', # sand, window glass, plywood
 'scad-174','scad-175', # same masonry as normative entries
 'scad-027','scad-034','scad-035', # repeated roofing felt, bitumen and cement-sand mortar
 'scad-180', # aluminum
}
# m_23 and m_27 are same product w/ density bands (in between classes are found in SP)
items['m_23']['density_min']=40.0
items['m_23']['density_max']=180.0
# scad direct weights with clear thickness: convert to an actual layer if density is interpretable
# Keep fine / roofing coatings direct if original value reflects coverage/number of applications.
def layer(id,name,density,thickness,cat=None):
 m=items[id]
 m.update(name=name,kind='layer',density_min=density,density_max=density,default_thickness_mm=thickness)
 m.pop('g_norm',None)
 if cat: m['category_id']=cat
# Substantial simplification without losing the reference density.
layer('scad-043','Минераловатные плиты на битумном связующем',400,100,'c2')
items['scad-043']['density_min']=300.0 # 70mm / 21kgf and 100mm /40kgf
layer('scad-045','Минераловатные плиты (расчётный ориентир)',200,80,'c2')
layer('scad-047','Экструзионный пенополистирол (по данным покрытия)',40,50,'c1')
# XPS 100mm -> same density 40, reference same material
items['scad-047']['default_thickness_mm']=100
# XPS 50mm 2.0kgf and 100mm 4.0kgf
# scad 047 is the authoritative density-40 reference, 048 removed as duplicate.
drops.add('scad-048')
layer('scad-049','Утеплитель фасадный ТЕХНОВЕНТ',80,100,'c2')
layer('scad-050','Плиты ПЕНОПЛЭКС 35',37.5,80,'c1')
layer('scad-051','Плиты ТЕХНОРУФ Н',120,50,'c2')
drops.add('scad-052')  # same ТЕХНОРУФ Н, ρ=121 at 100mm. Use 120 by rounded reference
layer('scad-069','Армированная цементно-песчаная стяжка',1900,40,'c9')
layer('scad-120','ДСП (плита)',780,18,'c11')
layer('scad-031','Окраска горячим битумом (2 прохода)',1200,4,'c22')
layer('scad-032','Обмазка битумно-эмульсионной пастой (2 прохода)',1200,2,'c22')
layer('scad-033','Обмазка битумно-каучуковой мастикой',1200,1,'c22')
layer('scad-036','Жидкое стекло с уплотняющей добавкой',1600,10,'c22')
layer('scad-037','Обмазка на основе синтетических смол (релакстоп)',750,4,'c22')
items['m_235']['density_min']=1200.0; items['m_235']['density_max']=1400.0;items['m_235']['default_thickness_mm']=5
# Otherwise retain prefabricated objects as direct dead load per m² without fictitious density
# Dedup identical-density seed materials.
items['m_64']['density_min']=300.0; items['m_64']['density_max']=600.0; items['m_64']['default_thickness_mm']=100
# explicit larger density range from source SCAD: 300 and 500
items['scad-053']['name']='Шлак топливный'; items['scad-053']['density_min']=700.0
items['scad-055']['name']='Шлак доменный гранулированный'; items['scad-055']['density_min']=500.0
items['scad-057']['density_min']=300.0;items['scad-057']['density_max']=500.0
# m_64 already covers c4, so remove second kерамзит entry
drops.add('scad-057')
# Shared source materials: keep a single catalog line, using density ranges and editable t.
items['m_213']['density_min']=600.0; items['m_213']['density_max']=700.0
items['m_38']['density_min']=200.0; items['m_38']['default_thickness_mm']=12
# Two sources for XPS indicate densities up to 40; one selection is enough.
items['m_7']['density_min']=35.0; items['m_7']['density_max']=40.0
drops.add('scad-047')
# Brand-specific "Пеноплекс 35" remains as a separate optional choice.
# Rename misleading source and inconsistent generic category placeholders.
items['scad-025']['name']='Геотекстиль ТЭТ (мембранный слой)'
items['scad-043']['default_thickness_mm']=70
items['scad-150']['density_min']=1530.0; items['scad-150']['density_max']=1530.0
items['m_99']['default_thickness_mm']=50
items['m_218']['default_thickness_mm']=40
items['scad-169']['name']='Гипсовая сухая штукатурка (лист)'
items['scad-170']['name']='Древесно-волокнистая сухая штукатурка (лист)'
items['scad-029']['name']='Битумный праймер (пароизоляционный слой)'
# Extras define additional locations for a *single* material record, not cloned rows.
extra={
 'm_38':['c3','c14'], 'm_51':['c31','c28'], 'm_64':['c5'], 'm_88':['c3'],
 'm_95':['c5'], 'm_99':['c4'], 'm_216':['c17','c30'],
 'm_209':['c20'], 'm_211':['c20'], 'm_213':['c14','c29'],
 'm_221':['c14'], 'm_222':['c14'], 'm_235':['c27'], 'm_218':['c28','c22'], 'm_238':['c27','c14'],
 'm_240':['c14'], 'm_249':['c29'], 'm_250':['c16'],
 'scad-001':['c20'], 'scad-012':['c24','c20'], 'scad-004':['c19'],
 'scad-013':['c24'], 'scad-014':['c24'], 'scad-015':['c24'],
 'scad-018':['c24'], 'scad-019':['c24'], 'scad-023':['c24','c23'],
 'scad-024':['c24','c23'], 'scad-025':['c26'], 'scad-026':['c13'],
 'scad-027':['c13'], 'scad-029':['c22'], 'scad-030':['c13'],
 'scad-035':['c28','c9'], 'scad-036':['c22'], 'scad-038':['c13','c24'],
 'scad-039':['c13','c24'], 'scad-040':['c13'], 'scad-041':['c13','c24'],
 'scad-042':['c13','c26'], 'scad-043':['c3'], 'scad-045':['c3'],
 'scad-049':['c29'], 'scad-050':['c3'], 'scad-051':['c23'],
 'scad-053':['c5'], 'scad-055':['c5'], 'scad-059':['c5'],
 'scad-063':['c17'], 'scad-065':['c28'], 'scad-066':['c28'],
 'scad-067':['c28'], 'scad-068':['c28'], 'scad-069':['c14'],
 'scad-070':['c17'], 'scad-071':['c9','c29'],
 'scad-100':['c3','c14','c28'], 'scad-103':['c30'],
 'scad-104':['c17'], 'scad-105':['c30'], 'scad-106':['c30'],
 'scad-107':['c29'], 'scad-108':['c23'], 'scad-109':['c23'],
 'scad-110':['c24'], 'scad-111':['c21','c30'], 'scad-112':['c21','c10'],
 'scad-113':['c20','c30'], 'scad-114':['c21'], 'scad-115':['c20'],
 'scad-116':['c30'], 'scad-117':['c21','c17'], 'scad-118':['c30'],
 'scad-119':['c24','c14'], 'scad-120':['c14','c29'], 'scad-121':['c11'],
 'scad-122':['c14'], 'scad-123':['c29'], 'scad-124':['c23'],
 'scad-125':['c16'], 'scad-150':['c28'], 'scad-151':['c5'],
 'scad-152':['c12'], 'scad-153':['c12'], 'scad-154':['c4'],
 'scad-155':['c33'], 'scad-156':['c33'], 'scad-157':['c28','c33'],
 'scad-158':['c5'], 'scad-159':['c20'], 'scad-160':['c20'],
 'scad-161':['c20'], 'scad-162':['c20'], 'scad-163':['c28'],
 'scad-165':['c4','c3'], 'scad-167':['c4'], 'scad-168':['c33'],
 'scad-169':['c29'], 'scad-170':['c11','c29'], 'scad-173':['c10'],
 'scad-176':['c31'], 'scad-177':['c31'], 'scad-181':['c33'],
 'scad-182':['c30'],
}
# Source errors that could not be reconciled: explicitly keep separate configurations
# instead of averaging unequal weights (asbestos sheets 10/12 mm, etc.).
out=[]
for v in data['items']:
 if v['id'] in drops:continue
 if v['category_id']=='c25':v['category_id']='c3'
 refs=[x for x in extra.get(v['id'],[]) if x != v['category_id']]
 v['category_ids']=list(dict.fromkeys(x for x in refs if x in valid))
 if not v['category_ids']:v.pop('category_ids',None)
 if v['category_id'] not in valid:raise ValueError('unknown category '+str(v))
 if v['kind']=='layer' and (v.get('density_max') is None or v['density_max']<0):
  raise ValueError('missing density '+str(v))
 if v['kind']=='layer' and v.get('density_min') is not None and v['density_min']>v['density_max']:
  raise ValueError('density min>max '+str(v))
 out.append(v)
data['items']=out
# Repeat-report: any true exact duplicates after consolidation are errors.
seen=set()
for m in out:
 key=(m['name'].strip().casefold(),m['kind'])
 if key in seen:raise ValueError('duplicate '+str(key))
 seen.add(key)
(root/'Catalog/materials.json').write_text(json.dumps(data,ensure_ascii=False,indent=2)+'\n',encoding='utf-8')
print(f"Catalog: {len(items)} -> {len(out)} material identities; {len(data['categories'])} categories; {sum(bool(x.get('category_ids')) for x in out)} multi-category entries")
from collections import Counter
counts=Counter(k for x in out for k in [x['category_id']]+x.get('category_ids',[]))
for cat in data['categories']:
 print(cat['id'],cat['group'],'/',cat['name'],counts[cat['id']])

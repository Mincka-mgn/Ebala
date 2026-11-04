import os,zipfile,json
ROOT=os.path.dirname(__file__); rows=[]
def add(group,title,src,params,notes='',status='reference',code='',wheels=None):
 rows.append(dict(id=f'T-{len(rows)+1:05}',group=group,title=str(title),source=src,params=params,notes=str(notes or ''),status=status,code=str(code),wheels=wheels or []))
def jfrom(path,name):
 with zipfile.ZipFile(path) as z:return json.loads(z.read(name))
z='/mnt/data/TEMP_LOADS_STAGE3_DB_v0_1.zip'
sp20=jfrom(z,'SP20_TEMPORARY/SP20_TEMPORARY.json')
for r in sp20['tables']['loads']:
 params={k:r[k] for k in ('q_kPa','P_kN','line_load_kN_per_m','gamma_f') if r.get(k) is not None}
 if not params:continue
 add('СП 20 · временные',r['description'],f"СП 20.13330.2016 {r['source']}",params,(str(r.get('notes') or '')+'; Схемы равномерной и сосредоточенной нагрузки проверяются отдельно, не одновременно.' if r.get('category')=='parking' else r.get('notes','')),'reference',r.get('variant',''))
for r in sp20['tables']['coefficients']:
 add('СП 20 · коэффициенты',r.get('note') or r['key'],r['source'],{'value':r['value'],'unit':r['unit']},'Коэффициент / правило, не отдельная нагрузка','coefficient',r['key'])
sp35=jfrom(z,'SP35_TRANSPORT/SP35_TRANSPORT.json')
for r in sp35['tables']['load_models']:
 add('СП 35 · транспорт',r['title'],f"{r['source']}, {r['source_clause']}",r['parameters'],r.get('notes',''),'reference_only' if not r.get('simulation_ready') else 'reference',r['code'])
for r in sp35['tables']['AB_vehicles']:
 add('СП 35 · транспорт',r['code'],r['source'],{k:v for k,v in r.items() if k not in ('code','source','verification')},'Схема требует проверки условий приложения по СП 35.','reference_only',r['code'])
hand=jfrom(z,'HANDBOOK_CRANES/HANDBOOK_CRANES.json')
for r in hand['tables']['crane_parameters']:
 params={k:r.get(k) for k in ('crane_capacity_t','crane_span_m','crane_mode','lift_variant','wheel_load_F_kN','wheel_load_F1_kN','horizontal_load_Qb_kN','base_A_mm','spacing_A2_mm','spacing_A3_mm','spacing_A4_mm') if r.get(k) is not None}
 title=f"{r.get('crane_capacity_t')} т · Lкр={r.get('crane_span_m')} м · {r.get('crane_mode','')} · {r.get('lift_variant','')}"
 add('Краны · справочник',title,f"Кузнецов, том 2, 1998, табл. {r.get('table')}, PDF стр. {r.get('pdf_page')}",params,'Нет подтверждённой полной геометрии колёс. Только справочные характеристики.','no_wheel_scheme')
gost=jfrom('/mnt/data/KRANY_GOST_DB_v0_1.zip','krany_gost_v1.json')
for r in gost['records']:
 params={k:r.get(k) for k in ('capacity_t','crane_span_m','p_wheel_kN','mass_t','wheel_count_one_rail','wheelbase_mm','overall_length_mm','track_load_kN') if r.get(k) is not None}
 add('Краны · ГОСТ',f"{r.get('crane_kind','Кран')} · {r.get('capacity_t','?')} т · {r.get('crane_span_m','?')} м",f"{r.get('source')} табл.{r.get('table')} стр.{r.get('page')}",params,r.get('note') or r.get('status'),'draft',r['id'])
ser=jfrom('/mnt/data/KRANY_SERII_DB_v0_1.zip','KRANY_SERII_v0_1.json')
for r in ser['cranes']:
 positions=r.get('wheel_x_mm') or [];forces=r.get('wheel_forces_kn') or []
 wheels=[{'x_mm':x,'p_kN':p} for x,p in zip(positions,forces)] if len(positions)==len(forces) else []
 add('Краны · серии',f"{r.get('crane_capacity_t')} т · Lкр={r.get('crane_span_m')} м · {r.get('crane_group')}",f"{r.get('source_document')} PDF стр.{r.get('pdf_page')}, лист {r.get('drawing_sheet')}",{'wheel_count':r.get('wheel_count'),'crane_span_m':r.get('crane_span_m'),'capacity_t':r.get('crane_capacity_t')},r.get('remarks',''),'draft_see_multiwheel' if 'выпуск 5' in str(r.get('source_document')).lower() else 'series_reference',r['id'],wheels)
multi=jfrom('/mnt/data/KRANY_MULTI_WHEEL_DB_v0_1.zip','KRANY_MULTI_WHEEL_v0_1/multiwheel_cranes.json')
for r in multi:
 positions=r.get('wheel_x_mm') or [];forces=r.get('wheel_p_kN') or []
 wheels=[{'x_mm':x,'p_kN':p} for x,p in zip(positions,forces)] if len(positions)==len(forces) else []
 add('Краны · многоколёсные',f"{r.get('crane_capacity_t')} т · Lкр={r.get('crane_span_m')} м · {r.get('crane_regime')}",f"{r.get('source_document')} PDF стр.{r.get('source_page_pdf')}, {r.get('source_sheet_or_table')}",{'wheel_count':r.get('wheel_count_one_rail'),'capacity_t':r.get('crane_capacity_t'),'crane_span_m':r.get('crane_span_m'),'wheel_envelope_mm':r.get('wheel_envelope_mm')},'Справочная схема из подготовленной базы. Требует инженерной сверки перед расчётом.','draft_multiwheel',r['id'],wheels)
path=os.path.join(ROOT,'Catalog','temporary_loads.json');os.makedirs(os.path.dirname(path),exist_ok=True)
with open(path,'w',encoding='utf8') as f:json.dump({'version':1,'sources':['СП20','СП35','ГОСТ','Серии','Кузнецов 1998'],'records':rows},f,ensure_ascii=False,indent=1)
print('rows',len(rows),'size',os.stat(path).st_size)

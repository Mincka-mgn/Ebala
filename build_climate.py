from bs4 import BeautifulSoup
from collections import defaultdict,Counter
from pathlib import Path
root=Path('/mnt/data')
regions={line.split('|')[0] for line in (root/'Ebala_003_alpha/region_capitals.tsv').read_text().splitlines() if '|' in line}
fix={
 'Адыгея':'Республика Адыгея',
 'Кемеровская область':'Кемеровская область — Кузбасс',
 'Костромкая область':'Костромская область',
 'Республика Татарстан ( Татарстан )':'Республика Татарстан',
 'Республика Татарстан (Татарстан)':'Республика Татарстан',
 'Республика Северная Осетия-Алания':'Республика Северная Осетия — Алания',
 'Республика Саза ( Якутия )':'Республика Саха (Якутия)',
 'Республика Саза (Якутия)':'Республика Саха (Якутия)',
 'Чувашская Республика - Чувашия':'Чувашская Республика',
 'Чувашская Республика- Чувашия':'Чувашская Республика',
 'Ханты-Мансийский автономный округ- Югра':'Ханты-Мансийский автономный округ — Югра',
 'Алтайский край , Республика Алтай':'Алтайский край',
 'Алтайский край, Республика Алтай':'Алтайский край',
}
# Record erroneous spelling corrections are REGIONAL ONLY; climate values are never corrected.
items=defaultdict(dict)
for name,col in [('Снеговой район', 'snow'),('Ветровой район','wind')]:
 t=BeautifulSoup((root/(name+'.html')).read_text(encoding='utf-16'),'lxml').select_one('table')
 for tr in t.select('tr')[1:]:
  row=[x.get_text(' ',strip=True) for x in tr.select('td')]
  if len(row)!=3: continue
  raw,city,zone=row
  if city=="Ростов- на - Дону": city="Ростов-на-Дону"
  raw=' '.join(raw.split())
  region=fix.get(raw,raw)
  if city=='Горно-Алтайск': region='Республика Алтай'
  if city in ('Москва','Санкт-Петербург','Севастополь'): region=city
  if region not in regions: print('MISSING REGION',repr(region),repr(raw),city)
  key=(region,city)
  if col in items[key] and items[key][col]!=zone: print('CONFLICT',key,col,zone)
  items[key][col]=zone
rows=[]
for (reg,city),data in sorted(items.items()):
 rows.append((reg,city,data.get('snow',''),data.get('wind','')))
out=root/'Ebala_003_alpha/Maps/Climate/settlements.tsv'
with out.open('w',encoding='utf-8') as f:
 f.write('# EVRAZ Steel Box, справочный каталог. Регион|Населённый пункт|Снеговой район|Ветровой район\n')
 f.write('# Районирование сверять по картам приложения Е СП 20. Для приложения К снег Sg берётся из СП.\n')
 for row in rows:f.write('|'.join(row)+'\n')
print('records',len(rows), 'snow',sum(bool(x[2]) for x in rows),'wind',sum(bool(x[3]) for x in rows),'regions',len(set(x[0] for x in rows)))
for city in ['Краснодар','Майкоп','Тихорецк','Красноярск','Москва','Казань','Ростов-на-Дону']:
 print(city,[x for x in rows if x[1]==city]);

from pathlib import Path
import cairosvg
root=Path(__file__).resolve().parents[1]/'Resources'/'Wind'
side=root/'wind_house_side_B_H_h.svg'
s=side.read_text(encoding='utf-8')
s=s.replace('<line x1="104" y1="77" x2="424" y2="77" stroke-dasharray="6 7" stroke-width="1.5"/>','')
if '<rect x="180" y="158"' not in s:
 s=s.replace('<line x1="76" y1="218"', '<rect x="180" y="158" width="38" height="34"/><line x1="199" y1="158" x2="199" y2="192"/><line x1="180" y1="175" x2="218" y2="175"/><rect x="310" y="158" width="38" height="34"/><line x1="329" y1="158" x2="329" y2="192"/><line x1="310" y1="175" x2="348" y2="175"/><line x1="76" y1="218"')
side.write_text(s,encoding='utf-8')
cairosvg.svg2png(bytestring=s.encode(),write_to=str(side.with_suffix('.png')),output_width=1080,output_height=550)
fence='''<svg xmlns="http://www.w3.org/2000/svg" width="540" height="275" viewBox="0 0 540 275">
<rect width="540" height="275" fill="white"/>
<g stroke="black" fill="none" stroke-width="2" stroke-linejoin="round">
 <path d="M80 202 L80 96 L90 81 L100 96 L100 202 M106 202 L106 96 L116 81 L126 96 L126 202 M132 202 L132 96 L142 81 L152 96 L152 202 M158 202 L158 96 L168 81 L178 96 L178 202 M184 202 L184 96 L194 81 L204 96 L204 202 M210 202 L210 96 L220 81 L230 96 L230 202 M236 202 L236 96 L246 81 L256 96 L256 202 M262 202 L262 96 L272 81 L282 96 L282 202 M288 202 L288 96 L298 81 L308 96 L308 202 M314 202 L314 96 L324 81 L334 96 L334 202 M340 202 L340 96 L350 81 L360 96 L360 202 M366 202 L366 96 L376 81 L386 96 L386 202 M392 202 L392 96 L402 81 L412 96 L412 202 M418 202 L418 96 L428 81 L438 96 L438 202"/>
 <line x1="78" y1="140" x2="440" y2="140" stroke-width="5"/><line x1="78" y1="180" x2="440" y2="180" stroke-width="5"/>
 <line x1="63" y1="206" x2="460" y2="206"/>
 <line x1="84" y1="217" x2="84" y2="254"/><line x1="440" y1="217" x2="440" y2="254"/><line x1="84" y1="243" x2="440" y2="243"/>
 <path d="M91 239 L84 243 L91 247 M433 239 L440 243 L433 247"/>
 <line x1="466" y1="81" x2="466" y2="206"/><line x1="459" y1="81" x2="473" y2="81"/><line x1="459" y1="206" x2="473" y2="206"/>
</g>
<g font-family="Arial,sans-serif" text-anchor="middle" fill="black"><text x="262" y="268" font-size="19">L</text><text x="486" y="147" font-size="19">H</text></g>
</svg>'''
f=(root/'wind_fence_H_L.svg');f.write_text(fence,encoding='utf-8')
cairosvg.svg2png(bytestring=fence.encode(),write_to=str(f.with_suffix('.png')),output_width=1080,output_height=550)
print(side, f)

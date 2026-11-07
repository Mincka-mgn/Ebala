from pathlib import Path
from PIL import Image, ImageDraw, ImageFont
root=Path(__file__).resolve().parent.parent / 'Resources' / 'Wind'
root.mkdir(parents=True, exist_ok=True)
original=Image.open(root/'wind_V3_norm.png').convert('RGB')
# The two crops preserve the original normative graphics, without the
# not-yet-implemented reverse-sloping facade or the pulsation diagram.
original.crop((0,0,252,245)).save(root/'wind_plan_only.png')
original.crop((271,28,474,240)).save(root/'wind_side_zones.png')
# An original dimensions-only schematic. SVG retained in sources.
svg='''<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 520 300">
<rect width="520" height="300" fill="white"/>
<g stroke="#111" stroke-width="3" fill="none" stroke-linejoin="round"><path d="M80 242V136L230 44L380 136V242Z"/><path d="M80 136H380" stroke-dasharray="6 7"/><path d="M58 249H402" stroke-width="2"/></g>
<g stroke="#111" stroke-width="1.5" fill="none"><path d="M408 242V44M402 44h12M402 242h12M425 136V44M419 44h12M419 136h12"/></g>
<g font-family="DejaVu Sans, sans-serif" font-size="20" fill="#111"><text x="442" y="151">H</text><text x="449" y="96">h</text><text x="92" y="282" font-size="15">H — до конька; h — подъём крыши</text></g>
</svg>'''
(root/'wind_house_H_h.svg').write_text(svg,encoding='utf-8')
# Rasterized image uses identical geometry and labels for Win32 image renderer.
im=Image.new('RGB',(1040,600),'white'); d=ImageDraw.Draw(im)
font='/usr/share/fonts/truetype/dejavu/DejaVuSans.ttf'
f=ImageFont.truetype(font,38); small=ImageFont.truetype(font,27)
scale=2
p=lambda x,y:(int(x*scale),int(y*scale))
d.line([p(80,242),p(80,136),p(230,44),p(380,136),p(380,242),p(80,242)],fill='#111111',width=5,joint='curve')
d.line([p(80,136),p(380,136)],fill='#777777',width=3)
d.line([p(60,249),p(401,249)],fill='#111111',width=3)
for coords in [[(408,242),(408,44)],[(425,136),(425,44)],[(402,44),(414,44)],[(402,242),(414,242)],[(419,44),(431,44)],[(419,136),(431,136)]]:d.line([p(*xy) for xy in coords],fill='#222222',width=3)
d.text(p(438,140),'H',font=f,fill='black');d.text(p(447,88),'h',font=f,fill='black')
d.text(p(78,266),'H — до конька; h — подъём крыши',font=small,fill='black')
im.save(root/'wind_house_H_h.png')
print(root)

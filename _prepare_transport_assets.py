from PIL import Image,ImageOps,ImageDraw,ImageFont
import os,json
root='/mnt/data/ebala_015_work/Resources/Transport';os.makedirs(root,exist_ok=True)
src=Image.open('/mnt/data/image(20261030-125012).png').convert('RGB')
boxes={'AK':(128,0,451,171),'NK':(129,175,458,319),'Metro':(23,327,516,465),'Tram':(18,470,531,615)}
for key,box in boxes.items():
 im=src.crop(box); im.save(f'{root}/{key}.png')
 # presentation wide aspect 400x170 (not squeeze, white letterbox)
 thumbnail=Image.new('RGB',(420,180),'white');im.thumbnail((410,172)); thumbnail.paste(im,((420-im.width)//2,(180-im.height)//2));thumbnail.save(f'{root}/{key}.bmp')
for key,inp in [('AB', 'image(20261030-125152).png'),('AB_Distances','image(20261030-125239).png')]:
 im=Image.open('/mnt/data/'+inp).convert('RGB');im.save(f'{root}/{key}.png'); im.thumbnail((410,170));thumbnail=Image.new('RGB',(420,180),'white'); thumbnail.paste(im,((420-im.width)//2,(180-im.height)//2));thumbnail.save(f'{root}/{key}.bmp')
print(os.listdir(root))

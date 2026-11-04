//go:build !windows

package main

import (
	"bytes"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"image"
	"image/draw"
	"image/png"
	"io"
	"os"
	"os/exec"
	"sync"
)

// The Linux-only report test adapter keeps one real TrueType renderer alive.
// The Windows executable uses native GDI and needs no Python or PIL.
var snowTestFonts struct {
	sync.Mutex
	cmd *exec.Cmd
	in  io.WriteCloser
	out io.ReadCloser
}

const snowPILWorker = `import sys,io,json,struct
from PIL import Image,ImageDraw,ImageFont
fonts={};sizes={3:34,4:37,5:38,6:41}
read=sys.stdin.buffer.read;write=sys.stdout.buffer.write
while True:
 p=read(4)
 if not p:break
 length=struct.unpack('<I',p)[0]
 src=read(length)
 tlen=struct.unpack('<I',read(4))[0]
 labels=json.loads(read(tlen))
 im=Image.open(io.BytesIO(src)).convert('RGBA')
 d=ImageDraw.Draw(im)
 for t in labels:
  sz=sizes.get(min(t['Size'],6),34)
  if sz not in fonts:fonts[sz]=ImageFont.truetype('/usr/share/fonts/truetype/dejavu/DejaVuSans.ttf',sz)
  c=t['Ink'];d.text((t['X'],t['Y']),t['Text'],font=fonts[sz],fill=(c['R'],c['G'],c['B'],255))
 buf=io.BytesIO();im.save(buf,'PNG');ans=buf.getvalue()
 write(struct.pack('<I',len(ans)));write(ans);sys.stdout.buffer.flush()
`

func snowRenderPlatformText(dst *image.RGBA, captions []snowTextCmd) error {
	if len(captions) == 0 {
		return nil
	}
	snowTestFonts.Lock()
	defer snowTestFonts.Unlock()
	if snowTestFonts.cmd == nil {
		cmd := exec.Command("python3", "-u", "-c", snowPILWorker)
		input, e := cmd.StdinPipe()
		if e != nil {
			return e
		}
		output, e := cmd.StdoutPipe()
		if e != nil {
			return e
		}
		cmd.Stderr = os.Stderr
		if e = cmd.Start(); e != nil {
			return e
		}
		snowTestFonts.cmd = cmd
		snowTestFonts.in = input
		snowTestFonts.out = output
	}
	var imagebuf bytes.Buffer
	if e := png.Encode(&imagebuf, dst); e != nil {
		return e
	}
	capbuf, e := json.Marshal(captions)
	if e != nil {
		return e
	}
	if e = binary.Write(snowTestFonts.in, binary.LittleEndian, uint32(imagebuf.Len())); e != nil {
		return e
	}
	if _, e = snowTestFonts.in.Write(imagebuf.Bytes()); e != nil {
		return e
	}
	if e = binary.Write(snowTestFonts.in, binary.LittleEndian, uint32(len(capbuf))); e != nil {
		return e
	}
	if _, e = snowTestFonts.in.Write(capbuf); e != nil {
		return e
	}
	var n uint32
	if e = binary.Read(snowTestFonts.out, binary.LittleEndian, &n); e != nil {
		return e
	}
	if n > 32*1024*1024 {
		return fmt.Errorf("image too large: %d", n)
	}
	result := make([]byte, n)
	if _, e = io.ReadFull(snowTestFonts.out, result); e != nil {
		return e
	}
	finished, e := png.Decode(bytes.NewReader(result))
	if e != nil {
		return e
	}
	draw.Draw(dst, dst.Bounds(), finished, image.Point{}, draw.Src)
	return nil
}

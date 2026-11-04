package main

import (
	"bytes"
	"image"
	"image/color"
	"image/draw"
	"image/png"
	"math"
)

func reservoirPlanImage() ([]byte, error) {
	const w, h = 680, 615
	im := image.NewRGBA(image.Rect(0, 0, w, h))
	draw.Draw(im, im.Bounds(), image.NewUniform(color.White), image.Point{}, draw.Src)
	xc, yc, r := 330, 290, 210
	put := func(x, y int, c color.RGBA) {
		if image.Pt(x, y).In(im.Bounds()) {
			im.SetRGBA(x, y, c)
		}
	}
	black := color.RGBA{32, 37, 44, 255}
	blue := color.RGBA{225, 238, 250, 255}
	gray := color.RGBA{155, 165, 175, 255}
	// Only the upper sector (0…180°) is active in the spatial snow-drift case.
	for y := yc - r; y <= yc; y++ {
		for x := xc - r; x <= xc+r; x++ {
			dx, dy := x-xc, y-yc
			if dx*dx+dy*dy <= r*r {
				put(x, y, blue)
			}
		}
	}
	line := func(x1, y1, x2, y2 int, c color.RGBA) {
		n := int(math.Max(math.Abs(float64(x2-x1)), math.Abs(float64(y2-y1))))
		if n == 0 {
			return
		}
		for j := 0; j <= n; j++ {
			u := float64(j) / float64(n)
			put(int(math.Round(float64(x1)+float64(x2-x1)*u)), int(math.Round(float64(y1)+float64(y2-y1)*u)), c)
		}
	}
	for i := 0; i < 2160; i++ {
		th := 2 * math.Pi * float64(i) / 2160
		x := xc + int(math.Round(float64(r)*math.Cos(th)))
		y := yc - int(math.Round(float64(r)*math.Sin(th)))
		put(x, y, black)
		put(x+1, y, black)
	}
	line(xc-r, yc, xc+r, yc, black)
	line(xc, yc, xc, yc-r, gray)
	line(xc, yc, xc+int(.707*float64(r)), yc-int(.707*float64(r)), black)
	// sector arcs beta=0...45...90...180 (the angle is shown from the rightward axis)
	for i := 0; i <= 180; i++ {
		th := math.Pi * float64(i) / 180
		x := xc + int(66*math.Cos(th))
		y := yc - int(66*math.Sin(th))
		put(x, y, black)
	}
	labels := []snowTextCmd{
		{X: 45, Y: 26, Size: 6, Text: "Вид сверху. Пространственный снегозанос", Ink: black},
		{X: 540, Y: 279, Size: 5, Text: "β = 0°", Ink: black},
		{X: 42, Y: 279, Size: 5, Text: "β = 180°", Ink: black},
		{X: 351, Y: 73, Size: 5, Text: "β = 90°", Ink: black},
		{X: 392, Y: 153, Size: 5, Text: "β", Ink: black},
		{X: 335, Y: 130, Size: 5, Text: "μ = μmax", Ink: black},
		{X: 215, Y: 432, Size: 5, Text: "β = 180°…360°: μ = 0", Ink: black},
		{X: 48, Y: 556, Size: 4, Text: "Для неравномерного пространственного варианта загружения", Ink: black},
	}
	if err := snowRenderPlatformText(im, labels); err != nil {
		return nil, err
	}
	var b bytes.Buffer
	err := png.Encode(&b, im)
	return b.Bytes(), err
}

func croppedRoofImage(data []byte) (image.Image, error) {
	im, err := png.Decode(bytes.NewReader(data))
	if err != nil {
		return nil, err
	}
	b := im.Bounds()
	left, right, top, bottom := b.Max.X, b.Min.X, b.Max.Y, b.Min.Y
	for y := b.Min.Y; y < b.Max.Y; y++ {
		for x := b.Min.X; x < b.Max.X; x++ {
			r, g, bb, _ := im.At(x, y).RGBA()
			if r < 59000 || g < 59000 || bb < 59000 {
				if x < left {
					left = x
				}
				if x > right {
					right = x
				}
				if y < top {
					top = y
				}
				if y > bottom {
					bottom = y
				}
			}
		}
	}
	if right <= left || bottom <= top {
		return im, nil
	}
	pad := 18
	if left-pad < b.Min.X {
		left = b.Min.X
	} else {
		left -= pad
	}
	if top-pad < b.Min.Y {
		top = b.Min.Y
	} else {
		top -= pad
	}
	if right+pad >= b.Max.X {
		right = b.Max.X - 1
	} else {
		right += pad
	}
	if bottom+pad >= b.Max.Y {
		bottom = b.Max.Y - 1
	} else {
		bottom += pad
	}
	return im.(interface {
		SubImage(image.Rectangle) image.Image
	}).SubImage(image.Rect(left, top, right+1, bottom+1)), nil
}

// High-resolution compositing preserves source geometry and makes the plan readable in Word and Windows.
func reservoirGeometryWithPlan(roof []byte) ([]byte, error) {
	src, e := croppedRoofImage(roof)
	if e != nil {
		return nil, e
	}
	planData, e := reservoirPlanImage()
	if e != nil {
		return nil, e
	}
	plan, e := png.Decode(bytes.NewReader(planData))
	if e != nil {
		return nil, e
	}
	const height = 615
	sw, sh := src.Bounds().Dx(), src.Bounds().Dy()
	pw, ph := plan.Bounds().Dx(), plan.Bounds().Dy()
	leftW := int(math.Round(float64(sw) * float64(height) / float64(sh)))
	if leftW > 1290 {
		leftW = 1290
	}
	const rightW = 680
	out := image.NewRGBA(image.Rect(0, 0, leftW+rightW+20, height))
	draw.Draw(out, out.Bounds(), image.NewUniform(color.White), image.Point{}, draw.Src)
	for y := 0; y < height; y++ {
		for x := 0; x < leftW; x++ {
			sx := src.Bounds().Min.X + x*sw/leftW
			sy := src.Bounds().Min.Y + y*sh/height
			out.Set(x, y, src.At(sx, sy))
		}
	}
	for y := 0; y < height; y++ {
		for x := 0; x < rightW; x++ {
			out.Set(leftW+20+x, y, plan.At(plan.Bounds().Min.X+x*pw/rightW, plan.Bounds().Min.Y+y*ph/height))
		}
	}
	var b bytes.Buffer
	e = png.Encode(&b, out)
	return b.Bytes(), e
}

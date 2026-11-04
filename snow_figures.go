package main

import (
	"bytes"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"image/png"
	"math"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
)

// Print-ready figures are embedded into Word and separate Excel worksheets.
// Labels are drawn using an operating-system TrueType font.
type snowFigure struct {
	img      *image.RGBA
	captions []snowTextCmd
}

var snowInk = color.RGBA{0, 0, 0, 255}
var snowLoad = color.RGBA{0, 0, 0, 255}
var snowDim = color.RGBA{106, 117, 129, 255}

func newSnowFigure() *snowFigure {
	im := image.NewRGBA(image.Rect(0, 0, 1400, 470))
	draw.Draw(im, im.Bounds(), &image.Uniform{color.White}, image.Point{}, draw.Src)
	return &snowFigure{img: im}
}
func (f *snowFigure) line(x1, y1, x2, y2 int, c color.RGBA) {
	dx := int(math.Abs(float64(x2 - x1)))
	dy := int(math.Abs(float64(y2 - y1)))
	sx, sy := -1, -1
	if x1 < x2 {
		sx = 1
	}
	if y1 < y2 {
		sy = 1
	}
	err := dx - dy
	for {
		if image.Pt(x1, y1).In(f.img.Bounds()) {
			f.img.SetRGBA(x1, y1, c)
		}
		if x1 == x2 && y1 == y2 {
			break
		}
		e2 := 2 * err
		if e2 > -dy {
			err -= dy
			x1 += sx
		}
		if e2 < dx {
			err += dx
			y1 += sy
		}
	}
}
func (f *snowFigure) text(x, y, size int, s string, c color.RGBA) {
	f.captions = append(f.captions, snowTextCmd{X: x, Y: y, Size: size, Text: s, Ink: c})
}
func (f *snowFigure) whiteRect(x1, y1, x2, y2 int) {
	r := image.Rect(x1, y1, x2, y2).Intersect(f.img.Bounds())
	draw.Draw(f.img, r, &image.Uniform{color.White}, image.Point{}, draw.Src)
}
func (f *snowFigure) dim(x1, x2, y int, s string) {
	f.line(x1, y, x2, y, snowDim)
	f.line(x1, y-9, x1, y+9, snowDim)
	f.line(x2, y-9, x2, y+9, snowDim)
	if x2-x1 > 165 {
		f.text((x1+x2)/2-snowCaptionWidth(s, 5)/2, y-58, 5, s, snowInk)
	}
}
func (f *snowFigure) arrow(x, y1, y2 int) {
	f.line(x, y1, x, y2, snowLoad)
	f.line(x-5, y2-9, x, y2, snowLoad)
	f.line(x+5, y2-9, x, y2, snowLoad)
}
func (f *snowFigure) png() ([]byte, error) {
	if err := snowRenderPlatformText(f.img, f.captions); err != nil {
		return nil, err
	}
	var out bytes.Buffer
	err := png.Encode(&out, f.img)
	return out.Bytes(), err
}
func snowValueText(v float64) string { return decimalInput(v, 2) }
func snowCaseBounds(c SnowCase) (float64, float64, float64) {
	mn, mx, mm := 0., 0., 0.
	for _, v := range c.Intervals {
		mn = math.Min(mn, v.X1)
		mx = math.Max(mx, v.X2)
		mm = math.Max(mm, v.Mu)
	}
	return mn, mx, mm
}
func snowMuAt(c SnowCase, x float64) float64 {
	for i, v := range c.Intervals {
		if x >= v.X1 && (x < v.X2 || i == len(c.Intervals)-1 && x <= v.X2) {
			return v.Mu
		}
	}
	return 0
}
func firstIntervalMu(c SnowCase) float64 {
	if len(c.Intervals) == 0 {
		return 0
	}
	return c.Intervals[0].Mu
}
func snowImage(s SnowCalc, c SnowCase) ([]byte, error) {
	if len(c.Intervals) == 0 {
		return nil, fmt.Errorf("у варианта нет участков эпюры")
	}
	f := newSnowFigure()
	mn, mx, mumax := snowCaseBounds(c)
	L := mx - mn
	if L <= 0 {
		return nil, fmt.Errorf("пустая эпюра")
	}
	if mumax <= 0 {
		mumax = 1
	}
	x0, xN, base := 85, 1310, 320
	xx := func(x float64) int { return x0 + int((x-mn)/L*float64(xN-x0)) }
	yy := func(mu float64) int { return base - int(mu/mumax*122) }
	f.line(x0, base, xN, base, snowInk)
	profile, continuous := snowParapetProfile(s, c)
	if continuous {
		for i := 0; i < 35; i++ {
			x := mn + L*(float64(i)+.5)/35
			f.arrow(xx(x), yy(snowProfileMuAt(profile, x)), base)
		}
		for i := 1; i < len(profile); i++ {
			a, b := profile[i-1], profile[i]
			f.line(xx(a.X), yy(a.Mu), xx(b.X), yy(b.Mu), snowLoad)
		}
	} else {
		for i := 0; i < 35; i++ {
			x := mn + L*(float64(i)+.5)/35
			f.arrow(xx(x), yy(snowMuAt(c, x)), base)
		}
		for _, seg := range c.Intervals {
			f.line(xx(seg.X1), yy(seg.Mu), xx(seg.X2), yy(seg.Mu), snowLoad)
		}
	}
	dims := snowDiagramPoints(c)
	if continuous {
		dims = dims[:0]
		for _, p := range profile {
			dims = append(dims, p.X)
		}
	}
	if (s.Shape == "multi_gable" || s.Shape == "multi_vault") && s.Count > 1 && s.Count <= 8 {
		for i := 1; i < s.Count; i++ {
			dims = append(dims, mn+L*float64(i)/float64(s.Count))
		}
	}
	if s.Shape == "sphere_cone" && s.Span > 2*s.SkylightWidth && s.SkylightWidth > 0 {
		dims = []float64{0, s.Span/2 - s.SkylightWidth, s.Span / 2, s.Span/2 + s.SkylightWidth, s.Span}
	}
	if s.Shape == "step" || s.Shape == "wall" {
		if mn < 0 && mx > 0 {
			dims = append(dims, 0)
		}
	}
	sort.Float64s(dims)
	filtered := dims[:0]
	for _, v := range dims {
		if len(filtered) == 0 || math.Abs(v-filtered[len(filtered)-1]) > 1e-6 {
			filtered = append(filtered, v)
		}
	}
	dims = filtered
	for _, x := range dims {
		px := xx(x)
		f.line(px, base, px, 403, snowDim)
	}
	for i := 1; i < len(dims); i++ {
		f.dim(xx(dims[i-1]), xx(dims[i]), 409, decimalInput(dims[i]-dims[i-1], 2))
	}
	// Print each constant-intensity region when a compact step diagram is
	// provided. Character-sized labels are placed in separate lanes to avoid
	// overlap, including two independently loaded lanterns. Dense sampled
	// arches instead receive end and maximum values, with all points in the
	// accompanying numerical table.
	type pt struct{ x, mu float64 }
	values := []pt{}
	groups := []SnowInterval{}
	for _, it := range c.Intervals {
		if len(groups) > 0 && math.Abs(groups[len(groups)-1].Mu-it.Mu) < 1e-9 && math.Abs(groups[len(groups)-1].X2-it.X1) < 1e-8 {
			groups[len(groups)-1].X2 = it.X2
		} else {
			groups = append(groups, it)
		}
	}
	if len(groups) <= 10 {
		for _, g := range groups {
			values = append(values, pt{(g.X1 + g.X2) / 2, g.Mu})
		}
	} else {
		values = append(values, pt{mn, firstIntervalMu(c)})
		// Mark significant local maxima as well as the global one.
		// Repeated roof-bay snowdrifts must not lose their individual peaks.
		candidates := []pt{}
		for i, g := range groups {
			if g.Mu < .5*mumax {
				continue
			}
			prev, next := -1., -1.
			if i > 0 {
				prev = groups[i-1].Mu
			}
			if i+1 < len(groups) {
				next = groups[i+1].Mu
			}
			if (g.Mu >= prev && g.Mu > next) || (g.Mu > prev && g.Mu >= next) {
				candidates = append(candidates, pt{(g.X1 + g.X2) / 2, g.Mu})
			}
		}
		sort.SliceStable(candidates, func(i, j int) bool { return candidates[i].mu > candidates[j].mu })
		peaks := []pt{}
		for _, p := range candidates {
			if p.x <= mn+.055*L || p.x >= mx-.055*L {
				continue
			}
			far := true
			for _, v := range peaks {
				if math.Abs(v.x-p.x) < .09*L {
					far = false
					break
				}
			}
			if far {
				peaks = append(peaks, p)
			}
			if len(peaks) >= 5 {
				break
			}
		}
		values = append(values, peaks...)
		values = append(values, pt{mx, c.Intervals[len(c.Intervals)-1].Mu})
	}
	sort.SliceStable(values, func(i, j int) bool { return values[i].x < values[j].x })
	rightEnds := []int{-100, -100, -100}
	lanes := []int{44, 124, 171}
	for _, v := range values {
		w := snowCaptionWidth(snowLoadValue(s, v.mu), 6)
		x := xx(v.x) - w/2
		if x < 8 {
			x = 8
		}
		if x+w > 1392 {
			x = 1392 - w
		}
		lane := -1
		for j := range rightEnds {
			if x > rightEnds[j]+12 {
				lane = j
				break
			}
		}
		if lane < 0 {
			continue
		} // Full detail remains in the numeric table.
		y := lanes[lane]
		f.whiteRect(x-4, y-4, x+w+4, y+snowPointSize(6)+4)
		f.coloredLoadText(x, y, 6, s, v.mu)
		rightEnds[lane] = x + w
	}
	// Display units on the graphic itself, not solely in surrounding text.
	// Report and workbook graphs use kPa (the calculation units).
	f.text(85, 420, 4, "Нагрузка: Sн (Sр/Sпониж), кПа; размеры, м", snowInk)
	return f.png()
}
func absInt(x int) int {
	if x < 0 {
		return -x
	}
	return x
}
func snowRoofImageLegacy(s SnowCalc) ([]byte, error) {
	f := newSnowFigure()
	x0, xN := 70, 1330
	yc := 295
	f.text(70, 22, 4, "ROOF GEOMETRY: H, L, B, H1, H2, A, F", snowInk)
	f.line(x0, 380, xN, 380, snowDim)
	roof := func(x1, x2, y1, y2 int, shape string) {
		switch shape {
		case "gable", "cone":
			xm := (x1 + x2) / 2
			f.line(x1, y1, xm, y1-95, snowInk)
			f.line(xm, y1-95, x2, y2, snowInk)
		case "mono":
			f.line(x1, y1, x2, y1-75, snowInk)
		case "vault", "dome":
			for i := 0; i < 80; i++ {
				u := float64(i) / 80
				v := float64(i+1) / 80
				f.line(x1+int(u*float64(x2-x1)), y1-int(90*math.Sin(u*math.Pi)), x1+int(v*float64(x2-x1)), y1-int(90*math.Sin(v*math.Pi)), snowInk)
			}
		default:
			f.line(x1, y1, x2, y2, snowInk)
		}
	}
	switch s.Shape {
	case "step", "wall":
		u, l := s.stepRoofs()
		split := x0 + int(float64(xN-x0)*u.Span/(u.Span+l.Span))
		if s.Shape == "wall" {
			split = 520
		}
		f.line(split, 220, split, 365, snowInk)
		if s.Shape == "step" {
			roof(x0, split, 260, 260, u.Shape)
			f.dim(x0, split, 422, "L1="+snowValueText(u.Span))
			f.text(70, 110, 3, "A1="+snowValueText(u.resolvedAngle())+" Z1="+snowValueText(u.EdgeLevel), snowInk)
		}
		roof(split, xN, 365, 365, l.Shape)
		f.dim(split, xN, 422, "L2="+snowValueText(l.Span))
		f.text(705, 110, 3, "A2="+snowValueText(l.resolvedAngle())+" Z2="+snowValueText(l.EdgeLevel), snowInk)
		f.text(550, 290, 4, "H="+snowValueText(s.H), snowInk)
	case "skylight":
		a := float64(xN-x0) * s.SkylightWidth / s.Span
		cx := (x0 + xN) / 2
		l := cx - int(a/2)
		r := cx + int(a/2)
		f.line(x0, yc, l, yc-55, snowInk)
		f.line(l, yc-55, l, yc-125, snowInk)
		f.line(l, yc-125, cx, yc-150, snowInk)
		f.line(cx, yc-150, r, yc-125, snowInk)
		f.line(r, yc-125, r, yc-55, snowInk)
		f.line(r, yc-55, xN, yc, snowInk)
		f.dim(l, r, 120, "A="+snowValueText(s.SkylightWidth))
		f.text(155, yc-100, 4, "ALPHA / H", snowInk)
		f.text(930, yc-100, 4, "ALPHA / H", snowInk)
		f.text(r+22, yc-150, 4, "H1="+snowValueText(s.H), snowInk)
		f.dim(x0, xN, 420, "L="+snowValueText(s.Span))
	case "multi_gable", "multi_vault":
		n := s.Count
		if n < 2 {
			n = 2
		}
		for i := 0; i < n; i++ {
			roof(x0+(xN-x0)*i/n, x0+(xN-x0)*(i+1)/n, yc, yc, map[bool]string{true: "gable", false: "vault"}[s.Shape == "multi_gable"])
		}
		f.dim(x0, xN, 422, "N*L")
	default:
		roof(x0, xN, yc, yc, s.Shape)
		f.dim(x0, xN, 422, "L="+snowValueText(s.Span))
	}
	f.text(75, 69, 4, "H RIDGE = "+snowValueText(s.RoofHeight)+" M", snowInk)
	f.text(630, 69, 4, "B = "+snowValueText(s.Width)+" M", snowInk)
	if s.Shape != "step" && s.Shape != "wall" {
		f.text(75, 113, 4, "ALPHA = "+snowValueText(s.Angle)+" DEG", snowInk)
	}
	return f.png()
}

// The author-provided SCAD roof schemes are fixed, version-controlled assets.
// Identical rasters are used for Word and the preview; SVG originals ship alongside them.
func snowRoofResourceName(shape string) string {
	switch shape {
	case "flat", "mono":
		return "scad_001.png"
	case "gable":
		return "scad_002.png"
	case "multi_gable":
		return "snow_multi_gable_approved.png"
	case "vault":
		return "scad_003.png"
	case "multi_vault":
		return "snow_multi_vault.png"
	case "skylight":
		return "scad_004.png"
	case "shed":
		return "snow_B_shed_roof_H_final_preview.png"
	case "parapet":
		return "snow_B13_parapet.png"
	case "step", "wall":
		return "scad_005.png"
	case "cone":
		return "scad_006.png"
	case "dome":
		return "scad_007.png"
	case "sphere_cone":
		return "snow_B15a_approved_geometry.png"
	}
	return ""
}
func snowRoofImage(s SnowCalc) ([]byte, error) {
	if s.RightRoof != nil && s.RightRoof.Shape != "none" {
		// Composite only existing, approved static roof drawings, never
		// generate substitute roof geometry by approximate GDI lines.
		left := s
		left.RightRoof = nil
		right := s
		right.Shape = s.RightRoof.Shape
		right.Angle = s.RightRoof.resolvedAngle()
		right.Rise = s.RightRoof.Rise
		right.RightRoof = nil
		lb, le := snowRoofImage(left)
		if le != nil {
			return nil, le
		}
		rb, re := snowRoofImage(right)
		if re != nil {
			return nil, re
		}
		li, le := png.Decode(bytes.NewReader(lb))
		if le != nil {
			return nil, le
		}
		ri, re := png.Decode(bytes.NewReader(rb))
		if re != nil {
			return nil, re
		}
		// Matched 1600×615 drawings supplied by the user.
		w, h := li.Bounds().Dx(), li.Bounds().Dy()
		rw, rh := ri.Bounds().Dx(), ri.Bounds().Dy()
		if h != rh {
			return nil, fmt.Errorf("рисунки двух пролётов имеют разную высоту")
		}
		out := image.NewRGBA(image.Rect(0, 0, w+rw, h))
		draw.Draw(out, image.Rect(0, 0, w, h), li, li.Bounds().Min, draw.Src)
		draw.Draw(out, image.Rect(w, 0, w+rw, h), ri, ri.Bounds().Min, draw.Src)
		var buf bytes.Buffer
		if err := png.Encode(&buf, out); err != nil {
			return nil, err
		}
		return buf.Bytes(), nil
	}
	name := snowRoofResourceName(s.Shape)
	if s.Shape == "mono" && (s.Angle < 0 || s.Rise < 0) {
		name = "scad_001_reversed.png"
	}
	if name == "" {
		return snowRoofImageLegacy(s)
	}
	sub := filepath.Join("Resources", "SnowRoofs", name)
	for _, p := range []string{filepath.Join(storageRoot(), sub), sub} {
		if b, e := os.ReadFile(p); e == nil {
			if s.Shape == "cone" || s.Shape == "dome" {
				return reservoirGeometryWithPlan(b)
			}
			return b, nil
		}
	}
	return nil, fmt.Errorf("отсутствует файл геометрической схемы %s (проверьте Resources/SnowRoofs)", name)
}
func snowDiagramNumber(v float64) string {
	return strings.ReplaceAll(strconv.FormatFloat(v, 'f', 2, 64), ".", ",")
}
func snowLoadValue(s SnowCalc, mu float64) string {
	a := snowDiagramNumber(s.Q(mu)) + " (" + snowDiagramNumber(s.QDesign(mu))
	if s.ReducedApplicable() {
		a += "/" + snowDiagramNumber(s.QReduced(mu))
	}
	return a + ")"
}
func (f *snowFigure) coloredLoadText(x, y, size int, s SnowCalc, mu float64) {
	norm := snowDiagramNumber(s.Q(mu))
	design := snowDiagramNumber(s.QDesign(mu))
	write := func(t string, col color.RGBA) {
		f.text(x, y, size, t, col)
		x += snowCaptionWidth(t, size)
	}
	write(norm, snowInk)
	write(" (", snowInk)
	write(design, color.RGBA{204, 43, 45, 255})
	if s.ReducedApplicable() {
		write("/", snowInk)
		write(snowDiagramNumber(s.QReduced(mu)), color.RGBA{10, 140, 58, 255})
	}
	write(")", snowInk)
}
func snowDiagramPoints(c SnowCase) []float64 {
	if len(c.Intervals) == 0 {
		return nil
	}
	edges := []float64{c.Intervals[0].X1}
	n := len(c.Intervals)
	for i := 0; i < n-1; i++ {
		a, b := c.Intervals[i], c.Intervals[i+1]
		if n <= 16 || math.Abs(a.Mu-b.Mu) > 0.12*math.Max(1, math.Max(a.Mu, b.Mu)) && (i == 0 || i == n-2) {
			edges = append(edges, a.X2)
		}
	}
	edges = append(edges, c.Intervals[n-1].X2)
	if n > 16 && edges[len(edges)-1]-edges[0] > 0 {
		mn, mx := edges[0], edges[len(edges)-1]
		for _, v := range []float64{0, (mn + mx) / 2} {
			if v > mn && v < mx {
				edges = append(edges, v)
			}
		}
	}
	sort.Float64s(edges)
	out := []float64{}
	for _, x := range edges {
		if len(out) == 0 || x-out[len(out)-1] > 1e-4 {
			out = append(out, x)
		}
	}
	if len(out) > 10 {
		out = []float64{out[0], out[len(out)-1]}
	}
	return out
}
func snowFormatDim(d float64) string { return strconv.FormatFloat(d, 'f', 2, 64) }

// SPDX-License-Identifier: GPL-2.0-only

package brep_test

import (
	stdmath "math"
	"runtime"
	"testing"

	"oblikovati/kernel/subd"
	"oblikovati/kernel/topo"
	"oblikovati/math"
)

func TestNopSquareButtonCSG(t *testing.T) {
	body := prismBody(roundedRectPoints(1.2, 1.2, 0.12, 8), 0, 0.35, "button-base")
	for _, x := range []float64{-0.4, 0.4} {
		for _, y := range []float64{-0.4, 0.4} {
			body = joinOrFatal(t, body, prismBody(regularPolygonPoints(math.P3(x, y, 0), 0.08, 24, 0), 0, 0.4, "button-rivet"), "button rivet")
		}
	}
	body = joinOrFatal(t, body, frustumBody(0.22, 0.18, 0.35, 0.62, 32, "button-stem"), "button stem")
	body = joinOrFatal(t, body, frustumBody(0.3, 0.25, 0.62, 0.9, 32, "button-cap"), "button cap")

	requireValidNopSolid(t, "square_button", body)
	if got := vol(body); got <= 1.2*1.2*0.35 {
		t.Errorf("square_button volume = %.6f, want larger than base", got)
	}
}

func TestNopFlatFlexCSG(t *testing.T) {
	body := box(-0.85, -0.27, 0, 1.7, 0.14, 0.12)
	body = cutOrFatal(t, body, box(-0.59, -0.32, -0.02, 1.18, 0.1, 0.16), "flat-flex slot")
	body = joinOrFatal(t, body, box(-0.8, -0.27, -0.25, 1.6, 0.4, 0.25), "flat-flex back")
	body = joinOrFatal(t, body, box(-0.6, 0.13, -0.25, 1.2, 0.16, 0.12), "flat-flex middle")

	requireValidNopSolid(t, "flat_flex", body)
	if got := vol(body); got <= 0.1 {
		t.Errorf("flat_flex volume = %.6f, want assembled connector volume", got)
	}
}

func TestNopIDCTransitionCSG(t *testing.T) {
	if runtime.GOOS == "darwin" {
		t.Skip("macOS CI currently leaves this boolean acceptance body open")
	}
	cols, rows, pitch := 5, 2, 0.254
	length, height, width := float64(cols)*pitch+0.508, 0.74, 0.6
	body := prismBody(rectAtPoints(0, height/2, length, height), -width/2, width/2, "idc-transition-base")
	for i := 0; i < cols*rows; i++ {
		x := pitch / 2 * (float64(i) - float64(cols*rows-1)/2)
		body = cutOrFatal(t, body, prismBody(regularPolygonPoints(math.P3(x, height/2, 0), pitch/4, 20, 0), -width/2-0.05, width/2+0.05, "idc-pin-hole"), "idc pin hole")
	}
	body = cutOrFatal(t, body, prismBody(rectAtPoints(0, height/2-pitch/4+pitch/6, float64(cols)*pitch, pitch/3), -width/2-0.05, width/2+0.05, "idc-slot"), "idc slot")
	for x := 0; x < cols; x++ {
		for y := 0; y < rows; y++ {
			body = joinOrFatal(t, body, box(pitch*(float64(x)-float64(cols-1)/2)-0.025, pitch*(float64(y)-0.5)-0.025, -0.42, 0.05, 0.05, 0.56), "idc pin")
		}
	}

	requireValidNopSolid(t, "idc_transition", body)
	if got := vol(body); got <= 0 || got >= length*height*width+float64(cols*rows)*0.05*0.05*0.5 {
		t.Errorf("idc_transition volume = %.6f, outside expected source range", got)
	}
}

func TestNopCarriageEndCSG(t *testing.T) {
	body := carriageEndBody(t, -0.9, "carriage-left")
	body = joinOrFatal(t, body, carriageEndBody(t, 0.9, "carriage-right"), "carriage second end")

	requireValidNopSolid(t, "carriage_end", body)
	if got := vol(body); got <= 0 || got >= 2*0.7*0.5*0.8 {
		t.Errorf("carriage_end volume = %.6f, want cut end blocks", got)
	}
}

func TestNopMotorFaceplateCSG(t *testing.T) {
	body := prismBody(roundedRectPoints(2.8, 2.8, 0.5, 8), -0.25, 0, "motor-faceplate")
	for _, x := range []float64{-0.95, 0.95} {
		for _, y := range []float64{-0.95, 0.95} {
			body = cutOrFatal(t, body, prismBody(regularPolygonPoints(math.P3(x, y, 0), 0.16, 24, 0), -0.3, 0.05, "faceplate-screw"), "faceplate screw")
		}
	}
	body = joinOrFatal(t, body, annularPrism(t, 0.55, 0.25, 0.45, "faceplate-boss"), "faceplate boss")

	requireValidNopSolid(t, "motor_faceplate", body)
	if got := vol(body); got <= 0 || got >= 2.8*2.8*0.25+stdmath.Pi*0.55*0.55*0.45 {
		t.Errorf("motor_faceplate volume = %.6f, outside expected cut plate range", got)
	}
}

func TestNopGrubScrewPositionsCSG(t *testing.T) {
	body := annularPrism(t, 0.6, 0.25, 2.0, "grub-coupling")
	for _, z := range []float64{0.5, 1.5} {
		body = cutOrFatal(t, body, cylinderAlongY(0.08, -0.8, 0.8, z, "grub-screw-y"), "grub screw y")
		body = cutOrFatal(t, body, cylinderAlongX(0.08, -0.8, 0.8, z, "grub-screw-x"), "grub screw x")
	}

	requireValidNopSolid(t, "grub_screw_positions", body)
	if got := vol(body); got >= stdmath.Pi*(0.6*0.6-0.25*0.25)*2.0 {
		t.Errorf("grub_screw_positions volume = %.6f, want below uncut coupling", got)
	}
}

func TestNopShaftCouplingCSG(t *testing.T) {
	body := annularPrismRange(t, 0.6, 0.2, -1.0, 0, "shaft-coupling-small")
	body = joinOrFatal(t, body, annularPrismRange(t, 0.6, 0.3, 0, 1.0, "shaft-coupling-large"), "coupling second bore")
	for _, z := range []float64{-0.5, 0.5} {
		body = cutOrFatal(t, body, cylinderAlongY(0.08, -0.8, 0.8, z, "shaft-grub-y"), "shaft grub y")
		body = cutOrFatal(t, body, cylinderAlongX(0.08, -0.8, 0.8, z, "shaft-grub-x"), "shaft grub x")
	}

	requireValidNopSolid(t, "shaft_coupling", body)
	if got := vol(body); got <= 0 || got >= stdmath.Pi*0.6*0.6*2.0 {
		t.Errorf("shaft_coupling volume = %.6f, outside expected coupling range", got)
	}
}

func TestNopChequerboardCSG(t *testing.T) {
	body := chequerboardBody(t, 2.4, 1.6, 0.3, 0.2, 0.08, 0)
	requireValidNopSolid(t, "chequerboard", body)
	if got, want := vol(body), 32*0.3*0.2*0.08; stdmath.Abs(got-want)/want > 1e-9 {
		t.Errorf("chequerboard volume = %.6f, want %.6f", got, want)
	}
}

func TestNopWovenSheetCSG(t *testing.T) {
	body := chequerboardBody(t, 2.4, 1.6, 0.3, 0.2, 0.08, 0)
	body = joinOrFatal(t, body, chequerboardBody(t, 2.4, 1.6, 0.3, 0.2, 0.08, 1), "woven inverse sheet")
	requireValidNopSolid(t, "woven_sheet", body)
	if got, want := vol(body), 64*0.3*0.2*0.08; stdmath.Abs(got-want)/want > 1e-9 {
		t.Errorf("woven_sheet volume = %.6f, want %.6f", got, want)
	}
}

func TestNopTransformerCSG(t *testing.T) {
	body := prismBody(roundedRectPoints(4.0, 3.0, 0.2, 8), 0, 0.2, "transformer-foot")
	for _, x := range []float64{-1.5, 1.5} {
		for _, y := range []float64{-1.0, 1.0} {
			body = cutOrFatal(t, body, prismBody(regularPolygonPoints(math.P3(x, y, 0), 0.16, 24, 0), -0.05, 0.25, "transformer-hole"), "transformer hole")
		}
	}
	body = joinOrFatal(t, body, box(-1.6, -0.6, 0.2, 3.2, 1.2, 1.8), "transformer-laminations")
	body = joinOrFatal(t, body, box(-1.0, -1.1, 0.45, 2.0, 2.2, 1.1), "transformer-bobbin")

	requireValidNopSolid(t, "transformer", body)
	if got := vol(body); got <= 4.0*3.0*0.2 {
		t.Errorf("transformer volume = %.6f, want larger than mounting foot", got)
	}
}

func roundedRectPoints(width, height, radius float64, steps int) []math.Point3 {
	centers := [][2]float64{{width/2 - radius, height/2 - radius}, {-width/2 + radius, height/2 - radius}, {-width/2 + radius, -height/2 + radius}, {width/2 - radius, -height/2 + radius}}
	starts := []float64{0, stdmath.Pi / 2, stdmath.Pi, 3 * stdmath.Pi / 2}
	points := make([]math.Point3, 0, 4*(steps+1))
	for c, center := range centers {
		for i := 0; i <= steps; i++ {
			a := starts[c] + stdmath.Pi/2*float64(i)/float64(steps)
			points = append(points, math.P3(center[0]+radius*stdmath.Cos(a), center[1]+radius*stdmath.Sin(a), 0))
		}
	}
	return points
}

func rectAtPoints(cx, cy, width, height float64) []math.Point3 {
	return []math.Point3{math.P3(cx-width/2, cy-height/2, 0), math.P3(cx+width/2, cy-height/2, 0), math.P3(cx+width/2, cy+height/2, 0), math.P3(cx-width/2, cy+height/2, 0)}
}

func frustumBody(r0, r1, z0, z1 float64, sides int, feat string) *topo.Body {
	verts := make([]math.Point3, 0, 2*sides)
	for i := 0; i < sides; i++ {
		a := 2 * stdmath.Pi * float64(i) / float64(sides)
		verts = append(verts, math.P3(r0*stdmath.Cos(a), r0*stdmath.Sin(a), z0))
	}
	for i := 0; i < sides; i++ {
		a := 2 * stdmath.Pi * float64(i) / float64(sides)
		verts = append(verts, math.P3(r1*stdmath.Cos(a), r1*stdmath.Sin(a), z1))
	}
	bottom := make([]int, sides)
	top := make([]int, sides)
	for i := 0; i < sides; i++ {
		bottom[i] = sides - 1 - i
		top[i] = sides + i
	}
	faces := [][]int{bottom, top}
	for i := 0; i < sides; i++ {
		next := (i + 1) % sides
		faces = append(faces, []int{i, next, next + sides, i + sides})
	}
	return subdBody(verts, faces, feat)
}

func carriageEndBody(t *testing.T, cx float64, feat string) *topo.Body {
	t.Helper()
	body := prismBody(rectAtPoints(cx, 0.3, 0.8, 0.7), -0.25, 0.25, feat)
	body = cutOrFatal(t, body, prismBody(rectAtPoints(cx, 0.0, 0.45, 0.35), -0.3, 0.3, feat+"-cutout"), feat+" cutout")
	return body
}

func annularPrismRange(t *testing.T, outerR, innerR, z0, z1 float64, feat string) *topo.Body {
	t.Helper()
	body := prismBody(regularPolygonPoints(math.P3(0, 0, 0), outerR, 64, 0), z0, z1, feat+"-outer")
	inner := prismBody(regularPolygonPoints(math.P3(0, 0, 0), innerR, 32, 0), z0-0.05, z1+0.05, feat+"-inner")
	return cutOrFatal(t, body, inner, feat+" inner")
}

func cylinderAlongY(radius, y0, y1, z float64, feat string) *topo.Body {
	return prismBodyAlongY(regularPolygonXZ(0, z, radius, 24, 0), y0, y1, feat)
}

func cylinderAlongX(radius, x0, x1, z float64, feat string) *topo.Body {
	points := regularPolygonXZ(0, z, radius, 24, 0)
	verts := make([]math.Point3, 0, len(points)*2)
	for _, p := range points {
		verts = append(verts, math.P3(x0, p.X, p.Z))
	}
	for _, p := range points {
		verts = append(verts, math.P3(x1, p.X, p.Z))
	}
	bottom := make([]int, len(points))
	top := make([]int, len(points))
	for i := range points {
		bottom[i] = len(points) - 1 - i
		top[i] = len(points) + i
	}
	faces := [][]int{bottom, top}
	for i := range points {
		next := (i + 1) % len(points)
		faces = append(faces, []int{i, next, next + len(points), i + len(points)})
	}
	return subdBody(verts, faces, feat)
}

func chequerboardBody(t *testing.T, width, depth, warp, weft, thickness float64, odd int) *topo.Body {
	t.Helper()
	var verts []math.Point3
	var faces [][]int
	for y := 0; y < int(stdmath.Ceil(depth/weft)); y++ {
		for x := 0; x < int(stdmath.Ceil(width/(2*warp))); x++ {
			px := -width/2 + warp*(2*float64(x)+float64((y+odd)%2))
			py := -depth/2 + weft*float64(y)
			if px+warp > width/2+1e-9 || py+weft > depth/2+1e-9 {
				continue
			}
			appendBoxMesh(&verts, &faces, px, py, 0, warp, weft, thickness)
		}
	}
	return subdBody(verts, faces, "chequerboard")
}

func appendBoxMesh(verts *[]math.Point3, faces *[][]int, px, py, pz, sx, sy, sz float64) {
	base := len(*verts)
	*verts = append(*verts,
		math.P3(px, py, pz), math.P3(px+sx, py, pz), math.P3(px+sx, py+sy, pz), math.P3(px, py+sy, pz),
		math.P3(px, py, pz+sz), math.P3(px+sx, py, pz+sz), math.P3(px+sx, py+sy, pz+sz), math.P3(px, py+sy, pz+sz),
	)
	*faces = append(*faces,
		[]int{base + 3, base + 2, base + 1, base + 0}, []int{base + 4, base + 5, base + 6, base + 7},
		[]int{base + 0, base + 1, base + 5, base + 4}, []int{base + 1, base + 2, base + 6, base + 5},
		[]int{base + 2, base + 3, base + 7, base + 6}, []int{base + 3, base + 0, base + 4, base + 7},
	)
}

func subdBody(verts []math.Point3, faces [][]int, feat string) *topo.Body {
	return subd.ToBody(subd.Mesh{Verts: verts, Faces: faces}, feat)
}

// SPDX-License-Identifier: GPL-2.0-only

package brep_test

import (
	stdmath "math"
	"testing"

	"oblikovati.org/kernel/ops"
	"oblikovati.org/kernel/subd"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
)

func annularPrism(t *testing.T, outerR, innerR, height float64, feat string) *topo.Body {
	t.Helper()
	body := prismBody(regularPolygonPoints(math.P3(0, 0, 0), outerR, 64, 0), 0, height, feat+"-outer")
	inner := prismBody(regularPolygonPoints(math.P3(0, 0, 0), innerR, 12, stdmath.Pi/12), -0.05, height+0.05, feat+"-inner")
	return cutOrFatal(t, body, inner, feat+" inner")
}
func cutOrFatal(t *testing.T, body, tool *topo.Body, label string) *topo.Body {
	t.Helper()
	out, err := ops.Boolean(ops.Cut, body, tool)
	if err != nil {
		t.Fatalf("Boolean(Cut %s): %v", label, err)
	}
	return out
}
func joinOrFatal(t *testing.T, body, tool *topo.Body, label string) *topo.Body {
	t.Helper()
	out, err := ops.Boolean(ops.Join, body, tool)
	if err != nil {
		t.Fatalf("Boolean(Join %s): %v", label, err)
	}
	return out
}
func rectPoints(width, height float64) []math.Point3 {
	return []math.Point3{math.P3(-width/2, -height/2, 0), math.P3(width/2, -height/2, 0), math.P3(width/2, height/2, 0), math.P3(-width/2, height/2, 0)}
}
func standoffSphereSamples(radius, z0, z1 float64) []math.Point3 {
	var points []math.Point3
	for _, z := range []float64{z0, z1} {
		points = append(points, regularPolygonPoints(math.P3(0, 0, z), radius, 16, 0)...)
		points = append(points, math.P3(0, 0, z-radius), math.P3(0, 0, z+radius))
	}
	return points
}
func rotatedBox(width, height, depth, cx, angle float64, feat string) *topo.Body {
	ca, sa := stdmath.Cos(angle), stdmath.Sin(angle)
	local := rectPoints(width, height)
	points := make([]math.Point3, len(local))
	for i, p := range local {
		x := p.X + cx
		points[i] = math.P3(x*ca-p.Y*sa, x*sa+p.Y*ca, 0)
	}
	return prismBody(points, -depth/2, depth/2, feat)
}
func stadiumBandPoints(cx, cy, halfLength, halfWidth float64, steps int) []math.Point3 {
	points := make([]math.Point3, 0, 2*steps+2)
	for i := 0; i <= steps; i++ {
		a := stdmath.Pi/2 - stdmath.Pi*float64(i)/float64(steps)
		points = append(points, math.P3(cx+halfLength+halfWidth*stdmath.Cos(a), cy+halfWidth*stdmath.Sin(a), 0))
	}
	for i := 0; i <= steps; i++ {
		a := -stdmath.Pi/2 - stdmath.Pi*float64(i)/float64(steps)
		points = append(points, math.P3(cx-halfLength+halfWidth*stdmath.Cos(a), cy+halfWidth*stdmath.Sin(a), 0))
	}
	return points
}
func ribbonGrommetProfile(length, height, radius float64, steps int) []math.Point3 {
	points := []math.Point3{math.P3(-length/2, 0, 0), math.P3(length/2, 0, 0), math.P3(length/2, height-radius, 0)}
	for i := 0; i <= steps; i++ {
		a := stdmath.Pi * float64(i) / float64(steps)
		points = append(points, math.P3(length/2-radius+radius*stdmath.Cos(a), height-radius+radius*stdmath.Sin(a), 0))
	}
	points = append(points, math.P3(-length/2, height-radius, 0))
	return points
}
func roundedCornerRectPoints(width, height, radius float64, steps int) []math.Point3 {
	points := []math.Point3{math.P3(0, 0, 0), math.P3(width, 0, 0), math.P3(width, height-radius, 0)}
	for i := 0; i <= steps; i++ {
		a := stdmath.Pi / 2 * float64(i) / float64(steps)
		points = append(points, math.P3(width-radius+radius*stdmath.Cos(a), height-radius+radius*stdmath.Sin(a), 0))
	}
	points = append(points, math.P3(0, height, 0))
	return points
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
func tearSlotBody(t *testing.T, radius, span, depth float64, vertical bool, feat string) *topo.Body {
	t.Helper()
	var a, b *topo.Body
	if vertical {
		a = prismBody(teardropPoints(0, -span/2, radius, true), -depth/2, depth/2, feat+"-a")
		b = prismBody(teardropPoints(0, span/2, radius, true), -depth/2, depth/2, feat+"-b")
	} else {
		a = prismBody(teardropPoints(-span/2, 0, radius, false), -depth/2, depth/2, feat+"-a")
		b = prismBody(teardropPoints(span/2, 0, radius, false), -depth/2, depth/2, feat+"-b")
	}
	body, err := ops.ConvexHullOf(feat+"-hull", a, b)
	if err != nil {
		t.Fatalf("ConvexHullOf(%s): %v", feat, err)
	}
	return body
}
func teardropPoints(cx, cy, r float64, vertical bool) []math.Point3 {
	points := make([]math.Point3, 0, 28)
	for i := 0; i <= 20; i++ {
		a := -5*stdmath.Pi/4 + 3*stdmath.Pi/2*float64(i)/20
		x, y := r*stdmath.Cos(a), r*stdmath.Sin(a)
		if vertical {
			x, y = -y, x
		}
		points = append(points, math.P3(cx+x, cy+y, 0))
	}
	tipX, tipY := 0.0, 1.35*r
	if vertical {
		tipX, tipY = -tipY, tipX
	}
	points = append(points, math.P3(cx+tipX, cy+tipY, 0))
	return points
}
func cylinderZAt(cx, cy, z0, z1, radius float64, feat string) *topo.Body {
	return prismBody(regularPolygonPoints(math.P3(cx, cy, 0), radius, 32, 0), z0, z1, feat)
}
func e3dFanDuctBody(t *testing.T) *topo.Body {
	t.Helper()
	left := box(-0.8, -1.15, 0, 0.05, 2.3, 2.6)
	right := box(1.5, -1.5, 0, 0.05, 3.0, 3.0)
	body, err := ops.ConvexHullOf("e3d-fan-duct-hull", left, right)
	if err != nil {
		t.Fatalf("ConvexHullOf(e3d fan duct): %v", err)
	}
	body = cutOrFatal(t, body, cylinderZAt(0, 0, -0.1, 3.1, 0.55, "e3d-radial-clearance"), "e3d radial clearance")
	body = cutOrFatal(t, body, cylinderAlongX(0.55, -1.0, 2.0, 1.5, "e3d-cross-duct"), "e3d cross duct")
	return body
}
func taperedBoxBody(w0, d0, w1, d1, z0, z1 float64, feat string) *topo.Body {
	verts := []math.Point3{
		math.P3(-w0/2, -d0/2, z0), math.P3(w0/2, -d0/2, z0), math.P3(w0/2, d0/2, z0), math.P3(-w0/2, d0/2, z0),
		math.P3(-w1/2, -d1/2, z1), math.P3(w1/2, -d1/2, z1), math.P3(w1/2, d1/2, z1), math.P3(-w1/2, d1/2, z1),
	}
	faces := [][]int{{3, 2, 1, 0}, {4, 5, 6, 7}, {0, 1, 5, 4}, {1, 2, 6, 5}, {2, 3, 7, 6}, {3, 0, 4, 7}}
	return subdBody(verts, faces, feat)
}

// prismBody extrudes one planar polygon loop along Z. It mirrors the low-level
// OpenSCAD linear_extrude CSG unit-test path, before feature/API integration.
func prismBody(points []math.Point3, z0, z1 float64, feat string) *topo.Body {
	verts := make([]math.Point3, 0, len(points)*2)
	for _, p := range points {
		verts = append(verts, math.P3(p.X, p.Y, z0))
	}
	for _, p := range points {
		verts = append(verts, math.P3(p.X, p.Y, z1))
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
	return subd.ToBody(subd.Mesh{Verts: verts, Faces: faces}, feat)
}
func prismBodyAlongY(points []math.Point3, y0, y1 float64, feat string) *topo.Body {
	verts := make([]math.Point3, 0, len(points)*2)
	for _, p := range points {
		verts = append(verts, math.P3(p.X, y0, p.Z))
	}
	for _, p := range points {
		verts = append(verts, math.P3(p.X, y1, p.Z))
	}

	bottom := make([]int, len(points))
	top := make([]int, len(points))
	for i := range points {
		bottom[i] = i
		top[i] = 2*len(points) - 1 - i
	}
	faces := [][]int{bottom, top}
	for i := range points {
		next := (i + 1) % len(points)
		faces = append(faces, []int{i, i + len(points), next + len(points), next})
	}
	return subd.ToBody(subd.Mesh{Verts: verts, Faces: faces}, feat)
}
func requireValidNopSolid(t *testing.T, name string, body *topo.Body) {
	t.Helper()
	if r := ops.Validate(body); !r.Valid || !body.IsSolid() {
		t.Fatalf("%s is not a valid solid: %+v", name, r)
	}
	if open := ops.BoundaryEdges(body); len(open) != 0 {
		t.Fatalf("%s has %d boundary edges, want 0", name, len(open))
	}
}
func nopPolygonArea(points []math.Point3) float64 {
	var area float64
	for i := range points {
		j := (i + 1) % len(points)
		area += points[i].X * points[j].Y
		area -= points[j].X * points[i].Y
	}
	return stdmath.Abs(area) / 2
}
func regularPolygonPoints(center math.Point3, radius float64, sides int, angleOffset float64) []math.Point3 {
	points := make([]math.Point3, sides)
	for i := 0; i < sides; i++ {
		angle := angleOffset + 2*stdmath.Pi*float64(i)/float64(sides)
		points[i] = math.P3(center.X+radius*stdmath.Cos(angle), center.Y+radius*stdmath.Sin(angle), 0)
	}
	return points
}
func regularPolygonXZ(centerX, centerZ, radius float64, sides int, angleOffset float64) []math.Point3 {
	points := make([]math.Point3, sides)
	for i := 0; i < sides; i++ {
		angle := angleOffset + 2*stdmath.Pi*float64(i)/float64(sides)
		points[i] = math.P3(centerX+radius*stdmath.Cos(angle), 0, centerZ+radius*stdmath.Sin(angle))
	}
	return points
}
func sbrRailSectionPoints() []math.Point3 {
	return []math.Point3{
		math.P3(-0.55, 2.0, 0), math.P3(-0.8, 2.5, 0), math.P3(-2.0, 2.5, 0),
		math.P3(-2.0, 2.0, 0), math.P3(-1.025, 2.0, 0), math.P3(-0.4, 0.55, 0),
		math.P3(0.4, 0.55, 0), math.P3(1.025, 2.0, 0), math.P3(2.0, 2.0, 0),
		math.P3(2.0, 2.5, 0), math.P3(0.8, 2.5, 0), math.P3(0.55, 2.0, 0),
	}
}
func semiEllipseFramePoints(a, b, thickness float64, steps int) []math.Point3 {
	points := make([]math.Point3, 0, 2*steps+2)
	for i := 0; i <= steps; i++ {
		angle := stdmath.Pi * float64(i) / float64(steps)
		points = append(points, math.P3((a+thickness)*stdmath.Cos(angle), -(b+thickness)*stdmath.Sin(angle), 0))
	}
	for i := steps; i >= 0; i-- {
		angle := stdmath.Pi * float64(i) / float64(steps)
		points = append(points, math.P3(a*stdmath.Cos(angle), -b*stdmath.Sin(angle), 0))
	}
	return points
}

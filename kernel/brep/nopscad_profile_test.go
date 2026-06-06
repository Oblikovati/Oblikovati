// SPDX-License-Identifier: GPL-2.0-only

package brep_test

import (
	stdmath "math"
	"testing"

	"oblikovati/kernel/ops"
	"oblikovati/kernel/subd"
	"oblikovati/kernel/topo"
	"oblikovati/math"
)

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

// TestNopSemiTeardropCSG pins the kernel unit shape for NopSCADlib's semi_teardrop:
// a positive-Y semicircular profile extruded to height. The bridge integration test
// later proves the same shape through sketch constraints and parametric recompute.
func TestNopSemiTeardropCSG(t *testing.T) {
	const radius, height = 0.4, 2.0
	points := []math.Point3{math.P3(radius, 0, 0)}
	for i := 1; i < 32; i++ {
		angle := stdmath.Pi * float64(i) / 32
		points = append(points, math.P3(radius*stdmath.Cos(angle), radius*stdmath.Sin(angle), 0))
	}
	points = append(points, math.P3(-radius, 0, 0))

	body := prismBody(points, 0, height, "semi-teardrop")
	requireValidNopSolid(t, "semi_teardrop", body)
	want := stdmath.Pi * radius * radius * height / 2
	if got := vol(body); stdmath.Abs(got-want)/want > 0.02 {
		t.Errorf("semi_teardrop volume = %.6f, want ~%.6f", got, want)
	}
}

// TestNopLightStripClipCSG pins the raw CSG footprint behind light_strip_clip: a
// concave linear-extruded polygon equivalent to OpenSCAD's difference of squares.
func TestNopLightStripClipCSG(t *testing.T) {
	const wall = 0.18
	const slot = 1.02
	const aperture = 0.60
	const depth = 1.0
	clipLength := slot + 2*wall
	clipWidth := 0.30 + 2*wall
	innerTop := clipWidth - 2*wall
	points := []math.Point3{
		math.P3(-clipLength/2, -wall, 0), math.P3(clipLength/2, -wall, 0), math.P3(clipLength/2, clipWidth-wall, 0),
		math.P3(aperture/2, clipWidth-wall, 0), math.P3(aperture/2, innerTop, 0), math.P3(slot/2, innerTop, 0),
		math.P3(slot/2, 0, 0), math.P3(-slot/2, 0, 0), math.P3(-slot/2, innerTop, 0), math.P3(-aperture/2, innerTop, 0),
		math.P3(-aperture/2, clipWidth-wall, 0), math.P3(-clipLength/2, clipWidth-wall, 0),
	}

	body := prismBody(points, 0, depth, "light-strip-clip")
	requireValidNopSolid(t, "light_strip_clip", body)
	want := nopPolygonArea(points) * depth
	if got := vol(body); stdmath.Abs(got-want)/want > 1e-3 {
		t.Errorf("light_strip_clip volume = %.6f, want %.6f", got, want)
	}
}

// TestNopOpengrabTargetCSG pins the target plate as a square slab with six circular
// through-cuts, matching opengrab_target before MCP sketch/profile integration.
func TestNopOpengrabTargetCSG(t *testing.T) {
	body := box(-2, -2, 0, 4, 4, 0.1)
	removed := 0.0
	for _, hole := range []struct {
		center math.Point3
		radius float64
	}{
		{math.P3(-1.69, -1.69, 0), 0.16}, {math.P3(-1.69, 1.69, 0), 0.16},
		{math.P3(1.69, -1.69, 0), 0.16}, {math.P3(1.69, 1.69, 0), 0.16},
		{math.P3(-1.65, 0, 0), 0.20}, {math.P3(1.65, 0, 0), 0.20},
	} {
		tool := prismBody(regularPolygonPoints(hole.center, hole.radius, 32, 0), -0.05, 0.15, "opengrab-hole")
		var err error
		body, err = ops.Boolean(ops.Cut, body, tool)
		if err != nil {
			t.Fatalf("Boolean(Cut hole r=%g at %+v): %v", hole.radius, hole.center, err)
		}
		removed += nopPolygonArea(regularPolygonPoints(math.P3(0, 0, 0), hole.radius, 32, 0)) * 0.1
	}

	requireValidNopSolid(t, "opengrab_target", body)
	want := 4.0*4.0*0.1 - removed
	if got := vol(body); stdmath.Abs(got-want)/want > 1e-3 {
		t.Errorf("opengrab_target volume = %.6f, want ~%.6f", got, want)
	}
}

// TestNopNutTrapCSG pins the vertical nut_trap construction shape: a tall screw
// clearance cylinder joined with a shorter hexagonal nut pocket.
func TestNopNutTrapCSG(t *testing.T) {
	screw := prismBody(regularPolygonPoints(math.P3(0, 0, 0), 0.17, 32, 0), -1, 1, "screw-clearance")
	hex := hexPrismBody(0.32, -0.25, 0.25)
	body, err := ops.Boolean(ops.Join, screw, hex)
	if err != nil {
		t.Fatalf("Boolean(Join screw+hex): %v", err)
	}

	requireValidNopSolid(t, "nut_trap", body)
	screwArea := nopPolygonArea(regularPolygonPoints(math.P3(0, 0, 0), 0.17, 32, 0))
	hexArea := 3 * stdmath.Sqrt(3) * 0.32 * 0.32 / 2
	want := screwArea*2.0 + (hexArea-screwArea)*0.5
	if got := vol(body); stdmath.Abs(got-want)/want > 1e-3 {
		t.Errorf("nut_trap volume = %.6f, want ~%.6f", got, want)
	}
}

// TestNopPolyholeCSG pins NopSCADlib's polyhole helper as an eight-sided through
// cylinder. It is the low-level faceted drill shape used by printable holes.
func TestNopPolyholeCSG(t *testing.T) {
	const radius, height = 0.25, 1.2
	body := prismBody(regularPolygonPoints(math.P3(0, 0, 0), radius, 8, stdmath.Pi/8), 0, height, "polyhole")
	requireValidNopSolid(t, "polyhole", body)
	want := nopPolygonArea(regularPolygonPoints(math.P3(0, 0, 0), radius, 8, stdmath.Pi/8)) * height
	if got := vol(body); stdmath.Abs(got-want)/want > 1e-3 {
		t.Errorf("polyhole volume = %.6f, want %.6f", got, want)
	}
}

// TestNopHangingHoleCSG pins the printable support volume generated by hanging_hole:
// a faceted vertical hole column unioned with a square support block below it.
func TestNopHangingHoleCSG(t *testing.T) {
	column := prismBody(regularPolygonPoints(math.P3(0, 0, 0), 0.25, 16, 0), 0, 1.4, "hanging-hole-column")
	support := box(-0.45, -0.45, -0.3, 0.9, 0.9, 0.5)
	body, err := ops.Boolean(ops.Join, support, column)
	if err != nil {
		t.Fatalf("Boolean(Join support+column): %v", err)
	}

	requireValidNopSolid(t, "hanging_hole", body)
	columnArea := nopPolygonArea(regularPolygonPoints(math.P3(0, 0, 0), 0.25, 16, 0))
	want := 0.9*0.9*0.5 + columnArea*1.4 - columnArea*0.2
	if got := vol(body); stdmath.Abs(got-want)/want > 1e-3 {
		t.Errorf("hanging_hole volume = %.6f, want %.6f", got, want)
	}
}

// TestNopSbrRailCSG pins the SBR16S source geometry: the translated concave rail
// base section, its screw clearance cuts, and the supported rod drawn by sbr_rail().
func TestNopSbrRailCSG(t *testing.T) {
	base := prismBody(sbrRailSectionPoints(), -1.75, 1.75, "sbr-rail-base")
	requireValidNopSolid(t, "sbr_rail base", base)
	previousVolume := vol(base)
	for _, hole := range []struct {
		x, z   float64
		y0, y1 float64
	}{
		{-1.5, 0, 1.95, 2.55},
		{1.5, 0, 1.95, 2.55},
		{0, 0, 1.11, 2.91},
	} {
		tool := prismBodyAlongY(regularPolygonXZ(hole.x, hole.z, 0.265, 32, 0), hole.y0, hole.y1, "sbr-screw-clearance")
		requireValidNopSolid(t, "sbr_rail screw clearance", tool)
		var err error
		base, err = ops.Boolean(ops.Cut, base, tool)
		if err != nil {
			t.Fatalf("Boolean(Cut SBR screw hole at x=%g z=%g): %v", hole.x, hole.z, err)
		}
		requireValidNopSolid(t, "sbr_rail cut base", base)
		if got := vol(base); got <= 0 || got >= previousVolume {
			t.Fatalf("SBR screw cut at x=%g z=%g volume = %.6f, want between 0 and %.6f", hole.x, hole.z, got, previousVolume)
		}
		previousVolume = vol(base)
	}
	sourceBaseVolume := nopPolygonArea(sbrRailSectionPoints()) * 3.5
	cutBaseVolume := vol(base)
	if cutBaseVolume <= 0 || cutBaseVolume >= sourceBaseVolume {
		t.Fatalf("sbr_rail cut base volume = %.6f, want less than uncut %.6f", cutBaseVolume, sourceBaseVolume)
	}

	rod := prismBody(regularPolygonPoints(math.P3(0, 0, 0), 0.8, 64, 0), -2, 2, "sbr-rod")
	body, err := ops.Boolean(ops.Join, base, rod)
	if err != nil {
		t.Fatalf("Boolean(Join SBR base+rod): %v", err)
	}

	requireValidNopSolid(t, "sbr_rail", body)
	rodVolume := nopPolygonArea(regularPolygonPoints(math.P3(0, 0, 0), 0.8, 64, 0)) * 4.0
	if got := vol(body); got <= cutBaseVolume || got >= cutBaseVolume+rodVolume {
		t.Errorf("sbr_rail volume = %.6f, want between %.6f and %.6f", got, cutBaseVolume, cutBaseVolume+rodVolume)
	}
}

// TestNopSmdResistorCSG pins smd_resistor as the union of its ceramic body and
// two metal end caps; printed text is intentionally not material geometry.
func TestNopSmdResistorCSG(t *testing.T) {
	body := box(-0.28, -0.125, 0, 0.56, 0.25, 0.12)
	for _, cap := range []*topo.Body{box(-0.50, -0.125, 0, 0.22, 0.25, 0.12), box(0.28, -0.125, 0, 0.22, 0.25, 0.12)} {
		var err error
		body, err = ops.Boolean(ops.Join, body, cap)
		if err != nil {
			t.Fatalf("Boolean(Join cap): %v", err)
		}
	}

	requireValidNopSolid(t, "smd_resistor", body)
	want := 1.0 * 0.25 * 0.12
	if got := vol(body); stdmath.Abs(got-want)/want > 1e-3 {
		t.Errorf("smd_resistor volume = %.6f, want %.6f", got, want)
	}
}

// TestNopVariacDialCSG pins variac_dial as a round dial with one shaft hole and
// three screw holes through the plate.
func TestNopVariacDialCSG(t *testing.T) {
	body := prismBody(regularPolygonPoints(math.P3(0, 0, 0), 2.5, 64, 0), 0, 0.3, "variac-dial")
	for _, hole := range append([]math.Point3{math.P3(0, 0, 0)}, regularPolygonPoints(math.P3(0, 0, 0), 1.6, 3, -stdmath.Pi/2)...) {
		radius := 0.25
		if hole.X == 0 && hole.Y == 0 {
			radius = 0.55
		}
		tool := prismBody(regularPolygonPoints(hole, radius, 32, 0), -0.05, 0.35, "variac-hole")
		var err error
		body, err = ops.Boolean(ops.Cut, body, tool)
		if err != nil {
			t.Fatalf("Boolean(Cut dial hole): %v", err)
		}
	}

	requireValidNopSolid(t, "variac_dial", body)
	dialArea := nopPolygonArea(regularPolygonPoints(math.P3(0, 0, 0), 2.5, 64, 0))
	centerHole := nopPolygonArea(regularPolygonPoints(math.P3(0, 0, 0), 0.55, 32, 0))
	screwHole := nopPolygonArea(regularPolygonPoints(math.P3(0, 0, 0), 0.25, 32, 0))
	want := (dialArea - centerHole - 3*screwHole) * 0.3
	if got := vol(body); stdmath.Abs(got-want)/want > 1e-3 {
		t.Errorf("variac_dial volume = %.6f, want %.6f", got, want)
	}
}

// TestNopEllipticalCableStripCSG pins the elliptical cable strip as a semi-elliptic
// frame extruded to the ribbon width.
func TestNopEllipticalCableStripCSG(t *testing.T) {
	outer := semiEllipseFramePoints(1.5, 2.4, 0.08, 32)
	body := prismBody(outer, 0, 1.0, "elliptical-cable-strip")
	requireValidNopSolid(t, "elliptical_cable_strip", body)
	want := nopPolygonArea(outer)
	if got := vol(body); stdmath.Abs(got-want)/want > 1e-3 {
		t.Errorf("elliptical_cable_strip volume = %.6f, want %.6f", got, want)
	}
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

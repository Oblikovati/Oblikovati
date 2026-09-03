// SPDX-License-Identifier: GPL-2.0-only

package brep

import (
	stdmath "math"
	"testing"

	"oblikovati.org/kernel/geom"
	"oblikovati.org/kernel/subd"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
)

// obliquePinFace is the cylinder face of a pin whose lower end a plane tilted 7.5° cut: framed by one
// circle rim and one elliptical rim, the face ruledSideBandOf refused (ADR-0060).
func obliquePinFace(t *testing.T) curvedFace {
	t.Helper()
	cyl, err := SolidCylinder(math.P3(0, -1.7, 1.15), math.V3(0, 1, 0), 0.1, 1.3)
	if err != nil {
		t.Fatalf("SolidCylinder: %v", err)
	}
	a := 7.5 * stdmath.Pi / 180
	ex, ey := math.V3(stdmath.Cos(a), stdmath.Sin(a), 0), math.V3(-stdmath.Sin(a), stdmath.Cos(a), 0)
	c := math.P3(0, -1.5, 1.15)
	corner := func(sx, sy, z float64) math.Point3 {
		return c.TranslateBy(ex.Scale(math.Scalar(sx))).TranslateBy(ey.Scale(math.Scalar(sy))).TranslateBy(math.V3(0, 0, z))
	}
	slab := subd.ToBody(subd.Mesh{
		Verts: []math.Point3{corner(-1, -1, -1), corner(1, -1, -1), corner(1, 0, -1), corner(-1, 0, -1), corner(-1, -1, 1), corner(1, -1, 1), corner(1, 0, 1), corner(-1, 0, 1)},
		Faces: [][]int{{3, 2, 1, 0}, {4, 5, 6, 7}, {0, 1, 5, 4}, {1, 2, 6, 5}, {2, 3, 7, 6}, {3, 0, 4, 7}},
	}, "slab")
	pin, err := Boolean(Difference, cyl, slab)
	if err != nil {
		t.Fatalf("oblique cut: %v", err)
	}
	return sideFaceOf(t, pin)
}

func TestRuledFaceOfTakesAnObliquelyCutSide(t *testing.T) {
	t.Parallel()
	f := obliquePinFace(t)
	rs, ok := ruledFaceOf(f)
	if !ok {
		t.Fatal("a cylinder side framed by a circle and an ellipse must be a wall")
	}
	rise := 0.1 * stdmath.Tan(7.5*stdmath.Pi/180)
	if stdmath.Abs(rs.band.vMin-(0.2-rise)) > 1e-12 || stdmath.Abs(rs.band.vMax-1.3) > 1e-12 {
		t.Errorf("axial window = [%g, %g], want [%g, 1.3] (the ellipse dips r·tanα below its station)", rs.band.vMin, rs.band.vMax, 0.2-rise)
	}
	if rs.frame.RadConst != 0.1 || rs.frame.RadSlope != 0 {
		t.Errorf("frame = %+v, want a cylinder of radius 0.1", rs.frame)
	}
}

func TestRuledFaceOfDeclinesAnOpenLoop(t *testing.T) {
	t.Parallel()
	surf, _ := geom.NewCylinder(math.P3(0, 0, 0), math.V3(0, 0, 1), 1)
	open := curvedFace{surface: surf, loops: []curvedLoop{{edges: []loopEdge{
		{curve: geom.NewLineSegment(math.P3(1, 0, 0), math.P3(1, 0, 1)), t0: 0, t1: 1},
	}}}}
	if _, ok := ruledFaceOf(open); ok {
		t.Error("a loop that does not close cannot frame a wall")
	}
}

func TestFoldIntoStripKeepsASeamEndExact(t *testing.T) {
	t.Parallel()
	twoPi := 2 * stdmath.Pi
	s := foldIntoStrip(uvSeg{a: math.P2(twoPi, 0.5), b: math.P2(twoPi+0.1, 0.6)})
	if float64(s.a.X) != 0 || stdmath.Abs(float64(s.b.X)-0.1) > 1e-15 {
		t.Errorf("segment past the seam folded to [%g, %g], want [0, 0.1]", float64(s.a.X), float64(s.b.X))
	}
	s = foldIntoStrip(uvSeg{a: math.P2(6.1, 0.5), b: math.P2(twoPi, 0.6)})
	if float64(s.a.X) != 6.1 || float64(s.b.X) != twoPi {
		t.Errorf("segment ending on the seam moved to [%g, %g]", float64(s.a.X), float64(s.b.X))
	}
}

func TestInjectedParamsFollowTraversal(t *testing.T) {
	t.Parallel()
	circle, _ := geom.NewCircle(math.P3(0, 0, 0), math.V3(0, 0, 1), 1)
	ps := injectedParams(circle, 1, 0, []float64{0.123, 0.5, 2}) // reversed traversal; 2 is out of span
	if ps[0] != 1 || ps[len(ps)-1] != 0 {
		t.Fatalf("reversed traversal runs %g → %g, want 1 → 0", ps[0], ps[len(ps)-1])
	}
	seen := map[float64]bool{}
	for i, p := range ps {
		seen[p] = true
		if i > 0 && ps[i] >= ps[i-1] {
			t.Fatalf("params not strictly decreasing at %d: %v", i, ps[i-1:i+1])
		}
	}
	if !seen[0.123] || !seen[0.5] || seen[2] {
		t.Errorf("injected 0.123 and 0.5 must appear, 2 (outside the span) must not: %v", seen)
	}
}

func TestCurvedFaceLineIntervalsClipsAnEllipticalCap(t *testing.T) {
	t.Parallel()
	pinFace := obliquePinFace(t)
	// The pin's oblique cap is the planar face whose plane is tilted off the pin's cross-section.
	var cap curvedFace
	for _, f := range facesOfAny(pinBodyOf(t, pinFace)) {
		if pl, isPlane := f.surface.(geom.Plane); isPlane && stdmath.Abs(float64(pl.Normal().Y)) < 0.999 {
			cap = f
		}
	}
	if cap.surface == nil {
		t.Fatal("no elliptical cap found on the pin")
	}
	// An in-plane line through the cap's centre along its major axis crosses the ellipse at ±major radius.
	a := 7.5 * stdmath.Pi / 180
	ivs, exact := curvedFaceLineIntervals(cap, math.P3(0, -1.5, 1.15), math.V3(stdmath.Cos(a), stdmath.Sin(a), 0))
	if !exact || len(ivs) != 1 {
		t.Fatalf("intervals = %v exact=%v, want one exact interval", ivs, exact)
	}
	major := 0.1 / stdmath.Cos(7.5*stdmath.Pi/180)
	if stdmath.Abs(ivs[0][0]+major) > 1e-12 || stdmath.Abs(ivs[0][1]-major) > 1e-12 {
		t.Errorf("chord = [%g, %g], want ±%g (the section's major radius)", ivs[0][0], ivs[0][1], major)
	}
}

// pinBodyOf rebuilds the pin body the face came from, so the test can reach its other faces.
func pinBodyOf(t *testing.T, _ curvedFace) *topo.Body {
	t.Helper()
	cyl, _ := SolidCylinder(math.P3(0, -1.7, 1.15), math.V3(0, 1, 0), 0.1, 1.3)
	a := 7.5 * stdmath.Pi / 180
	ex, ey := math.V3(stdmath.Cos(a), stdmath.Sin(a), 0), math.V3(-stdmath.Sin(a), stdmath.Cos(a), 0)
	c := math.P3(0, -1.5, 1.15)
	corner := func(sx, sy, z float64) math.Point3 {
		return c.TranslateBy(ex.Scale(math.Scalar(sx))).TranslateBy(ey.Scale(math.Scalar(sy))).TranslateBy(math.V3(0, 0, z))
	}
	slab := subd.ToBody(subd.Mesh{
		Verts: []math.Point3{corner(-1, -1, -1), corner(1, -1, -1), corner(1, 0, -1), corner(-1, 0, -1), corner(-1, -1, 1), corner(1, -1, 1), corner(1, 0, 1), corner(-1, 0, 1)},
		Faces: [][]int{{3, 2, 1, 0}, {4, 5, 6, 7}, {0, 1, 5, 4}, {1, 2, 6, 5}, {2, 3, 7, 6}, {3, 0, 4, 7}},
	}, "slab")
	pin, err := Boolean(Difference, cyl, slab)
	if err != nil {
		t.Fatalf("oblique cut: %v", err)
	}
	return pin
}

func TestOrientFaceSignsTurnsEveryLumpOutward(t *testing.T) {
	t.Parallel()
	// Two disjoint blocks in one face set: each shell must integrate positive on its own, whatever
	// colouring it arrived with (the second block's loops are reversed here).
	a, _ := SolidBlock(math.P3(0, 0, 0), math.P3(1, 1, 1), "a")
	b, _ := SolidBlock(math.P3(3, 0, 0), math.P3(4, 1, 1), "b")
	faces := append(facesOfAny(a), reverseCurvedFaces(facesOfAny(b))...)
	ff := make([]fluxFace, 0, len(faces))
	for _, f := range faces {
		region := faceTrimRegion(f)
		u0, u1, v0, v1, ok := fluxDomain(f, region)
		if !ok {
			t.Fatalf("no flux domain for %T", f.surface)
		}
		ff = append(ff, fluxFace{cf: f, region: region, u0: u0, u1: u1, v0: v0, v1: v1, sign: 1})
	}
	signs := orientFaceSigns(ff)
	for _, shell := range fluxShellsLargestFirst(ff, signs) {
		if v := shellSignedVolume(ff, signs, shell); v <= 0 {
			t.Errorf("a lump of %d faces integrates %g, want positive", len(shell), v)
		}
	}
}

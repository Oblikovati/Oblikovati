// SPDX-License-Identifier: GPL-2.0-only

package feature

import (
	stdmath "math"
	"testing"

	"oblikovati.org/kernel/geom"
	"oblikovati.org/kernel/ops"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
	"oblikovati.org/model/sketch"
)

// Analytic PARTIAL revolve (#2019): a partial-angle revolve of a line-only meridian keeps true
// surface-of-revolution walls (partial cylinder / cone / disk sector) plus two planar caps, instead
// of the faceted section sweep whose walls project onto a sketch as chorded arcs.

// planarFaceCount counts the geom.Plane faces of a body.
func planarFaceCount(b *topo.Body) int {
	n := 0
	for _, f := range b.Faces() {
		if _, ok := f.Geometry().(geom.Plane); ok {
			n++
		}
	}
	return n
}

// partialTubeBody revolves the washer profile (x∈[2,4], y∈[0,2]) through angle about the Y axis.
func partialTubeBody(t *testing.T, angle float64, dir ExtentDirection) *topo.Body {
	t.Helper()
	fs := NewPartFeatures(nil)
	sk := offsetSquareSketch(2, 2)
	cl := sk.Lines().AddByTwoPoints(math.P2(0, 0), math.P2(0, 2))
	cl.SetCenterline(true)
	def := &RevolveDefinition{Sketch: sk, ProfileIndex: 0, Angle: func() float64 { return angle }, Direction: dir, Operation: ops.NewBody}
	rf := &RevolveFeature{def: def}
	pf := fs.Add(rf)
	pf.SetName(fs.UniqueName("Revolution"))
	rf.featName = pf.name
	fs.Recompute()
	if !pf.Health().OK() {
		t.Fatalf("partial revolve sick: %+v", pf.Health())
	}
	return fs.Result()[0]
}

// TestPartialRevolveTubeKeepsCylinderWalls: a quarter revolve of the tube profile is the analytic
// sector — bore + outer cylinder walls, two annulus-sector caps at z, two planar side caps: exactly
// six faces, two of them cylinders — not a faceted prism.
func TestPartialRevolveTubeKeepsCylinderWalls(t *testing.T) {
	body := partialTubeBody(t, stdmath.Pi/2, PositiveDir)
	if r := ops.Validate(body); !r.Valid || !body.IsSolid() {
		t.Fatalf("partial revolved tube is not a valid solid: %+v", r.Issues)
	}
	if got := len(body.Faces()); got != 6 {
		t.Fatalf("partial tube has %d faces, want 6 (2 walls + 2 annulus sectors + 2 caps) — faceted", got)
	}
	if got := cylinderFaceCount(body); got != 2 {
		t.Fatalf("partial tube has %d cylinder faces, want 2 (bore + outer wall)", got)
	}
	if got := planarFaceCount(body); got != 4 {
		t.Fatalf("partial tube has %d planar faces, want 4 (2 annulus sectors + 2 caps)", got)
	}
	want := 0.25 * stdmath.Pi * (4*4 - 2*2) * 2 // quarter of the full tube = 6π
	if got := bodyVolume(body); relErr(got, want) > 0.03 {
		t.Errorf("partial tube volume = %g, want ≈%g (quarter tube, 6π)", got, want)
	}
}

// TestPartialRevolveSymmetric checks the start-angle offset (SymmetricDir sweeps −A/2…+A/2): the
// sector must stay a valid analytic solid with the same volume regardless of where it starts.
func TestPartialRevolveSymmetric(t *testing.T) {
	body := partialTubeBody(t, stdmath.Pi/2, SymmetricDir)
	if r := ops.Validate(body); !r.Valid || !body.IsSolid() {
		t.Fatalf("symmetric partial tube is not a valid solid: %+v", r.Issues)
	}
	if got := cylinderFaceCount(body); got != 2 {
		t.Fatalf("symmetric partial tube has %d cylinder faces, want 2", got)
	}
	want := 0.25 * stdmath.Pi * (4*4 - 2*2) * 2
	if got := bodyVolume(body); relErr(got, want) > 0.03 {
		t.Errorf("symmetric partial tube volume = %g, want ≈%g", got, want)
	}
}

// coneTriangleSketch is a right triangle (0,0)-(2,0)-(0,2) with the y-axis side as centerline: its
// hypotenuse revolves to a cone, its base to a disk, its axis side traces nothing.
func coneTriangleSketch() *sketch.Sketch {
	s := sketch.NewSketches().Add(sketch.XYPlane())
	base := s.Points().Add(math.P2(0, 0))
	rim := s.Points().Add(math.P2(2, 0))
	apex := s.Points().Add(math.P2(0, 2))
	s.Lines().Add(base, rim)  // base disk sector
	s.Lines().Add(rim, apex)  // hypotenuse → cone
	s.Lines().Add(apex, base) // closes the profile along the axis
	cl := s.Lines().AddByTwoPoints(math.P2(0, 0), math.P2(0, 2))
	cl.SetCenterline(true) // separate centerline coincident with the axis side
	return s
}

// TestPartialRevolveConeSector exercises the apex collapse and the oblique-edge cone: a quarter
// revolve of a right triangle touching the axis is a partial cone (one geom.Cone face) plus a disk
// sector and two caps.
func TestPartialRevolveConeSector(t *testing.T) {
	fs := NewPartFeatures(nil)
	pf := NewRevolveFeatures(fs).AddAboutCenterline(coneTriangleSketch(), 0, func() float64 { return stdmath.Pi / 2 }, ops.NewBody)
	fs.Recompute()
	if !pf.Health().OK() {
		t.Fatalf("partial cone revolve sick: %+v", pf.Health())
	}
	body := fs.Result()[0]
	if r := ops.Validate(body); !r.Valid || !body.IsSolid() {
		t.Fatalf("partial cone is not a valid solid: %+v", r.Issues)
	}
	if got := coneFaceCount(body); got != 1 {
		t.Fatalf("partial cone has %d cone faces, want 1 (apex collapse / oblique edge faceted)", got)
	}
	want := 0.25 * (1.0 / 3.0) * stdmath.Pi * 2 * 2 * 2 // quarter of a cone r=2 h=2 = 2π/3
	if got := bodyVolume(body); relErr(got, want) > 0.03 {
		t.Errorf("partial cone volume = %g, want ≈%g (quarter cone, 2π/3)", got, want)
	}
}

// coneFaceCount counts geom.Cone faces.
func coneFaceCount(b *topo.Body) int {
	n := 0
	for _, f := range b.Faces() {
		if _, ok := f.Geometry().(geom.Cone); ok {
			n++
		}
	}
	return n
}

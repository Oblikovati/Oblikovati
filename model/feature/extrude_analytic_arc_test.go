// SPDX-License-Identifier: GPL-2.0-only

package feature

import (
	stdmath "math"
	"testing"

	"oblikovati.org/kernel/geom"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
	"oblikovati.org/model/sketch"
)

// Analytic extrude for line+arc profiles (#2164): extruding a profile whose outer loop is arcs and
// lines must keep the arcs ANALYTIC in the B-rep — the cap faces bounded by real arc edges and the
// side walls true cylinders — not sampled into a chord polygon. Before this, an arc profile flattened
// to ~60 straight edges (a piston crown projected as 63 tiny line segments, so a downstream project +
// offset produced faceted arcs and corner slivers).

// topCapPlanarFace returns the planar cap face whose every boundary vertex lies at z (the far cap of
// a +Z extrude). It fails the test when no such face exists.
func topCapPlanarFace(t *testing.T, b *topo.Body, z float64) *topo.Face {
	t.Helper()
	for _, f := range b.Faces() {
		if _, ok := f.Geometry().(geom.Plane); !ok {
			continue
		}
		if faceVerticesAllAtZ(f, z) {
			return f
		}
	}
	t.Fatalf("no planar cap face with all vertices at z=%.3f", z)
	return nil
}

// faceVerticesAllAtZ reports whether every vertex on the face's outer loop is at height z.
func faceVerticesAllAtZ(f *topo.Face, z float64) bool {
	for _, u := range f.Loops()[0].EdgeUses() {
		for _, v := range u.Edge().Vertices() {
			if stdmath.Abs(float64(v.Point().Z)-z) > 1e-6 {
				return false
			}
		}
	}
	return true
}

// capEdgeCurveCounts counts the analytic curve kinds on a face's outer loop.
func capEdgeCurveCounts(f *topo.Face) (arcs, lines, other int) {
	for _, u := range f.Loops()[0].EdgeUses() {
		switch u.Edge().Geometry().(type) {
		case geom.Arc3d:
			arcs++
		case geom.LineSegment:
			lines++
		default:
			other++
		}
	}
	return arcs, lines, other
}

// TestExtrudeStadiumKeepsAnalyticArcCapEdges: the stadium (two arcs + two lines) must extrude to a
// cap bounded by exactly two arc edges and two line edges — not a chord polygon (#2164).
func TestExtrudeStadiumKeepsAnalyticArcCapEdges(t *testing.T) {
	const l, r, h = 10.0, 3.0, 4.0
	sk := stadiumSketch(l, r)
	b := extrudeProfile(t, sk, 0, h)
	assertWatertightSolid(t, b)

	cap := topCapPlanarFace(t, b, h)
	arcs, lines, other := capEdgeCurveCounts(cap)
	if arcs != 2 || lines != 2 || other != 0 {
		t.Fatalf("stadium end-cap has %d arc + %d line + %d other edges, want 2 arc + 2 line (the arcs faceted into chords, #2164)", arcs, lines, other)
	}
}

// concaveNotchSketch is a 10×10 square (x∈[-5,5], y∈[0,10]) with a semicircular bite of radius 5
// taken out of its TOP edge — a CONCAVE boundary arc (its centre lies outside the material), so the
// side wall must be a reversed cylinder. Region = 100 − πr²/2.
func concaveNotchSketch() *sketch.Sketch {
	s := sketch.NewSketches().Add(sketch.XYPlane())
	bl := s.Points().Add(math.P2(-5, 0))
	br := s.Points().Add(math.P2(5, 0))
	tr := s.Points().Add(math.P2(5, 10))
	tl := s.Points().Add(math.P2(-5, 10))
	s.Lines().Add(bl, br)                                       // bottom
	s.Lines().Add(br, tr)                                       // right
	s.Arcs().Add(s.Points().Add(math.P2(0, 10)), tr, tl, false) // concave top bite, through (0,5)
	s.Lines().Add(tl, bl)                                       // left
	return s
}

// TestExtrudeConcaveArcNotchReversedCylinder exercises the concave branch: a boundary arc whose centre
// is OUTSIDE the material must build a REVERSED cylinder wall. A wrong orientation makes the solid
// non-watertight or adds the bite instead of removing it — both caught here.
func TestExtrudeConcaveArcNotchReversedCylinder(t *testing.T) {
	const h = 3.0
	sk := concaveNotchSketch()
	b := extrudeProfile(t, sk, 0, h)
	assertWatertightSolid(t, b) // fails inside-out if the concave cylinder faces the wrong way

	cap := topCapPlanarFace(t, b, h)
	if arcs, lines, other := capEdgeCurveCounts(cap); arcs != 1 || lines != 3 || other != 0 {
		t.Fatalf("concave-notch cap = %d arc + %d line + %d other, want 1 arc + 3 line", arcs, lines, other)
	}
	assertAnalyticPrismVolume(t, b, 100-stdmath.Pi*25/2, h) // square minus the semicircular bite
}

// TestExtrudeStadiumAnalyticVolume: the analytic prism is the geometrically correct stadium solid —
// its (tessellated) volume tracks the TRUE analytic area (2rl + πr²) × h within the tessellation
// tolerance the codebase uses for curved bodies, confirming the swept solid is neither collapsed nor
// inverted. (Analyticity itself is asserted by the cap-edge test; meshing an arc always undershoots
// the exact value slightly, so this uses the 2% curved-volume tolerance, not an exact match.)
func TestExtrudeStadiumAnalyticVolume(t *testing.T) {
	const l, r, h = 10.0, 3.0, 4.0
	sk := stadiumSketch(l, r)
	b := extrudeProfile(t, sk, 0, h)
	want := (2*r*l + stdmath.Pi*r*r) * h
	if got := meshVolume(b); relErr(got, want) > 0.02 {
		t.Errorf("analytic stadium volume = %.6f, want ≈ %.6f (2rl+πr²)·h", got, want)
	}
}

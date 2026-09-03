// SPDX-License-Identifier: GPL-2.0-only

package brep

import (
	stdmath "math"
	"testing"

	"oblikovati.org/kernel/geom"
	"oblikovati.org/math"
)

// The planeFaceUV chart (ADR-0058): a planar face with full-circle boundary edges splits by straight
// imprint segments through the shared (u,v) trimmer, re-emitting EXACT sub-arcs that terminate on the
// closed-form circle∩segment crossing points.

// uvDiscFace builds a z=0 disc of radius r as a curvedFace (one full-circle outer loop, CCW about +z).
func uvDiscFace(t *testing.T, r float64) (curvedFace, geom.Circle) {
	t.Helper()
	pl, err := geom.NewPlane(math.P3(0, 0, 0), math.V3(0, 0, 1))
	if err != nil {
		t.Fatal(err)
	}
	circ, err := geom.NewCircle(math.P3(0, 0, 0), math.V3(0, 0, 1), r)
	if err != nil {
		t.Fatal(err)
	}
	return curvedFace{surface: pl, loops: []curvedLoop{{edges: []loopEdge{{curve: circ, t0: 0, t1: 1}}}}}, circ
}

// chordSegment is a straight imprint at x=x0 spanning y∈[−span, span] in the z=0 plane.
func chordSegment(x0, span float64) geom.Curve3 {
	return geom.NewLineSegment(math.P3(math.Scalar(x0), math.Scalar(-span), 0), math.P3(math.Scalar(x0), math.Scalar(span), 0))
}

// trimPlaneFace runs the shared trimmer over a planeFaceUV chart with the given keep predicate.
func trimPlaneFace(t *testing.T, f curvedFace, imprint []geom.Curve3, keep func(math.Point3) bool) []curvedFace {
	t.Helper()
	c, ok := newPlaneFaceUV(f, geom.ResolutionForSize(10))
	if !ok {
		t.Fatal("newPlaneFaceUV declined the fixture frame")
	}
	if !planeFaceContactOK(c, imprint) {
		t.Fatal("planeFaceContactOK declined a transversal fixture")
	}
	faces, _, err := trimByImprint(c, f, f.surface, imprint, planeFaceMaterial(c, keep))
	if err != nil {
		t.Fatalf("trimByImprint: %v", err)
	}
	return faces
}

// TestPlaneFaceUVChordSplitsDisc: a disc split by a chord, keeping the x<0.5 side, yields ONE face
// bounded by an EXACT major sub-arc of the rim circle and the chord segment, meeting exactly at the
// closed-form crossing points (0.5, ±√(r²−0.25)).
func TestPlaneFaceUVChordSplitsDisc(t *testing.T) {
	t.Parallel()
	f, circ := uvDiscFace(t, 2)
	faces := trimPlaneFace(t, f, []geom.Curve3{chordSegment(0.5, 2.5)}, func(p math.Point3) bool { return float64(p.X) < 0.5 })
	if len(faces) != 1 || len(faces[0].loops) != 1 {
		t.Fatalf("kept %d faces (loops %v), want 1 face with 1 loop", len(faces), len(faces[0].loops))
	}
	arcs, chords := 0, 0
	for _, e := range faces[0].loops[0].edges {
		switch e.curve.(type) {
		case geom.Circle:
			arcs++
			assertOnCircle(t, e, circ)
		case geom.LineSegment:
			chords++
		default:
			t.Errorf("unexpected edge curve %T in split loop", e.curve)
		}
	}
	if arcs != 1 || chords != 1 {
		t.Errorf("split loop has %d arcs + %d chords, want 1 + 1 (exact sub-arc + chord)", arcs, chords)
	}
}

// assertOnCircle checks a re-emitted sub-arc stays EXACTLY on its rim circle (radius at endpoints and
// midpoint) — the sagitta-free guarantee the exact re-emission provides.
func assertOnCircle(t *testing.T, e loopEdge, circ geom.Circle) {
	t.Helper()
	for _, tp := range []float64{e.t0, (e.t0 + e.t1) / 2, e.t1} {
		p := e.curve.PointAt(tp)
		if r := float64(circ.Center.DistanceTo(p)); stdmath.Abs(r-circ.Radius) > 1e-12 {
			t.Fatalf("sub-arc point at t=%g off the rim circle: radius %g want %g", tp, r, circ.Radius)
		}
	}
}

// TestPlaneFaceUVKeepAllIsWholeDisc: with a keep-everything predicate the imprint dissolves (both cell
// sides kept) and the whole disc comes back as one loop on its exact rim circle. The rim keeps the two
// vertices where the chord met it: the face that laid that chord meets the rim there too, and a shared
// edge subdivides identically on both faces (ADR-0060) — so the loop is the rim in exact sub-arcs
// (the rim's own parameter origin stays a vertex as well).
func TestPlaneFaceUVKeepAllIsWholeDisc(t *testing.T) {
	t.Parallel()
	f, circ := uvDiscFace(t, 2)
	faces := trimPlaneFace(t, f, []geom.Curve3{chordSegment(0.5, 2.5)}, func(math.Point3) bool { return true })
	if len(faces) != 1 || len(faces[0].loops) != 1 || len(faces[0].loops[0].edges) < 2 {
		t.Fatalf("keep-all: got %d faces, want the whole disc as one loop of rim arcs", len(faces))
	}
	span := 0.0
	for _, e := range faces[0].loops[0].edges {
		if _, isCircle := e.curve.(geom.Circle); !isCircle {
			t.Fatalf("keep-all loop edge = %T, want the rim circle", e.curve)
		}
		assertOnCircle(t, e, circ)
		span += stdmath.Abs(e.t1 - e.t0)
	}
	if stdmath.Abs(span-1) > 1e-12 {
		t.Fatalf("keep-all rim arcs span %g of the circle, want the whole turn", span)
	}
}

// TestPlaneFaceUVHoleSurvivesSplit: a square face with a circular hole, split by a chord away from the
// hole — the kept fragment containing the hole carries it as the EXACT full circle (reversed), while
// the fragment boundary carries the chord.
func TestPlaneFaceUVHoleSurvivesSplit(t *testing.T) {
	t.Parallel()
	pl, _ := geom.NewPlane(math.P3(0, 0, 0), math.V3(0, 0, 1))
	hole, _ := geom.NewCircle(math.P3(0, 0, 0), math.V3(0, 0, 1), 0.5)
	square := [][2]float64{{-2, -2}, {2, -2}, {2, 2}, {-2, 2}}
	outer := curvedLoop{}
	for i := range square {
		a, b := square[i], square[(i+1)%len(square)]
		seg := geom.NewLineSegment(math.P3(math.Scalar(a[0]), math.Scalar(a[1]), 0), math.P3(math.Scalar(b[0]), math.Scalar(b[1]), 0))
		outer.edges = append(outer.edges, loopEdge{curve: seg, t0: 0, t1: 1})
	}
	f := curvedFace{surface: pl, loops: []curvedLoop{outer, {edges: []loopEdge{{curve: hole, t0: 1, t1: 0}}}}}

	faces := trimPlaneFace(t, f, []geom.Curve3{chordSegment(1.5, 2.5)}, func(p math.Point3) bool { return float64(p.X) < 1.5 })
	if len(faces) != 1 {
		t.Fatalf("kept %d faces, want 1 (the x<1.5 fragment with the hole)", len(faces))
	}
	holeLoops := 0
	for _, l := range faces[0].loops {
		if len(l.edges) == 1 {
			if _, isCircle := l.edges[0].curve.(geom.Circle); isCircle && isFullDomain(l.edges[0].t0, l.edges[0].t1) {
				holeLoops++
			}
		}
	}
	if holeLoops != 1 {
		t.Fatalf("fragment carries %d exact full-circle hole loops, want 1", holeLoops)
	}
}

// TestCurvedFaceLineIntervals: exact even-odd line-in-face intervals on a disc and a holed square.
func TestCurvedFaceLineIntervals(t *testing.T) {
	t.Parallel()
	f, _ := uvDiscFace(t, 2)
	// A diameter line through the centre: one interval spanning [−2, 2] exactly.
	iv, ok := curvedFaceLineIntervals(f, math.P3(0, 0, 0), math.V3(1, 0, 0))
	if !ok || len(iv) != 1 || stdmath.Abs(iv[0][0]+2) > 1e-9 || stdmath.Abs(iv[0][1]-2) > 1e-9 {
		t.Fatalf("disc diameter intervals = %v ok=%v, want [[-2,2]]", iv, ok)
	}
	// A line missing the disc entirely: no intervals.
	iv, ok = curvedFaceLineIntervals(f, math.P3(0, 3, 0), math.V3(1, 0, 0))
	if !ok || len(iv) != 0 {
		t.Fatalf("miss intervals = %v ok=%v, want none", iv, ok)
	}
	// The holed square (hole r=0.5 at centre): a centre line yields TWO intervals around the hole.
	pl, _ := geom.NewPlane(math.P3(0, 0, 0), math.V3(0, 0, 1))
	hole, _ := geom.NewCircle(math.P3(0, 0, 0), math.V3(0, 0, 1), 0.5)
	square := [][2]float64{{-2, -2}, {2, -2}, {2, 2}, {-2, 2}}
	outer := curvedLoop{}
	for i := range square {
		a, b := square[i], square[(i+1)%len(square)]
		outer.edges = append(outer.edges, loopEdge{curve: geom.NewLineSegment(
			math.P3(math.Scalar(a[0]), math.Scalar(a[1]), 0), math.P3(math.Scalar(b[0]), math.Scalar(b[1]), 0)), t0: 0, t1: 1})
	}
	holed := curvedFace{surface: pl, loops: []curvedLoop{outer, {edges: []loopEdge{{curve: hole, t0: 1, t1: 0}}}}}
	iv, ok = curvedFaceLineIntervals(holed, math.P3(0, 0, 0), math.V3(1, 0, 0))
	if !ok || len(iv) != 2 || stdmath.Abs(iv[0][1]+0.5) > 1e-9 || stdmath.Abs(iv[1][0]-0.5) > 1e-9 {
		t.Fatalf("holed-square intervals = %v ok=%v, want two intervals split at ±0.5", iv, ok)
	}
}

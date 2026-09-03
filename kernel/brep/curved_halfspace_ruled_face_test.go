// SPDX-License-Identifier: GPL-2.0-only

package brep

import (
	stdmath "math"
	"testing"

	"oblikovati.org/kernel/geom"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
)

// holedWallFace is a cylinder wall band carrying a HOLE: its two rim circles plus a rectangular window
// cut in the wall, three closed loops in all. That is the shape the boundary walk cannot pair —
// splitLoopByPlane needs exactly ONE loop — so before ADR-0061 stage 2 a plane cutting such a wall
// declined to the CSG fallback. It is built directly rather than by a boolean, because the boolean that
// would drill it is itself one of the configurations the retirement has still to model.
func holedWallFace(t *testing.T) curvedFace {
	t.Helper()
	ax, _ := math.NewUnitVector3(0, 0, 1)
	ref, _ := math.NewUnitVector3(1, 0, 0)
	cyl := geom.Cylinder{Origin: math.P3(0, 0, 0), AxisDir: ax, Ref: ref, Radius: 3}
	rim := func(z float64) curvedLoop {
		c, err := geom.NewCircle(math.P3(0, 0, math.Scalar(z)), math.V3(0, 0, 1), 3)
		if err != nil {
			t.Fatalf("rim at z=%g: %v", z, err)
		}
		return curvedLoop{edges: []loopEdge{{curve: c, t0: 0, t1: 1, v0: c.PointAt(0), v1: c.PointAt(1)}}}
	}
	// The window: two constant-z arcs joined by two rulings, a closed circuit on the wall.
	at := func(deg, z float64) math.Point3 {
		a := deg * stdmath.Pi / 180
		return math.P3(math.Scalar(3*stdmath.Cos(a)), math.Scalar(3*stdmath.Sin(a)), math.Scalar(z))
	}
	arc := func(z, from, to float64) geom.Curve3 {
		c, err := geom.NewArc3d(math.P3(0, 0, math.Scalar(z)), math.V3(0, 0, 1), math.V3(1, 0, 0), 3,
			from*stdmath.Pi/180, (to-from)*stdmath.Pi/180)
		if err != nil {
			t.Fatalf("arc at z=%g: %v", z, err)
		}
		return c
	}
	window := curvedLoop{edges: []loopEdge{
		{curve: arc(4, 20, 60), t0: 0, t1: 1, v0: at(20, 4), v1: at(60, 4)},
		{curve: geom.NewLineSegment(at(60, 4), at(60, 6)), t0: 0, t1: 1, v0: at(60, 4), v1: at(60, 6)},
		{curve: arc(6, 60, 20), t0: 0, t1: 1, v0: at(60, 6), v1: at(20, 6)},
		{curve: geom.NewLineSegment(at(20, 6), at(20, 4)), t0: 0, t1: 1, v0: at(20, 6), v1: at(20, 4)},
	}}
	return curvedFace{surface: cyl, lineage: topo.NewLineage(topo.Tok("holedwall", "f", 0)),
		loops: []curvedLoop{rim(0), rim(10), window}}
}

// TestHalfSpaceCutsAWallTheBoundaryWalkCannotPair pins ADR-0061 stage 2's first migration: a ruled wall
// with more than one loop is trimmed in the frame its own loops make, where the boundary walk declined.
func TestHalfSpaceCutsAWallTheBoundaryWalkCannotPair(t *testing.T) {
	t.Parallel()
	f := holedWallFace(t)
	res := geom.ResolutionForBox(faceLoopBox(f))
	plane, err := geom.NewPlane(math.P3(0, 0, 7), math.V3(0, 0, 1)) // keep z ≤ 7, above the window
	if err != nil {
		t.Fatalf("plane: %v", err)
	}
	n := plane.Normal()
	if boundaryWalkPairs(f, plane, n, res) {
		t.Fatal("premise: a three-loop wall must NOT be sent to the single-loop boundary walk")
	}
	curves, handled := curvedImprint(f, curvedFace{surface: plane}, res)
	if !handled || len(curves) == 0 {
		t.Fatalf("premise: the plane must section the wall (handled=%v curves=%d)", handled, len(curves))
	}
	pieces, _, ok := ruledHalfSpaceSplit(f, curves, plane, n)
	if !ok {
		t.Fatal("the loop-framed chart declined a wall the boundary walk cannot pair")
	}
	if len(pieces) == 0 {
		t.Fatal("the chart kept nothing, but the plane crosses the wall")
	}
	// The window sits at z 4..6, wholly below the cut, so it must survive as a hole in the kept piece.
	holes := 0
	for _, p := range pieces {
		holes += len(p.loops) - 1
	}
	if holes < 1 {
		t.Errorf("the kept wall has %d hole loop(s); the window at z 4..6 is below the cut and must survive", holes)
	}
	t.Logf("kept %d face(s) carrying %d hole loop(s)", len(pieces), holes)
}

// TestBoundaryWalkPairsClassifiesRatherThanRetries: the classification must SEPARATE the two paths — a
// plain band the walk pairs, a multi-loop wall it does not — so neither runs as the other's fallback.
func TestBoundaryWalkPairsClassifiesRatherThanRetries(t *testing.T) {
	t.Parallel()
	plain, err := SolidCylinder(math.P3(0, 0, 0), math.V3(0, 0, 1), 3, 10)
	if err != nil {
		t.Fatalf("cylinder: %v", err)
	}
	res := geom.ResolutionForBox(plain.RangeBox())
	plane, err := geom.NewPlane(math.P3(1.5, 0, 0), math.V3(1, 0, 0))
	if err != nil {
		t.Fatalf("plane: %v", err)
	}
	for _, f := range facesOfAny(plain) {
		if _, isPlane := f.surface.(geom.Plane); isPlane {
			continue
		}
		if !boundaryWalkPairs(f, plane, plane.Normal(), res) {
			t.Error("a plain cylinder wall must go to the boundary walk, not the loop frame")
		}
	}
	holed := holedWallFace(t)
	if boundaryWalkPairs(holed, plane, plane.Normal(), geom.ResolutionForBox(faceLoopBox(holed))) {
		t.Errorf("a %d-loop face must NOT be sent to the single-loop boundary walk", len(holed.loops))
	}
}

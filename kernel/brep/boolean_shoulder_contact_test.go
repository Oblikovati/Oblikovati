// SPDX-License-Identifier: GPL-2.0-only

package brep

import (
	"testing"

	"oblikovati.org/kernel/geom"
	"oblikovati.org/math"
)

// A rod stopping PART WAY through a ball's shoulder: its end cap is neither inside the ball nor outside
// it, and the ball's own surface crosses it. The cap's section with the ball is a circle strictly inside
// the cap's DISC, so the contact test has to see a point inside a face bounded by ONE closed circle.
//
// The polygon containment test reads one point per boundary edge, and a disc's "polygon" is that single
// point — it contains nothing. So the section read as no contact at all, no imprint was planned, and the
// ball passed through WHOLE. Containment now goes through faceContainsExact, which meets an arc boundary
// by exact ray intervals (ADR-0061 stage 4).
func TestShoulderRodMeetsTheBallThroughItsCapDisc(t *testing.T) {
	t.Parallel()
	const ballR, rodR, stop = 0.5, 0.3, 0.45
	for _, op := range []Op{Union, Difference, Intersection} {
		t.Run(opName(op), func(t *testing.T) {
			t.Parallel()
			ball, err := SolidSphere(math.P3(0, 0, 0), ballR, "ball")
			if err != nil {
				t.Fatalf("SolidSphere: %v", err)
			}
			rod, err := SolidCylinder(math.P3(0, 0, 0), math.V3(0, 1, 0), rodR, stop)
			if err != nil {
				t.Fatalf("SolidCylinder: %v", err)
			}
			res, err := Boolean(op, ball, rod)
			if err != nil {
				t.Fatalf("Boolean(%v): %v", op, err)
			}
			assertWatertight(t, res)
			cut := 0
			for _, f := range res.Faces() {
				if _, ok := f.Geometry().(geom.Sphere); !ok {
					continue
				}
				if len(f.Loops()) == 0 {
					t.Errorf("%v: the ball's surface came back UNCUT (no boundary loop)", op)
					continue
				}
				cut++
			}
			if cut == 0 {
				t.Errorf("%v: no trimmed sphere face survived — the ball was never imprinted", op)
			}
		})
	}
}

// opName is the operation's name for a subtest.
func opName(op Op) string {
	switch op {
	case Union:
		return "union"
	case Intersection:
		return "intersect"
	default:
		return "difference"
	}
}

// The ball's surviving surface after a shoulder rod takes a BELT out of it is two disconnected caps —
// one below the seam, one beyond the rod's end — and that is two FACES.
//
// The emission grouped a trim's loops by (u,v) containment over the whole kept set, and two loops in
// different components need not contain one another at all: the two caps' rim circles are horizontal
// lines across the chart, so neither contained the other and both were filed on one face. A disconnected
// region's components are faces; containment files the holes WITHIN a component (ADR-0061 stage 4).
func TestShoulderCutLeavesTwoSphereFaces(t *testing.T) {
	t.Parallel()
	ball, err := SolidSphere(math.P3(0, 0, 0), 0.5, "ball")
	if err != nil {
		t.Fatalf("SolidSphere: %v", err)
	}
	rod, err := SolidCylinder(math.P3(0, 0, 0), math.V3(0, 1, 0), 0.3, 0.45)
	if err != nil {
		t.Fatalf("SolidCylinder: %v", err)
	}
	res, err := Boolean(Difference, ball, rod)
	if err != nil {
		t.Fatalf("Boolean(Difference): %v", err)
	}
	assertWatertight(t, res)
	n := 0
	for _, f := range res.Faces() {
		if _, ok := f.Geometry().(geom.Sphere); ok {
			n++
		}
	}
	if n != 2 {
		t.Errorf("ball − shoulder rod has %d sphere faces, want 2 (the cap below the seam and the tip beyond the rod)", n)
	}
}

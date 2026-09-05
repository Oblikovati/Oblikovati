// SPDX-License-Identifier: GPL-2.0-only

package brep

import (
	"testing"

	"oblikovati.org/math"
)

// cutUVFromNotched builds a cutCylinderUV over a top-notched cylinder (bottom rim intact, top wedge clipped by
// the plane x+z=9.5), with a dummy second-cut membership. Its prior loop is the notched top boundary and its
// seam is placed clear of the notch — the fixture the survival predicate and segment assembly run on.
func cutUVFromNotched(t *testing.T) cutCylinderUV {
	t.Helper()
	f := firstCylinderFace(t, notchedCylinder(t, true))
	cyl, band, prior, ok := cutCylinderSideBand(f)
	if !ok {
		t.Fatal("notched cylinder side not recognised")
	}
	c := newCutCylinderUVSolid(cyl, band, prior, Difference, false, func(math.Point3) bool { return false })
	c.placeSeams(nil) // no new imprint yet; seam goes to the widest gap clear of the notch
	return c
}

// TestBelowPriorClassifiesSurvivalAgainstFirstCut: the survival predicate must agree with the first cut's
// half-space x+z≤9.5 — a point below the notched top boundary survived, one above it was removed. Points are
// mapped through the SAME paramOf the prior polyline uses, so the seam shift cancels.
func TestBelowPriorClassifiesSurvivalAgainstFirstCut(t *testing.T) {
	t.Parallel()
	c := cutUVFromNotched(t)
	poly := c.priorUVSegments()
	// (surface point, want-survived). x+z ≤ 9.5 survived the first cut; > 9.5 was removed.
	cases := []struct {
		p        math.Point3
		survived bool
	}{
		{math.P3(-3, 0, 5), true},   // back wall, mid height: x+z=2, deep in surviving material
		{math.P3(-3, 0, 9.9), true}, // back wall near the top: rim intact here (x+z=6.9)
		{math.P3(3, 0, 5), true},    // front wall below the notch: x+z=8 ≤ 9.5
		{math.P3(3, 0, 8), false},   // front wall inside the notch: x+z=11 > 9.5, removed
		{math.P3(3, 0, 6.4), true},  // front wall just under the notch floor (~6.5)
		{math.P3(3, 0, 6.6), false}, // front wall just above the notch floor
	}
	// The composed material predicate (survival ∧ second cut) reduces to belowPrior here: the dummy inside is
	// always false, so under Difference the target keeps every point (keep(Difference,false,false)=true).
	mat := cutCylinderMaterial(&c)()
	for _, tc := range cases {
		uv := c.paramOf(tc.p)
		if got := belowPrior(poly, uv); got != tc.survived {
			t.Errorf("belowPrior at %v (uv=%.3f,%.3f) = %v, want survived=%v", tc.p, uv.X, uv.Y, got, tc.survived)
		}
		if got := mat(uv); got != tc.survived {
			t.Errorf("composed material at %v = %v, want %v (base is all-keep, so it must equal survival)", tc.p, got, tc.survived)
		}
	}
}

// TestCutFrameHasNoTopRim: an already-cut side has no full top circle, so the frame carries the bottom rim and
// the two seam verticals but NO second rim — the prior loop is the top boundary the arrangement uses instead.
func TestCutFrameHasNoTopRim(t *testing.T) {
	t.Parallel()
	c := cutUVFromNotched(t)
	frame := c.cutFrameSegments()
	rims, seams := 0, 0
	for _, s := range frame {
		switch s.kind {
		case segRim:
			rims++
			if s.a.Y != c.band.vMin {
				t.Errorf("the only rim must be the bottom (v=%v); got a rim at v=%v", c.band.vMin, s.a.Y)
			}
		case segSeam:
			seams++
		}
	}
	if rims != 1 || seams != 2 {
		t.Errorf("cut frame has %d rims + %d seams; want 1 bottom rim + 2 seams (no top rim)", rims, seams)
	}
}

// TestAssembleSegmentsIngestsPriorLoop: the second cut's segment set must include the prior loop's edges as
// constraint segments so the arrangement subdivides at the surviving boundary; without them a cell straddling
// the old notch would classify as one material and re-include removed geometry.
func TestAssembleSegmentsIngestsPriorLoop(t *testing.T) {
	t.Parallel()
	c := cutUVFromNotched(t)
	if got, want := len(c.priorUVSegments()), 1; got < want {
		t.Fatalf("prior loop produced %d (u,v) segments; want at least the notched boundary", got)
	}
	// With no new imprint the assembled set is exactly prior segments + the cut frame (bottom rim + 2 seams).
	segs := c.assembleSegments(nil)
	if len(segs) != len(c.priorUVSegments())+3 {
		t.Errorf("assembled %d segments; want prior(%d) + frame(3)", len(segs), len(c.priorUVSegments()))
	}
}

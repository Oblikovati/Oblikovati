// SPDX-License-Identifier: GPL-2.0-only

package brep

import (
	"testing"

	"oblikovati.org/kernel/geom"
	"oblikovati.org/math"
)

// A rod drilled through an ALREADY-NOTCHED cylinder, positioned so its exit bite crosses the notch: the
// rod's exit is bounded by the wall crossing over part of the turn and by the notch plane's section over
// the rest, the two meeting at two triple points shared by the wall, the notch cap and the tunnel
// (EPIC #1738, ADR-0048).
//
// The general pipeline reaches it once two imprints on one face co-refine against each other — the
// arrangement welds vertices, it does not split a segment where another crosses it — and once BOTH sides
// of that meeting are solved in their own parameters rather than by inverting an approximate candidate
// (ADR-0061 stage 4).
func TestCornerJunctionCutWeldsThroughTheGeneralPath(t *testing.T) {
	t.Parallel()
	bare, err := SolidCylinder(math.P3(0, 0, 0), math.V3(0, 0, 1), 3, 10)
	if err != nil {
		t.Fatalf("SolidCylinder: %v", err)
	}
	pl, err := geom.NewPlane(math.P3(1.5, 0, 8), math.V3(1, 0, 1))
	if err != nil {
		t.Fatalf("NewPlane: %v", err)
	}
	target, err := HalfSpaceCut(bare, pl)
	if err != nil {
		t.Fatalf("first cut (the notch): %v", err)
	}
	rod, err := SolidCylinder(math.P3(-6, 0, 7), math.V3(1, 0, 0), 1, 12)
	if err != nil {
		t.Fatalf("SolidCylinder rod: %v", err)
	}
	res, err := Boolean(Difference, target, rod)
	if err != nil {
		t.Fatalf("Boolean(Difference) on the corner junction: %v", err)
	}
	assertWatertight(t, res)
	if got := len(res.Faces()); got != 5 {
		t.Errorf("corner-junction cut has %d faces, want 5 (holed wall + 3 caps + tunnel)", got)
	}
}

// A ruled-quadric crossing lies on TWO surfaces and must name both incidences: which of them is the
// chart's own host — and so carries no information at all, being satisfied by every point of it —
// depends on which side is asking. Naming only the carried quadric made the crossing's meeting with a
// section on its OWN base unsolvable: the condition was identically zero along the walk, its sign
// flipped on float noise, and the roots were seven pieces of noise instead of the two triple points.
func TestRuledCrossingNamesBothIncidences(t *testing.T) {
	t.Parallel()
	cyl, err := geom.NewCylinder(math.P3(0, 0, 0), math.V3(0, 0, 1), 3)
	if err != nil {
		t.Fatalf("NewCylinder: %v", err)
	}
	rod, err := geom.NewCylinder(math.P3(0, 0, 7), math.V3(1, 0, 0), 1)
	if err != nil {
		t.Fatalf("NewCylinder rod: %v", err)
	}
	curves, handled := geom.IntersectSurfacesAnalytic(cyl, rod, geom.ResolutionForSize(12))
	if !handled || len(curves) == 0 {
		t.Fatalf("IntersectSurfacesAnalytic(cylinder, rod): handled=%v curves=%d", handled, len(curves))
	}
	arc := curves[0]
	conds := geom.CurveIncidence(arc)
	if len(conds) != 2 {
		t.Fatalf("%T names %d incidences, want 2 (both surfaces it lies on)", arc, len(conds))
	}
	// Both must vanish along the crossing, and each must discriminate somewhere off it.
	lo, hi := arc.Domain()
	for i, on := range conds {
		for k := 0; k <= 8; k++ {
			p := arc.PointAt(lo + (hi-lo)*float64(k)/8)
			if v := on(p); v > 1e-9 || v < -1e-9 {
				t.Errorf("incidence %d is %.3e on the crossing itself, want 0", i, v)
			}
		}
	}
}

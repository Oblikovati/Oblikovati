// SPDX-License-Identifier: GPL-2.0-only

package ops

import (
	stdmath "math"
	"strings"
	"testing"

	"oblikovati.org/kernel/geom"
	m "oblikovati.org/math"
)

// TestAdoptedCurveRunsTheStoredEdgeSense is the gate on the ONE thing that separates a working
// nil-vs-curve adoption from a broken one.
//
// ★ Both consumers of a manifold edge traverse it in OPPOSITE directions, and each offers its
// curve in ITS OWN direction (survivorCurve's from→to convention). The edge, however, was welded
// once, from→to of whichever consumer reached the catalog first. So a second consumer's curve is
// backwards relative to the edge it is being hung on whenever rec.from != a — which is the case for
// EVERY one of the 272 adoptions in the corpus. Adopting it without that reversal hands the
// receiving face a boundary that runs the wrong way round, its loop walk then bounds a different
// REGION, and simple/T3's blend torus inflates 2827.227365 → 13816.882599 (mesh area at
// PropertyQuality). Adopted the right way round the same face reads 2826.791716, a 0.015 %
// improvement, and its two mirror-twin blend cylinders come from 227.809448 / 227.851401
// (4.2e-02 apart, on a symmetric fixture) to 227.764913 / 227.764925 (1.2e-05 apart).
//
// The rule is CONDITIONAL, so both arms are driven: an offer that runs WITH the stored sense must
// be adopted untouched. A mutation that reverses unconditionally fails the second subtest.
func TestAdoptedCurveRunsTheStoredEdgeSense(t *testing.T) {
	_, major := complementaryArcPair()
	p0, p1 := major.PointAt(0), major.PointAt(1)
	for _, tc := range []struct {
		name     string
		sndA     int // the second consumer's from-vertex: 1 = it walks the edge backwards
		sndCurve geom.Curve3
	}{
		{"the second consumer walks the edge backwards", 1, reversedArc3d(major)},
		{"the second consumer walks it the same way", 0, major},
	} {
		t.Run(tc.name, func(t *testing.T) {
			ec := newSeamCatalog([]m.Point3{p0, p1})
			ec.use(0, 1, nil, 0) // the consumer with no curve welds the edge 0→1
			u := ec.use(tc.sndA, 1-tc.sndA, tc.sndCurve, 0)
			assertEdgeCarriesArcForward(t, u.Edge.Geometry(), major)
		})
	}
}

// assertEdgeCarriesArcForward requires the edge's geometry to BE want, traversed want's own way:
// the same endpoints in the same order, and the same path between them (sampledCurveGap forward,
// which is exactly what a face's loop walk reads).
func assertEdgeCarriesArcForward(t *testing.T, got geom.Curve3, want geom.Arc3d) {
	t.Helper()
	if _, isLine := got.(geom.LineSegment); isLine || got == nil {
		t.Fatalf("edge still carries %T, want the offered arc adopted — a nil offer is an absence of "+
			"information, not an assertion of straightness", got)
	}
	lo, hi := got.Domain()
	if d := float64(got.PointAt(lo).DistanceTo(want.PointAt(0))); d > 1e-9 {
		t.Fatalf("the adopted curve starts %.6g from the edge's own start vertex %v (it starts at %v) — it was "+
			"hung on the edge in the OFFERING consumer's direction, not the edge's stored sense",
			d, want.PointAt(0), got.PointAt(lo))
	}
	if d := float64(got.PointAt(hi).DistanceTo(want.PointAt(1))); d > 1e-9 {
		t.Fatalf("the adopted curve ends %.6g from the edge's own end vertex %v (it ends at %v)",
			d, want.PointAt(1), got.PointAt(hi))
	}
	if gap := sampledCurveGap(got, want, false); gap > 1e-9 {
		t.Errorf("the adopted curve is %.6g from the offered arc at matched fractions, want ≈0 — the edge "+
			"carries a DIFFERENT path between the same two vertices", gap)
	}
}

// reversedArc3d is the same arc traversed backwards, built the way survivorCurve builds it for a
// reversed use (start at the far end, negate the sweep) — a genuine Arc3d, not a wrapper, so the
// test drives the shape the corpus's consumers really hand the catalog.
func reversedArc3d(a geom.Arc3d) geom.Arc3d {
	return geom.Arc3d{Center: a.Center, Normal: a.Normal, RefDir: a.RefDir, Radius: a.Radius,
		StartAngle: a.StartAngle + a.SweepAngle, SweepAngle: -a.SweepAngle}
}

// TestOfferWithinTheWeldIsNeitherAdoptedNorRecorded pins the threshold the nil branch was missing.
//
// ★ The conflict branch has always compared its two offers against the model weld; the nil branch
// applied NO threshold at all, so 90 of the corpus's 362 recorded offers were curves departing
// their own chord by 4.4e-16…2.0e-15 — three orders below every model weld in the corpus
// (5.1e-08…1.3e-07) and pure float noise. The gate's own docstring claimed "every entry is a real
// consumer-side gap", and for those 90 it was false. This is a correctness fix to the detector's
// stated contract, so it is gated in BOTH directions: an offer a hair BELOW the weld must be
// silent and must leave the edge alone; an offer a hair ABOVE it must still be recorded AND
// adopted, so the fix cannot be mistaken for (or mutated into) blanket suppression.
func TestOfferWithinTheWeldIsNeitherAdoptedNorRecorded(t *testing.T) {
	for _, tc := range []struct {
		name       string
		sagitta    float64
		wantRecord bool
	}{
		{"a hair below the weld: float noise, not a gap", 1e-4, false},
		{"a hair above the weld: a real gap", 2e-3, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			bow := bowedArcOverUnitChord(t, tc.sagitta)
			ec := newSeamCatalog([]m.Point3{bow.PointAt(0), bow.PointAt(1)}) // weld 1e-3
			ec.use(0, 1, nil, 0)
			u := ec.use(1, 0, geom.ReverseCurve3(bow), 0)
			_, stillStraight := u.Edge.Geometry().(geom.LineSegment)
			if stillStraight == tc.wantRecord {
				t.Errorf("sagitta %.6g (weld 1e-3): edge carries %T, adopted=%v want adopted=%v",
					tc.sagitta, u.Edge.Geometry(), !stillStraight, tc.wantRecord)
			}
			got := ec.bld.Build().BuildDiagnostics()
			if (len(got) > 0) != tc.wantRecord {
				t.Errorf("sagitta %.6g (weld 1e-3): recorded %v, want recorded=%v — the nil branch must use the "+
					"SAME weld threshold the conflict branch does, and must still fire on a real gap",
					tc.sagitta, got, tc.wantRecord)
			}
		})
	}
}

// bowedArcOverUnitChord is the circular arc from (0,0,0) to (1,0,0) bowed by sagitta s at mid-span,
// i.e. an offer whose curveChordDeparture is s by construction.
func bowedArcOverUnitChord(t *testing.T, s float64) geom.Curve3 {
	t.Helper()
	arc, err := geom.Arc3dByThreePoints(m.P3(0, 0, 0), m.P3(0.5, s, 0), m.P3(1, 0, 0))
	if err != nil {
		t.Fatalf("Arc3dByThreePoints for sagitta %.6g: %v", s, err)
	}
	return arc
}

// TestClosedSeamOfferIsDeclinedNotAdopted pins the one shape where a nil MEANS something.
//
// A welded pair a==b is one vertex, and subdividedSurvivorCurve drops a closed-conic rim to nil
// there ON PURPOSE: the inserts re-trace that rim as straight chords, so hanging the full circle on
// the degenerate edge would make that one edge tessellate the WHOLE circle and self-cross the loop.
// A deliberate decline is therefore distinguishable from an absent offer by SHAPE, and needs no new
// flag: adoption refuses at a==b, and says so in the record rather than silently doing nothing.
func TestClosedSeamOfferIsDeclinedNotAdopted(t *testing.T) {
	c := geom.Arc3d{Center: m.P3(0, 0, 0), Normal: seamAxis(t, m.V3(0, 0, 1)),
		RefDir: seamAxis(t, m.V3(1, 0, 0)), Radius: 5, StartAngle: 0, SweepAngle: 2 * stdmath.Pi}
	ec := newSeamCatalog([]m.Point3{c.PointAt(0)})
	ec.use(0, 0, nil, 0)
	u := ec.use(0, 0, c, 0)
	lo, hi := u.Edge.Geometry().Domain()
	if d := float64(u.Edge.Geometry().PointAt(lo).DistanceTo(u.Edge.Geometry().PointAt(hi))); d > 1e-9 {
		t.Errorf("a full circle was adopted onto a pair welded to ONE vertex (its ends are %.6g apart) — that "+
			"edge would tessellate the whole circle and self-cross its loop", d)
	}
	got := ec.bld.Build().BuildDiagnostics()
	if len(got) != 1 || !strings.Contains(got[0].Detail, "DECLINED") {
		t.Errorf("recorded %v, want exactly one record naming the decline — a refusal must be visible, "+
			"not silence", got)
	}
}

// seamAxis is a unit vector for the closed-seam fixture's circle frame.
func seamAxis(t *testing.T, v m.Vector3) m.UnitVector3 {
	t.Helper()
	u, err := m.UnitVector3FromVector(v)
	if err != nil {
		t.Fatalf("UnitVector3FromVector(%v): %v", v, err)
	}
	return u
}

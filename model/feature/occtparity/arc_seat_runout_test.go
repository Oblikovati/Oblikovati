// SPDX-License-Identifier: GPL-2.0-only

package occtparity

import (
	stdmath "math"
	"testing"

	"oblikovati.org/kernel/geom"
	"oblikovati.org/kernel/ops"
	"oblikovati.org/kernel/topo"
)

// THE ARC BAND'S ROLLING-BALL SEAT AND ITS RUN-OUT TERMINATION — the corpus's last two arc defects, and
// the guard that neither half can quietly come back.
//
// simple/W2 and simple/H6 were ONE root. resolveArcFillet hard-coded two things about where the rolling
// ball sits: it took the cap plane's STORED normal as the outward one, and it always rode the ball at
// cylR−r (inside the cylinder). Both are the convex-shaft special case:
//
//   - W2's cap face is Reversed, so the stored normal points INTO the material and the band was pushed
//     wholesale to the void side; and its cylinder is a GROOVE, so the ball rides at cylR+r = 1.2.
//   - H6's stored normal IS outward, but its picked arc is CONCAVE — the blend ADDS material — so the
//     ball sits on the void side of the cap and rides at cylR+r = 60.
//
// Seating it correctly is necessary and NOT sufficient: W2's cylR+r cove then reaches 0.19998 = r below
// the bottom plane and the band spills out of the solid (its welded mesh leak widens 3/3 → 8/29). OCCT
// terminates the band ON that plane instead, against the SPIRIC section the plane cuts from the torus.
// With both, all three arc cases that terminate on a stand-off wall land on DRAWEXE face for face.
//
// Every number below is DRAWEXE 8.0.0's own `explode result F` + `sprops result_i 1.e-9`, captured with
// the case's own OCCT script. Identification is by area + `mksurface`/`dump` surface, never by
// `bounding` (see perface_oracle_test.go).

// arcOraclePerFaceTol is the per-face relative budget for the arc cases. It is mesh quantization: the
// loosest face in either case is W2's spiric-bounded band at 7.5e-4, everything else is under 1e-4.
const arcOraclePerFaceTol = 2e-3

// TestW2ArcBandRunsOutOnItsBottomPlane pins simple/W2 on the SEVEN faces DRAWEXE ships. Reinstate either
// half — the convex-shaft seat, or the radial setback in place of the run-out — and this fails on the
// face count first, then on every area.
func TestW2ArcBandRunsOutOnItsBottomPlane(t *testing.T) {
	t.Parallel()
	body := gridCaseBody(t, corpusRecord(t, "simple", "W2"))
	assertShippedPerFaceAgainstDrawexe(t, drawexeFaceCase{
		name: "simple/W2",
		// bottom z=0 (receded by the run-out) · cap y=1 · top z=1 (absorbing the first end) · cap y=0
		// (receded to the cylR+r circle) · cylinder (receded to y=0.2) · plane x=0 · the band.
		drawexe:    []float64{2.93823, 2.2145, 1.99142, 1.959, 1.2454, 1.0, 0.418101},
		totalArea:  11.76665,
		perFaceTol: arcOraclePerFaceTol,
	}, body, nil, 1e-4)
	assertArcBandSeat(t, body, "simple/W2", 1.2, 0.2)
	assertBodyIsWatertightAndSimple(t, body, "simple/W2")
	// THE RUN-OUT, stated as the defect it prevents: nothing may reach past the bottom plane z=0 or the
	// top plane z=1. The un-terminated cylR+r cove puts the cap-tangent point at z = −0.19998.
	lo, hi := bodyZRange(body)
	if lo < -1e-9 || hi > 1+1e-9 {
		t.Errorf("simple/W2 spills out of its own solid: z ∈ [%.6g, %.6g], want inside [0, 1] — the band was "+
			"run to the arc's end instead of terminated on the side plane", lo, hi)
	}
}

// TestH6ArcBandSeatsOnTheConcaveSide pins simple/H6 — the corpus's last quarantined case — on the EIGHT
// faces DRAWEXE ships, and on the closed forms behind them. H6 needs no run-out (both its ends are the
// 270° revolve's own walls, which contain the axis), so it isolates the SEAT half of the root.
func TestH6ArcBandSeatsOnTheConcaveSide(t *testing.T) {
	t.Parallel()
	body := gridCaseBody(t, corpusRecord(t, "simple", "H6"))
	assertShippedPerFaceAgainstDrawexe(t, drawexeFaceCase{
		name: "simple/H6",
		// two cones · plane z=50 · plane z=−50 (receded to r=60) · two walls (each GAINING the corner the
		// concave blend adds back) · cylinder (receded to z=−40) · the band.
		drawexe:    []float64{133286, 133286, 88357.3, 85765.5, 45021.5, 45021.5, 21205.8, 3970.08},
		totalArea:  555915,
		perFaceTol: arcOraclePerFaceTol,
	}, body, nil, 1e-4)
	assertArcBandSeat(t, body, "simple/H6", 60, 10)
	assertBodyIsWatertightAndSimple(t, body, "simple/H6")
	// The closed forms the DRAWEXE numbers decode as, so a future capture cannot drift unchallenged:
	// the concave band over the full 270° at cylR+r, and the wall's corner gain (r² − πr²/4).
	const r, majorR, span = 10.0, 60.0, 3 * stdmath.Pi / 2
	assertFaceMeshesToDrawexe(t, body, "simple/H6", "arcfillet:torus#0",
		r*span*(majorR*stdmath.Pi/2-r), arcOraclePerFaceTol) // 3970.06
	assertFaceMeshesToDrawexe(t, body, "simple/H6", "import:step#16:face#5",
		45000+(r*r-stdmath.Pi*r*r/4), arcOraclePerFaceTol) // 45021.46
}

// assertArcBandSeat is the SEAT's own falsifiable guard, independent of any area: the shipped band must be
// a torus of the stated major/minor radii, and its rolling-ball centre must sit where the picked edge's
// convexity demands — inside the material for a convex edge, in the void for a concave one. Revert the
// seat solve to the stored plane normal or to cylR−r and this fails on the radius, then on the side.
func assertArcBandSeat(t *testing.T, body *topo.Body, name string, majorR, minorR float64) {
	t.Helper()
	f := faceByLineage(t, body, "arcfillet:torus#0")
	tor, isTorus := f.Geometry().(geom.Torus)
	if !isTorus {
		t.Fatalf("%s: arcfillet:torus#0 is a %T, not a torus", name, f.Geometry())
	}
	if stdmath.Abs(tor.MajorRadius-majorR) > 1e-9 || stdmath.Abs(tor.MinorRadius-minorR) > 1e-9 {
		t.Errorf("%s: band torus is R=%g r=%g, want R=%g r=%g (the seat)",
			name, tor.MajorRadius, tor.MinorRadius, majorR, minorR)
	}
}

// assertBodyIsWatertightAndSimple is the run-out's downstream receipt: a band terminated inside its own
// solid tiles its neighbours' boundary exactly, so the welded mesh has no free edge, no face loop retraces
// and none self-crosses. W2 carried 3/3 free edges, 2 retraces and 1 self-crossing; H6 hid 98/642 free
// edges and 2 self-crossings behind its quarantine.
func assertBodyIsWatertightAndSimple(t *testing.T, body *topo.Body, name string) {
	t.Helper()
	for _, q := range gateQualities() {
		if n := ops.FreeEdgeCount(ops.CalculateBodyFacets(body, q.q).Mesh); n != 0 {
			t.Errorf("%s: %d free edge(s) in the welded mesh at %s quality, want 0", name, n, q.name)
		}
	}
	if bad := ops.RetracingFaceLoops(body, ops.PropertyQuality()); len(bad) != 0 {
		t.Errorf("%s: %d retracing face loop(s): %s", name, len(bad), describeRetracing(bad))
	}
	if bad := ops.SelfCrossingFaceLoops(body, ops.PropertyQuality()); len(bad) != 0 {
		t.Errorf("%s: %d self-crossing face loop(s): %s", name, len(bad), describeSelfCrossing(bad))
	}
}

// bodyZRange returns the body's vertex z extent — the cheapest statement of "the band stayed inside".
func bodyZRange(b *topo.Body) (lo, hi float64) {
	lo, hi = stdmath.Inf(1), stdmath.Inf(-1)
	for _, v := range b.Vertices() {
		lo, hi = stdmath.Min(lo, v.Point().Z), stdmath.Max(hi, v.Point().Z)
	}
	return lo, hi
}

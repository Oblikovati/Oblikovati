// SPDX-License-Identifier: GPL-2.0-only

package tessellate_test

import (
	"fmt"
	"testing"

	"oblikovati.org/kernel/brep"
	"oblikovati.org/kernel/diag"
	"oblikovati.org/kernel/geom"
	"oblikovati.org/kernel/ops"
	"oblikovati.org/kernel/ops/tessellate"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
)

// The TANGENT-PLANE FAMILY, swept over aspect ratio (Oblikovati/Oblikovati#3551).
//
// A torus cut by the plane tangent to its inner equator is the self-touching boundary this wave's work
// turns on, and one aspect ratio is not a family: R=5 r=2 was the only one in the corpus and it is the
// one ratio that never showed the defect. This row sweeps R/r from 100 down to 2 and holds the WHOLE
// SET's free-edge count, so a regression anywhere in the family fails here rather than waiting for
// someone to pick the right radii.
//
// Its number is a two-sided pin, not a bound, because the set still carries a residue this slice did
// not close and a residue that nobody is watching becomes a floor. A RISE is a regression. A FALL is a
// fix, and the pin comes down with the change that earned it.
//
// WHAT THE SET MEASURES, at the wave base and now. Review 2 built the same construction independently
// (19 aspect ratios × 2 operations × 2 facetings) and read 3170 free edges at the wave base `4b3b1819`
// and 364 at the #3551 commit — 85 % removed, with every "intersect at PropertyQuality" row of the
// family going to zero. This row's own set reads 368.

// tangentFamilyFreeEdges is the whole set's free-edge total at both facetings. See the file doc for
// what the residue is; TestTheTangentPlaneFamilyHolds also asserts that every torn row REPORTS its
// tear, so the residue is a capability gap and not a silent one.
//
// 368 → 728 (#3517's rebase onto this branch). It is a RISE, and it was RULED on rather than decided by
// whoever moved the number: #3517's coverShear tilts the frame the covering TRIANGULATES in by 1/1024,
// and on this family that costs exactly ONE row — R=50 r=1 intersect at PropertyQuality, 0 → 360 free
// edges. Bisected: every other row of the 76 is bit-identical with the shear on and off, the
// one-location invariant (dup = 0) holds throughout, and #3517's structured interior changes nothing
// here either way.
//
// THE SHEAR IS KEPT FOR DETERMINISM, NOT FOR A FREE-EDGE COUNT, and the ordering of those two reasons
// is the whole of the ruling. With the shear at 0 a covering's lattice cell has two EXACTLY equal
// diagonals — measured, 0.6444989408828234 against itself — so the structured interior's split has no
// tie-breaker left. "Output is byte-identical across runs and platforms: explicit total orders for every
// tie-break" is a ground rule, and a tie with nothing to break it is not a worse number, it is
// nondeterminism. That is a correctness failure, so it does not compete with a free-edge total at all.
//
// The other consequences of shear = 0 are real and would NOT have been enough on their own, which is
// why they are recorded second: #3542's closure is given back (5 of the near-pinch corpus's 16 rows
// carry 32 seam edges again), near-pinch crossing rods ∪ tears by 992 free edges at the fine faceting,
// and its face 6 area FALLS under refinement, 154.21847 → 104.32692. A capability handed back is a
// tradeable cost, and 992 against 360 compares free-edge totals across different families, which are
// not commensurable — neither is a reason to settle a design.
//
// WHAT MAKES THE RISE ACCEPTABLE rather than merely ruled is that R=50 r=1 is the SAME residue one
// aspect ratio along, not a new defect: see the density bullet in the split below, which now carries it
// beside R=100 r=1 with its recovery numbers. A defect that disappears under refinement while getting
// CHEAPER is a sampling limit.
//
// No value inside coverShear's own four-decade plateau avoids the row either — 1/65536, 1/16384, 1/4096
// and 1/1024 all read 728 — because breaking an exact tie is a discrete choice, not a magnitude. If the
// ruling is ever revisited, the change is one constant and this pin comes back to 368.
const tangentFamilyFreeEdges = 728

// tangentFamilyAspects is the swept set: R/r from 100 down to 2, spanning the thin-tube end where the
// covering's cells are most anisotropic and the fat-tube end where the pinch corner is widest.
func tangentFamilyAspects() []struct{ R, r float64 } {
	return []struct{ R, r float64 }{
		{100, 1}, {50, 1}, {30, 2}, {20, 1}, {10, 1}, {10, 1.5}, {10, 3}, {10, 5},
		{5, 0.5}, {5, 1}, {5, 1.25}, {5, 1.5}, {5, 1.75}, {5, 2}, {5, 2.5},
		{3, 1}, {2, 1}, {1, 0.5}, {0.5, 0.2},
	}
}

// tangentPlanePiece builds one piece of one aspect ratio: the torus (R, r) cut by y = R − r, the plane
// tangent to its inner equator. The box is sized from the torus so every ratio is cut the same way.
func tangentPlanePiece(t *testing.T, ringR, tubeR float64, op ops.PartFeatureOperation) (*topo.Body, error) {
	t.Helper()
	ring, err := brep.SolidTorus(math.P3(0, 0, 0), math.V3(0, 0, 1), ringR, tubeR, "ring")
	if err != nil {
		return nil, err
	}
	far := 20 * (ringR + tubeR)
	box, err := brep.SolidBlock(math.P3(-far, ringR-tubeR, -far), math.P3(far, far, far), "box")
	if err != nil {
		return nil, err
	}
	return ops.Boolean(op, ring, box)
}

// tangentFamilyTally is what one walk of the sweep collects. The three questions are asked on ONE walk
// rather than in three rows: each row rebuilt all 38 bodies and re-tessellated all 76, which cost
// 440 s of the six-package tier between them (review 3, M2). Sharing the built bodies across parallel
// ROWS is the other way to save it and is not taken: a face memoises its trim and metric scale while it
// is tessellated, so two parallel rows over one body would race on writes the gate does not look for.
type tangentFamilyTally struct {
	rows, freeEdges, torn, declinedEdges, chartFaces, duplicates int
}

// TestTheTangentPlaneFamilyHolds is the sweep, and it asks all three questions on one walk: the whole
// set's free-edge total, the split of what is left by cause, and the one-location invariant across
// every ratio rather than the four the corpus happens to carry.
func TestTheTangentPlaneFamilyHolds(t *testing.T) {
	if testing.Short() {
		t.Skip("corpus tier (~3 min): `make test-corpus`")
	}
	t.Parallel()
	var got tangentFamilyTally
	forEachTangentPlanePiece(t, func(name string, b *topo.Body, q ops.Quality, qName string) {
		got.rows++
		mesh, _ := tessellate.TessellateBody(b, q)
		tallyOneTangentRow(t, &got, name+" "+qName, b, mesh, q)
	})
	assertTangentFamilyTally(t, got)
}

// tallyOneTangentRow adds one (ratio, operation, faceting) to the tally and makes the two assertions
// that are per-row rather than per-sweep: a torn row must REPORT its tear, and every torus face must
// lay one vertex per covering location.
func tallyOneTangentRow(t *testing.T, got *tangentFamilyTally, row string, b *topo.Body, mesh *tessellate.Mesh, q ops.Quality) {
	t.Helper()
	free := tessellate.FreeEdgeCount(mesh)
	got.freeEdges += free
	if free > 0 {
		got.torn++
		if !tangentMeshReports(mesh, tessellate.CodeMeshNotWatertight) {
			t.Errorf("%s tears by %d and does not report %q; a degradation that reaches nobody is worse "+
				"than the tear: %v", row, free, tessellate.CodeMeshNotWatertight, mesh.Diagnostics)
		}
		if tangentMeshReports(mesh, tessellate.CodeChartMesherDeclined) {
			got.declinedEdges += free
		}
	}
	for _, f := range b.Faces() {
		if _, isTorus := f.Geometry().(geom.Torus); !isTorus {
			continue
		}
		dup, verts, ok := tessellate.ChartCoveringLocationCount(f, q)
		if !ok {
			continue
		}
		got.chartFaces++
		got.duplicates += dup
		if dup != 0 {
			t.Errorf("%s: the covering lays %d of its %d vertices on top of another (#3551)", row, dup, verts)
		}
	}
}

// assertTangentFamilyTally holds the three per-sweep numbers, each of which fails in both directions,
// and refuses a walk that covered nothing.
func assertTangentFamilyTally(t *testing.T, got tangentFamilyTally) {
	t.Helper()
	if want := 4 * len(tangentFamilyAspects()); got.rows != want {
		t.Fatalf("the sweep presented %d rows; want %d (%d ratios × 2 operations × 2 facetings)",
			got.rows, want, len(tangentFamilyAspects()))
	}
	if got.chartFaces == 0 {
		t.Fatal("no torus face of the family reached the chart covering; the one-location invariant covers nothing")
	}
	if got.freeEdges != tangentFamilyFreeEdges {
		t.Errorf("the tangent-plane family meshes with %d free edges over the whole sweep; the "+
			"measurement is %d. A rise is a regression. A FALL is a fix — bring this pin down with it",
			got.freeEdges, tangentFamilyFreeEdges)
	}
	if got.torn != tangentFamilyTornRows || got.declinedEdges != tangentFamilyDeclinedEdges {
		t.Errorf("%d rows of the sweep tear, %d of their free edges behind a chart-mesher decline; the "+
			"measurement is %d rows and %d edges. The split matters: one number is the covering's density "+
			"limit and the other is a shared chord at the pinch", got.torn, got.declinedEdges,
			tangentFamilyTornRows, tangentFamilyDeclinedEdges)
	}
	if got.torn == 0 {
		t.Error("no row of the sweep tears, so the reporting assertion covers nothing — bring the pins down with it")
	}
}

// forEachTangentPlanePiece runs visit on every (aspect ratio, operation, faceting) of the sweep. A
// ratio whose boolean is REFUSED is skipped with a name rather than silently: that is #3552's gap, and
// a ratio that stops building would otherwise quietly shrink this sweep.
func forEachTangentPlanePiece(t *testing.T, visit func(name string, b *topo.Body, q ops.Quality, qName string)) {
	t.Helper()
	for _, g := range tangentFamilyAspects() {
		for _, op := range []ops.PartFeatureOperation{ops.Cut, ops.Intersect} {
			name := fmt.Sprintf("R=%g r=%g %v", g.R, g.r, op)
			b, err := tangentPlanePiece(t, g.R, g.r, op)
			if err != nil {
				t.Errorf("%s: the boolean refused a tangent-plane cut this sweep counts on: %v", name, err)
				continue
			}
			for _, gq := range []struct {
				n string
				q ops.Quality
			}{{"default", ops.DefaultQuality()}, {"property", ops.PropertyQuality()}} {
				visit(name, b, gq.q, gq.n)
			}
		}
	}
}

// tangentFamilyTornRows and tangentFamilyDeclinedEdges split the residue into the two DIFFERENT
// defects behind it, so neither can grow inside the other's number.
//
//   - ONE row is the chart mesher's: R=100 r=1 intersect at PropertyQuality, 360 free edges. Its torus
//     face is REFUSED by the mesher's own rim gate (3 unpaired edges that are no rim segment) and the
//     router falls back to the surface's whole domain, which is what tears. Measured, it is a covering
//     DENSITY limit and not a defect of this slice's kind: the face meshes watertight with extra = 0 and
//     no diagnostic at all once the chord reaches 2e-4 (272 free edges at chord 0.05, 360 at 1e-3, 400
//     at 5e-4, then 0 at 2e-4, 1e-4 and 5e-5 — and the triangle count FALLS 524644 → 40668 there,
//     because the decline and its whole-domain fall-through both stop). The covering's cells are
//     25.00 : 1 anisotropic at the quality's own chord and 12.50 : 1 where it goes watertight, and
//     `balancedCoverGrid` does not act on any row of this family — its raw and balanced grids are equal
//     throughout, because it refines only a FLOORED axis and both of these are chord-subdivided. So
//     squaring these cells means refining a CHORD-sized axis, which is the case that file records as
//     tried and reverted (it drove the figure-eight band from 110.947 mm² to the whole torus). Closing
//     this row means revisiting that decision with its own corpus, not widening anything here.
//
//     A SECOND row joined this bullet with #3517's rebase, and it belongs here rather than in a total:
//     R=50 r=1 intersect at PropertyQuality, also 360 free edges, refused by the same rim gate (3
//     unpaired edges that are no rim segment). It is the same limit at the next aspect ratio down, and
//     it behaves the same way under refinement — 292 free edges at chord 0.05, 360 at 1e-3, then 0 at
//     5e-4, 2e-4 and 1e-4, with the triangle count FALLING 262500 → 30504 where it recovers, because
//     the decline and its whole-domain fall-through both stop. It recovers at a COARSER chord than
//     R=100 r=1 does (5e-4 against 2e-4), so it sits nearer the edge: coverShear moves the edge of this
//     density limit from R/r = 100 to R/r = 50, it does not open a new one.
//
//     WHAT CLOSES BOTH is a facet count, and naming it is the point of putting them together.
//     adaptiveParams subdivides by HALVING, so it can only land on a power of two: the general path
//     samples ~1.9× more than the tolerance demands on some faces and leaves these two short, because
//     one dyadic ladder cannot be right for a cell 25 : 1 anisotropic. A non-dyadic per-axis
//     discretizer derived from tolerance is what takes both rows to zero without refining a chord-sized
//     axis, and it moves every curved-face pin in the repo (ADR-0061 §R4.5). Until then this bullet is
//     where the family's density residue is counted, and it is two rows.
//
//   - FIVE rows, 8 edges between them: a SHARED CHORD at the pinch, and the per-face tally is what
//     names it. Every one of the eight is an edge of degree FOUR whose four uses split two-and-two
//     between the cut plane and the torus — `{box:face#2 (Plane): 2, ring:face#0 (Torus): 2}` on every
//     row, never three-and-one — and within each patch the two uses run in opposite directions, so the
//     edge is an ordinary INTERIOR edge of both. Neither patch is doubled on its own; the two patches
//     draw the same segment. It is not a weld artefact: the raw endpoints that weld together are
//     bit-identical (spread 0.0 against welds of 1.4e-7 and 1.5e-8). The segment is rim-to-rim, between
//     two points of the figure eight on opposite branches at one ring angle, so it lies in the cut plane
//     y = R − r by construction — which is why the plane's chord and the torus's chord ARE the same
//     segment. The torus face draws it because near the pinch the kept strip is thinner than the
//     covering's across-cell and the chart mesher spans it in one triangle; refining does not remove it,
//     it moves the chord closer to the pinch (z = ±0.707 coarse, ±0.195 fine on R=50 r=1), because the
//     strip is arbitrarily thin arbitrarily close to the tangency.
//
//     An earlier version of this comment called these edges the planar lid's and "NOT the chart
//     mesher's at all". That was wrong and the way it was wrong is worth keeping: the evidence offered
//     for it — the torus face is not declined, its rim mismatch is (0, 0), its area is right — is all
//     true and none of it discriminates between "the lid doubled" and "the two patches share a chord".
//     Evidence consistent with a claim is not evidence for it (review 3, I1). The tally discriminates;
//     it says the chart mesher is a co-equal contributor.
//
// Neither is the duplicate covering location #3551 fixed: the same walk holds dup = 0 on every torus
// face of every ratio, torn rows included.
//
// 6 → 7 rows and 360 → 720 declined edges (#3517's rebase): R=50 r=1 intersect at PropertyQuality joins
// the FIRST bullet — the covering density limit — which names it there with its own recovery numbers
// rather than letting it disappear into a total. The five-row shared-chord bullet is unchanged at 8
// edges between them, and that is the property this split exists for: one defect must not be able to
// grow inside another's number, and the rise is entirely inside the density number.
const (
	tangentFamilyTornRows      = 7
	tangentFamilyDeclinedEdges = 720
)

// tangentMeshReports reports whether a mesh carries the given code at Defect severity.
func tangentMeshReports(m *tessellate.Mesh, code diag.Code) bool {
	for _, d := range m.Diagnostics {
		if d.Code == code && d.Severity == diag.Defect {
			return true
		}
	}
	return false
}

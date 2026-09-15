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
// what the residue is and TestTheTangentPlaneFamilyResidueIsRefusedAndReported for why it is not silent.
const tangentFamilyFreeEdges = 368

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

// TestTheTangentPlaneFamilyMeshesWatertightAcrossItsAspectRatios is the sweep's ratchet.
func TestTheTangentPlaneFamilyMeshesWatertightAcrossItsAspectRatios(t *testing.T) {
	if testing.Short() {
		t.Skip("corpus tier (~4 min): `make test-corpus`")
	}
	t.Parallel()
	total, rows := 0, 0
	forEachTangentPlanePiece(t, func(_ string, b *topo.Body, q ops.Quality, _ string) {
		rows++
		mesh, _ := tessellate.TessellateBody(b, q)
		total += tessellate.FreeEdgeCount(mesh)
	})
	if rows != 4*len(tangentFamilyAspects()) {
		t.Fatalf("the sweep presented %d rows; want %d (%d ratios × 2 operations × 2 facetings)",
			rows, 4*len(tangentFamilyAspects()), len(tangentFamilyAspects()))
	}
	if total != tangentFamilyFreeEdges {
		t.Errorf("the tangent-plane family meshes with %d free edges over the whole sweep; the "+
			"measurement is %d. A rise is a regression. A FALL is a fix — bring this pin down with it",
			total, tangentFamilyFreeEdges)
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
//     DENSITY limit and not a defect of this slice's kind: the covering's cells on a torus of R/r = 100
//     are 25:1 anisotropic at the quality's own chord, and the face meshes watertight with extra = 0 and
//     no diagnostic at all once the chord reaches 2e-4 (free edges 400 at chord 5e-4, then 0 at 2e-4,
//     1e-4 and 5e-5). Cells that square up is `balancedCoverGrid`, which refines only a FLOORED axis on
//     purpose — re-balancing a torus's tube against its ring is recorded there as tried and reverted,
//     because it drove the figure-eight band from 110.947 mm² to the whole torus. So closing this row
//     means revisiting that decision with its own corpus, not widening anything here.
//
//   - FIVE rows are the LID's, not the torus's, and they are 8 free edges between them — 1 or 2 each,
//     and every one is an OVER-MERGED edge (three or more triangles) on `box:face#2` and `ring:face#0`
//     rather than a crack. Their torus faces are clean: rim-only triangles 0, rim mismatch (0, 0), not
//     declined, area right. What doubles is the planar cap whose own boundary is the figure eight —
//     the self-touching boundary in the PLANAR mesher, which is the other half of #3519's original
//     "the FIG8 lid's tangential self-touch" and a different mesher from this one.
//
// Neither is the duplicate covering location #3551 fixed: TestTheTangentPlaneFamilyLaysOneVertexPerLocation
// holds dup = 0 on every row of the sweep, torn ones included.
const (
	tangentFamilyTornRows      = 6
	tangentFamilyDeclinedEdges = 360
)

// TestTheTangentPlaneFamilyResidueIsRefusedAndReported is the half that says the residue is a
// CAPABILITY gap and not a silent one: every torn row of the sweep reports the tear as a Defect, and
// the one row whose chart face is refused says THAT too, so a reader who meets either is told what gave
// up. It is the ground rules' "never degrade silently" applied to what is left rather than to what is
// fixed, and it is what makes the count above a residue one can live beside.
func TestTheTangentPlaneFamilyResidueIsRefusedAndReported(t *testing.T) {
	if testing.Short() {
		t.Skip("corpus tier (~4 min): `make test-corpus`")
	}
	t.Parallel()
	torn, declinedEdges := 0, 0
	forEachTangentPlanePiece(t, func(name string, b *topo.Body, q ops.Quality, qName string) {
		mesh, _ := tessellate.TessellateBody(b, q)
		free := tessellate.FreeEdgeCount(mesh)
		if free == 0 {
			return
		}
		torn++
		if !tangentMeshReports(mesh, tessellate.CodeMeshNotWatertight) {
			t.Errorf("%s %s tears by %d and does not report %q; a degradation that reaches nobody is "+
				"worse than the tear: %v", name, qName, free, tessellate.CodeMeshNotWatertight, mesh.Diagnostics)
		}
		if tangentMeshReports(mesh, tessellate.CodeChartMesherDeclined) {
			declinedEdges += free
		}
	})
	if torn != tangentFamilyTornRows || declinedEdges != tangentFamilyDeclinedEdges {
		t.Errorf("%d rows of the sweep tear, %d of their free edges behind a chart-mesher decline; the "+
			"measurement is %d rows and %d edges. The split matters: one number is the covering's density "+
			"limit and the other is the planar lid's self-touch", torn, declinedEdges,
			tangentFamilyTornRows, tangentFamilyDeclinedEdges)
	}
}

// tangentMeshReports reports whether a mesh carries the given code at Defect severity.
func tangentMeshReports(m *tessellate.Mesh, code diag.Code) bool {
	for _, d := range m.Diagnostics {
		if d.Code == code && d.Severity == diag.Defect {
			return true
		}
	}
	return false
}

// TestTheTangentPlaneFamilyLaysOneVertexPerLocation is the invariant #3551 fixed, held across the whole
// family rather than on the four ratios the corpus happens to carry. It is what says the residue above
// is a DIFFERENT defect: every torus face of every ratio lays one vertex per covering location, torn
// rows included.
func TestTheTangentPlaneFamilyLaysOneVertexPerLocation(t *testing.T) {
	if testing.Short() {
		t.Skip("corpus tier (~4 min): `make test-corpus`")
	}
	t.Parallel()
	checked := 0
	forEachTangentPlanePiece(t, func(name string, b *topo.Body, q ops.Quality, qName string) {
		for _, f := range b.Faces() {
			if _, isTorus := f.Geometry().(geom.Torus); !isTorus {
				continue
			}
			dup, verts, ok := tessellate.ChartCoveringLocationCount(f, q)
			if !ok {
				continue
			}
			checked++
			if dup != 0 {
				t.Errorf("%s %s: the covering lays %d of its %d vertices on top of another (#3551)",
					name, qName, dup, verts)
			}
		}
	})
	if checked == 0 {
		t.Error("no torus face of the family reached the chart covering; the invariant covers nothing")
	}
}

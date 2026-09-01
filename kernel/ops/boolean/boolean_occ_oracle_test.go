// SPDX-License-Identifier: GPL-2.0-only

package boolean_test

import (
	"encoding/json"
	stdmath "math"
	"os"
	"path/filepath"
	"testing"

	"oblikovati.org/kernel/geom"
	"oblikovati.org/kernel/ops"
	"oblikovati.org/kernel/ops/query"
	"oblikovati.org/kernel/topo"
)

// OCC getMass boolean oracle (M2 Phase 3, Oblikovati/Oblikovati#1336 — the independent ground-truth
// validation #1320 asks for). The exact analytic curved booleans are checked against OpenCASCADE's exact
// volume (BRepGProp::VolumeProperties, "getMass") on the SAME inputs. testdata/occ_boolean_oracle.json
// holds {case: volume} produced offline by experiments/occ-boolean-oracle (a tiny OCCT C++ driver), so the
// test needs no OCCT at run time. The cases ARE curvedExactCases — the same matrix the exactness guard
// runs — so each documented curved boolean is validated both for staying exact AND for matching OCC mass.
//
// The budget depends on how the result's volume was measured. Where it integrates the ANALYTIC B-rep it
// must match OCC outright (occAnalyticTolerance). Where the analytic integrator still declines and the
// tessellation measures it, the volume sits UNDER OCC's exact value by the inscribed-facet chord deficit —
// never far over — and the per-case budget below reflects how much curved area that result carries.

func loadOCCBooleanOracle(t *testing.T) map[string]float64 {
	t.Helper()
	raw, err := os.ReadFile(filepath.Join("testdata", "occ_boolean_oracle.json"))
	if err != nil {
		t.Fatalf("read occ_boolean_oracle.json: %v", err)
	}
	oracle := map[string]float64{}
	if err := json.Unmarshal(raw, &oracle); err != nil {
		t.Fatalf("parse occ_boolean_oracle.json: %v", err)
	}
	return oracle
}

// occBooleanTolerance is the allowed relative volume error vs OCC getMass. Our value is the TESSELLATED
// B-rep's volume (inscribed facets at DefaultQuality), so it sits below OCC's exact analytic value by the
// chord deficit — the more curved area a result carries, the larger the (one-sided) deficit. The B-rep
// itself is exact; this budget is the meshing error, matching the per-shape budgets the STEP OCC oracle
// already uses (singly-curved solids ~5%). A result that exceeds it has a geometry error, not mere faceting.
func occBooleanTolerance(name string) float64 {
	switch name {
	case "box − cylinder (drilled plate)", "cylinder boss ∪", "cylinder − box tunnel":
		return 0.015 // planar-dominant: only one cylinder wall to facet
	case "torus ∩ box (axis-∥ oval cap)":
		// A small DOUBLY-curved cap (the torus is the suite's only doubly-curved surface): its facets
		// deflect in both u and v, so the one-sided inscribed deficit at DefaultQuality runs ~7% — larger
		// than a singly-curved band of the same size. Proven pure faceting: monotone-convergent to OCC
		// (rel falls 0.07→0.0006 as the chord tolerance tightens 0.05→0.0002), so the B-rep is exact.
		return 0.08
	default:
		return 0.05 // a singly-curved result (cylinder/cone bands, saddle lobes) at DefaultQuality
	}
}

// occAnalyticTolerance is the budget for a result whose volume comes from the ANALYTIC B-rep
// (M48/C3 #3453). There is no faceting deficit to absorb then — the integral agrees with the closed
// form to ~1e-11 relative — so the only slack needed is for OCC's own rounding in the stored oracle,
// which carries four decimals. A case that misses THIS is a geometry error, and the loose budgets
// above no longer hide it.
const occAnalyticTolerance = 1e-4

// TestCurvedBooleanVolumesMatchOCC is the corpus gate for the analytic boolean. Each case is held to
// the tight analytic budget once its result integrates over the exact B-rep, and only a result that
// still falls back to the tessellation keeps the old faceting budget — so the log below reads as a
// scoreboard of which booleans are analytic yet, and tightens on its own as more of them become so.
func TestCurvedBooleanVolumesMatchOCC(t *testing.T) {
	if testing.Short() {
		t.Skip("corpus tier (~9s): `make test-corpus`")
	}
	t.Parallel()
	oracle := loadOCCBooleanOracle(t)
	for _, c := range curvedExactCases() {
		t.Run(c.name, func(t *testing.T) {
			want, ok := oracle[c.name]
			if !ok {
				t.Fatalf("no OCC oracle entry for %q (regenerate testdata/occ_boolean_oracle.json)", c.name)
			}
			target, tool := c.build(t)
			res, err := ops.Boolean(c.op, target, tool)
			if err != nil {
				t.Fatalf("%s: Boolean(%s): %v", c.name, c.op, err)
			}
			got := query.BodyGeometryProperties(res, ops.DefaultQuality()).Volume
			budget, source := occBudgetFor(c.name, want, target, tool, res)
			rel := stdmath.Abs(got-want) / want
			t.Logf("%-32s ours=%.6f  occ=%.6f  rel=%.6f (%s budget %.4f)", c.name, got, want, rel, source, budget)
			if rel > budget {
				t.Errorf("%s: volume %.6f vs OCC getMass %.6f (rel %.6f > %.6f, %s)", c.name, got, want, rel, budget, source)
			}
		})
	}
}

// occBudgetFor picks the budget by which regime the result is in, and names it so a failure says
// which. A result DEMOTED to a faceted fallback is all-planar, so the analytic integrator accepts it
// happily — its volume is the exact volume of an inscribed polyhedron, which is not the exact volume
// of the shape. Asking whether the integrator accepted the body therefore cannot separate the
// regimes; asking whether the result still carries the curved geometry its operands had, can.
func occBudgetFor(name string, want float64, target, tool, res *topo.Body) (float64, string) {
	if curvedFaceTotal(res) == 0 && curvedFaceTotal(target)+curvedFaceTotal(tool) > 0 {
		return occBooleanTolerance(name), "faceted" // demoted: the curved operands left no curved face
	}
	if _, analytic := query.AnalyticGeometryProperties(res); analytic {
		// A body measured analytically is held to the exact budget, WIDENED by what its own boundary
		// approximation can account for: an edge built from a marched intersection carries its achieved
		// deviation, and a body whose boundary is only good to 4.5e-4 cannot report a volume better than
		// that however exactly it is integrated. An all-analytic body adds nothing here (#3489).
		if slack := query.AchievedBoundarySlack(res) / stdmath.Abs(want); slack > occAnalyticTolerance {
			return slack, "analytic (boundary-limited)"
		}
		return occAnalyticTolerance, "analytic"
	}
	return occBooleanTolerance(name), "tessellated"
}

// curvedFaceTotal counts a body's non-planar faces — the analytic geometry a faceted fallback loses.
func curvedFaceTotal(b *topo.Body) int {
	if b == nil {
		return 0
	}
	n := 0
	for _, f := range b.Faces() {
		if _, planar := f.Geometry().(geom.Plane); !planar {
			n++
		}
	}
	return n
}

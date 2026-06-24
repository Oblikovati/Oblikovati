// SPDX-License-Identifier: GPL-2.0-only

package ops_test

import (
	"encoding/json"
	stdmath "math"
	"os"
	"path/filepath"
	"testing"

	"oblikovati.org/kernel/ops"
)

// OCC getMass boolean oracle (M2 Phase 3, Oblikovati/Oblikovati#1336 — the independent ground-truth
// validation #1320 asks for). The exact analytic curved booleans are checked against OpenCASCADE's exact
// volume (BRepGProp::VolumeProperties, "getMass") on the SAME inputs. testdata/occ_boolean_oracle.json
// holds {case: volume} produced offline by experiments/occ-boolean-oracle (a tiny OCCT C++ driver), so the
// test needs no OCCT at run time. The cases ARE curvedExactCases — the same matrix the exactness guard
// runs — so each documented curved boolean is validated both for staying exact AND for matching OCC mass.
//
// Our volume is the tessellated B-rep's (inscribed facets at DefaultQuality), so it sits slightly UNDER
// OCC's exact analytic value by the chord deficit — never far over. The per-case budget reflects how much
// curved area each result carries (a doubly-curved or many-arc result faceted more than a singly-curved
// one); it is one-sided in spirit but written as |rel| for simplicity.

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
	default:
		return 0.05 // a singly-curved result (cylinder/cone bands, saddle lobes) at DefaultQuality
	}
}

func TestCurvedBooleanVolumesMatchOCC(t *testing.T) {
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
			got := ops.BodyGeometryProperties(res, ops.DefaultQuality()).Volume
			rel := stdmath.Abs(got-want) / want
			t.Logf("%-32s ours=%.4f  occ=%.4f  rel=%.4f (budget %.3f)", c.name, got, want, rel, occBooleanTolerance(c.name))
			if rel > occBooleanTolerance(c.name) {
				t.Errorf("%s: volume %.4f vs OCC getMass %.4f (rel %.4f > %.4f)", c.name, got, want, rel, occBooleanTolerance(c.name))
			}
		})
	}
}

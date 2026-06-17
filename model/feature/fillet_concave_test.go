// SPDX-License-Identifier: GPL-2.0-only

package feature

import (
	stdmath "math"
	"testing"

	"oblikovati.org/api/types"
	"oblikovati.org/kernel/ops"
	"oblikovati.org/kernel/topo"
)

// filletConcaved fillets the L-extrude's concave edge (radius r) with the given strategy and
// returns the result, failing on a sick feature or an invalid solid. lExtrude (chamfer_concave_test.go)
// is the small L: a 2×2 square less its 1×1 corner, height 1 — volume 3, concave edge at (1,1).
func filletConcaved(t *testing.T, r float64, strategy types.FilletConcaveStrategy) *topo.Body {
	t.Helper()
	body, edge := lExtrude(t)
	fs := NewPartFeatures(nil, nil)
	NewBaseFeatures(fs).AddBase(body)
	fl := NewDressUpFeatures(fs).AddFilletConcave([][]byte{edge}, func() float64 { return r }, strategy)
	fs.Recompute()
	if !fl.Health().OK() {
		t.Fatalf("concave fillet (%v) sick: %+v", strategy, fl.Health())
	}
	res := fs.Result()[0]
	if rep := ops.Validate(res); !rep.Valid || !res.IsSolid() {
		t.Fatalf("concave fillet (%v) not a valid solid: %+v", strategy, rep)
	}
	return res
}

// concaveFilletNotch is the rolling-ball fillet's quarter-corner cross-section for a 90° concave
// edge of radius r: the square corner minus the inscribed quarter-disc (r² − πr²/4).
func concaveFilletNotch(r float64) float64 { return r*r - stdmath.Pi*r*r/4 }

// TestFilletConcaveOutwardFills is the regression for the inverted-fillet bug: an internal
// (concave) edge filleted outward must FILL the inside corner with an exact rolling-ball cylinder
// face, adding the notch cross-section over the edge length — not scoop material out as the
// convex-only path (which left the centre in the material) did. The L is volume 3, edge length 1.
func TestFilletConcaveOutwardFills(t *testing.T) {
	const r = 0.6
	res := filletConcaved(t, r, types.FilletConcaveOutward)
	want := 3 + concaveFilletNotch(r) // L volume 3 + fillet fill over edge length 1
	got := ops.BodyGeometryProperties(res, ops.Quality{ChordTolerance: 1e-4}).Volume
	if stdmath.Abs(got-want) > 2e-3 {
		t.Errorf("outward concave fillet volume = %g, want %g±2e-3 (notch should be filled)", got, want)
	}
}

// TestFilletConcaveInwardDegenerate documents that an inward recess is NOT geometrically realizable
// at a generic concave edge: the rolling ball rolls into the material, but the two bounded faces
// extend only toward the void, so its tangent points fall off them. The feature goes sick honestly
// rather than emitting a malformed solid. (A concave edge's valid fillet is the outward fill.)
func TestFilletConcaveInwardDegenerate(t *testing.T) {
	body, edge := lExtrude(t)
	fs := NewPartFeatures(nil, nil)
	NewBaseFeatures(fs).AddBase(body)
	fl := NewDressUpFeatures(fs).AddFilletConcave([][]byte{edge}, func() float64 { return 0.6 }, types.FilletConcaveInward)
	fs.Recompute()
	if fl.Health().OK() {
		t.Error("inward fillet of this L should be sick (recess tangents fall off the bounded faces)")
	}
}

// TestFilletConcaveOutwardWatertight verifies the filled concave fillet validates at the
// tessellating geometry level across qualities — tessellation correctness is paramount (a
// faceted-but-open mesh is a defect). ValidateBodyEntities at CheckGeometry tessellates and runs
// the manifold/orientation/closure + self-intersection checks.
func TestFilletConcaveOutwardWatertight(t *testing.T) {
	res := filletConcaved(t, 0.6, types.FilletConcaveOutward)
	for _, tol := range []float64{0.05, 1e-2, 1e-3} {
		if ok, problems := ops.ValidateBodyEntities(res, ops.CheckGeometry, ops.Quality{ChordTolerance: tol}); !ok {
			t.Errorf("outward concave fillet at tol %g is not watertight/valid: %+v", tol, problems)
		}
	}
}

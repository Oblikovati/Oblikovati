// SPDX-License-Identifier: GPL-2.0-only

package ops_test

import (
	"strings"
	"testing"

	"oblikovati.org/kernel/ops"
	"oblikovati.org/kernel/topo"
)

// Comprehensive coverage of the three fillet CORNER strategies (miter / round / setback) and the
// run-out they are built on. The strategy controls how a vertex where two filleted edges meet — its
// third edge sharp — is treated:
//   - miter:   the two cylinders mutually trim at a crease (exact rolling ball; no sphere);
//   - round:   the sharp third edge is rounded at constant radius → a full 3-edge sphere;
//   - setback: the sharp third edge is tapered to a run-out → a smooth sphere that fades to sharp.
//
// The fillet-of-a-fillet (refillet) cases live with their features: TestFilletEdgesRoutesArc /
// TestFilletCurvedAdjacentReported (arc cap → torus, smooth tangent line → rejected) and
// TestFilletEdgesRoutesRim / TestRimFilletTorusBand (cylinder rim → toroidal band).

// assertWatertight fails t if res has any open mesh edges across coarse-to-fine tolerances.
func assertWatertight(t *testing.T, res *topo.Body, label string) {
	t.Helper()
	for _, tol := range []float64{0.05, 1e-2, 1e-3} {
		m, _ := ops.TessellateBody(res, ops.Quality{ChordTolerance: tol})
		if open := meshOpenEdges(m); open != 0 {
			t.Errorf("%s at tol %g: %d open edges", label, tol, open)
		}
	}
}

// topLoopKeys returns the four top edges of a box (both endpoints at z == top).
func topLoopKeys(t *testing.T, b *topo.Body, top float64) [][]byte {
	t.Helper()
	var keys [][]byte
	for _, e := range b.Edges() {
		if a, c := e.StartVertex().Point(), e.EndVertex().Point(); a.Z == top && c.Z == top {
			keys = append(keys, e.ReferenceKey())
		}
	}
	if len(keys) != 4 {
		t.Fatalf("expected 4 top edges at z=%g, found %d", top, len(keys))
	}
	return keys
}

// cornerStrategyOf maps a wire-style spelling to the kernel strategy (the test's own small table).
func cornerStrategyOf(name string) ops.CornerStrategy {
	switch name {
	case "round":
		return ops.CornerRound
	case "setback":
		return ops.CornerSetback
	default:
		return ops.CornerMiter
	}
}

// cornerStrategyPicks selects the fillet edges for a scenario: two of one vertex's three edges
// ("oneCorner"), or the whole top loop ("allTop"), all at radius 0.3.
func cornerStrategyPicks(t *testing.T, box *topo.Body, scenario string) []ops.EdgeFilletRadii {
	t.Helper()
	keys := topLoopKeys(t, box, 2)
	if scenario == "oneCorner" {
		keys = cornerEdgeKeys(t, box)[:2]
	}
	picks := make([]ops.EdgeFilletRadii, len(keys))
	for i, k := range keys {
		picks[i] = ops.EdgeFilletRadii{Key: k, R0: 0.3, R1: 0.3}
	}
	return picks
}

// TestFilletCornerStrategiesValidWatertight drives every strategy across a single-corner selection
// (two of a vertex's three edges) and a whole top-loop selection (four corners), asserting a valid
// watertight solid, the expected sphere count, and that material was removed.
func TestFilletCornerStrategiesValidWatertight(t *testing.T) {
	cases := []struct {
		strategy   string
		scenario   string
		wantSphere int
	}{
		{"miter", "oneCorner", 0}, {"round", "oneCorner", 1}, {"setback", "oneCorner", 1},
		{"miter", "allTop", 0}, {"round", "allTop", 4}, {"setback", "allTop", 4},
	}
	for _, c := range cases {
		t.Run(c.strategy+"-"+c.scenario, func(t *testing.T) {
			box := shellBox(2, 2, 2)
			res, err := ops.FilletEdgesCorner(box, cornerStrategyPicks(t, box, c.scenario), cornerStrategyOf(c.strategy), ops.FillConcaveOutward)
			if err != nil {
				t.Fatalf("%s/%s: %v", c.strategy, c.scenario, err)
			}
			if r := ops.Validate(res); !r.Valid || !res.IsSolid() {
				t.Fatalf("%s/%s not a valid solid: %+v", c.strategy, c.scenario, r)
			}
			if s := hasSphereFaces(res); s != c.wantSphere {
				t.Errorf("%s/%s: %d sphere faces, want %d", c.strategy, c.scenario, s, c.wantSphere)
			}
			assertWatertight(t, res, c.strategy+"/"+c.scenario)
			if v := ops.BodyGeometryProperties(res, ops.Quality{ChordTolerance: 1e-3}).Volume; v >= 8 {
				t.Errorf("%s/%s volume = %g, want < 8 (material removed)", c.strategy, c.scenario, v)
			}
		})
	}
}

// TestFilletCornerVolumeOrdering checks the strategies differ as expected on the SAME selection: a
// miter keeps the most material (it only rounds the two picked edges); a round removes the most (it
// rounds the third edge full length too); a setback sits strictly between (it rounds the third edge
// only near the corner, tapering to a run-out). So vol(miter) > vol(setback) > vol(round).
func TestFilletCornerVolumeOrdering(t *testing.T) {
	vol := func(strategy string) float64 {
		box := shellBox(2, 2, 2)
		res, err := ops.FilletEdgesCorner(box, cornerStrategyPicks(t, box, "oneCorner"), cornerStrategyOf(strategy), ops.FillConcaveOutward)
		if err != nil {
			t.Fatalf("%s: %v", strategy, err)
		}
		return ops.BodyGeometryProperties(res, ops.Quality{ChordTolerance: 1e-3}).Volume
	}
	miter, setback, round := vol("miter"), vol("setback"), vol("round")
	if !(miter > setback && setback > round) {
		t.Errorf("expected vol(miter) > vol(setback) > vol(round), got %g, %g, %g", miter, setback, round)
	}
}

// TestFilletRunOutBothEndsZeroRejected: a fillet of radius 0 at BOTH ends is no fillet — a precise
// error, not a degenerate body (a single end at 0 is the valid run-out, TestFilletRunOutToZero).
func TestFilletRunOutBothEndsZeroRejected(t *testing.T) {
	box := shellBox(2, 2, 2)
	_, err := ops.FilletEdgesVarying(box, []ops.EdgeFilletRadii{{Key: verticalEdgeKey(t, box), R0: 0, R1: 0}})
	if err == nil || !strings.Contains(err.Error(), "at least one") {
		t.Fatalf("both-ends-zero err = %v, want the >=0 / at-least-one rejection", err)
	}
}

// TestFilletCornerRadiusMismatchRejected: at a shared corner every edge must carry the SAME radius
// there (the corner sphere has one radius) — differing corner radii are a precise error.
func TestFilletCornerRadiusMismatchRejected(t *testing.T) {
	box := shellBox(2, 2, 2)
	keys := cornerEdgeKeys(t, box)
	_, err := ops.FilletEdgesCorner(box, []ops.EdgeFilletRadii{
		{Key: keys[0], R0: 0.3, R1: 0.3},
		{Key: keys[1], R0: 0.3, R1: 0.3},
		{Key: keys[2], R0: 0.5, R1: 0.5},
	}, ops.CornerMiter, ops.FillConcaveOutward)
	// A mixed-radius TRIHEDRAL corner (r0.3/0.3/0.5) needs a torus corner patch (A4) — still declined,
	// now with the reason it defers rather than the old blanket "one radius" guard (which also rejected the
	// 2-edge miter that P9/V9 now green). It must not silently build a wrong (equal-radius) sphere.
	if err == nil || !strings.Contains(err.Error(), "torus corner patch") {
		t.Fatalf("radius-mismatch err = %v, want the mixed-radius-trihedral (torus patch) decline", err)
	}
}

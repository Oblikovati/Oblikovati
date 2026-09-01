// SPDX-License-Identifier: GPL-2.0-only

package ops_test

import (
	"strings"
	"testing"

	"oblikovati.org/math"
	"oblikovati.org/test-utilities/brepfixture"

	"oblikovati.org/kernel/ops"
	"oblikovati.org/kernel/ops/blend"
	"oblikovati.org/kernel/ops/query"
	"oblikovati.org/kernel/ops/tessellate"
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
		m, _ := tessellate.TessellateBody(res, ops.Quality{ChordTolerance: tol})
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
func cornerStrategyOf(name string) blend.CornerStrategy {
	switch name {
	case "round":
		return blend.CornerRound
	case "setback":
		return blend.CornerSetback
	default:
		return blend.CornerMiter
	}
}

// cornerStrategyPicks selects the fillet edges for a scenario: two of one vertex's three edges
// ("oneCorner"), or the whole top loop ("allTop"), all at radius 0.3.
func cornerStrategyPicks(t *testing.T, box *topo.Body, scenario string) []blend.EdgeFilletRadii {
	t.Helper()
	keys := topLoopKeys(t, box, 2)
	if scenario == "oneCorner" {
		keys = cornerEdgeKeys(t, box)[:2]
	}
	picks := make([]blend.EdgeFilletRadii, len(keys))
	for i, k := range keys {
		picks[i] = blend.EdgeFilletRadii{Key: k, R0: 0.3, R1: 0.3}
	}
	return picks
}

// TestFilletCornerStrategiesValidWatertight drives every strategy across a single-corner selection
// (two of a vertex's three edges) and a whole top-loop selection (four corners), asserting a valid
// watertight solid, the expected sphere count, and that material was removed.
func TestFilletCornerStrategiesValidWatertight(t *testing.T) {
	t.Parallel()
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
			box := brepfixture.Box(math.P3(0, 0, 0), 2, 2, 2)
			res, err := blend.FilletEdgesCorner(box, cornerStrategyPicks(t, box, c.scenario), cornerStrategyOf(c.strategy), blend.FillConcaveOutward)
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
			if v := query.BodyGeometryProperties(res, ops.Quality{ChordTolerance: 1e-3}).Volume; v >= 8 {
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
	t.Parallel()
	vol := func(strategy string) float64 {
		box := brepfixture.Box(math.P3(0, 0, 0), 2, 2, 2)
		res, err := blend.FilletEdgesCorner(box, cornerStrategyPicks(t, box, "oneCorner"), cornerStrategyOf(strategy), blend.FillConcaveOutward)
		if err != nil {
			t.Fatalf("%s: %v", strategy, err)
		}
		return query.BodyGeometryProperties(res, ops.Quality{ChordTolerance: 1e-3}).Volume
	}
	miter, setback, round := vol("miter"), vol("setback"), vol("round")
	if !(miter > setback && setback > round) {
		t.Errorf("expected vol(miter) > vol(setback) > vol(round), got %g, %g, %g", miter, setback, round)
	}
}

// TestFilletRunOutBothEndsZeroRejected: a fillet of radius 0 at BOTH ends is no fillet — a precise
// error, not a degenerate body (a single end at 0 is the valid run-out, TestFilletRunOutToZero).
func TestFilletRunOutBothEndsZeroRejected(t *testing.T) {
	t.Parallel()
	box := brepfixture.Box(math.P3(0, 0, 0), 2, 2, 2)
	_, err := blend.FilletEdgesVarying(box, []blend.EdgeFilletRadii{{Key: verticalEdgeKey(t, box), R0: 0, R1: 0}})
	if err == nil || !strings.Contains(err.Error(), "at least one") {
		t.Fatalf("both-ends-zero err = %v, want the >=0 / at-least-one rejection", err)
	}
}

// TestFilletCornerRadiusMismatchRejected: at a shared corner every edge must carry the SAME radius
// there (the corner sphere has one radius) — differing corner radii are a precise error, UNLESS the
// radii fit the [rB, rS, rS] torus-corner closed form (fillet_corner_radiustorus.go, cluster-A wave):
// three FULLY DISTINCT radii have no common sphere AND no common torus, so they stay the guard's
// target. (r0.3/0.3/0.5 — this test's fixture before the torus corner shipped — now legitimately
// BUILDS instead of declining; see TestFilletCornerRadiusTorusBuilds for that positive case.)
func TestFilletCornerRadiusMismatchRejected(t *testing.T) {
	t.Parallel()
	box := brepfixture.Box(math.P3(0, 0, 0), 2, 2, 2)
	keys := cornerEdgeKeys(t, box)
	_, err := blend.FilletEdgesCorner(box, []blend.EdgeFilletRadii{
		{Key: keys[0], R0: 0.3, R1: 0.3},
		{Key: keys[1], R0: 0.4, R1: 0.4},
		{Key: keys[2], R0: 0.5, R1: 0.5},
	}, blend.CornerMiter, blend.FillConcaveOutward)
	// A mixed-radius TRIHEDRAL corner with three DISTINCT radii (r0.3/0.4/0.5) has no common sphere
	// and no [rB,rS,rS] torus pattern (no two radii are equal) — still declined. It must not silently
	// build a wrong (equal-radius) sphere.
	if err == nil || !strings.Contains(err.Error(), "torus corner patch") {
		t.Fatalf("radius-mismatch err = %v, want the mixed-radius-trihedral (torus patch) decline", err)
	}
}

// TestFilletCornerRadiusTorusBuilds is the positive twin of TestFilletCornerRadiusMismatchRejected:
// on a plain orthogonal box corner, r0.3/0.3/0.5 (the [rB,rS,rS] pattern — one strictly-larger
// radius on the edge joining two walls, equal smaller radii on the two arms sharing the third face)
// now builds a valid watertight solid with an analytic torus corner patch (major R=rB−rS=0.2, minor
// rS=0.3), independent of the OCCT corpus fixtures — a minimal, kernel-only proof this generalizes
// beyond simple/A4's specific dimensions.
func TestFilletCornerRadiusTorusBuilds(t *testing.T) {
	t.Parallel()
	box := brepfixture.Box(math.P3(0, 0, 0), 2, 2, 2)
	keys := cornerEdgeKeys(t, box)
	res, err := blend.FilletEdgesCorner(box, []blend.EdgeFilletRadii{
		{Key: keys[0], R0: 0.3, R1: 0.3},
		{Key: keys[1], R0: 0.3, R1: 0.3},
		{Key: keys[2], R0: 0.5, R1: 0.5},
	}, blend.CornerMiter, blend.FillConcaveOutward)
	if err != nil {
		t.Fatalf("radius-torus corner: %v", err)
	}
	if r := ops.Validate(res); !r.Valid || !res.IsSolid() {
		t.Fatalf("radius-torus corner not a valid solid: %+v", r)
	}
	assertWatertight(t, res, "radius-torus")
}

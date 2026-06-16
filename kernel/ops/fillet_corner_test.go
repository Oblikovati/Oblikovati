// SPDX-License-Identifier: GPL-2.0-only

package ops_test

import (
	stdmath "math"
	"strings"
	"testing"

	"oblikovati.org/kernel/brep"
	"oblikovati.org/kernel/geom"
	"oblikovati.org/kernel/ops"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
)

// Comprehensive coverage of the three fillet CORNER strategies (miter / round / setback) and the
// run-out they are built on, plus the fillet-of-a-fillet (refillet) matrix. The corner strategy
// controls how a vertex where two filleted edges meet — its third edge sharp — is treated:
//   - miter:   the two cylinders mutually trim at a crease (exact rolling ball; no sphere);
//   - round:   the sharp third edge is rounded at constant radius → a full 3-edge sphere;
//   - setback: the sharp third edge is tapered to a run-out → a smooth sphere that fades to sharp.

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

// countTorusFaces counts the body's analytic torus faces.
func countTorusFaces(res *topo.Body) int {
	n := 0
	for _, f := range res.Faces() {
		if _, ok := f.Geometry().(geom.Torus); ok {
			n++
		}
	}
	return n
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
			var picks []ops.EdgeFilletRadii
			if c.scenario == "oneCorner" {
				for _, k := range cornerEdgeKeys(t, box)[:2] {
					picks = append(picks, ops.EdgeFilletRadii{Key: k, R0: 0.3, R1: 0.3})
				}
			} else {
				for _, k := range topLoopKeys(t, box, 2) {
					picks = append(picks, ops.EdgeFilletRadii{Key: k, R0: 0.3, R1: 0.3})
				}
			}
			res, err := ops.FilletEdgesCorner(box, picks, cornerStrategyOf(c.strategy))
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
		var picks []ops.EdgeFilletRadii
		for _, k := range cornerEdgeKeys(t, box)[:2] {
			picks = append(picks, ops.EdgeFilletRadii{Key: k, R0: 0.3, R1: 0.3})
		}
		res, err := ops.FilletEdgesCorner(box, picks, cornerStrategyOf(strategy))
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
// error, not a degenerate body (a single end at 0 is the valid run-out).
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
	}, ops.CornerMiter)
	if err == nil || !strings.Contains(err.Error(), "one radius") {
		t.Fatalf("radius-mismatch err = %v, want the one-radius-at-corner rejection", err)
	}
}

// TestFilletOfAFilletArcCap: rounding a box's vertical edge leaves a quarter-cylinder whose top cap
// edge is a sharp ARC (cylinder ∩ top plane); filleting THAT arc (a fillet of a fillet) rounds it
// into a torus + setback end-caps — a valid watertight solid with exactly one torus face.
func TestFilletOfAFilletArcCap(t *testing.T) {
	f1 := filletBoxVertical(t, 4, 3, 0.3)
	var arc []byte
	for _, e := range f1.Edges() {
		m := e.RangeBox().Center()
		if stdmath.Abs(m.X-3.85) < 1e-6 && stdmath.Abs(m.Y-2.85) < 1e-6 && m.Z > 1.9 {
			arc = e.ReferenceKey()
		}
	}
	if arc == nil {
		t.Fatal("no sharp arc cap on the filleted box")
	}
	res, err := ops.FilletEdges(f1, [][]byte{arc}, 0.1)
	if err != nil {
		t.Fatalf("arc-cap refillet: %v", err)
	}
	if r := ops.Validate(res); !r.Valid || !res.IsSolid() {
		t.Fatalf("arc-cap refillet not a valid solid: %+v", r)
	}
	if tor := countTorusFaces(res); tor != 1 {
		t.Errorf("torus faces = %d, want 1", tor)
	}
	assertWatertight(t, res, "arc-cap refillet")
}

// TestFilletOfAFilletSmoothLineRejected: the cylinder a vertical-edge fillet leaves runs G1-smooth
// into the side plane; that tangent line has no corner to round, so filleting it (a fillet of a
// fillet) is rejected cleanly as smooth — not a misleading invalid-solid.
func TestFilletOfAFilletSmoothLineRejected(t *testing.T) {
	f1 := filletBoxVertical(t, 4, 3, 0.3)
	var line []byte
	for _, e := range f1.Edges() {
		m := e.RangeBox().Center()
		if stdmath.Abs(m.X-4) < 1e-6 && stdmath.Abs(m.Y-2.7) < 1e-6 {
			line = e.ReferenceKey()
		}
	}
	if line == nil {
		t.Fatal("no smooth tangent line on the filleted box")
	}
	if _, err := ops.FilletEdges(f1, [][]byte{line}, 0.1); err == nil || !strings.Contains(err.Error(), "smooth") {
		t.Errorf("smooth tangent line should be rejected as smooth, got: %v", err)
	}
}

// TestFilletOfAFilletRim: rounding the top rim of an analytic cylinder is a fillet of a (degenerate)
// closed edge — it routes to the toroidal-band rim fillet, a valid watertight solid with one torus.
func TestFilletOfAFilletRim(t *testing.T) {
	cyl, err := brep.SolidCylinder(math.P3(0, 0, 0), math.V3(0, 0, 1), 1.0, 2.0)
	if err != nil {
		t.Fatal(err)
	}
	res, err := ops.FilletEdges(cyl, [][]byte{topRimKey(t, cyl, 2.0)}, 0.3)
	if err != nil {
		t.Fatalf("rim refillet: %v", err)
	}
	if r := ops.Validate(res); !r.Valid || !res.IsSolid() {
		t.Fatalf("rim refillet not a valid solid: %+v", r)
	}
	if tor := countTorusFaces(res); tor != 1 {
		t.Errorf("rim torus faces = %d, want 1", tor)
	}
}

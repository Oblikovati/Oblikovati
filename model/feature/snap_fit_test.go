// SPDX-License-Identifier: GPL-2.0-only

package feature

import (
	"testing"

	"oblikovati.org/kernel/ops"
)

// cf wraps a constant as a func() float64 dimension.
func cf(v float64) func() float64 { return func() float64 { return v } }

// TestSnapFitBuildsValidHookSolid: a standalone cantilever snap-fit is one valid solid whose volume
// is the beam plus the catch lip (#486).
func TestSnapFitBuildsValidHookSolid(t *testing.T) {
	t.Parallel()
	fs := NewPartFeatures(nil)
	NewPlasticFeatures(fs).AddCantileverSnapFit(cf(20), cf(6), cf(2), cf(3), cf(1.5))
	fs.Recompute()

	res := fs.Result()
	if len(res) != 1 {
		t.Fatalf("snap fit → %d bodies, want 1", len(res))
	}
	if r := ops.Validate(res[0]); !r.Valid || !res[0].IsSolid() {
		t.Fatalf("snap-fit hook is not a valid solid: %+v", r.Issues)
	}
	beam, catch := 20.0*2*6, 3.0*1.5*6
	if got := ops.BodyGeometryProperties(res[0], ops.Quality{ChordTolerance: 1e-4}).Volume; relErr(got, beam+catch) > 1e-3 {
		t.Errorf("snap-fit volume = %g, want ≈ %g (beam %g + catch %g)", got, beam+catch, beam, catch)
	}
}

// TestSnapFitJoinsRunningBody: a snap fit added after a base box merges into one body.
func TestSnapFitJoinsRunningBody(t *testing.T) {
	t.Parallel()
	fs, _ := boxAndPlanarFace(t) // a 2×2×2 base box at the origin
	NewPlasticFeatures(fs).AddCantileverSnapFit(cf(6), cf(1), cf(1), cf(1), cf(0.5))
	fs.Recompute()
	if got := len(fs.Result()); got != 1 {
		t.Fatalf("base + snap fit → %d bodies, want 1 merged body", got)
	}
}

// TestSnapFitRejectsBadDimensions: non-positive dims and an over-long catch are clean errors.
func TestSnapFitRejectsBadDimensions(t *testing.T) {
	t.Parallel()
	for _, c := range []struct {
		name             string
		l, w, th, cl, ch float64
	}{
		{"zero thickness", 10, 4, 0, 2, 1},
		{"catch longer than beam", 5, 4, 2, 6, 1},
	} {
		fs := NewPartFeatures(nil)
		pf := NewPlasticFeatures(fs).AddCantileverSnapFit(cf(c.l), cf(c.w), cf(c.th), cf(c.cl), cf(c.ch))
		fs.Recompute()
		if pf.Health().OK() {
			t.Errorf("%s: expected the feature to be sick", c.name)
		}
	}
}

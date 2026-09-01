// SPDX-License-Identifier: GPL-2.0-only

package blend

import (
	stdmath "math"
	"testing"

	"oblikovati.org/kernel/brep"
	"oblikovati.org/kernel/geom"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
)

// abs is a tiny local helper mirroring the brief's TestExtractTrihedral assertion; the ops test
// package otherwise reaches for stdmath.Abs directly, so keep this narrow to this recognition test.
func abs(x float64) float64 { return stdmath.Abs(x) }

// sphereCornerBlendFixture builds a planar-trihedral *cornerBlend whose sphere is Sphere{origin, r}
// and whose three blendArcs are the SAME three quarter great-circles sphereTriLoop encodes — the
// exact shape analyticSphereProvider recognises — packaged as a cornerBlend so extractTrihedral can
// be exercised end-to-end. Recognition is therefore guaranteed by construction (the guard in
// TestExtractTrihedralRecognizesSphere).
func sphereCornerBlendFixture(t *testing.T, r float64) *cornerBlend {
	t.Helper()
	o := math.P3(0, 0, 0)
	d := r / stdmath.Sqrt2
	sph, err := geom.NewSphere(o, r)
	if err != nil {
		t.Fatalf("sphere: %v", err)
	}
	return &cornerBlend{
		center: o,
		sphere: sph,
		tan:    map[uint64]math.Point3{},
		arcs: []blendArc{
			{ta: math.P3(r, 0, 0), mid: math.P3(d, d, 0), tb: math.P3(0, r, 0)}, // XY great circle
			{ta: math.P3(0, r, 0), mid: math.P3(0, d, d), tb: math.P3(0, 0, r)}, // YZ great circle
			{ta: math.P3(0, 0, r), mid: math.P3(d, 0, d), tb: math.P3(r, 0, 0)}, // XZ great circle
		},
	}
}

// TestExtractTrihedralRecognizesSphere proves the extracted 3-arc loop is claimed by the exact sphere
// tier and yields the SAME sphere as the cornerBlend (center + radius within weld).
func TestExtractTrihedralRecognizesSphere(t *testing.T) {
	t.Parallel()
	cb := sphereCornerBlendFixture(t, 4)
	loop, ok := extractTrihedral(cb)
	if !ok {
		t.Fatal("extractTrihedral declined the planar trihedral")
	}
	if loop.Valence() != 3 {
		t.Fatalf("valence = %d, want 3", loop.Valence())
	}
	patch, ok := resolveBlend(loop, blendScale())
	if !ok || patch.Kind != BlendKindSphere {
		t.Fatalf("resolveBlend kind = %q ok=%v, want analytic-sphere", patch.Kind, ok)
	}
	got := patch.Surface.(geom.Sphere)
	if got.Center.DistanceTo(cb.sphere.Center) > blendScale().Weld() || abs(got.Radius-cb.sphere.Radius) > blendScale().Weld() {
		t.Fatalf("recognized sphere %+v != cornerBlend sphere %+v", got, cb.sphere)
	}
}

// TestExtractTrihedralDeclinesNonTriadic pins the honest-reject floor: a cornerBlend without exactly
// three arcs is not a planar trihedral, so extractTrihedral declines and the strangler falls back.
func TestExtractTrihedralDeclinesNonTriadic(t *testing.T) {
	t.Parallel()
	cb := sphereCornerBlendFixture(t, 4)
	cb.arcs = cb.arcs[:2]
	if _, ok := extractTrihedral(cb); ok {
		t.Fatal("extractTrihedral must decline a non-triadic cornerBlend")
	}
}

// realTrihedralBlend drives the ACTUAL fillet corner pipeline on a genuine box corner (three filleted
// edges meeting at a valence-3 vertex) and returns the solved *cornerBlend whose arcs are in NATURAL
// registration order — the order registerBlendArc appends them per edge-pick (fillet.go), NOT the
// hand-chained order the synthetic fixture uses. This is the seam-proof witness on real data: with the
// pre-fix append-order extractor, RailLoop.Closed is false on these arcs → the sphere certificate fails
// → resolveBlend declines, so TestExtractTrihedralRecognizesRealCorner FAILS pre-fix and PASSES post-fix.
func realTrihedralBlend(t *testing.T) *cornerBlend {
	t.Helper()
	box, err := brep.SolidBlock(math.P3(0, 0, 0), math.P3(5, 5, 5), "box")
	if err != nil {
		t.Fatalf("SolidBlock: %v", err)
	}
	picks := cornerEdgePicks(t, box, math.P3(0, 0, 0), 1)
	blends, miters, err := computeCorners(box, picks)
	if err != nil {
		t.Fatalf("computeCorners: %v", err)
	}
	if _, err := computeFillets(box, picks, blends, miters, FillConcaveOutward, nil); err != nil {
		t.Fatalf("computeFillets: %v", err) // populates blend.arcs in natural registration order
	}
	return singleBlend(t, blends)
}

// cornerEdgePicks builds a constant-radius filletPick for each of the three box edges incident to the
// corner vertex at point p (a valence-3 trihedral corner), erroring unless exactly three are found.
func cornerEdgePicks(t *testing.T, b *topo.Body, corner math.Point3, r float64) []filletPick {
	t.Helper()
	var picks []filletPick
	for _, e := range b.Edges() {
		if e.StartVertex().Point().DistanceTo(corner) < 1e-9 || e.EndVertex().Point().DistanceTo(corner) < 1e-9 {
			picks = append(picks, filletPick{edge: e, r0: r, r1: r})
		}
	}
	if len(picks) != 3 {
		t.Fatalf("corner %v: got %d incident edges, want 3", corner, len(picks))
	}
	return picks
}

// singleBlend returns the one cornerBlend the corner solve produced, erroring unless there is exactly one.
func singleBlend(t *testing.T, blends map[uint64]*cornerBlend) *cornerBlend {
	t.Helper()
	if len(blends) != 1 {
		t.Fatalf("got %d blends, want exactly 1 trihedral corner", len(blends))
	}
	for _, cb := range blends {
		return cb
	}
	return nil
}

// TestExtractTrihedralRecognizesRealCorner is the seam proof on REAL data: a cornerBlend from the live
// fillet pipeline (arcs in natural registration order) must, once chain-ordered by extractTrihedral, be
// claimed by the exact sphere tier and yield the SAME sphere as the cornerBlend. This fails with the
// pre-fix append-order extractor (RailLoop.Closed false) and passes after the chain-order fix.
func TestExtractTrihedralRecognizesRealCorner(t *testing.T) {
	t.Parallel()
	cb := realTrihedralBlend(t)
	if len(cb.arcs) != 3 {
		t.Fatalf("real corner produced %d arcs, want 3", len(cb.arcs))
	}
	loop, ok := extractTrihedral(cb)
	if !ok {
		t.Fatal("extractTrihedral declined a real planar-trihedral corner")
	}
	patch, ok := resolveBlend(loop, sphereScale(cb))
	if !ok || patch.Kind != BlendKindSphere {
		t.Fatalf("resolveBlend on real corner: kind=%q ok=%v, want analytic-sphere", patch.Kind, ok)
	}
	got := patch.Surface.(geom.Sphere)
	weld := blendScale().Weld()
	if got.Center.DistanceTo(cb.sphere.Center) > weld || abs(got.Radius-cb.sphere.Radius) > weld {
		t.Fatalf("recognized sphere %+v != real cornerBlend sphere %+v (weld %g)", got, cb.sphere, weld)
	}
}

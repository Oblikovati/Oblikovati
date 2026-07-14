// SPDX-License-Identifier: GPL-2.0-only

package ops

import (
	stdmath "math"
	"testing"

	"oblikovati.org/kernel/geom"
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
	cb := sphereCornerBlendFixture(t, 4)
	cb.arcs = cb.arcs[:2]
	if _, ok := extractTrihedral(cb); ok {
		t.Fatal("extractTrihedral must decline a non-triadic cornerBlend")
	}
}

// SPDX-License-Identifier: GPL-2.0-only

package ops

import (
	stdmath "math"
	"testing"

	"oblikovati.org/kernel/geom"
	"oblikovati.org/math"
)

// TestTorusTangencyRoots_SyntheticFourRoots exercises the quartic path directly against
// torusSyntheticFixture's own numbers, checking it finds ALL FOUR real tangent points (the two
// radial branches × the ±t symmetry) — a genuine multi-root corner, not merely the single admitted
// candidate the corner solve ultimately picks.
func TestTorusTangencyRoots_SyntheticFourRoots(t *testing.T) {
	t.Parallel()
	const rm, rt, r = 50.0, 20.0, 5.0
	rho := rt - r
	tor, _ := geom.NewTorus(math.P3(0, 0, 0), math.V3(0, 0, 1), rm, rt)
	p0 := math.P3(0, -r, -r) // on the plane-pair line (x free, y=-r, z=-r)
	d := math.V3(1, 0, 0)
	res := ResolutionForSize(100)
	roots := torusTangencyRoots(tor, p0, d, rho, res)
	if len(roots) != 4 {
		t.Fatalf("got %d real physical roots %v, want 4 (two radial branches × ±t)", len(roots), roots)
	}
	radial1 := rm + stdmath.Sqrt(rho*rho-r*r)
	radial2 := rm - stdmath.Sqrt(rho*rho-r*r)
	want := []float64{
		stdmath.Sqrt(radial1*radial1 - r*r), -stdmath.Sqrt(radial1*radial1 - r*r),
		stdmath.Sqrt(radial2*radial2 - r*r), -stdmath.Sqrt(radial2*radial2 - r*r),
	}
	for _, w := range want {
		if !closeToAny(roots, w, 1e-6) {
			t.Fatalf("roots %v missing expected root %.6f", roots, w)
		}
	}
}

// closeToAny reports whether xs contains a value within tol of want.
func closeToAny(xs []float64, want, tol float64) bool {
	for _, x := range xs {
		if stdmath.Abs(x-want) < tol {
			return true
		}
	}
	return false
}

// TestTorusCandidatesWellSeparated_GrazingBand is the mutation witness for the point-space grazing
// gate (torusQuarticGraceBand·res.Weld()): two distinct points closer than the band must reject;
// two points at or beyond it must accept.
func TestTorusCandidatesWellSeparated_GrazingBand(t *testing.T) {
	t.Parallel()
	res := ResolutionForSize(1000) // Weld() ≈ 1e-6
	band := torusQuarticGraceBand * res.Weld()
	close2 := []math.Point3{math.P3(0, 0, 0), math.P3(band*0.5, 0, 0)}
	if torusCandidatesWellSeparated(close2, res) {
		t.Fatalf("points %.3e apart (< band %.3e) accepted as well-separated; want reject", band*0.5, band)
	}
	far2 := []math.Point3{math.P3(0, 0, 0), math.P3(band*10, 0, 0)}
	if !torusCandidatesWellSeparated(far2, res) {
		t.Fatalf("points %.3e apart (> band %.3e) rejected as too close; want accept", band*10, band)
	}
	identical := []math.Point3{math.P3(1, 2, 3), math.P3(1, 2, 3)}
	if !torusCandidatesWellSeparated(identical, res) {
		t.Fatalf("two IDENTICAL points (the same physical root) rejected; want the coincident-point no-op to pass")
	}
}

// twoCandArms builds a single straight arm along the x-axis through onSpine's own perpendicular
// foot, with `far` placed so the candidate that sits ON the spine (within weld) and on the far side
// of vp is admitted — the fixture for every torusCornerTiebreak test below.
func twoCandArms(t *testing.T, onSpine math.Point3) ([]cornerArm, math.Point3) {
	t.Helper()
	spine, err := geom.NewCylinder(onSpine, math.V3(1, 0, 0), 1)
	if err != nil {
		t.Fatalf("spine cylinder: %v", err)
	}
	far := onSpine.TranslateBy(math.V3(1, 0, 0).Scale(5))
	return []cornerArm{{spine: spine, far: far}}, far
}

// TestTorusCornerTiebreak_ResolvesViaArms is the tiebreak's core proof: given two candidates, only
// one of which lies on the injected arm's spine, the in-domain one must be chosen — even when it is
// FARTHER from vp than the off-spine one (so this is provably not a disguised nearest-vertex pick).
func TestTorusCornerTiebreak_ResolvesViaArms(t *testing.T) {
	t.Parallel()
	onSpine := math.P3(10, 0, 0)
	offSpine := math.P3(1, 5, 0) // NEARER to vp=(0,0,0) but off the spine (not collinear with it) — must NOT be chosen
	arms, _ := twoCandArms(t, onSpine)
	vp := math.P3(0, 0, 0)
	res := ResolutionForSize(100)
	got, ok := torusCornerTiebreak(arms, vp, 5, res, []math.Point3{offSpine, onSpine})
	if !ok {
		t.Fatalf("tiebreak declined; want the unique in-domain candidate %v", onSpine)
	}
	if got.DistanceTo(onSpine) > 1e-6 {
		t.Fatalf("tiebreak picked %v, want the on-spine candidate %v (proves it is NOT nearest-vertex)", got, onSpine)
	}
}

// TestTorusCornerTiebreak_BothInDomainRejects: two candidates BOTH on a colinear spine (station
// toward far on both) is ambiguous — honest-reject rather than guess (the N7 lesson).
func TestTorusCornerTiebreak_BothInDomainRejects(t *testing.T) {
	t.Parallel()
	onSpine1 := math.P3(10, 0, 0)
	onSpine2 := math.P3(20, 0, 0)
	arms, _ := twoCandArms(t, math.P3(0, 0, 0)) // far = (5,0,0); both stations are toward +x
	vp := math.P3(0, 0, 0)
	res := ResolutionForSize(100)
	if _, ok := torusCornerTiebreak(arms, vp, 5, res, []math.Point3{onSpine1, onSpine2}); ok {
		t.Fatalf("both candidates in-domain: tiebreak accepted one; want ambiguous reject")
	}
}

// TestTorusCornerTiebreak_NeitherInDomainRejects: neither candidate lies on the arm's spine —
// honest-reject rather than fall back to nearest-vertex (an arm IS present, so its witness governs).
func TestTorusCornerTiebreak_NeitherInDomainRejects(t *testing.T) {
	t.Parallel()
	arms, _ := twoCandArms(t, math.P3(10, 0, 0))
	cands := []math.Point3{math.P3(0, 50, 0), math.P3(0, -50, 0)}
	vp := math.P3(0, 0, 0)
	res := ResolutionForSize(100)
	if _, ok := torusCornerTiebreak(arms, vp, 5, res, cands); ok {
		t.Fatalf("neither candidate in-domain: tiebreak accepted one; want reject")
	}
}

// TestTorusCornerTiebreak_NoArmsFallsBackToNearest: with NO arms built at the corner (no straight
// cylinder arm to witness against — the real E6/E8/F1/F3 fixtures' actual situation, since none of
// their edges are Plane∧Cylinder lines), the legacy nearer-vertex pick applies, mirroring
// sphereCornerRoot/coneCornerRoot's own "no witness" fallback.
func TestTorusCornerTiebreak_NoArmsFallsBackToNearest(t *testing.T) {
	t.Parallel()
	near := math.P3(1, 0, 0)
	far := math.P3(100, 0, 0)
	vp := math.P3(0, 0, 0)
	res := ResolutionForSize(100)
	got, ok := torusCornerTiebreak(nil, vp, 5, res, []math.Point3{far, near})
	if !ok || got.DistanceTo(near) > 1e-9 {
		t.Fatalf("no-arms tiebreak = %v (ok=%v), want nearest-vertex pick %v", got, ok, near)
	}
}

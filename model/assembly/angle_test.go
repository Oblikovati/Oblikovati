// SPDX-License-Identifier: GPL-2.0-only

package assembly

import (
	stdmath "math"
	"testing"

	"oblikovati.org/api/types"
	"oblikovati.org/math"
	"oblikovati.org/model/occurrence"
)

// signedWorldAngle measures the signed angle from dA to dB about axis (the value the constraint
// residual drives to the target), for asserting a solved configuration.
func signedWorldAngle(dA, dB, axis math.Vector3) float64 {
	u, _ := math.UnitVector3FromVector(axis)
	return stdmath.Atan2(dA.Cross(dB).Dot(u.AsVector()), dA.Dot(dB))
}

// angleClose reports whether two angles are equal modulo 2π (to a tight tolerance).
func angleClose(got, want float64) bool {
	return stdmath.Abs(wrapToPi(got-want)) < 1e-4
}

// TestUndirectedAngleResidualUnchanged pins the undirected residual to the cosine form — the
// regression guard that the signed-angle work (#1972) left the default solution untouched.
func TestUndirectedAngleResidualUnchanged(t *testing.T) {
	dA, dB := math.V3(1, 0, 0), math.V3(0, 1, 0) // 90° apart
	got := undirectedAngleResiduals(dA, dB, stdmath.Pi/3)[0]
	want := stdmath.Cos(stdmath.Pi/2) - stdmath.Cos(stdmath.Pi/3) // cosθ − cos(target)
	if stdmath.Abs(got-want) > 1e-12 {
		t.Errorf("undirected residual = %v, want %v (cosine form)", got, want)
	}
}

// TestSignedAngleResidualSignsAndWraps checks the signed residual is zero at the target and at the
// target plus a full turn (so a past-180° target is reachable), and that reversing the axis flips
// the measured sign (#1972).
func TestSignedAngleResidualSignsAndWraps(t *testing.T) {
	dA := math.V3(1, 0, 0)
	dB := math.V3(0, 1, 0)   // +90° about +Z
	axis := math.V3(0, 0, 1) // +Z

	if r := signedAngleResiduals(dA, dB, axis, stdmath.Pi/2)[0]; stdmath.Abs(r) > 1e-9 {
		t.Errorf("residual at the exact target = %v, want ~0", r)
	}
	// The same geometry against a target one full turn away is still satisfied (wrap).
	if r := signedAngleResiduals(dA, dB, axis, stdmath.Pi/2+2*stdmath.Pi)[0]; stdmath.Abs(r) > 1e-9 {
		t.Errorf("residual at target+2π = %v, want ~0 (wrap)", r)
	}
	// Reversing the axis flips the measured angle: +90° becomes −90°, so the +90° target now has a
	// residual of a half turn's worth away from satisfied (|residual| = π ... measured−target).
	flipped := signedAngleResiduals(dA, dB, math.V3(0, 0, -1), stdmath.Pi/2)[0]
	if stdmath.Abs(wrapToPi(flipped-(-stdmath.Pi))) > 1e-9 {
		t.Errorf("axis-reversed residual = %v, want −90°−90° wrapped to ±π", flipped)
	}
}

// TestImpliedAngleAxisFallback the implied axis is the cross product, or a perpendicular of dA when
// the two directions start parallel (#1972).
func TestImpliedAngleAxisFallback(t *testing.T) {
	ax := impliedAngleAxis(math.V3(1, 0, 0), math.V3(0, 1, 0))
	if stdmath.Abs(ax.Dot(math.V3(0, 0, 1))) < 0.9 { // ≈ +Z
		t.Errorf("implied axis of +X,+Y = %v, want ≈ +Z", ax)
	}
	par := impliedAngleAxis(math.V3(1, 0, 0), math.V3(2, 0, 0)) // parallel
	if stdmath.Abs(par.Dot(math.V3(1, 0, 0))) > 1e-9 {
		t.Errorf("parallel-input implied axis %v is not perpendicular to dA", par)
	}
	if l := par.Length(); l < 1e-6 {
		t.Errorf("parallel-input implied axis is degenerate (length %v)", l)
	}
}

// hingeAboutZ grounds a base and pins a moving occurrence to spin about +Z at the origin (an
// insert leaving only the spin free), the setup a signed angle then drives deterministically.
func hingeAboutZ(t *testing.T) (*ConstraintSet, *occurrence.Occurrence, *occurrence.Occurrence) {
	t.Helper()
	occs := occurrence.NewOccurrences()
	base := place(occs, "base:1", math.Identity4())
	base.SetGrounded(true)
	moving := place(occs, "moving:1", math.Identity4())
	set := NewConstraintSet(occs, nil)
	set.AddInsert(ref(base, LinePrimitive(math.P3(0, 0, 0), unit(t, 0, 0, 1))),
		ref(moving, LinePrimitive(math.P3(0, 0, 0), unit(t, 0, 0, 1))), 0, true)
	return set, base, moving
}

// TestReferenceVectorAngleSignsPast180 drives a hinge to 210° about an explicit +Z reference
// vector: the solved spin lands on the 210° side (a negative sine), which the undirected cosine
// solution — equally satisfied by 150° — cannot reach (#1972).
func TestReferenceVectorAngleSignsPast180(t *testing.T) {
	set, base, moving := hingeAboutZ(t)
	target := 210 * stdmath.Pi / 180
	set.AddAngleAbout(
		ref(base, LinePrimitive(math.P3(0, 0, 0), unit(t, 1, 0, 0))),
		ref(moving, LinePrimitive(math.P3(0, 0, 0), unit(t, 1, 0, 0))),
		ref(base, LinePrimitive(math.P3(0, 0, 0), unit(t, 0, 0, 1))),
		target)
	if rep := set.Solve(); !rep.Converged {
		t.Fatalf("solve did not converge: %+v", rep)
	}
	dMov := moving.Transform().TransformVector(math.V3(1, 0, 0))
	signed := signedWorldAngle(math.V3(1, 0, 0), dMov, math.V3(0, 0, 1))
	if !angleClose(signed, target) {
		t.Errorf("solved signed angle = %.4f rad, want %.4f (210°)", signed, target)
	}
	if dMov.Y >= 0 {
		t.Errorf("solved direction %v has a non-negative Y — landed on the 150° side, not 210°", dMov)
	}
}

// TestDirectedAngleSignsPast180 drives a hinge to 210° with the directed solution: the moving
// input starts perpendicular so the implied axis resolves to the spin axis, and the signed angle
// reaches the 210° configuration (#1972).
func TestDirectedAngleSignsPast180(t *testing.T) {
	set, base, moving := hingeAboutZ(t)
	target := 210 * stdmath.Pi / 180
	// The moving angle input points +Y (perpendicular to the base +X), so dA×dB = +Z = the spin axis.
	set.AddAngle(
		ref(base, LinePrimitive(math.P3(0, 0, 0), unit(t, 1, 0, 0))),
		ref(moving, LinePrimitive(math.P3(0, 0, 0), unit(t, 0, 1, 0))),
		target, types.AngleSolutionDirected)
	if rep := set.Solve(); !rep.Converged {
		t.Fatalf("solve did not converge: %+v", rep)
	}
	dMov := moving.Transform().TransformVector(math.V3(0, 1, 0))
	signed := signedWorldAngle(math.V3(1, 0, 0), dMov, math.V3(0, 0, 1))
	if !angleClose(signed, target) {
		t.Errorf("solved directed angle = %.4f rad, want %.4f (210°)", signed, target)
	}
}

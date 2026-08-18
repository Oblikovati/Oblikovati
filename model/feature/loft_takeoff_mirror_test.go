// SPDX-License-Identifier: GPL-2.0-only

package feature

import (
	stdmath "math"
	"testing"

	"oblikovati.org/kernel/geom"
	"oblikovati.org/kernel/ops"
	"oblikovati.org/math"
)

// Regression for Oblikovati#2082. A loft end condition describes how the surface LEAVES its
// section. +u leaves the first section but ARRIVES at the last one, so the last section's tangent
// is the mirror of its leaving direction. applyApexCondition always mirrored; the profile branches
// (angle takeoff, face continuity) did not, so an equal condition at both ends built a lopsided
// body — and, Reversed, one that cut through its own end cap.

// ringOfRadius is a circle of n points on the plane z, in +u (counter-clockwise) order.
func ringOfRadius(r, z float64, n int) []math.Point3 {
	out := make([]math.Point3, n)
	for k := range out {
		a := 2 * stdmath.Pi * float64(k) / float64(n)
		out[k] = math.P3(r*stdmath.Cos(a), r*stdmath.Sin(a), z)
	}
	return out
}

// angleTakeoffTangents returns the end tangents of a two-section loft over equal circles, with the
// same condition applied at both ends.
func angleTakeoffTangents(end LoftEnd) (first, last []math.Vector3) {
	secs := [][]math.Point3{ringOfRadius(2, 0, 8), ringOfRadius(2, 4, 8)}
	up := math.V3(0, 0, 1).AsUnit()
	tan := sectionTangents(secs, false, loftEnds{first: end, last: end, firstN: up, lastN: up}, 0)
	return tan[0], tan[1]
}

// TestAngleTakeoffMirrorsTheRadialComponentAtTheLastSection is the defect in one assertion. With
// the same 45 deg takeoff at both ends, the two tangents must agree through the plane (both aim
// into the body) and OPPOSE across it (both flare outward as the surface leaves its own section).
// Before the fix both radial components pointed the same way, which is what tilted the body.
func TestAngleTakeoffMirrorsTheRadialComponentAtTheLastSection(t *testing.T) {
	first, last := angleTakeoffTangents(LoftEnd{Condition: LoftAngle, Angle: rad(45)})
	for j := range first {
		radial := math.V3(float64(first[j].X), float64(first[j].Y), 0)
		lastRadial := math.V3(float64(last[j].X), float64(last[j].Y), 0)
		if got := float64(radial.Dot(lastRadial)); got >= 0 {
			t.Fatalf("track %d: radial takeoffs agree (dot = %g); the last section must flare the "+
				"opposite way in +u, or the loft leans instead of bulging", j, got)
		}
		if float64(first[j].Z) <= 0 || float64(last[j].Z) <= 0 {
			t.Fatalf("track %d: through-plane components %g and %g should both aim along +u",
				j, first[j].Z, last[j].Z)
		}
	}
}

// TestAngleTakeoffBuildsASymmetricBarrel is the same fact on the delivered body. Equal circles with
// an equal takeoff at both ends are symmetric about the mid plane, so the shape must be too.
// Measured before the fix: 2.27 at the bottom against 1.75 at the top — a pear, not a barrel.
func TestAngleTakeoffBuildsASymmetricBarrel(t *testing.T) {
	const span = 4.0
	end := LoftEnd{Condition: LoftAngle, Angle: rad(45)}
	b := conditionedLoft(t, twoCircles(2), false, end, end)
	// Radius by height, keyed on z to the micron. Comparing halves would only measure where the
	// samples happen to fall; mirrored heights compare the SHAPE.
	radiusAt := map[int64]float64{}
	for _, v := range b.Vertices() {
		p := v.Point()
		key := int64(stdmath.Round(float64(p.Z) * 1e6))
		radiusAt[key] = stdmath.Max(radiusAt[key], stdmath.Hypot(float64(p.X), float64(p.Y)))
	}
	widest := 0.0
	for key, r := range radiusAt {
		mirror, ok := radiusAt[int64(stdmath.Round(span*1e6))-key]
		if !ok {
			t.Fatalf("no sample at the mirror of z = %.6f — the loft is not sampled symmetrically",
				float64(key)/1e6)
		}
		if stdmath.Abs(r-mirror) > 1e-9 {
			t.Errorf("at z = %.4f the radius is %.6f against %.6f at the mirrored height — the barrel "+
				"leans", float64(key)/1e6, r, mirror)
		}
		widest = stdmath.Max(widest, r)
	}
	if widest < 2.15 {
		t.Errorf("max radius %.4f: the takeoff stopped bulging at all", widest)
	}
}

// TestReversedTakeoffOverhangsOutsideTheCap is the #2082 headline. A reversed takeoff must hang
// over the end cap, not come back down through it: every point above the end plane has to sit
// OUTSIDE the cap's own radius. Before the fix the wall re-crossed z=4 at radius 1.74, inside the
// radius-2 disc, for 24 interpenetrating face pairs on a body that passed ops.Validate.
func TestReversedTakeoffOverhangsOutsideTheCap(t *testing.T) {
	rev := LoftEnd{Condition: LoftAngle, Angle: rad(45), Reversed: true}
	b := conditionedLoft(t, twoCircles(2), false, rev, rev)
	overhang := false
	for _, v := range b.Vertices() {
		p := v.Point()
		if float64(p.Z) <= 4+1e-9 {
			continue
		}
		r := stdmath.Hypot(float64(p.X), float64(p.Y))
		if r < 2-1e-9 {
			t.Fatalf("a point above the end cap sits at radius %.4f, inside the cap's 2 — the side "+
				"wall came back down through it at %v", r, p)
		}
		overhang = true
	}
	if !overhang {
		t.Error("the reversed takeoff produced no overhang above the end plane at all")
	}
}

// sphereFaceContinuity runs faceContinuity for one end of a loft leaving a sphere face, returning
// the tangents it wrote plus the second and third derivatives.
func sphereFaceContinuity(isStart bool) (tan []math.Vector3, second, third []math.Vector3) {
	const R, v0, n = 2.0, 0.3, 12
	sph, _ := geom.NewSphere(math.P3(0, 0, 0), R)
	sec := make([]math.Point3, n)
	for k := range sec {
		sec[k] = sph.PointAt(2*stdmath.Pi*float64(k)/n, v0)
	}
	neighbor := ringOfRadius(R*stdmath.Cos(v0)+1, R*stdmath.Sin(v0)+1, n)
	tan = make([]math.Vector3, n)
	end := LoftEnd{Condition: LoftG3, Impact: 1}
	second, third = faceContinuity(tan, sec, neighbor, sph, end, isStart)
	return tan, second, third
}

// TestFaceContinuityMirrorsTheOddDerivativesAtTheLastSection: face continuity has the same shape of
// defect, and the same fix has to reach the DERIVATIVES it feeds the quintic/septic end blend.
// P(u)=γ(±c·u) — so the odd orders flip at a last section and the even ones do not. Getting the
// second derivative to flip too would break the curvature match the G2 end exists for.
func TestFaceContinuityMirrorsTheOddDerivativesAtTheLastSection(t *testing.T) {
	startTan, startA, startJ := sphereFaceContinuity(true)
	lastTan, lastA, lastJ := sphereFaceContinuity(false)
	opposed := func(name string, a, b []math.Vector3) {
		t.Helper()
		for j := range a {
			if d := float64(a[j].Add(b[j]).Length()); d > 1e-9 {
				t.Errorf("%s track %d: last is %v against the start's %v, want the exact negation "+
					"(off by %g)", name, j, b[j], a[j], d)
			}
		}
	}
	same := func(name string, a, b []math.Vector3) {
		t.Helper()
		for j := range a {
			if d := float64(a[j].Sub(b[j]).Length()); d > 1e-9 {
				t.Errorf("%s track %d: last is %v against the start's %v, want them equal (off by %g)",
					name, j, b[j], a[j], d)
			}
		}
	}
	opposed("tangent", startTan, lastTan)
	same("second derivative", startA, lastA)
	opposed("third derivative", startJ, lastJ)
}

// TestFaceContinuityKeepsTheSeamCurvature guards the mirror against the cheap fix of negating
// everything: geometric curvature |m×a|/|m|³ at the seam must still equal the sphere's 1/R at a
// LAST section, exactly as it does at a first one.
func TestFaceContinuityKeepsTheSeamCurvature(t *testing.T) {
	const R = 2.0
	tan, second, _ := sphereFaceContinuity(false)
	for j := range tan {
		k := float64(tan[j].Cross(second[j]).Length()) / stdmath.Pow(float64(tan[j].Length()), 3)
		if d := stdmath.Abs(k - 1/R); d > 1e-6 {
			t.Fatalf("track %d seam curvature = %.6f at a last section, want 1/R = %.6f", j, k, 1/R)
		}
	}
}

// TestTakeoffSignIsTheOnlyDifferenceBetweenTheEnds pins the helper itself: a start section keeps
// its leaving direction, a last one mirrors it. Everything above rides on these two values.
func TestTakeoffSignIsTheOnlyDifferenceBetweenTheEnds(t *testing.T) {
	if got := takeoffSign(true); got != 1 {
		t.Errorf("takeoffSign(start) = %g, want 1 — +u leaves the first section", got)
	}
	if got := takeoffSign(false); got != -1 {
		t.Errorf("takeoffSign(last) = %g, want -1 — +u arrives at the last section", got)
	}
}

// TestFaceTangentTakeoffMirrorsToo: the approximate (no source surface) face-continuity branch
// shares the defect and the fix, so it gets the same assertion — the two ends' in-plane tangents
// must oppose.
func TestFaceTangentTakeoffMirrorsToo(t *testing.T) {
	secs := [][]math.Point3{ringOfRadius(2, 0, 8), ringOfRadius(2, 4, 8)}
	up := math.V3(0, 0, 1).AsUnit()
	end := LoftEnd{Condition: LoftTangent, Impact: 1}
	tan := sectionTangents(secs, false, loftEnds{first: end, last: end, firstN: up, lastN: up}, 0)
	for j := range tan[0] {
		if got := float64(tan[0][j].Dot(tan[1][j])); got >= 0 {
			t.Fatalf("track %d: the two face tangents agree (dot = %g); a last section must take off "+
				"the opposite way in +u", j, got)
		}
	}
}

// TestReversedTakeoffStillUndercuts keeps the behaviour the mirror must not cost: the reversed
// takeoff exists to dip the surface below the start plane, and it still does.
func TestReversedTakeoffStillUndercuts(t *testing.T) {
	rev := LoftEnd{Condition: LoftAngle, Angle: rad(45), Reversed: true}
	b := conditionedLoft(t, twoCircles(2), false, rev, rev)
	if z := float64(b.RangeBox().Min.Z); z > -0.02 {
		t.Errorf("min z = %.4f, want < -0.02 — the undercut was lost to the mirror", z)
	}
	if v := ops.BodyGeometryProperties(b, ops.DefaultQuality()).Volume; v <= 0 {
		t.Errorf("the undercut body has volume %g", v)
	}
}

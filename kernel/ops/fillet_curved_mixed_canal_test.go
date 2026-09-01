// SPDX-License-Identifier: GPL-2.0-only

package ops

import (
	stdmath "math"
	"testing"

	"oblikovati.org/kernel/geom"
	"oblikovati.org/math"
)

// The DECLINE tests for newN4BallFrame's three class guards. The O1-reuse safety story rests entirely on
// these: O1's third arm is a concave cove torus over cylinder hosts, so its ball may NOT roll on the lateral
// arm's tube the way N4's does, and the frame is what must refuse the corner rather than emit a wrong patch.
// A guard whose failing branch is never executed is an assumption, not a guard — so each test BRACKETS its
// guard: the same geometry perturbed just under the trip point still ACCEPTS, and just over it DECLINES,
// which is what proves the decline came from that guard and not from an unrelated degeneracy downstream.

// n4BallFrameInputs is the exact N4 frame input tuple (lateral torus arm, shared vertical plane, the two
// terminating arms' ball centres) taken from the same solver the corner uses, so a decline test perturbs
// REAL accepted geometry rather than a hand-built approximation of it.
type n4BallFrameInputs struct {
	torus  geom.Torus
	vplane geom.Plane
	m0, m1 math.Point3 // band-arm and concave-cyl-arm ball centres
	tol    float64     // the model-relative weld distance the corner passes down
}

// n4AcceptedBallFrameInputs solves the real N4 corner points and returns the ball-frame inputs they yield,
// asserting up front that they ARE accepted — the baseline every decline test is measured against.
func n4AcceptedBallFrameInputs(t *testing.T) n4BallFrameInputs {
	t.Helper()
	arms, ok := classifyN4MixedArms(n4TestArms(t))
	if !ok {
		t.Fatal("classifyN4MixedArms rejected the N4 corner")
	}
	res := ResolutionForPoints([]math.Point3{math.P3(0, 0, 0), math.P3(200, 200, 60)})
	pts := n4TestCornerPts(t, arms)
	in := n4BallFrameInputs{
		torus: arms.torus.armSurface.(geom.Torus), vplane: arms.band.a.Geometry().(geom.Plane),
		m0: pts.ballAB, m1: pts.ballCD, tol: res.Weld() * 5,
	}
	if _, ok := newN4BallFrame(in.torus, in.vplane, in.m0, in.m1, in.tol); !ok {
		t.Fatal("newN4BallFrame declined the unperturbed N4 corner — the decline tests have no baseline")
	}
	return in
}

// TestN4BallFrameDeclinesVPlaneNotParallelToTorusAxis exercises the n̂·k̂ CLASS guard: the closed-form curve
// is {n̂·(P−O) = a} ∩ {2r-torus} only while the host plane contains the torus axis direction, because only
// then is the plane constraint ρ·cosθ = a. It brackets the predicate itself at its own seamAngularTol AND
// asserts that a host the predicate rejects makes newN4BallFrame decline.
//
// The predicate is bracketed separately from the frame on purpose: an out-of-plane tilt of ε also moves the
// offset plane by |m₀−O|·ε ≈ 25ε, so at any tilt near the tolerance the STATION checks would fire too and a
// frame-level bracket could not attribute the decline. Testing the named predicate at the threshold, plus the
// frame's decline on a host it rejects, does attribute it.
func TestN4BallFrameDeclinesVPlaneNotParallelToTorusAxis(t *testing.T) {
	t.Parallel()
	in := n4AcceptedBallFrameInputs(t)
	for _, tc := range []struct {
		name string
		tilt float64 // radians of normal rotation toward the torus axis
		want bool
	}{
		{"an untilted host is in class", 0, true},
		{"half an angular tolerance of tilt is in class", 0.5 * seamAngularTol, true},
		{"twice the angular tolerance is out of class", 2 * seamAngularTol, false},
		{"a grossly tilted host is out of class", 0.5, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			tilted := tiltPlaneNormalTowardAxis(t, in.vplane, in.torus.AxisDir.AsVector(), tc.tilt)
			if got := vplaneParallelToTorusAxis(in.torus, tilted); got != tc.want {
				t.Fatalf("vplaneParallelToTorusAxis = %v for a host tilted %.3e rad out of the torus "+
					"equatorial plane, want %v (n̂·k̂ = %.3e vs seamAngularTol %.0e)",
					got, tc.tilt, tc.want, stdmath.Sin(tc.tilt), seamAngularTol)
			}
			if tc.want {
				return
			}
			if _, ok := newN4BallFrame(in.torus, tilted, in.m0, in.m1, in.tol); ok {
				t.Fatalf("newN4BallFrame built a frame on an out-of-class host (tilt %.3e rad): the plane "+
					"constraint is not ρ·cosθ = a there, so the patch would be a wrong envelope", tc.tilt)
			}
		})
	}
}

// tiltPlaneNormalTowardAxis rotates a plane's normal by `tilt` radians toward `axis`, keeping the plane
// through the same origin. Only the normal's axial component matters to the guard under test.
func tiltPlaneNormalTowardAxis(t *testing.T, pl geom.Plane, axis math.Vector3, tilt float64) geom.Plane {
	t.Helper()
	n := unit(pl.Normal())
	k := unit(axis)
	return mustPlane(t, pl.Origin,
		n.Scale(math.Scalar(stdmath.Cos(tilt))).Add(k.Scale(math.Scalar(stdmath.Sin(tilt)))))
}

// TestN4BallFrameDeclinesStationOffTheDerivedCurve exercises holdsStation, the guard that proves the class
// HYPOTHESIS instead of assuming it: the two arm ball centres must actually lie on {offset plane} ∩
// {2r-torus}, i.e. the ball must really roll on the lateral torus arm's tube. Pushing the concave-cyl arm's
// station RADIALLY away from the torus axis takes its spine-circle distance off 2r by exactly that amount,
// so the ball no longer rides that tube.
//
// The direction matters, and the obvious one is a trap: on N4 that station sits at ψ=0, where dC/dψ = 2r·k̂,
// so displacing it along the torus AXIS moves it ALONG the curve and holdsStation correctly still accepts
// (a 1e-3 axial lift leaves it only 5e-8 off — measured). Radial is transverse to the curve there, which is
// what actually falsifies the hypothesis. The sub-weld case, which still accepts, shows the guard is
// thresholded rather than merely present; meridianOf's ρ and azimuth-side checks are unaffected by a radial
// push and the meridian span stays wide, so holdsStation is the only guard that can fire.
func TestN4BallFrameDeclinesStationOffTheDerivedCurve(t *testing.T) {
	t.Parallel()
	in := n4AcceptedBallFrameInputs(t)
	for _, tc := range []struct {
		name  string
		axial bool
		d     float64
		want  bool
	}{
		{"a radial push well inside weld still accepts", false, 0.1 * in.tol, true},
		{"a radial push off the tube declines", false, 1e-3, false},
		{"a gross radial push off the tube declines", false, 1.0, false},
		{"the same displacement AXIALLY runs along the curve and still accepts", true, 1e-3, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			m1 := displaceStation(t, in.torus, in.m1, tc.d, tc.axial)
			if _, ok := newN4BallFrame(in.torus, in.vplane, in.m0, m1, in.tol); ok != tc.want {
				t.Fatalf("newN4BallFrame accepted=%v with the concave-cyl station displaced %.3e "+
					"(axial=%v) from the 2r tube, want accepted=%v (weld-scaled tol %.3e)",
					ok, tc.d, tc.axial, tc.want, in.tol)
			}
		})
	}
}

// displaceStation moves p by d about the torus frame — along the axis when axial, otherwise along p's own
// cylindrical-radial direction. Radial is transverse to the ball-centre curve at the equatorial station, so
// it changes dist(p, spine circle) by exactly d; axial is tangent to it there, which is why the two
// directions of the same size have opposite expected verdicts.
func displaceStation(t *testing.T, torus geom.Torus, p math.Point3, d float64, axial bool) math.Point3 {
	t.Helper()
	k := unit(torus.AxisDir.AsVector())
	if axial {
		return p.TranslateBy(k.Scale(math.Scalar(d)))
	}
	rel := torus.Center.VectorTo(p)
	radial, err := math.UnitVector3FromVector(rel.Sub(k.Scale(rel.Dot(k))))
	if err != nil {
		t.Fatalf("station %v sits on the torus axis, so it has no radial direction: %v", p, err)
	}
	return p.TranslateBy(radial.AsVector().Scale(math.Scalar(d)))
}

// TestN4BallFrameDeclinesDegenerateMeridianSpan exercises the meridian-SPAN guard: a corner whose two arm
// stations sit at the same ψ has no span to loft over. It also pins the ADR-0042 unit fix — that guard
// compares two ANGLES and is thresholded by seamAngularTol, never by the model-scaled weld DISTANCE — by
// bracketing it at the angular tolerance while the passed-in distance tol (1.4e-6 here, five times larger)
// stays fixed.
func TestN4BallFrameDeclinesDegenerateMeridianSpan(t *testing.T) {
	t.Parallel()
	in := n4AcceptedBallFrameInputs(t)
	f, ok := newN4BallFrame(in.torus, in.vplane, in.m0, in.m1, in.tol)
	if !ok {
		t.Fatalf("newN4BallFrame declined the baseline N4 frame (torus %+v, m0 %v, m1 %v, tol %.3e), so the "+
			"span cases have no curve to place stations on", in.torus, in.m0, in.m1, in.tol)
	}
	for _, tc := range []struct {
		name string
		span float64
		want bool
	}{
		{"the same station twice declines", 0, false},
		{"half an angular tolerance of span declines", 0.5 * seamAngularTol, false},
		{"ten angular tolerances of span accepts", 10 * seamAngularTol, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			m1, ok := f.centerAt(f.psi0 + tc.span) // a station ON the curve, so only the span guard can fire
			if !ok {
				t.Fatalf("centerAt(psi0+%.3e) has no point on the derived curve", tc.span)
			}
			if _, ok := newN4BallFrame(in.torus, in.vplane, in.m0, m1, in.tol); ok != tc.want {
				t.Fatalf("newN4BallFrame accepted=%v for a meridian span of %.3e rad, want accepted=%v "+
					"(seamAngularTol %.0e, distance tol %.3e)", ok, tc.span, tc.want, seamAngularTol, in.tol)
			}
		})
	}
}

// TestN4CanalSurfaceDoesNotMutateTheCallerPath is the regression guard for the aliasing bug the pinning
// introduced: cornerBallPath is passed BY VALUE but its three slices share their backing arrays with the
// caller's, so pinning the end stations in place wrote THROUGH and destroyed the two derived end stations —
// which is also what left them unverified by TestN4BallPathRollsOnPlaneAndTorusArm.
func TestN4CanalSurfaceDoesNotMutateTheCallerPath(t *testing.T) {
	t.Parallel()
	arms, _ := classifyN4MixedArms(n4TestArms(t))
	res := ResolutionForPoints([]math.Point3{math.P3(0, 0, 0), math.P3(200, 200, 60)})
	pts := n4TestCornerPts(t, arms)
	path, ok := n4CornerBallPath(arms.torus.armSurface.(geom.Torus), arms.band.a.Geometry().(geom.Plane),
		pts.ballAB, pts.ballCD, res.Weld()*5)
	if !ok {
		t.Fatal("n4CornerBallPath declined the N4 corner")
	}
	before := ballPathEnds(path)
	if _, ok := cornerCanalSurface(path, pts, 5, res.Weld()); !ok {
		t.Fatal("cornerCanalSurface declined the N4 corner")
	}
	for i, name := range [3]string{"centers", "feetMid", "feetLateral"} {
		for j, was := range before[i] {
			if d := was.DistanceTo(ballPathEnds(path)[i][j]); d != 0 {
				t.Fatalf("cornerCanalSurface moved path.%s end %d by %v — it must pin a COPY", name, j, d)
			}
		}
	}
}

// ballPathEnds snapshots the first and last entry of each of the path's three station columns — the six
// entries end-pinning would overwrite if it wrote through the shared backing arrays.
func ballPathEnds(path cornerBallPath) [3][2]math.Point3 {
	return [3][2]math.Point3{
		{path.centers[0], path.centers[len(path.centers)-1]},
		{path.feetMid[0], path.feetMid[len(path.feetMid)-1]},
		{path.feetLateral[0], path.feetLateral[len(path.feetLateral)-1]},
	}
}

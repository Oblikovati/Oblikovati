// SPDX-License-Identifier: GPL-2.0-only

package ops

import (
	stdmath "math"

	"oblikovati.org/kernel/geom"
	"oblikovati.org/math"
)

// The N4 corner patch as the TRUE ROLLING-BALL CANAL (not a Coons fill).
//
// DRAWEXE 8.0.0 forensic (`restore data/CFI_e5678fil.rle s ; tscale s 0 0 0 10 ; explode s e ;
// blend result s 5 s_4 5 s_13 5 s_2 ; explode result f ; mksurface sf5 result_5 ; dump sf5`) shows
// OCCT's corner patch is a 3×9 pole net, u degree 2 RATIONAL × v degree 8, one Bézier span each way.
// Evaluating it proves what that shape means: at EVERY v the u-isocurve is an exact circular arc of
// radius 5.000000 (isoceles control triangle, end weights 1 and middle weight cos of the half-angle),
// i.e. the patch is the canal (pipe) surface swept by the rolling ball's own cross-section — NOT a
// variational/Coons fill. Pole row u=0 lies exactly in the vertical plane host; pole row u=1 lies
// exactly on the lateral torus arm (distance from its spine circle 5.000000). The ball centre
// C(v) = row0(v) + r·n̂_vplane then satisfies, to 5.5e-3 (OCCT's own G1-approximation residual; the two
// ends are exact):
//
//	n̂_vplane·(C − O) = const   and   dist(C, torus spine circle) = 2r
//
// So the ball transits the corner tangent to the vertical plane and ROLLING ON THE LATERAL TORUS ARM
// (the cap-rim fillet tube), and its centre curve is the exact intersection of the plane offset r from
// the vplane with the torus of the same spine circle and tube radius 2r. Numerically integrating the
// canal over that exact curve gives area 80.7725 against OCCT's 80.7328 (+0.049%) — versus 59.273 for
// the chord-projected coons4 fill this replaces (−27%). Measured surface-to-surface against result_5
// rebuilt from the dump above, the Hausdorff distance falls from 1.032/2.997 to 0.0056/0.0019.
//
// The two on-host rails are then the ball's CONTACT LOCI and fall out of the same construction rather
// than being guessed: on the plane the contact is C − r·n̂ (a rigid offset of the centre curve, which is
// exactly why a straight chord was wrong); on the torus it is the closest point on the tube, which
// because |C − spine| = 2r is the midpoint of C and its spine foot. Both are just
// geom.ClosestPointOnSurface, and geom.LoftCanalStations asserts each foot sits at radius — the
// do-no-harm fidelity gate.

// n4CanalStationCount / n4CanalEndCluster set the station distribution along the ball-centre curve. Every
// station column is EXACT on the true envelope, so the patch's only residual is the loft's CUBIC
// v-interpolation between stations, and the certificate that gates it is the G1 crease against the four
// analytic neighbours (seamAngularTol = 1e-6 rad). Two distinct residuals compete there and they want
// opposite spacings, which is why the distribution is a BLEND rather than uniform:
//
//   - the two ARM sides (v=0 / v=1) hold G1 against their arm cylinder through the loft's END v-tangent,
//     whose interpolation error falls with the spacing AT THE ENDS → wants cosine end-clustering;
//   - the TORUS side (u=1) holds G1 through the foot-locus, which leaves the torus BETWEEN stations by the
//     interior interpolation error → wants uniform spacing.
//
// Measured on N4 (crease rad, arm side / torus side): uniform N=33 → 3.2e-5 / 2.3e-6; full cosine N=33 →
// 6.7e-8 / 3.4e-6. A 0.7 blend equalises them, and N=65 then measures 1.7e-7 / 3.2e-7 — a 3.1× margin
// under the gate. Raising the count is the only knob that improves both at once (N=81 → 8.0e-8 / 1.7e-7).
const (
	n4CanalStationCount = 65
	n4CanalEndCluster   = 0.7
)

// n4StationParam maps station i to its fraction along the meridian span, blending uniform spacing with
// cosine end-clustering per n4CanalEndCluster (see that constant for why the blend exists).
func n4StationParam(i int) float64 {
	x := float64(i) / float64(n4CanalStationCount-1)
	return (1-n4CanalEndCluster)*x + n4CanalEndCluster*(1-stdmath.Cos(stdmath.Pi*x))/2
}

// n4BallPath is the corner's rolling-ball path: the exact ball-centre curve stations plus the ball's two
// contact loci — vplane feet (the u=0 rail) and torus-arm feet (the u=1 rail) — one triple per station.
type n4BallPath struct {
	centers, feetVplane, feetTorus []math.Point3
}

// n4CornerBallPath derives the corner's exact rolling-ball path from the two terminating arms' ball
// centres m0 (band arm) and m1 (concave-cyl arm). The centre curve is
// {n̂·(P−O) = a} ∩ {dist(P, spine circle) = 2r}, parametrised by the 2r-torus MERIDIAN angle ψ:
//
//	ρ(ψ) = Rs + 2r·cosψ ,  h(ψ) = 2r·sinψ ,  cosθ(ψ) = a/ρ(ψ) ,  sinθ(ψ) = σ·√(1−cos²θ)
//	C(ψ) = O + ρ(ψ)·(cosθ·n̂ + sinθ·(k̂×n̂)) + h(ψ)·k̂
//
// ψ (not the azimuth θ) is the regular parameter: h(θ) carries a square-root branch point where the
// curve leaves the 2r-torus equator (dh/dθ → ∞), while ψ is smooth across the whole span. Tangency to
// both arm spines is then automatic — dC/dψ has no k̂ component at ψ=±π/2 (so it runs along the
// vplane∧cap-plane band spine) and equals 2r·k̂ at ψ=0 (the concave-cyl arm's own axis direction) — which
// is why the patch is G1 to both arms with no extra condition.
//
// ok=false (do-no-harm decline, the corner keeps its prior path) when the vplane is not parallel to the
// torus axis, when the offset plane misses the 2r-torus, when the azimuth degenerates, or when either
// arm station does not actually lie on the derived curve — that last check is what proves the class
// hypothesis instead of assuming it. tol is the model-relative weld distance (ADR-0042).
func n4CornerBallPath(torus geom.Torus, vplane geom.Plane, m0, m1 math.Point3, tol float64) (n4BallPath, bool) {
	f, ok := newN4BallFrame(torus, vplane, m0, m1, tol)
	if !ok {
		return n4BallPath{}, false
	}
	path := n4BallPath{}
	for i := 0; i < n4CanalStationCount; i++ {
		c, ok := f.centerAt(f.psi0 + (f.psi1-f.psi0)*n4StationParam(i))
		if !ok {
			return n4BallPath{}, false
		}
		_, _, fv := geom.ClosestPointOnSurface(vplane, c)
		_, _, ft := geom.ClosestPointOnSurface(torus, c)
		path.centers = append(path.centers, c)
		path.feetVplane = append(path.feetVplane, fv)
		path.feetTorus = append(path.feetTorus, ft)
	}
	return path, true
}

// n4BallFrame is the closed-form ball-centre curve: the torus frame (origin O, axis k̂, spine radius Rs,
// tube radius 2r), the vplane's in-plane offset a and normal n̂ with its azimuth binormal, the azimuth
// sign σ, and the meridian span [psi0, psi1] the two arm stations pin.
type n4BallFrame struct {
	origin     math.Point3
	axis       math.Vector3 // k̂
	normal     math.Vector3 // n̂, the vplane normal (⊥ k̂)
	binormal   math.Vector3 // k̂ × n̂
	spine      float64      // Rs
	tube       float64      // 2r
	offset     float64      // a = n̂·(C − O)
	sign       float64      // σ = sign of the azimuth component
	psi0, psi1 float64
}

// n4BallFrame.centerAt evaluates C(ψ). ok=false when |a| exceeds ρ(ψ) (the offset plane misses the
// 2r-torus at this meridian angle) — the curve does not exist there and the corner declines.
func (f n4BallFrame) centerAt(psi float64) (math.Point3, bool) {
	rho := f.spine + f.tube*stdmath.Cos(psi)
	if rho <= 0 || stdmath.Abs(f.offset) > rho {
		return math.Point3{}, false
	}
	cosT := f.offset / rho
	sinT := f.sign * stdmath.Sqrt(stdmath.Max(0, 1-cosT*cosT))
	radial := f.normal.Scale(math.Scalar(cosT)).Add(f.binormal.Scale(math.Scalar(sinT)))
	return f.origin.
		TranslateBy(radial.Scale(math.Scalar(rho))).
		TranslateBy(f.axis.Scale(math.Scalar(f.tube * stdmath.Sin(psi)))), true
}

// n4BallFrame.meridianOf resolves one arm ball centre into (ψ, azimuth sign): ψ = atan2(h, ρ−Rs) on the
// 2r-torus meridian, σ from the centre's component along k̂×n̂. ok=false when the azimuth degenerates
// (the centre sits on the n̂ axis, where dθ/dψ blows up).
func (f n4BallFrame) meridianOf(m math.Point3, tol float64) (psi, sign float64, ok bool) {
	rel := f.origin.VectorTo(m)
	h := float64(rel.Dot(f.axis))
	radial := rel.Sub(f.axis.Scale(math.Scalar(h)))
	rho := float64(radial.Length())
	side := float64(radial.Dot(f.binormal))
	if rho <= tol || stdmath.Abs(side) <= tol {
		return 0, 0, false
	}
	return stdmath.Atan2(h, rho-f.spine), stdmath.Copysign(1, side), true
}

// n4BallFrame.holdsStation reports whether the closed-form curve reproduces an arm's ball centre at its
// own meridian angle. This is the load-bearing class check: the two arm stations must actually LIE on
// {offset plane} ∩ {2r-torus}, so a corner that merely looks N4-shaped but whose ball never rolls on the
// lateral torus arm declines instead of receiving a wrong patch.
func (f n4BallFrame) holdsStation(m math.Point3, psi, tol float64) bool {
	c, ok := f.centerAt(psi)
	return ok && float64(c.DistanceTo(m)) <= tol
}

// vplaneParallelToTorusAxis is the N4 CLASS test, n̂·k̂ = 0: the host plane must contain the torus axis
// direction, because only then is the plane constraint n̂·(P−O) = a expressible as ρ(ψ)·cosθ = a — the step
// the whole closed-form ball-centre curve rests on. A named predicate rather than an inline condition so the
// branch can be exercised on its own (the O1-reuse safety story leans on it DECLINING, and O1's hosts are
// cylinders). It is a SINE and is therefore thresholded by the layer's angular tolerance, never by a
// model-scaled length (ADR-0042).
func vplaneParallelToTorusAxis(torus geom.Torus, vplane geom.Plane) bool {
	k := unit(torus.AxisDir.AsVector())
	n := unit(vplane.Normal())
	return stdmath.Abs(float64(n.Dot(k))) <= seamAngularTol
}

// newN4BallFrame builds the frame from the lateral torus arm, the shared vertical plane, and the two arm
// ball centres, then self-checks it against both stations. The tube radius is 2·MinorRadius because the
// rolling ball (radius r = the tube's own minor radius) rides on the OUTSIDE of that tube. tol is a
// DISTANCE (the model-relative weld); the plane-parallel test is a sine and so takes the layer's angular
// tolerance instead, never a length — and so does the meridian-SPAN degeneracy test below, which compares
// two ψ ANGLES and must therefore never be thresholded by a model-scaled length (ADR-0042: mixing the two
// makes the guard's trip point drift with the model's size).
func newN4BallFrame(torus geom.Torus, vplane geom.Plane, m0, m1 math.Point3, tol float64) (n4BallFrame, bool) {
	if !vplaneParallelToTorusAxis(torus, vplane) {
		return n4BallFrame{}, false
	}
	k := unit(torus.AxisDir.AsVector())
	n := unit(vplane.Normal())
	f := n4BallFrame{
		origin: torus.Center, axis: k, normal: n, binormal: k.Cross(n),
		spine: torus.MajorRadius, tube: 2 * torus.MinorRadius,
		offset: float64(torus.Center.VectorTo(m0).Dot(n)),
	}
	psi0, sign0, ok0 := f.meridianOf(m0, tol)
	psi1, sign1, ok1 := f.meridianOf(m1, tol)
	if !ok0 || !ok1 || sign0 != sign1 || stdmath.Abs(psi1-psi0) <= seamAngularTol {
		return n4BallFrame{}, false
	}
	f.sign, f.psi0, f.psi1 = sign0, psi0, psi1
	if !f.holdsStation(m0, psi0, tol) || !f.holdsStation(m1, psi1, tol) {
		return n4BallFrame{}, false
	}
	return f, true
}

// n4CanalSurface lofts the rolling-ball path into the corner's canal BSpline, PINNING the two end
// stations to the corner points the terminating arms already own. That pinning is the weld: the loft's
// v=0 / v=1 cross-sections are then the byte-same circles as pts.arcAB / pts.arcCD (same centre, same
// radius, same two endpoints), so the patch's own boundary coincides with the arcs the arm faces trim to
// rather than merely approximating them.
func n4CanalSurface(path n4BallPath, pts n4CornerPts, r, weld float64) (geom.BSplineSurface, bool) {
	n := len(path.centers)
	if n < 2 || len(path.feetVplane) != n || len(path.feetTorus) != n {
		return geom.BSplineSurface{}, false // a path whose three columns disagree has no end to pin
	}
	surf, err := geom.LoftCanalStations(
		pinnedEndStations(path.centers, pts.ballBand, pts.ballCcyl),
		pinnedEndStations(path.feetVplane, pts.a, pts.d),
		pinnedEndStations(path.feetTorus, pts.b, pts.c),
		r, weld)
	return surf, err == nil
}

// pinnedEndStations copies a station column and replaces its two ends with the corner points the arms own.
// The COPY is load-bearing: n4BallPath is passed by value but its three slices share their backing arrays
// with the caller's path, so pinning in place would write THROUGH and silently overwrite the caller's
// derived ball path — which would also make the two derived end stations unobservable to any caller (a
// test included) that measures the path after lofting it.
func pinnedEndStations(stations []math.Point3, first, last math.Point3) []math.Point3 {
	out := append(make([]math.Point3, 0, len(stations)), stations...)
	out[0], out[len(out)-1] = first, last
	return out
}

// n4CanalRails extracts the canal's two on-host boundary isoparms — the ball's contact loci, which lie ON
// their host by construction. u=u0 is the vplane locus A→D (returned REVERSED as the D→A rail the patch
// ring wants); u=u1 is the torus-arm locus B→C. Replacing the old chord-projection rails with these is
// the whole geometric fix: on a plane, projecting a straight chord returns the chord, so railDA came out
// exactly linear where the true contact locus bows 3.00 units.
func n4CanalRails(surf geom.BSplineSurface) (railBC, railDA geom.Curve3, ok bool) {
	u0, u1 := surf.UDomain()
	locusV, err := geom.SurfaceIsoCurve(surf, true, u0)
	if err != nil {
		return nil, nil, false
	}
	locusT, err := geom.SurfaceIsoCurve(surf, true, u1)
	if err != nil {
		return nil, nil, false
	}
	return locusT, geom.ReverseCurve3(locusV), true
}

// curveMidPoint is a curve's own mid-domain point — the ledger's `mid` witness for a non-arc rail, taken
// from Domain() rather than assuming a [0,1] parametrisation (a lofted isoparm carries chord-length
// knots).
func curveMidPoint(c geom.Curve3) math.Point3 {
	lo, hi := c.Domain()
	return c.PointAt((lo + hi) / 2)
}

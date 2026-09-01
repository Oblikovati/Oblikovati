// SPDX-License-Identifier: GPL-2.0-only

package ops

import (
	stdmath "math"

	"oblikovati.org/kernel/geom"
	"oblikovati.org/kernel/ops/tessellate"
	"oblikovati.org/math"
)

// The O1-class corner's ROLLING-BALL CANAL — the cylinder-on-cylinder sibling of the N4 frame in
// fillet_curved_mixed_canal.go.
//
// Both classes' corner patch is the envelope of ONE ball of radius r that transits the corner while staying
// tangent to (i) the MID host it rides and (ii) the LATERAL arm's tube it rolls on. What differs is which
// surfaces those are, and therefore the closed form of the ball-centre curve:
//
//	N4:  mid = a PLANE  , lateral = a TORUS arm    → {n̂·(P−O) = a} ∩ {dist(P, spine circle) = 2r}
//	O1:  mid = a CYLINDER, lateral = a CYLINDER arm → {dist(P, axis) = ρ}  ∩ {dist(P, arm axis) = 2r}
//
// So newN4BallFrame's two class assumptions (n̂·k̂ = 0, tube = 2·MinorRadius) are NOT transplanted; they are
// re-derived below for O1's own geometry and checked against it. DRAWEXE 8.0.0 confirms the family
// (`restore data/CFI_f5678fin.rle s ; tscale s 0 0 0 10 ; explode s e ; blend result s 5 s_7 5 s_6 5 s_14 ;
// explode result f ; mksurface sf result_5 ; bounds sf ; svalue sf u v`): OCCT's corner patch has v-domain
// EXACTLY [0, π/2] — the quarter turn α spans below — and every u-isocurve of it is a circular arc of radius
// 5.000000 (fitted residual ≤1.8e-7) whose centre lies at 54.975–55.000 from the boss axis and at
// 9.927–10.000 from the lateral arm's axis, exact at both ends. That is this canal, carrying OCCT's own
// ≤0.073 (1.5% of r) G1-approximation residual in the interior.
//
// The frame's parameter is α, the angle AROUND THE LATERAL ARM'S AXIS. It is the regular parameter for the
// same reason ψ is in the N4 frame: the ĵ-offset t(α) carries the square-root branch, while α is smooth
// across the whole span. And the span is a quarter turn by construction, not by luck — see o1CanalSpan.

// o1BallFrame is the closed-form ball-centre curve of an O1-class corner:
//
//	C(α) = A0 + t(α)·ĵ + 2r·(cos α·û + sin α·k̂),  (p + t(α))² + (q + 2r·cos α)² = ρ²
//
// with k̂ the mid host cylinder's axis, ĵ the lateral arm's axis (⊥ k̂), û = ĵ×k̂, and (p, q) the components
// of A0−O in that frame. t(α) takes the σ branch the corner's own stations pin.
type o1BallFrame struct {
	latOrigin math.Point3  // A0, a point on the lateral arm's axis
	latDir    math.Vector3 // ĵ
	binormal  math.Vector3 // û = ĵ × k̂
	axis      math.Vector3 // k̂, the mid host cylinder's axis
	p, q      float64      // (A0−O)·ĵ , (A0−O)·û
	rho       float64      // ρ, the ball-centre distance from the mid host axis
	tube      float64      // 2r, the ball-centre distance from the lateral arm's axis
	sign      float64      // σ, the branch of the ĵ offset
	a0, a1    float64      // the two terminating arms' α stations
}

// centerAt evaluates C(α). ok=false when |q + 2r·cos α| exceeds ρ — the ρ-cylinder and the 2r-tube do not
// meet at this α, so the curve does not exist there and the corner declines.
func (f o1BallFrame) centerAt(alpha float64) (math.Point3, bool) {
	across := f.q + f.tube*stdmath.Cos(alpha)
	disc := f.rho*f.rho - across*across
	if disc <= 0 {
		return math.Point3{}, false
	}
	t := -f.p + f.sign*stdmath.Sqrt(disc)
	return f.latOrigin.
		TranslateBy(f.latDir.Scale(math.Scalar(t))).
		TranslateBy(f.binormal.Scale(math.Scalar(across - f.q))).
		TranslateBy(f.axis.Scale(math.Scalar(f.tube * stdmath.Sin(alpha)))), true
}

// newO1BallFrame builds the frame from the two host axes, the lateral arm's axis, and ρ. ok=false when the
// lateral arm's axis is not perpendicular to the mid host's axis — the ONE class assumption the closed form
// rests on, because only then is the ρ constraint expressible in the (ĵ, û) plane at all. It is a SINE and
// is therefore thresholded by the layer's angular tolerance, never by a model-scaled length (ADR-0042).
func newO1BallFrame(midAxisOrigin math.Point3, midAxis math.Vector3, lat geom.Cylinder, rho, tube float64) (o1BallFrame, bool) {
	k := unit(midAxis)
	j := unit(lat.AxisDir.AsVector())
	if stdmath.Abs(float64(j.Dot(k))) > tessellate.SeamAngularTol {
		return o1BallFrame{}, false
	}
	u := j.Cross(k)
	rel := midAxisOrigin.VectorTo(lat.Origin)
	return o1BallFrame{
		latOrigin: lat.Origin, latDir: j, binormal: u, axis: k,
		p: float64(rel.Dot(j)), q: float64(rel.Dot(u)),
		rho: rho, tube: tube,
	}, true
}

// o1CanalSpan pins the frame's two stations and its σ branch, and is the load-bearing CLASS CHECK: both α's
// are forced by tangency, not chosen, so a corner that merely looks O1-shaped declines here.
//
//   - The TERMINATING CYLINDER arm's spine is a line ∥ k̂ on the ρ-cylinder. It meets the 2r-tube where
//     |q + 2r·cos α| = |spine·û|, and that contact is TANGENTIAL — both this spine and the lateral arm's axis
//     lie in planes offset r either side of their shared PLANE host, whose normal is ±û (it contains both ĵ
//     and k̂), so their closest approach is exactly 2r. Hence cos α0 = ±1 and sin α0 = 0.
//   - The COVE TORUS arm's spine circle lies in the plane offset r from the SAME shared plane on the void
//     side, at k̂-height 2r above the lateral axis, so sin α1 = ±1 and cos α1 = 0.
//
// The span is therefore exactly a quarter turn — which is independently what OCCT's patch v-domain measures
// (see this file's header). ok=false when either tangency does not hold to tol.
func (f *o1BallFrame) o1CanalSpan(spineOrigin math.Point3, coveCenter math.Point3, tol float64) bool {
	rel := f.latOrigin.VectorTo(spineOrigin)
	across, along := float64(rel.Dot(f.binormal)), float64(rel.Dot(f.latDir))
	lift := float64(f.latOrigin.VectorTo(coveCenter).Dot(f.axis))
	if stdmath.Abs(stdmath.Abs(across)-f.tube) > tol || stdmath.Abs(stdmath.Abs(lift)-f.tube) > tol {
		return false
	}
	f.a0 = quarterTurn(across, 0)
	f.a1 = quarterTurn(0, lift)
	f.sign = stdmath.Copysign(1, along+f.p)
	return true
}

// quarterTurn returns the exact quadrant angle atan2(lift, across) rounds to when one of the two is zero —
// 0, π/2, π or −π/2. Computing it from the SIGNS rather than from atan2 of measured values keeps both
// stations at machine-exact quadrant angles, which is what makes the two pinned end cross-sections coincide
// with the arms' own arcs rather than merely approximate them.
func quarterTurn(across, lift float64) float64 {
	if lift != 0 {
		return stdmath.Copysign(stdmath.Pi/2, lift)
	}
	if across < 0 {
		return stdmath.Pi
	}
	return 0
}

// holdsStations reports whether the closed-form curve's two end centres really lie on the two terminating
// arms' own spines — the check that proves the class hypothesis instead of assuming it (the o1 analogue of
// n4BallFrame.holdsStation). m0 must sit on the cylinder arm's spine LINE, m1 on the cove arm's spine
// CIRCLE; both must sit at ρ from the mid host axis, which is what makes the ball ride that host.
func (f o1BallFrame) holdsStations(spine geom.Cylinder, cove geom.Torus, midAxisOrigin math.Point3, tol float64) bool {
	m0, ok0 := f.centerAt(f.a0)
	m1, ok1 := f.centerAt(f.a1)
	if !ok0 || !ok1 {
		return false
	}
	onLine := float64(m0.DistanceTo(footOnLine(m0, spine.Origin, spine.AxisDir.AsVector()))) <= tol
	onCircle := stdmath.Abs(float64(m1.DistanceTo(cove.Center))-cove.MajorRadius) <= tol &&
		stdmath.Abs(float64(cove.Center.VectorTo(m1).Dot(unit(cove.AxisDir.AsVector())))) <= tol
	return onLine && onCircle && f.holdsRadius(m0, midAxisOrigin, tol) && f.holdsRadius(m1, midAxisOrigin, tol)
}

// holdsRadius reports whether a station sits at ρ from the mid host's axis.
func (f o1BallFrame) holdsRadius(m, midAxisOrigin math.Point3, tol float64) bool {
	rel := midAxisOrigin.VectorTo(m)
	radial := rel.Sub(f.axis.Scale(rel.Dot(f.axis)))
	return stdmath.Abs(float64(radial.Length())-f.rho) <= tol
}

// o1CanalStationCount / o1CanalEndCluster set the station distribution along O1's ball-centre curve. O1 gets
// its OWN pair rather than reusing N4's because the two knobs trade off against different residuals per class
// and N4's are pinned by its fingerprint. See o1StationParam for the measurements.
const (
	o1CanalStationCount = 65
	o1CanalEndCluster   = 0.7
)

// o1StationParam maps station i to its fraction along the α span.
func o1StationParam(i int) float64 {
	x := float64(i) / float64(o1CanalStationCount-1)
	return (1-o1CanalEndCluster)*x + o1CanalEndCluster*(1-stdmath.Cos(stdmath.Pi*x))/2
}

// o1CornerBallPath samples the frame into the corner's exact rolling-ball path: the centre stations plus the
// ball's two contact loci — the MID host feet (the u=0 rail) and the LATERAL arm feet (the u=1 rail). Both
// loci lie on the ball's characteristic circle by construction: the centre curve keeps a CONSTANT distance
// to each of the two axes, so each radial offset is orthogonal to dC/dα, which is exactly the envelope
// condition (X−C)·C′ = 0. It reuses the N4 frame's station distribution, whose blend of uniform and cosine
// end-clustering equalises the arm-side and locus-side interpolation residuals (see n4CanalEndCluster).
func o1CornerBallPath(f o1BallFrame, mid, lateral geom.Surface) (cornerBallPath, bool) {
	path := cornerBallPath{}
	for i := range o1CanalStationCount {
		c, ok := f.centerAt(f.a0 + (f.a1-f.a0)*o1StationParam(i))
		if !ok {
			return cornerBallPath{}, false
		}
		_, _, fm := geom.ClosestPointOnSurface(mid, c)
		_, _, fl := geom.ClosestPointOnSurface(lateral, c)
		path.centers = append(path.centers, c)
		path.feetMid = append(path.feetMid, fm)
		path.feetLateral = append(path.feetLateral, fl)
	}
	return path, true
}

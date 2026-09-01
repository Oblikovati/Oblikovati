// SPDX-License-Identifier: GPL-2.0-only

package blend

import (
	stdmath "math"

	"oblikovati.org/kernel/geom"
	opstol "oblikovati.org/kernel/ops/internal/tol"
	"oblikovati.org/math"
)

// The torus∩cylinder miter corner-ball centre — split out of fillet_miter_curved.go (which now owns
// just the arm-pair RECOGNITION/dispatch) so both files stay under the 500-line/one-responsibility
// rule. Family D (torus∧torus, fillet_miter_curved_torustorus.go) solves its own corner-ball centre by
// a circle∩circle intersection instead (both arm spines are already coplanar circles there), so it
// does not call anything in this file.

// miterCornerBallCenter is the corner ball centre m* — the crossing of the two arm spines (the torus
// major circle ∩ the cylinder axis line), where the one ball tangent to both hosts sits. It solves
// the line∩circle in closed form (the line pierces the torus plane, or lies in it — a quadratic) and
// keeps the root nearest the corner vertex. ok=false when they do not cross (no equal-r bisector).
func miterCornerBallCenter(arms curvedMiterArms, vp math.Point3, res opstol.Resolution) (math.Point3, bool) {
	roots, ok := lineTorusSpineCrossings(arms.cyl, arms.tor)
	if !ok {
		return math.Point3{}, false
	}
	best, found := math.Point3{}, false
	tol := res.Weld() * (arms.tor.MajorRadius + arms.tor.MinorRadius)
	for _, p := range roots {
		m, mok := armBallCenter(arms.tor, p)
		if !mok || float64(m.DistanceTo(p)) > tol {
			continue // pierce point is off the torus major circle: not a spine crossing
		}
		if !found || p.DistanceTo(vp) < best.DistanceTo(vp) {
			best, found = p, true
		}
	}
	return best, found
}

// lineTorusSpineCrossings returns the point(s) where the cylinder axis line meets the torus major
// circle's plane: the single pierce point when the line crosses the plane, or the two |·−C|=R roots
// when the line lies in the plane (a quadratic in the axial parameter). ok=false when the line is
// parallel to but off the plane (no crossing).
func lineTorusSpineCrossings(cyl geom.Cylinder, tor geom.Torus) ([]math.Point3, bool) {
	o2, d2 := cyl.Origin, cyl.AxisDir.AsVector()
	n := tor.AxisDir.AsVector()
	q := tor.Center.VectorTo(o2)
	denom := float64(d2.Dot(n))
	if stdmath.Abs(denom) > sinFloor {
		t := -float64(q.Dot(n)) / denom
		return []math.Point3{o2.TranslateBy(d2.Scale(math.Scalar(t)))}, true
	}
	if stdmath.Abs(float64(q.Dot(n))) > sinFloor*(tor.MajorRadius+1) {
		return nil, false // line parallel to the torus plane but offset from it — no crossing
	}
	b := float64(q.Dot(d2))
	disc := b*b - (float64(q.Dot(q)) - tor.MajorRadius*tor.MajorRadius)
	if disc < 0 {
		return nil, false
	}
	s := stdmath.Sqrt(disc)
	return []math.Point3{o2.TranslateBy(d2.Scale(math.Scalar(-b + s))), o2.TranslateBy(d2.Scale(math.Scalar(-b - s)))}, true
}

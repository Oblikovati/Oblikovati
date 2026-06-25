// SPDX-License-Identifier: GPL-2.0-only

package geom

import (
	stdmath "math"

	"oblikovati.org/math"
)

// Oblique cone–plane section (M2 Phase-1 follow-up, Oblikovati/Oblikovati#1375). A plane that is
// neither perpendicular to the cone axis (a circle, #1334) nor parallel to it (an axis-parallel
// hyperbola, #1372) cuts the cone in an ELLIPSE (tilt steeper than the generators wrt the axis), a
// HYPERBOLA (tilt shallower) or — at the boundary tilt, parallel to one generator — a PARABOLA.
//
// All three come from ONE derivation: parameterize the plane by its own orthonormal in-plane basis
// (UAxis ê1, VAxis ê2), P(s,t)=O+s·ê1+t·ê2, and substitute into the cone identity
// ((P−V)·â)² = cos²α·|P−V|². The result is a general 2D conic A·s²+B·st+C·t²+D·s+E·t+F=0 whose
// quadratic-form eigenvalues classify it (same-sign→ellipse, opposite→hyperbola, one zero→parabola)
// and whose principal axes and centre give the analytic conic in 3D. This unifies the section types
// the cone can produce without per-case trigonometry — the conic falls out of the algebra.

// coneObliqueConic returns the conic an oblique plane cuts from a cone (n the unit plane normal, axis
// the unit cone axis), as a geom.EllipseFull (elliptic tilt) or geom.Hyperbola branch on the cone's
// nappe (hyperbolic tilt). The parabolic boundary tilt is deferred (handled=false) — a measure-zero
// case left to the numeric tracer for now (Oblikovati/Oblikovati#1375 notes it may be its own slice).
func coneObliqueConic(pl Plane, cone Cone, n, axis math.Vector3) ([]Curve3, bool) {
	e1, e2 := pl.UAxis.AsVector(), pl.VAxis.AsVector()
	w := cone.Apex.VectorTo(pl.Origin) // O − V, the apex-to-plane-origin vector
	c2 := cosSquared(cone.HalfAngle)
	a1, a2, aw := float64(axis.Dot(e1)), float64(axis.Dot(e2)), float64(axis.Dot(w))
	w1, w2, ww := float64(w.Dot(e1)), float64(w.Dot(e2)), float64(w.Dot(w))
	q := conic2D{
		a: a1*a1 - c2, b: 2 * a1 * a2, c: a2*a2 - c2,
		d: 2*a1*aw - 2*c2*w1, e: 2*a2*aw - 2*c2*w2, f: aw*aw - c2*ww,
	}
	return q.curve3(pl.Origin, e1, e2, n, axis, cone.Apex)
}

// conic2D is a general planar conic a·s²+b·st+c·t²+d·s+e·t+f=0 in a plane's (s,t)=(ê1,ê2) coordinates.
type conic2D struct {
	a, b, c, d, e, f float64
}

// curve3 classifies the conic by its quadratic-form eigenvalues and lifts it to the analytic 3D curve:
// a same-sign pair is an ellipse, an opposite-sign pair a hyperbola (the branch on the cone's nappe), a
// single (near-)zero eigenvalue a parabola. handled=false only on a fully degenerate section (both
// eigenvalues zero, or the parabola collapses to parallel lines), where the caller keeps the fallback.
func (q conic2D) curve3(o math.Point3, e1, e2, n, axis math.Vector3, apex math.Point3) ([]Curve3, bool) {
	l1, l2, u1, u2 := symmetricEig2(q.a, q.b/2, q.c)
	dir1 := e1.Scale(math.Scalar(u1[0])).Add(e2.Scale(math.Scalar(u1[1])))
	dir2 := e1.Scale(math.Scalar(u2[0])).Add(e2.Scale(math.Scalar(u2[1])))
	z1, z2 := stdmath.Abs(l1) < 1e-9, stdmath.Abs(l2) < 1e-9
	if z1 && z2 {
		return nil, false // both eigenvalues zero: a fully degenerate section
	}
	if z2 { // λ1 is the nonzero (cross) eigenvalue, dir2 the parabola's opening (zero) direction
		return obliqueParabola(o, l1, u1, u2, dir1, dir2, q)
	}
	if z1 {
		return obliqueParabola(o, l2, u2, u1, dir2, dir1, q)
	}
	s0, t0, ok := q.center()
	if !ok {
		return nil, false
	}
	center := o.TranslateBy(e1.Scale(math.Scalar(s0))).TranslateBy(e2.Scale(math.Scalar(t0)))
	f0 := q.f + 0.5*(q.d*s0+q.e*t0) // the conic value at its centre: λ1·u²+λ2·v²=−f0 in principal axes
	if l1*l2 > 0 {
		return obliqueEllipse(center, n, l1, l2, f0, dir1, dir2)
	}
	return obliqueHyperbola(center, l1, l2, f0, dir1, dir2, axis, apex)
}

// obliqueParabola builds the parabolic section (plane parallel to a generator, the boundary tilt). In
// principal coords with the zero-eigenvalue direction w the conic is λ·u²+d_u·u+d_w·w+f=0 (no w²
// term), so w = quadratic(u): a parabola with cross axis u (crossDir), opening along w (zeroDir). Its
// vertex is at u*=−d_u/(2λ), w*=(d_u²/(4λ)−f)/d_w; the shape coefficient k=−λ/d_w gives the focal
// length f=1/(4|k|) and orients the opening. d_w≈0 means the section is two parallel lines — deferred.
func obliqueParabola(o math.Point3, lam float64, crossEig, zeroEig [2]float64, crossDir, zeroDir math.Vector3, q conic2D) ([]Curve3, bool) {
	duP := q.d*crossEig[0] + q.e*crossEig[1]
	dwP := q.d*zeroEig[0] + q.e*zeroEig[1]
	if stdmath.Abs(dwP) < 1e-12 {
		return nil, false // the quadratic in u has no linear w term: parallel lines, not a parabola
	}
	uStar := -duP / (2 * lam)
	wStar := (duP*duP/(4*lam) - q.f) / dwP
	vertex := o.TranslateBy(crossDir.Scale(math.Scalar(uStar))).TranslateBy(zeroDir.Scale(math.Scalar(wStar)))
	k := -lam / dwP
	axisDir := zeroDir
	if k < 0 {
		axisDir = zeroDir.Scale(-1) // open toward +w when k>0, −w when k<0
	}
	par, err := NewParabola(vertex, axisDir, crossDir, 1/(4*stdmath.Abs(k)))
	if err != nil {
		return nil, false
	}
	return []Curve3{par}, true
}

// center solves the 2×2 system [[a, b/2],[b/2, c]]·[s0,t0] = −[d/2, e/2] for the conic's centre in
// (s,t). ok=false when the form is singular (no unique centre — the parabolic/degenerate case).
func (q conic2D) center() (s0, t0 float64, ok bool) {
	h := q.b / 2
	det := q.a*q.c - h*h
	if stdmath.Abs(det) < 1e-12 {
		return 0, 0, false
	}
	s0 = (-q.d/2*q.c - h*(-q.e/2)) / det
	t0 = (q.a*(-q.e/2) - (-q.d/2)*h) / det
	return s0, t0, true
}

// obliqueEllipse builds the elliptic section: in principal axes λ1·u²+λ2·v²=−f0, so the semi-axis
// along eigenvector i is √(−f0/λi) (both positive for a real ellipse). The larger is the major axis.
func obliqueEllipse(center math.Point3, n math.Vector3, l1, l2, f0 float64, dir1, dir2 math.Vector3) ([]Curve3, bool) {
	r1, r2 := -f0/l1, -f0/l2
	if r1 <= 0 || r2 <= 0 {
		return nil, false // no real ellipse (the plane misses the nappe)
	}
	semi1, semi2 := stdmath.Sqrt(r1), stdmath.Sqrt(r2)
	majorDir, majorR, minorR := dir1, semi1, semi2
	if semi2 > semi1 {
		majorDir, majorR, minorR = dir2, semi2, semi1
	}
	el, err := NewEllipseFull(center, n, majorDir, majorR, minorR)
	if err != nil {
		return nil, false
	}
	return []Curve3{el}, true
}

// obliqueHyperbola builds the hyperbolic section's branch ON the cone's nappe. In principal axes
// λ1·u²+λ2·v²=−f0 with λ1,λ2 opposite signs: the transverse axis is the eigenvector whose −f0/λ is
// positive (real semi A=√(−f0/λ)), the other gives the conjugate semi B=√(|−f0/λ|). The branch opens
// toward whichever nappe-side vertex lies on the actual cone (apex+t·axis, t>0), so the transverse
// direction is signed to point from the centre to that vertex.
func obliqueHyperbola(center math.Point3, l1, l2, f0 float64, dir1, dir2, axis math.Vector3, apex math.Point3) ([]Curve3, bool) {
	transDir, transL, conjL := dir1, l1, l2
	if -f0/l1 <= 0 { // dir1 is the imaginary axis; swap so transDir is the real (transverse) one
		transDir, transL, conjL = dir2, l2, l1
	}
	a := stdmath.Sqrt(-f0 / transL)
	b := stdmath.Sqrt(f0 / conjL)
	if stdmath.IsNaN(a) || stdmath.IsNaN(b) || a <= 0 || b <= 0 {
		return nil, false
	}
	if float64(apex.VectorTo(center.TranslateBy(transDir.Scale(math.Scalar(a)))).Dot(axis)) < 0 {
		transDir = transDir.Scale(-1) // point the branch toward the vertex on the cone's real nappe
	}
	conjDir := axis.Cross(transDir)
	h, err := NewHyperbola(center, transDir, conjDir, a, b)
	if err != nil {
		return nil, false
	}
	return []Curve3{h}, true
}

// symmetricEig2 returns the eigenvalues and unit eigenvectors of the symmetric 2×2 [[a, h],[h, c]]
// (l1 with u1, l2 with u2). Used to reduce a planar conic to its principal axes.
func symmetricEig2(a, h, c float64) (l1, l2 float64, u1, u2 [2]float64) {
	tr, disc := a+c, stdmath.Hypot(a-c, 2*h)
	l1, l2 = (tr+disc)/2, (tr-disc)/2
	u1 = eigenvector2(a, h, c, l1)
	u2 = [2]float64{-u1[1], u1[0]} // the other eigenvector is orthogonal (symmetric matrix)
	return l1, l2, u1, u2
}

// eigenvector2 returns a unit eigenvector of [[a, h],[h, c]] for eigenvalue l. For h≈0 the matrix is
// already diagonal, so the axis is e1 or e2 by which diagonal entry l matches.
func eigenvector2(a, h, c, l float64) [2]float64 {
	if stdmath.Abs(h) < 1e-12 {
		if stdmath.Abs(l-a) <= stdmath.Abs(l-c) {
			return [2]float64{1, 0}
		}
		return [2]float64{0, 1}
	}
	vx, vy := l-c, h // (l−c, h) solves (a−l)vx+h·vy=0 ↔ h·vx+(c−l)vy=0
	norm := stdmath.Hypot(vx, vy)
	return [2]float64{vx / norm, vy / norm}
}

// cosSquared returns cos²(angle), the coefficient the cone identity ((P−V)·â)²=cos²α·|P−V|² uses.
func cosSquared(angle float64) float64 {
	c := stdmath.Cos(angle)
	return c * c
}

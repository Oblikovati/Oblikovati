// SPDX-License-Identifier: GPL-2.0-only

package geom

import (
	stdmath "math"

	"oblikovati.org/math"
)

// rayCoeffTol is the degeneracy guard on a quadratic's leading coefficient. Because every
// coefficient is built from the UNIT ray direction (D·A, |D⊥|², …), a and b are dimensionless
// in [0, 1], so an absolute floor is a rank test, not a model length.
const rayCoeffTol = 1e-12 // tol:numeric — quadratic degeneracy (coefficients are unit-direction dot products)

// RayHit is one crossing of a ray with a surface: the signed ray parameter T (the pierce point
// is ray.Origin + T·ray.Dir, and T ≥ 0 is forward because ray.Dir is unit), the pierce point,
// and the surface parameters (u, v) there. NormalAt(U, V) recovers the surface orientation at
// the hit, which the point-in-solid classifier uses to sign each crossing.
type RayHit struct {
	T     float64
	Point math.Point3
	U, V  float64
}

// RaySurfaceHits returns the forward crossings (0 ≤ T ≤ tMax, ascending) of the ray with the
// surface. It is the OCCT IntTools ray-cast analog and the primitive the analytic point-in-solid
// classifier builds on: closed forms for the quadric surfaces (plane, sphere, cylinder, cone) and
// a conditioning-gated general path (the numeric curve tracer) for the torus and NURBS surface,
// which have no simple root form. tMax may be +Inf for the closed forms; the general path needs a
// finite tMax to bound the segment it samples.
//
// Example — a +Z ray from the origin pierces a unit sphere centred at (0,0,3) at T=2 and T=4:
//
//	sp, _ := geom.NewSphere(math.P3(0, 0, 3), 1)
//	ray, _ := geom.NewLine(math.P3(0, 0, 0), math.V3(0, 0, 1))
//	hits := geom.RaySurfaceHits(sp, ray, stdmath.Inf(1)) // hits[0].T ≈ 2, hits[1].T ≈ 4
func RaySurfaceHits(s Surface, ray Line, tMax float64) []RayHit {
	switch surf := s.(type) {
	case Plane:
		return planeRayHits(surf, ray, tMax)
	case Sphere:
		return sphereRayHits(surf, ray, tMax)
	case Cylinder:
		return cylinderRayHits(surf, ray, tMax)
	case Cone:
		return coneRayHits(surf, ray, tMax)
	default:
		return numericRayHits(s, ray, tMax)
	}
}

// planeRayHits solves n·(O + T·D − Origin) = 0 for the single pierce parameter; a ray parallel to
// the plane (n·D ≈ 0) never crosses it.
func planeRayHits(p Plane, ray Line, tMax float64) []RayHit {
	d := ray.Dir.AsVector()
	n := p.Normal()
	denom := float64(n.Dot(d))
	if stdmath.Abs(denom) < rayCoeffTol {
		return nil
	}
	t := float64(ray.Origin.VectorTo(p.Origin).Dot(n)) / denom
	return rayHitsFromRoots(p, ray, tMax, t)
}

// sphereRayHits solves |O + T·D − C|² = R² (monic in T because D is unit).
func sphereRayHits(sp Sphere, ray Line, tMax float64) []RayHit {
	m := sp.Center.VectorTo(ray.Origin) // O − C
	b := 2 * float64(m.Dot(ray.Dir.AsVector()))
	c := float64(m.Dot(m)) - sp.Radius*sp.Radius
	return rayHitsFromRoots(sp, ray, tMax, rayQuadraticRoots(1, b, c)...)
}

// cylinderRayHits solves the radial-distance quadric of the infinite cylinder by projecting the
// ray perpendicular to the axis; a ray parallel to the axis (D⊥ ≈ 0) grazes no radial band.
func cylinderRayHits(cy Cylinder, ray Line, tMax float64) []RayHit {
	axis := cy.AxisDir.AsVector()
	dPerp := perpTo(ray.Dir.AsVector(), axis)
	wPerp := perpTo(cy.Origin.VectorTo(ray.Origin), axis) // (O − Pa)⊥
	a := float64(dPerp.Dot(dPerp))
	b := 2 * float64(wPerp.Dot(dPerp))
	c := float64(wPerp.Dot(wPerp)) - cy.Radius*cy.Radius
	return rayHitsFromRoots(cy, ray, tMax, rayQuadraticRoots(a, b, c)...)
}

// coneRayHits solves ((X−V)·A)² = cos²α·|X−V|² for the double cone, then keeps only roots on the
// physical nappe (axial coordinate ≥ 0), so the mirror nappe never yields a spurious crossing.
func coneRayHits(co Cone, ray Line, tMax float64) []RayHit {
	axis := co.AxisDir.AsVector()
	w := co.Apex.VectorTo(ray.Origin) // O − V
	d := ray.Dir.AsVector()
	dv, wv := float64(d.Dot(axis)), float64(w.Dot(axis))
	cos2 := stdmath.Cos(co.HalfAngle) * stdmath.Cos(co.HalfAngle)
	a := dv*dv - cos2
	b := 2 * (wv*dv - cos2*float64(d.Dot(w)))
	c := wv*wv - cos2*float64(w.Dot(w))
	return coneNappeHits(co, ray, tMax, wv, dv, rayQuadraticRoots(a, b, c))
}

// coneNappeHits keeps the cone roots whose pierce point lies on the apex-forward nappe
// (wv + T·dv ≥ 0), the axial coordinate measured from the apex.
func coneNappeHits(co Cone, ray Line, tMax, wv, dv float64, roots []float64) []RayHit {
	kept := roots[:0]
	for _, t := range roots {
		if wv+t*dv >= 0 {
			kept = append(kept, t)
		}
	}
	return rayHitsFromRoots(co, ray, tMax, kept...)
}

// numericRayHits is the general path for surfaces with no closed-form ray quadric (torus, NURBS):
// it samples the bounded segment O→O+tMax·D against the surface's signed-distance field. It needs
// a finite tMax to bound that segment.
func numericRayHits(s Surface, ray Line, tMax float64) []RayHit {
	if stdmath.IsInf(tMax, 0) || tMax <= 0 {
		return nil
	}
	d := ray.Dir.AsVector()
	seg := NewLineSegment(ray.Origin, ray.Origin.TranslateBy(d.Scale(tMax)))
	var out []RayHit
	for _, p := range IntersectCurveSurface(seg, s) {
		t := float64(ray.Origin.VectorTo(p).Dot(d))
		out = appendRayHit(out, s, ray, tMax, t)
	}
	return sortedRayHits(out)
}

// perpTo returns the component of v perpendicular to the unit axis (v − (v·axis)·axis).
func perpTo(v, axis math.Vector3) math.Vector3 {
	return v.Sub(axis.Scale(float64(v.Dot(axis))))
}

// rayQuadraticRoots returns the real roots of a·x² + b·x + c, using the sign-stable form
// (Numerical Recipes §5.6) that avoids cancellation between the two roots. A degenerate leading
// coefficient falls to the linear root; a negative discriminant yields none.
func rayQuadraticRoots(a, b, c float64) []float64 {
	if stdmath.Abs(a) < rayCoeffTol {
		if stdmath.Abs(b) < rayCoeffTol {
			return nil
		}
		return []float64{-c / b}
	}
	disc := b*b - 4*a*c
	if disc < 0 {
		return nil
	}
	sq := stdmath.Sqrt(disc)
	q := -0.5 * (b + stdmath.Copysign(sq, b))
	if stdmath.Abs(q) < rayCoeffTol {
		return []float64{q / a}
	}
	return []float64{q / a, c / q}
}

// rayHitsFromRoots turns solved ray parameters into forward hits within [0, tMax], sorted.
func rayHitsFromRoots(s Surface, ray Line, tMax float64, roots ...float64) []RayHit {
	var out []RayHit
	for _, t := range roots {
		out = appendRayHit(out, s, ray, tMax, t)
	}
	return sortedRayHits(out)
}

// appendRayHit adds the hit at parameter t when it is forward and within tMax; the surface
// parameters come from the on-surface inverse ParamAt at the pierce point.
func appendRayHit(out []RayHit, s Surface, ray Line, tMax, t float64) []RayHit {
	if t < 0 || t > tMax {
		return out
	}
	p := ray.PointAt(t)
	u, v := s.ParamAt(p)
	return append(out, RayHit{T: t, Point: p, U: u, V: v})
}

// sortedRayHits orders hits by ascending ray parameter, the order a classifier walks crossings in.
func sortedRayHits(hits []RayHit) []RayHit {
	for i := 1; i < len(hits); i++ {
		for j := i; j > 0 && hits[j-1].T > hits[j].T; j-- {
			hits[j-1], hits[j] = hits[j], hits[j-1]
		}
	}
	return hits
}

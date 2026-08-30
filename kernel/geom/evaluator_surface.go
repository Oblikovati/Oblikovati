// SPDX-License-Identifier: GPL-2.0-only

package geom

import (
	stdmath "math"

	"oblikovati.org/math"
)

// The member-level surface evaluator (M01-F06, #603): higher partials,
// principal curvatures, area, continuity, anomaly and range box. The analytic
// surfaces differentiate their trigonometric forms exactly; NURBS uses
// [BSplineSurface.SurfaceDersAt]. Query members (iso-curves, closest point)
// live in evaluator_surface_query.go.

// SurfaceSecondPartials returns ∂²P/∂u², ∂²P/∂u∂v and ∂²P/∂v² at (u, v).
func SurfaceSecondPartials(s Surface, u, v float64) (puu, puv, pvv math.Vector3) {
	switch g := s.(type) {
	case Plane:
		return math.Vector3{}, math.Vector3{}, math.Vector3{}
	case Cylinder:
		return g.radial(u).Scale(-g.Radius), math.Vector3{}, math.Vector3{}
	case Cone:
		return coneSecondPartials(g, u, v)
	case Sphere:
		return sphereSecondPartials(g, u, v)
	case Torus:
		return torusSecondPartials(g, u, v)
	case EllipticalCylinder:
		return ellipticRadial(g.Ref.AsVector(), g.binormal, g.MajorRadius, g.MinorRadius, u).Scale(-1),
			math.Vector3{}, math.Vector3{}
	case EllipticalCone:
		return ellipticalConeSecondPartials(g, u, v)
	case BSplineSurface:
		ders := g.SurfaceDersAt(u, v, 2, 2)
		return ders[2][0], ders[1][1], ders[0][2]
	default:
		return numericSecondPartials(s, u, v)
	}
}

// ellipticRadial returns a·cos u·ref + b·sin u·bin, the elliptical radial term.
func ellipticRadial(ref, bin math.Vector3, a, b, u float64) math.Vector3 {
	cos, sin := cosSin(u)
	return ref.Scale(a * cos).Add(bin.Scale(b * sin))
}

// coneSecondPartials differentiates P = Apex + v·axis + v·tanα·radial(u).
func coneSecondPartials(g Cone, u, v float64) (puu, puv, pvv math.Vector3) {
	tan := stdmath.Tan(g.HalfAngle)
	cos, sin := cosSin(u)
	radialTan := g.Ref.AsVector().Scale(-sin).Add(g.binormal.Scale(cos))
	return g.radial(u).Scale(-v * tan), radialTan.Scale(tan), math.Vector3{}
}

// sphereSecondPartials differentiates P = C + R·d(u, v).
func sphereSecondPartials(g Sphere, u, v float64) (puu, puv, pvv math.Vector3) {
	cu, su := cosSin(u)
	cv, sv := cosSin(v)
	puu = math.V3(-g.Radius*cv*cu, -g.Radius*cv*su, 0)
	puv = math.V3(g.Radius*sv*su, -g.Radius*sv*cu, 0)
	pvv = g.direction(u, v).Scale(-g.Radius)
	return puu, puv, pvv
}

// torusSecondPartials differentiates P = C + (R + r·cos v)·radial(u) + r·sin v·axis.
func torusSecondPartials(g Torus, u, v float64) (puu, puv, pvv math.Vector3) {
	cu, su := cosSin(u)
	cv, sv := cosSin(v)
	radialTan := g.Ref.AsVector().Scale(-su).Add(g.binormal.Scale(cu))
	puu = g.radial(u).Scale(-(g.MajorRadius + g.MinorRadius*cv))
	puv = radialTan.Scale(-g.MinorRadius * sv)
	pvv = g.radial(u).Scale(-g.MinorRadius * cv).Add(g.AxisDir.AsVector().Scale(-g.MinorRadius * sv))
	return puu, puv, pvv
}

// ellipticalConeSecondPartials differentiates
// P = Apex + v·axis + v·(tanA·cos u·ref + tanB·sin u·bin).
func ellipticalConeSecondPartials(g EllipticalCone, u, v float64) (puu, puv, pvv math.Vector3) {
	tMaj, tMin := stdmath.Tan(g.MajorAngle), stdmath.Tan(g.MinorAngle)
	cos, sin := cosSin(u)
	puu = ellipticRadial(g.Ref.AsVector(), g.binormal, tMaj, tMin, u).Scale(-v)
	puv = g.Ref.AsVector().Scale(-tMaj * sin).Add(g.binormal.Scale(tMin * cos))
	return puu, puv, math.Vector3{}
}

// numericSecondPartials estimates the second partials by central-differencing the
// first partials — the fallback for unknown [Surface] implementations (e.g. the
// offset and threaded surfaces). Each direction uses the first-difference-optimal
// step stepD1 = ε^{1/3} scaled by its OWN domain span, so the truncation/roundoff
// balance holds whether the direction runs over [0,1] or [0,2π] and at any model
// scale, rather than a fixed 1e-5 that was both non-optimal and scale-blind (#1402).
func numericSecondPartials(s Surface, u, v float64) (puu, puv, pvv math.Vector3) {
	hu := stepD1 * spanOr1(s.UDomain())
	hv := stepD1 * spanOr1(s.VDomain())
	pu := func(uu, vv float64) math.Vector3 { du, _ := s.DerivativesAt(uu, vv); return du }
	pv := func(uu, vv float64) math.Vector3 { _, dv := s.DerivativesAt(uu, vv); return dv }
	puu = pu(u+hu, v).Sub(pu(u-hu, v)).Scale(1 / (2 * hu))
	puv = pu(u, v+hv).Sub(pu(u, v-hv)).Scale(1 / (2 * hv))
	pvv = pv(u, v+hv).Sub(pv(u, v-hv)).Scale(1 / (2 * hv))
	return puu, puv, pvv
}

// SurfaceThirdPartials returns the corner partials ∂³P/∂u³ and ∂³P/∂v³.
// Every analytic surface is sinusoidal in its angular directions (so the third
// derivative is the negated first) and at most linear in the others (zero).
func SurfaceThirdPartials(s Surface, u, v float64) (puuu, pvvv math.Vector3) {
	if g, ok := s.(BSplineSurface); ok {
		ders := g.SurfaceDersAt(u, v, 3, 3)
		return ders[3][0], ders[0][3]
	}
	du, dv := s.DerivativesAt(u, v)
	uAngular, vAngular := surfaceAngularDirections(s)
	if uAngular {
		puuu = du.Scale(-1)
	}
	if vAngular {
		pvvv = dv.Scale(-1)
	}
	return puuu, pvvv
}

// surfaceAngularDirections reports which parameter directions wind around an
// axis (sinusoidal parameterization).
func surfaceAngularDirections(s Surface) (uAngular, vAngular bool) {
	switch s.(type) {
	case Cylinder, Cone, EllipticalCylinder, EllipticalCone:
		return true, false
	case Sphere, Torus:
		return true, true
	default:
		return false, false
	}
}

// SurfaceTangents returns the unit tangents along u and v (zero vectors at
// degeneracies such as a sphere pole or cone apex).
func SurfaceTangents(s Surface, u, v float64) (uTangent, vTangent math.Vector3) {
	du, dv := s.DerivativesAt(u, v)
	return unitOrZero(du), unitOrZero(dv)
}

// SurfaceCurvatures returns the principal curvatures at (u, v) and the unit
// tangent direction of the maximum one, from the first and second fundamental
// forms with the normal oriented by NormalAt.
func SurfaceCurvatures(s Surface, u, v float64) (maxDir math.Vector3, kMax, kMin float64) {
	du, dv := s.DerivativesAt(u, v)
	n := s.NormalAt(u, v)
	puu, puv, pvv := SurfaceSecondPartials(s, u, v)
	e, f, g := float64(du.Dot(du)), float64(du.Dot(dv)), float64(dv.Dot(dv))
	l, m, nn := float64(puu.Dot(n)), float64(puv.Dot(n)), float64(pvv.Dot(n))
	if degenerateFirstForm(e, f, g) {
		return math.Vector3{}, 0, 0 // parallel/collapsed tangents (scale-invariant test, #1402)
	}
	den := e*g - f*f
	mean := (e*nn - 2*f*m + g*l) / (2 * den)
	gauss := (l*nn - m*m) / den
	disc := stdmath.Sqrt(stdmath.Max(0, mean*mean-gauss))
	kMax, kMin = mean+disc, mean-disc
	return principalDirection(du, dv, e, f, g, l, m, nn, kMax), kMax, kMin
}

// principalDirection solves the shape-operator null system
// (L−kE)·a + (M−kF)·b = 0 for the (a, b) tangent coefficients of the maximum
// principal direction, falling back to the u tangent at an umbilic point.
func principalDirection(du, dv math.Vector3, e, f, g, l, m, n, k float64) math.Vector3 {
	a1, b1 := l-k*e, m-k*f
	a2, b2 := m-k*f, n-k*g
	a, b := b1, -a1
	if stdmath.Abs(a1)+stdmath.Abs(b1) < stdmath.Abs(a2)+stdmath.Abs(b2) {
		a, b = b2, -a2
	}
	dir := du.Scale(a).Add(dv.Scale(b))
	if float64(dir.Length()) < 1e-14 {
		return unitOrZero(du) // umbilic: every direction is principal
	}
	return unitOrZero(dir)
}

// SurfaceArea returns the total area: closed forms for the bounded analytic
// surfaces, +Inf for unbounded ones, Gauss–Legendre quadrature for NURBS.
func SurfaceArea(s Surface) float64 {
	switch g := s.(type) {
	case Sphere:
		return 2 * twoPi * g.Radius * g.Radius
	case Torus:
		return twoPi * twoPi * g.MajorRadius * g.MinorRadius
	case BSplineSurface:
		return bsplineSurfaceArea(g)
	default:
		return stdmath.Inf(1)
	}
}

// bsplineSurfaceArea integrates |∂P/∂u × ∂P/∂v| with 5-point Gauss–Legendre
// per knot-span cell (the integrand is smooth inside each span).
func bsplineSurfaceArea(g BSplineSurface) float64 {
	uSpans := spanEdges(g.UKnots, g.UDegree)
	vSpans := spanEdges(g.VKnots, g.VDegree)
	total := 0.0
	for ui := 0; ui+1 < len(uSpans); ui++ {
		for vi := 0; vi+1 < len(vSpans); vi++ {
			total += gaussCellArea(g, uSpans[ui], uSpans[ui+1], vSpans[vi], vSpans[vi+1])
		}
	}
	return total
}

// spanEdges returns the distinct knot values spanning the domain.
func spanEdges(knots []float64, degree int) []float64 {
	lo, hi := knots[degree], knots[len(knots)-1-degree]
	return append(append([]float64{lo}, interiorKnots(knots, degree)...), hi)
}

// gaussCellArea integrates the area element |∂P/∂u × ∂P/∂v| over one parameter cell
// with a tensor-product 5-point Gauss–Legendre rule (the integrand is smooth inside a span).
func gaussCellArea(s Surface, u0, u1, v0, v1 float64) float64 {
	hu, hv := (u1-u0)/2, (v1-v0)/2
	if hu <= 0 || hv <= 0 {
		return 0
	}
	x, w := GaussLegendre(5)
	sum := 0.0
	for i := range x {
		for j := range x {
			du, dv := s.DerivativesAt(u0+hu*(1+x[i]), v0+hv*(1+x[j]))
			sum += w[i] * w[j] * float64(du.Cross(dv).Length())
		}
	}
	return sum * hu * hv
}

// SurfaceContinuity returns the largest maintained continuity order: the
// minimum over both directions for NURBS, C∞ for analytic surfaces.
func SurfaceContinuity(s Surface) int {
	g, ok := s.(BSplineSurface)
	if !ok {
		return ContinuityInfinite
	}
	cu := bsplineContinuity(g.UKnots, g.UDegree)
	cv := bsplineContinuity(g.VKnots, g.VDegree)
	if cv < cu {
		return cv
	}
	return cu
}

// SurfaceAnomaly reports each parameter direction's irregularities.
func SurfaceAnomaly(s Surface) (uAnomaly, vAnomaly ParamAnomaly) {
	switch s.(type) {
	case Plane:
		return ParamAnomaly{Unbounded: true}, ParamAnomaly{Unbounded: true}
	case Cylinder, EllipticalCylinder:
		return ParamAnomaly{Periodic: true, Period: twoPi}, ParamAnomaly{Unbounded: true}
	case Cone, EllipticalCone:
		return ParamAnomaly{Periodic: true, Period: twoPi},
			ParamAnomaly{Singular: true, Unbounded: true} // apex at v = 0
	case Sphere:
		return ParamAnomaly{Periodic: true, Period: twoPi},
			ParamAnomaly{Singular: true} // poles at v = ±π/2
	case Torus:
		return ParamAnomaly{Periodic: true, Period: twoPi}, ParamAnomaly{Periodic: true, Period: twoPi}
	default:
		return ParamAnomaly{}, ParamAnomaly{}
	}
}

// SurfaceRangeBox returns an enclosing axis-aligned box: exact for the sphere,
// conservative per-axis bounds for the torus, the control-net box for NURBS
// (enclosing but not minimal), ±Inf faces for unbounded surfaces.
func SurfaceRangeBox(s Surface) math.Box {
	switch g := s.(type) {
	case Sphere:
		r := g.Radius
		return math.NewBox(
			math.P3(g.Center.X-r, g.Center.Y-r, g.Center.Z-r),
			math.P3(g.Center.X+r, g.Center.Y+r, g.Center.Z+r))
	case Torus:
		return torusRangeBox(g)
	case BSplineSurface:
		return bsplineNetBox(g)
	default:
		inf := stdmath.Inf(1)
		return math.Box{Min: math.P3(-inf, -inf, -inf), Max: math.P3(inf, inf, inf)}
	}
}

// torusRangeBox bounds the torus per world axis: the planar reach (R + r)
// scaled by the axis's in-plane share plus the tube reach along the axis.
func torusRangeBox(g Torus) math.Box {
	box := math.EmptyBox()
	reach := [3]float64{}
	for axis := range 3 {
		ref, bin := vectorComponent(g.Ref.AsVector(), axis), vectorComponent(g.binormal, axis)
		axial := vectorComponent(g.AxisDir.AsVector(), axis)
		reach[axis] = (g.MajorRadius+g.MinorRadius)*stdmath.Hypot(ref, bin) +
			g.MinorRadius*stdmath.Abs(axial)
	}
	box = box.ExtendPoint(math.P3(g.Center.X-reach[0], g.Center.Y-reach[1], g.Center.Z-reach[2]))
	return box.ExtendPoint(math.P3(g.Center.X+reach[0], g.Center.Y+reach[1], g.Center.Z+reach[2]))
}

// bsplineNetBox boxes the control net (the convex-hull property makes it an
// enclosing bound).
func bsplineNetBox(g BSplineSurface) math.Box {
	box := math.EmptyBox()
	for _, row := range g.Ctrl {
		box = box.Union(math.BoxFromPoints(row...))
	}
	return box
}

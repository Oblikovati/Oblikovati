// SPDX-License-Identifier: GPL-2.0-only

package geom

import (
	"fmt"
	stdmath "math"

	"oblikovati.org/math"
)

// Surface query members (M01-F06, #603): iso-curve extraction, classified
// closest point, and off-surface normal lookup.

// SurfaceIsoCurve extracts the curve of constant parameter: u = param when
// uDirection (the curve runs along v), else v = param. It errors at a
// degenerate parameter (a cone apex, a sphere pole, a pinched torus ring).
func SurfaceIsoCurve(s Surface, uDirection bool, param float64) (Curve3, error) {
	switch g := s.(type) {
	case Plane:
		return planeIso(g, uDirection, param), nil
	case Cylinder:
		return cylinderIso(g, uDirection, param), nil
	case Cone:
		return coneIso(g, uDirection, param)
	case Sphere:
		return sphereIso(g, uDirection, param)
	case Torus:
		return torusIso(g, uDirection, param)
	case EllipticalCylinder:
		return ellipticalCylinderIso(g, uDirection, param)
	case EllipticalCone:
		return ellipticalConeIso(g, uDirection, param)
	case BSplineSurface:
		return bsplineIso(g, uDirection, param)
	default:
		return nil, fmt.Errorf("geom: no iso-curve extraction for surface type %T", s)
	}
}

// planeIso returns the in-plane line at the fixed parameter.
func planeIso(g Plane, uDirection bool, param float64) Line {
	if uDirection {
		return Line{Origin: g.PointAt(param, 0), Dir: g.VAxis}
	}
	return Line{Origin: g.PointAt(0, param), Dir: g.UAxis}
}

// cylinderIso returns the axial ruling line at fixed angle, or the
// cross-section circle at fixed height.
func cylinderIso(g Cylinder, uDirection bool, param float64) Curve3 {
	if uDirection {
		return Line{Origin: g.PointAt(param, 0), Dir: g.AxisDir}
	}
	center := g.Origin.TranslateBy(g.AxisDir.AsVector().Scale(param))
	return Circle{Center: center, Normal: g.AxisDir, RefDir: g.Ref, Radius: g.Radius}
}

// coneIso returns the slant ruling at fixed angle or the cross-section circle
// at fixed height (the apex itself is degenerate).
func coneIso(g Cone, uDirection bool, param float64) (Curve3, error) {
	if uDirection {
		_, dv := g.DerivativesAt(param, 1)
		return NewLine(g.Apex, dv)
	}
	if param <= 0 {
		return nil, fmt.Errorf("geom: cone iso-curve at v = %g is degenerate (apex at v = 0)", param)
	}
	center := g.Apex.TranslateBy(g.AxisDir.AsVector().Scale(param))
	return Circle{Center: center, Normal: g.AxisDir, RefDir: g.Ref, Radius: param * stdmath.Tan(g.HalfAngle)}, nil
}

// sphereIso returns the meridian arc at fixed longitude or the latitude circle
// at fixed latitude (poles are degenerate).
func sphereIso(g Sphere, uDirection bool, param float64) (Curve3, error) {
	if uDirection {
		return sphereMeridian(g, param), nil
	}
	cv, sv := cosSin(param)
	if stdmath.Abs(cv) <= 1e-12 {
		return nil, fmt.Errorf("geom: sphere iso-curve at latitude %g is a pole point", param)
	}
	center := g.Center.TranslateBy(math.V3(0, 0, g.Radius*sv))
	xRef, _ := math.NewUnitVector3(1, 0, 0)
	zAxis, _ := math.NewUnitVector3(0, 0, 1)
	return Circle{Center: center, Normal: zAxis, RefDir: xRef, Radius: g.Radius * cv}, nil
}

// sphereMeridian builds the half-circle from south to north pole at longitude u
// as an arc: P(v) = C + R·(cos v·eᵣ + sin v·ẑ) over v ∈ [−π/2, π/2].
func sphereMeridian(g Sphere, u float64) Arc3d {
	cu, su := cosSin(u)
	radial, _ := math.NewUnitVector3(cu, su, 0)
	normal, _ := math.NewUnitVector3(su, -cu, 0) // eᵣ × ẑ … oriented so N×eᵣ = ẑ
	return Arc3d{
		Center: g.Center, Normal: normal, RefDir: radial, Radius: g.Radius,
		StartAngle: -stdmath.Pi / 2, SweepAngle: stdmath.Pi,
	}
}

// torusIso returns the tube circle at fixed axis angle or the axis circle at
// fixed tube angle (a self-intersecting torus pinches where R + r·cos v ≤ 0).
func torusIso(g Torus, uDirection bool, param float64) (Curve3, error) {
	if uDirection {
		radial := g.radial(param)
		center := g.Center.TranslateBy(radial.Scale(g.MajorRadius))
		return Circle{Center: center, Normal: radial.Cross(g.AxisDir.AsVector()).AsUnit(),
			RefDir: radial.AsUnit(), Radius: g.MinorRadius}, nil
	}
	cv, sv := cosSin(param)
	radius := g.MajorRadius + g.MinorRadius*cv
	if radius <= 0 {
		return nil, fmt.Errorf("geom: torus iso-curve at tube angle %g has radius %g <= 0", param, radius)
	}
	center := g.Center.TranslateBy(g.AxisDir.AsVector().Scale(g.MinorRadius * sv))
	return Circle{Center: center, Normal: g.AxisDir, RefDir: g.Ref, Radius: radius}, nil
}

// ellipticalCylinderIso returns the axial ruling or the cross-section ellipse.
func ellipticalCylinderIso(g EllipticalCylinder, uDirection bool, param float64) (Curve3, error) {
	if uDirection {
		return Line{Origin: g.PointAt(param, 0), Dir: g.AxisDir}, nil
	}
	center := g.Origin.TranslateBy(g.AxisDir.AsVector().Scale(param))
	return EllipseFull{Center: center, Normal: g.AxisDir, MajorAxis: g.Ref,
		MajorRadius: g.MajorRadius, MinorRadius: g.MinorRadius}, nil
}

// ellipticalConeIso returns the slant ruling or the cross-section ellipse.
func ellipticalConeIso(g EllipticalCone, uDirection bool, param float64) (Curve3, error) {
	if uDirection {
		_, dv := g.DerivativesAt(param, 1)
		return NewLine(g.Apex, dv)
	}
	if param <= 0 {
		return nil, fmt.Errorf("geom: elliptical cone iso-curve at v = %g is degenerate (apex at v = 0)", param)
	}
	center := g.Apex.TranslateBy(g.AxisDir.AsVector().Scale(param))
	return EllipseFull{Center: center, Normal: g.AxisDir, MajorAxis: g.Ref,
		MajorRadius: param * stdmath.Tan(g.MajorAngle), MinorRadius: param * stdmath.Tan(g.MinorAngle)}, nil
}

// bsplineIso collapses one parametric direction of the control net at the
// fixed parameter (in homogeneous space), yielding the iso NURBS curve.
func bsplineIso(g BSplineSurface, uDirection bool, param float64) (Curve3, error) {
	if uDirection {
		ctrl, weights := collapseU(g, param)
		return NewBSplineCurve(g.VDegree, ctrl, weights, append([]float64(nil), g.VKnots...))
	}
	ctrl, weights := collapseV(g, param)
	return NewBSplineCurve(g.UDegree, ctrl, weights, append([]float64(nil), g.UKnots...))
}

// collapseU evaluates the u basis at the fixed u, blending control-net rows
// into one homogeneous control polygon over v.
func collapseU(g BSplineSurface, u float64) (ctrl []math.Point3, weights []float64) {
	span := findSpan(len(g.Ctrl)-1, g.UDegree, u, g.UKnots)
	basis := basisFuns(span, g.UDegree, u, g.UKnots)
	cols := len(g.Ctrl[0])
	ctrl = make([]math.Point3, cols)
	weights = make([]float64, cols)
	for j := 0; j < cols; j++ {
		var h homog
		for k := 0; k <= g.UDegree; k++ {
			i := span - g.UDegree + k
			h.add(g.Ctrl[i][j], basis[k]*g.Weights[i][j])
		}
		ctrl[j], weights[j] = h.point(), h.w
	}
	return ctrl, weights
}

// collapseV evaluates the v basis at the fixed v, blending control-net columns.
func collapseV(g BSplineSurface, v float64) (ctrl []math.Point3, weights []float64) {
	span := findSpan(len(g.Ctrl[0])-1, g.VDegree, v, g.VKnots)
	basis := basisFuns(span, g.VDegree, v, g.VKnots)
	rows := len(g.Ctrl)
	ctrl = make([]math.Point3, rows)
	weights = make([]float64, rows)
	for i := 0; i < rows; i++ {
		var h homog
		for k := 0; k <= g.VDegree; k++ {
			j := span - g.VDegree + k
			h.add(g.Ctrl[i][j], basis[k]*g.Weights[i][j])
		}
		ctrl[i], weights[i] = h.point(), h.w
	}
	return ctrl, weights
}

// SurfaceParamAtPoint returns the (u, v) of the point on s closest to p,
// classifying degenerate queries (an axis point sees a whole ring of equally
// close answers; an elliptical axis point sees the two minor-direction feet).
func SurfaceParamAtPoint(s Surface, p math.Point3) (u, v float64, nature SolutionNature) {
	if nature, ok := surfaceDegenerateNature(s, p); ok {
		u, v = s.ParamAt(p)
		return u, v, nature
	}
	if g, ok := s.(BSplineSurface); ok {
		return bsplineParamAtPoint(g, p)
	}
	u, v, _ = ClosestPointOnSurface(s, p)
	return u, v, UniqueSolution
}

// surfaceDegenerateNature classifies the closed-form degeneracies of the
// analytic surfaces.
func surfaceDegenerateNature(s Surface, p math.Point3) (SolutionNature, bool) {
	switch g := s.(type) {
	case Cylinder:
		return axisDegeneracy(g.Origin, g.AxisDir, p, g.Radius, InfinitelyManySolutions)
	case Cone:
		return axisDegeneracy(g.Apex, g.AxisDir, p, 1, InfinitelyManySolutions)
	case Sphere:
		if g.Center.DistanceTo(p) <= 1e-12*g.Radius {
			return InfinitelyManySolutions, true
		}
	case Torus:
		return axisDegeneracy(g.Center, g.AxisDir, p, g.MajorRadius, InfinitelyManySolutions)
	case EllipticalCylinder:
		return axisDegeneracy(g.Origin, g.AxisDir, p, g.MajorRadius,
			ellipticNature(g.MajorRadius, g.MinorRadius))
	case EllipticalCone:
		return axisDegeneracy(g.Apex, g.AxisDir, p, 1,
			ellipticNature(stdmath.Tan(g.MajorAngle), stdmath.Tan(g.MinorAngle)))
	}
	return UnknownSolutionNature, false
}

// axisDegeneracy reports nature when p lies on the surface's axis (the
// rotational degeneracy), scale-relative to the surface size.
func axisDegeneracy(origin math.Point3, axis math.UnitVector3, p math.Point3, scale float64, nature SolutionNature) (SolutionNature, bool) {
	d := origin.VectorTo(p)
	planar := d.Sub(axis.AsVector().Scale(d.Dot(axis.AsVector())))
	if float64(planar.Length()) <= 1e-12*stdmath.Max(1, scale) {
		return nature, true
	}
	return UnknownSolutionNature, false
}

// ellipticNature distinguishes the circular degeneracy (a whole ring of feet)
// from the elliptical one (exactly two minor-direction feet).
func ellipticNature(major, minor float64) SolutionNature {
	if stdmath.Abs(major-minor) <= 1e-12*stdmath.Max(1, major) {
		return InfinitelyManySolutions
	}
	return DistinctlyManySolutions
}

// surfaceSeed is one refined closest-point candidate in parameter space.
type surfaceSeed struct{ u, v float64 }

// bsplineParamAtPoint searches a coarse grid for near-minimal cells, refines
// each with the seeded projector, and clusters the winners.
func bsplineParamAtPoint(g BSplineSurface, p math.Point3) (float64, float64, SolutionNature) {
	const grid = 24
	uLo, uHi := g.UDomain()
	vLo, vHi := g.VDomain()
	us, vs, ds, best := surfaceGridDistances(g, p, uLo, uHi, vLo, vHi, grid)
	tol := 1e-9 * stdmath.Max(1, best)
	var clusters []surfaceSeed
	bestU, bestV, refinedBest := us[0], vs[0], stdmath.Inf(1)
	for i, d := range ds {
		if d > best+tol {
			continue
		}
		ru, rv := g.ParamNear(p, us[i], vs[i])
		rd := g.PointAt(ru, rv).DistanceTo(p)
		if rd < refinedBest-tol {
			refinedBest, bestU, bestV, clusters = rd, ru, rv, clusters[:0]
		}
		if rd <= refinedBest+tol && !inSeedCluster(clusters, ru, rv, (uHi-uLo)/grid, (vHi-vLo)/grid) {
			clusters = append(clusters, surfaceSeed{ru, rv})
		}
	}
	if len(clusters) > 1 {
		return bestU, bestV, DistinctlyManySolutions
	}
	return bestU, bestV, UniqueSolution
}

// surfaceGridDistances samples the distance field on a uniform parameter grid.
func surfaceGridDistances(s Surface, p math.Point3, uLo, uHi, vLo, vHi float64, grid int) (us, vs, ds []float64, best float64) {
	best = stdmath.Inf(1)
	for i := 0; i <= grid; i++ {
		for j := 0; j <= grid; j++ {
			u := uLo + (uHi-uLo)*float64(i)/float64(grid)
			v := vLo + (vHi-vLo)*float64(j)/float64(grid)
			d := s.PointAt(u, v).DistanceTo(p)
			us, vs, ds = append(us, u), append(vs, v), append(ds, d)
			best = stdmath.Min(best, d)
		}
	}
	return us, vs, ds, best
}

// inSeedCluster reports whether (u, v) merges into an existing cluster.
func inSeedCluster(clusters []surfaceSeed, u, v, uWidth, vWidth float64) bool {
	for _, c := range clusters {
		if stdmath.Abs(c.u-u) <= 2*uWidth && stdmath.Abs(c.v-v) <= 2*vWidth {
			return true
		}
	}
	return false
}

// SurfaceNormalAtPoint returns the unit surface normal at the point on s
// closest to p.
func SurfaceNormalAtPoint(s Surface, p math.Point3) math.Vector3 {
	u, v, _ := ClosestPointOnSurface(s, p)
	return s.NormalAt(u, v)
}

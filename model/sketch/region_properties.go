// SPDX-License-Identifier: GPL-2.0-only

package sketch

import (
	"fmt"
	stdmath "math"

	"oblikovati.org/api/types"
	"oblikovati.org/math"
)

// Region (section) properties of a closed profile — area, perimeter,
// centroid, second moments and principal axes — computed by exact Green's
// theorem integrals over the loops' sampled polygons, with the accuracy
// setting scaling the curve sampling density (M06-F08,
// Oblikovati/Oblikovati#623). The integrals are exact for the polygon; the
// accuracy only controls how closely the polygon follows curved boundaries.

// RegionProperties is the computed property set; all values are in database
// units (cm-based) on the sketch plane and the inertia values are centroidal.
// It satisfies api/contract.RegionProperties.
type RegionProperties struct {
	accuracy      types.Accuracy
	area          float64
	perimeter     float64
	cx, cy        float64
	ixx, iyy, ixy float64
}

// Accuracy returns the accuracy the values were computed at.
func (r *RegionProperties) Accuracy() types.Accuracy { return r.accuracy }

// Area returns the enclosed area in cm² (holes subtracted).
func (r *RegionProperties) Area() float64 { return r.area }

// Perimeter returns the total boundary length in cm, holes' rims included.
func (r *RegionProperties) Perimeter() float64 { return r.perimeter }

// Centroid returns the area centroid in sketch-plane cm.
func (r *RegionProperties) Centroid() (x, y float64) { return r.cx, r.cy }

// MomentsOfInertia returns the centroidal second moments Ixx, Iyy and the
// product Ixy in cm⁴.
func (r *RegionProperties) MomentsOfInertia() (ixx, iyy, ixy float64) {
	return r.ixx, r.iyy, r.ixy
}

// PrincipalMoments returns the principal second moments I1 ≥ I2 in cm⁴.
func (r *RegionProperties) PrincipalMoments() (i1, i2 float64) {
	avg := (r.ixx + r.iyy) / 2
	rad := stdmath.Hypot((r.ixx-r.iyy)/2, r.ixy)
	return avg + rad, avg - rad
}

// RotationAngle returns the CCW angle (radians) from the sketch X axis to the
// first principal axis (the one carrying I1). An isotropic section (a circle,
// an annulus — every axis principal) reports 0 rather than the atan2 of
// floating-point noise; the threshold is relative to the moment magnitude.
func (r *RegionProperties) RotationAngle() float64 {
	deviator := stdmath.Hypot((r.ixx-r.iyy)/2, r.ixy)
	if deviator <= 1e-9*(r.ixx+r.iyy)/2 {
		return 0
	}
	return stdmath.Atan2(-2*r.ixy, r.ixx-r.iyy) / 2
}

// PrincipalAxes returns the two principal axis directions as unit vectors,
// the first carrying I1.
func (r *RegionProperties) PrincipalAxes() (first, second math.Vector2) {
	a := r.RotationAngle()
	cos, sin := math.Scalar(stdmath.Cos(a)), math.Scalar(stdmath.Sin(a))
	return math.V2(cos, sin), math.V2(-sin, cos)
}

// accuracyDensity maps the public accuracy onto the curve-sampling density
// multiplier: each step doubles the polygon resolution.
func accuracyDensity(acc types.Accuracy) int {
	switch acc {
	case types.AccuracyLow:
		return 1
	case types.AccuracyMedium:
		return 2
	case types.AccuracyVeryHigh:
		return 8
	default: // High is the default the wire layer documents
		return 4
	}
}

// RegionProperties computes the profile's section properties at the given
// accuracy. Open profiles enclose nothing and are rejected.
func (p *Profile) RegionProperties(acc types.Accuracy) (*RegionProperties, error) {
	if !p.IsClosed() {
		return nil, fmt.Errorf("region properties need a closed profile, got an open chain of %d entities",
			len(p.outer.entities))
	}
	mult := accuracyDensity(acc)
	sums := loopIntegrals(denseLoopPolygon(p.outer, mult))
	perimeter := sums.perimeter
	for _, hole := range p.inner {
		h := loopIntegrals(denseLoopPolygon(hole, mult))
		sums = sums.subtract(h)
		perimeter += h.perimeter
	}
	sums.perimeter = perimeter
	return regionFromIntegrals(sums, acc)
}

// RegionProperties computes the planar 3D profile's section properties in its
// own plane frame (x along the first edge, the plane normal as z). The loop's
// vertices are already realized points, so accuracy does not refine them.
func (p *Profile3D) RegionProperties(acc types.Accuracy) (*RegionProperties, error) {
	planar := projectLoopToPlane(p.Points(), p.normal)
	if len(planar) < 3 {
		return nil, fmt.Errorf("region properties need >= 3 loop vertices, got %d", len(planar))
	}
	return regionFromIntegrals(loopIntegrals(planar), acc)
}

// regionFromIntegrals shifts the origin-referenced integrals to the centroid
// and packages them. A degenerate (zero-area) region is rejected.
func regionFromIntegrals(sums regionIntegrals, acc types.Accuracy) (*RegionProperties, error) {
	if sums.area <= 0 {
		return nil, fmt.Errorf("region properties need an enclosing region, got area %g", sums.area)
	}
	cx, cy := sums.sx/sums.area, sums.sy/sums.area
	return &RegionProperties{
		accuracy:  acc,
		area:      sums.area,
		perimeter: sums.perimeter,
		cx:        cx,
		cy:        cy,
		// Parallel-axis shift: I_centroid = I_origin − A·d².
		ixx: sums.ixx - sums.area*cy*cy,
		iyy: sums.iyy - sums.area*cx*cx,
		ixy: sums.ixy - sums.area*cx*cy,
	}, nil
}

// regionIntegrals are signed Green's-theorem polygon integrals about the
// sketch origin: area, first moments (sx, sy), second moments, and the rim
// length.
type regionIntegrals struct {
	area, sx, sy  float64
	ixx, iyy, ixy float64
	perimeter     float64
}

// subtract removes a hole's contribution (its rim length is handled by the
// caller, which *adds* rims while subtracting the enclosed integrals).
func (r regionIntegrals) subtract(hole regionIntegrals) regionIntegrals {
	r.area -= hole.area
	r.sx -= hole.sx
	r.sy -= hole.sy
	r.ixx -= hole.ixx
	r.iyy -= hole.iyy
	r.ixy -= hole.ixy
	return r
}

// loopIntegrals integrates one loop polygon, normalized to positive (CCW)
// orientation so outer/hole accounting is a plain subtraction.
func loopIntegrals(poly []math.Point2) regionIntegrals {
	if signedPolygonArea(poly) < 0 {
		reverseInPlace2(poly)
	}
	var out regionIntegrals
	for i := range poly {
		p, q := poly[i], poly[(i+1)%len(poly)]
		xi, yi, xj, yj := float64(p.X), float64(p.Y), float64(q.X), float64(q.Y)
		cross := xi*yj - xj*yi
		out.area += cross / 2
		out.sx += (xi + xj) * cross / 6
		out.sy += (yi + yj) * cross / 6
		out.ixx += (yi*yi + yi*yj + yj*yj) * cross / 12
		out.iyy += (xi*xi + xi*xj + xj*xj) * cross / 12
		out.ixy += (xi*yj + 2*xi*yi + 2*xj*yj + xj*yi) * cross / 24
		out.perimeter += float64(p.DistanceTo(q))
	}
	return out
}

// denseLoopPolygon re-samples a loop's entities at the accuracy-scaled
// density. The stored l.polygon is the base-density polygon used for region
// detection; properties want the finer one.
func denseLoopPolygon(l Loop, mult int) []math.Point2 {
	if mult <= 1 {
		return append([]math.Point2(nil), l.polygon...)
	}
	if len(l.entities) == 1 {
		if poly, ok := denseStandalonePolygon(l.entities[0].Entity, mult); ok {
			return poly
		}
	}
	var out []math.Point2
	for _, pe := range l.entities {
		seg := denseNaturalPolyline(pe.Entity, mult)
		if pe.reversed {
			reverseInPlace2(seg)
		}
		if len(seg) > 0 {
			out = append(out, seg[:len(seg)-1]...) // the next entity owns the shared vertex
		}
	}
	return out
}

// denseStandalonePolygon densely samples a self-closing entity (the
// single-entity loops of detection.go's standaloneLoop).
func denseStandalonePolygon(e Entity, mult int) ([]math.Point2, bool) {
	switch t := e.(type) {
	case *Circle:
		return sampleCircleN(t, circleSamples*mult), true
	case *Ellipse:
		return sampleEllipseEntityN(t, curveSamples*mult), true
	case *Spline:
		return sampleSplineEntityN(t, splineSamplesPerSpan*mult), true
	}
	return nil, false
}

// denseNaturalPolyline is naturalPolyline at the accuracy-scaled density;
// kinds without a density knob (fixed/offset splines) keep their stored shape.
func denseNaturalPolyline(e Entity, mult int) []math.Point2 {
	switch t := e.(type) {
	case *Arc:
		return sampleArcEntityN(t, curveSamples*mult)
	case *Spline:
		return sampleSplineEntityN(t, splineSamplesPerSpan*mult)
	case *EllipticalArc:
		return sampleEllipticalArcEntityN(t, curveSamples*mult)
	case *EquationCurve:
		return t.Sample(curveSamples * mult)
	default:
		return naturalPolyline(e)
	}
}

// projectLoopToPlane expresses a planar 3D loop in 2D plane coordinates: the
// frame origin is the first vertex, x along the first edge, y = normal × x.
func projectLoopToPlane(pts []math.Point3, normal math.Vector3) []math.Point2 {
	if len(pts) < 3 {
		return nil
	}
	x := pts[0].VectorTo(pts[1])
	if l := float64(x.Length()); l == 0 {
		return nil
	} else {
		x = x.Scale(math.Scalar(1 / l))
	}
	y := normal.Cross(x)
	out := make([]math.Point2, len(pts))
	for i, p := range pts {
		v := pts[0].VectorTo(p)
		out[i] = math.P2(v.Dot(x), v.Dot(y))
	}
	return out
}

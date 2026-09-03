// SPDX-License-Identifier: GPL-2.0-only

package geom

import (
	stdmath "math"
	"sort"

	"oblikovati.org/math"
)

// Plane sections and rulings on a ruled quadric (ADR-0060). A cylinder or cone side is charted as
// (u, v) = (azimuth, axial distance), and EVERY curve a boolean lays on it — the face's own rims, a
// tool plane's section, a ruling — is either a straight line or a conic that lies in one plane. Two
// such curves meet where the LINE their two planes share pierces the surface, or where a ruling pierces
// the other curve's plane: closed forms, so a frame×imprint vertex is exact on both curves and the
// (u,v) arrangement never has to resolve one on a sampled chord. These accessors are what the chart
// asks; the geometry-kind switches stay in this package (kernel ground rules).

// RuledFrame is the parameterisation the (u,v) charts read off a cylinder or a cone:
//
//	P(u,v) = Base + v·Axis + (RadSlope·v + RadConst)·(cos u·Ref + sin u·(Axis×Ref))
//
// A cylinder has RadSlope 0 and RadConst its radius; a cone has Base at its apex, RadConst 0 and
// RadSlope tan(half-angle), so v is the axial distance from the apex.
type RuledFrame struct {
	Base               math.Point3
	Axis, Ref          math.Vector3
	RadSlope, RadConst float64
}

// RuledFrameOf returns the ruled frame of a cylinder or a cone; ok=false for any other surface.
//
// Example:
//
//	if fr, ok := geom.RuledFrameOf(face.Surface()); ok { v := fr.Base.VectorTo(p).Dot(fr.Axis) }
func RuledFrameOf(s Surface) (RuledFrame, bool) {
	switch x := s.(type) {
	case Cylinder:
		return RuledFrame{Base: x.Origin, Axis: x.AxisDir.AsVector(), Ref: x.Ref.AsVector(), RadConst: x.Radius}, true
	case Cone:
		return RuledFrame{Base: x.Apex, Axis: x.AxisDir.AsVector(), Ref: x.Ref.AsVector(), RadSlope: stdmath.Tan(x.HalfAngle)}, true
	}
	return RuledFrame{}, false
}

// Radius is the surface radius at axial distance v.
func (f RuledFrame) Radius(v float64) float64 { return f.RadSlope*v + f.RadConst }

// Ruling returns the straight generator of the surface at absolute azimuth u, parameterised by v
// (its point at parameter v is the surface point (u, v)).
func (f RuledFrame) Ruling(u float64) Line {
	radial := f.Ref.Scale(math.Scalar(stdmath.Cos(u))).Add(f.Axis.Cross(f.Ref).Scale(math.Scalar(stdmath.Sin(u))))
	origin := f.Base.TranslateBy(radial.Scale(math.Scalar(f.RadConst)))
	dir, _ := math.UnitVector3FromVector(f.Axis.Add(radial.Scale(math.Scalar(f.RadSlope))))
	return Line{Origin: origin, Dir: dir}
}

// SectionPlane returns the plane a planar conic lies in — the plane a section of a quadric was cut by.
// ok=false for a straight curve (which lies in every plane through it) and for any curve AsConic does
// not recognise.
//
// Example:
//
//	if pl, ok := geom.SectionPlane(rim); ok { hits := geom.RaySurfaceHits(cyl, ruling, inf) }
func SectionPlane(c Curve3) (Plane, bool) {
	cf, ok := AsConic(c)
	if !ok {
		return Plane{}, false
	}
	pl, err := NewPlane(cf.Center, cf.Major.AsVector().Cross(cf.Minor.AsVector()))
	return pl, err == nil
}

// CurveParamAt inverts a curve's own parameterisation at a point ON it: straight curves by projection
// (a LineSegment onto [0,1], a Line onto its signed distance), conics through ConicParamAt. ok=false for
// a curve kind neither can invert.
//
// Example:
//
//	t, ok := geom.CurveParamAt(edge.Geometry(), pierce)
func CurveParamAt(c Curve3, p math.Point3) (float64, bool) {
	switch x := c.(type) {
	case LineSegment:
		d := x.StartPoint.VectorTo(x.EndPoint)
		l2 := float64(d.Dot(d))
		if l2 == 0 {
			return 0, false
		}
		return float64(x.StartPoint.VectorTo(p).Dot(d)) / l2, true
	case Line:
		return float64(x.Origin.VectorTo(p).Dot(x.Dir.AsVector())), true
	}
	return ConicParamAt(c, p)
}

// SectionCrossingCandidates returns the points where two curves on a ruled surface CAN meet — the
// pierce points of the line their section planes share, or of a ruling through the other's plane —
// without regard to either curve's parameter bounds (the caller checks those). coincident=true reports
// two sections in ONE plane: on the surface they are the same conic, a contact no crossing describes.
// Two rulings never cross. A pair it cannot describe (a non-planar curve) returns nothing.
//
// Example:
//
//	pts, same := geom.SectionCrossingCandidates(cyl, rimArc, toolSection)
func SectionCrossingCandidates(s Surface, a, b Curve3) (pts []math.Point3, coincident bool) {
	sa, sb := IsStraightCurve(a), IsStraightCurve(b)
	switch {
	case sa && sb:
		return nil, false
	case sa:
		return linePlanePierce(a, b)
	case sb:
		return linePlanePierce(b, a)
	}
	pa, okA := SectionPlane(a)
	pb, okB := SectionPlane(b)
	if !okA || !okB {
		return nil, false
	}
	p0, dir, ok := PlanePlaneLine(pa, pb)
	if !ok {
		return nil, coplanarPlanes(pa, pb)
	}
	ln, err := NewLine(p0, dir)
	if err != nil {
		return nil, false
	}
	return lineSurfacePierces(s, ln), false
}

// linePlanePierce is the single point where a straight curve pierces a planar curve's plane, or nothing
// when it runs parallel to it.
func linePlanePierce(straight, planar Curve3) ([]math.Point3, bool) {
	pl, ok := SectionPlane(planar)
	if !ok {
		return nil, false
	}
	o, d := straight.PointAt(0), straight.PointAt(1)
	dir := o.VectorTo(d)
	denom := float64(pl.Normal().Dot(dir))
	if denom == 0 {
		return nil, false
	}
	t := float64(pl.Normal().Dot(o.VectorTo(pl.Origin))) / denom
	return []math.Point3{o.TranslateBy(dir.Scale(math.Scalar(t)))}, false
}

// lineSurfacePierces returns every point of a full line on the surface: the forward ray hits plus the
// backward ones (a ray reports only T ≥ 0), with a hit at the origin counted once.
func lineSurfacePierces(s Surface, ln Line) []math.Point3 {
	back, _ := NewLine(ln.Origin, ln.Dir.AsVector().Scale(-1))
	var pts []math.Point3
	for _, h := range RaySurfaceHits(s, ln, stdmath.Inf(1)) {
		pts = append(pts, h.Point)
	}
	for _, h := range RaySurfaceHits(s, back, stdmath.Inf(1)) {
		if h.T > 0 {
			pts = append(pts, h.Point)
		}
	}
	return pts
}

// coplanarPlanes reports whether two parallel planes are the same plane.
func coplanarPlanes(a, b Plane) bool {
	res := ResolutionForPoints([]math.Point3{a.Origin, b.Origin})
	return stdmath.Abs(float64(a.Normal().Dot(a.Origin.VectorTo(b.Origin)))) <= res.Sew()
}

// AxialExtent returns the least and greatest axial coordinate (distance along axis from origin) a curve
// reaches over its parameter span [t0, t1] (in either order): the endpoints for a straight curve, and
// for a conic also the interior stationary points of its axial coordinate, found in closed form. ok=false
// for a curve kind it cannot bound.
//
// Example:
//
//	lo, hi, ok := geom.AxialExtent(rim, 0, 1, frame.Base, frame.Axis)
func AxialExtent(c Curve3, t0, t1 float64, origin math.Point3, axis math.Vector3) (lo, hi float64, ok bool) {
	lo, hi = stdmath.Inf(1), stdmath.Inf(-1)
	take := func(t float64) {
		v := float64(origin.VectorTo(c.PointAt(t)).Dot(axis))
		lo, hi = stdmath.Min(lo, v), stdmath.Max(hi, v)
	}
	take(t0)
	take(t1)
	if IsStraightCurve(c) {
		return lo, hi, true
	}
	cf, isConic := AsConic(c)
	if !isConic {
		return 0, 0, false
	}
	for _, t := range conicAxialStationaryParams(c, cf, t0, t1, axis) {
		take(t)
	}
	return lo, hi, true
}

// conicAxialStationaryParams returns the curve parameters strictly inside (t0, t1) where the conic's
// axial coordinate is stationary. Along axis the elliptic form reads vc + α·cosθ + β·sinθ, stationary at
// θ* = atan2(β, α) and θ*+π; the hyperbolic form vc + α·coshθ + β·sinhθ is stationary at tanhθ = −β/α.
func conicAxialStationaryParams(c Curve3, cf ConicForm, t0, t1 float64, axis math.Vector3) []float64 {
	alpha := cf.A * float64(cf.Major.AsVector().Dot(axis))
	beta := cf.B * float64(cf.Minor.AsVector().Dot(axis))
	var thetas []float64
	if cf.Hyperbolic {
		if alpha != 0 && stdmath.Abs(beta/alpha) < 1 {
			thetas = append(thetas, stdmath.Atanh(-beta/alpha))
		}
	} else {
		th := stdmath.Atan2(beta, alpha)
		thetas = append(thetas, th, th+stdmath.Pi)
	}
	var out []float64
	for _, th := range thetas {
		if t, ok := conicParamOfAngle(c, th, t0, t1); ok {
			out = append(out, t)
		}
	}
	return out
}

// AxialWindowParams returns the parameter intervals of c whose AXIAL coordinate — the distance along
// axis from origin — lies within [vMin, vMax]. It is the inverse of [AxialExtent]: that reports the
// axial span a parameter range covers, this reports the parameter ranges an axial span covers.
//
// A plane cutting a ruled wall parallel to its axis sections it in a curve that runs to INFINITY: a
// ruling on a cylinder, a hyperbola arm on a cone. An imprint is sampled over its own domain, and an
// infinite domain samples to nothing, so such a section has to be clipped to the wall's own axial
// window before any chart can carry it (ADR-0062).
//
// The axial coordinate is not monotone along a conic — a hyperbola's arm turns at its vertex — so the
// stationary parameters split the curve into monotone pieces and each piece is inverted on its own.
// ok=false for a curve whose axial coordinate this cannot bracket (a non-conic with an open domain).
//
// Example — a hyperbola arm cut from a cone, clipped to the frustum's band:
//
//	spans, _ := geom.AxialWindowParams(arm, cone.Apex, axis, vMin, vMax)
func AxialWindowParams(c Curve3, origin math.Point3, axis math.Vector3, vMin, vMax float64) ([][2]float64, bool) {
	t0, t1, ok := axialSearchRange(c, origin, axis, vMin, vMax)
	if !ok {
		return nil, false
	}
	v := func(t float64) float64 { return float64(origin.VectorTo(c.PointAt(t)).Dot(axis)) }
	var out [][2]float64
	for _, piece := range monotonePieces(c, t0, t1, axis) {
		lo, hi := piece[0], piece[1]
		a, b := axialCross(v, lo, hi, vMin), axialCross(v, lo, hi, vMax)
		s, e := stdmath.Min(a, b), stdmath.Max(a, b)
		if e > s {
			out = append(out, [2]float64{s, e})
		}
	}
	return out, len(out) > 0
}

// axialSearchRange brackets the curve's own domain, or — where it is open — a parameter range whose
// axial extent COVERS [vMin, vMax], found by doubling out from the parameterisation's origin. The
// bracket is kept as tight as the doubling allows: a wider one costs bisection precision, since the
// inversion below converges relative to the bracket it starts from.
func axialSearchRange(c Curve3, origin math.Point3, axis math.Vector3, vMin, vMax float64) (float64, float64, bool) {
	lo, hi := c.Domain()
	if !stdmath.IsInf(lo, 0) && !stdmath.IsInf(hi, 0) {
		return lo, hi, true
	}
	span := 1.0
	for range axialSearchDoublings {
		covLo, covHi, ok := AxialExtent(c, -span, span, origin, axis)
		if ok && covLo <= vMin && covHi >= vMax {
			return -span, span, true
		}
		span *= 2
	}
	return 0, 0, false
}

// axialSearchDoublings caps the bracket search. Each doubling covers twice the parameter range, so this
// reaches 2^40 times the curve's own scale — far beyond any modelled body — before giving up.
const axialSearchDoublings = 40

// monotonePieces splits [t0, t1] at the parameters where the axial coordinate turns, so each piece can
// be inverted by bisection.
func monotonePieces(c Curve3, t0, t1 float64, axis math.Vector3) [][2]float64 {
	cuts := []float64{t0}
	if cf, isConic := AsConic(c); isConic && !IsStraightCurve(c) {
		cuts = append(cuts, conicAxialStationaryParams(c, cf, t0, t1, axis)...)
	}
	cuts = append(cuts, t1)
	sort.Float64s(cuts)
	var out [][2]float64
	for i := 0; i+1 < len(cuts); i++ {
		if cuts[i+1] > cuts[i] {
			out = append(out, [2]float64{cuts[i], cuts[i+1]})
		}
	}
	return out
}

// axialCross bisects a MONOTONE piece for the parameter whose axial coordinate is target, clamping to
// the piece's own end when the target lies beyond it. It converges RELATIVE to the bracket it is given,
// because an unbounded section's bracket is found by doubling and can start very wide: a fixed halving
// count would then leave the answer short of the window by a fraction of that bracket.
func axialCross(v func(float64) float64, lo, hi, target float64) float64 {
	vlo, vhi := v(lo), v(hi)
	if (vlo-target)*(vhi-target) > 0 { // the target is off this piece: clamp to the nearer end
		if stdmath.Abs(vlo-target) < stdmath.Abs(vhi-target) {
			return lo
		}
		return hi
	}
	for range axialBisectionCap {
		if hi-lo <= axialParamEps*(1+stdmath.Abs(lo)+stdmath.Abs(hi)) {
			break
		}
		mid := (lo + hi) / 2
		if (v(lo)-target)*(v(mid)-target) <= 0 {
			hi = mid
			continue
		}
		lo = mid
	}
	return (lo + hi) / 2
}

// axialParamEps is the bisection's relative convergence: the parameter is carried in double precision,
// so nothing is gained below a few ulps of its own magnitude.
const axialParamEps = 1e-15 // tol:numeric — relative parameter convergence (dimensionless)

// axialBisectionCap bounds the loop for a piece the convergence test cannot satisfy (a curve whose
// axial coordinate is flat over the bracket), so the inversion always terminates.
const axialBisectionCap = 200

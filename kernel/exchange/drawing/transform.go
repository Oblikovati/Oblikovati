// SPDX-License-Identifier: GPL-2.0-only

package drawing

import "math"

// Affine is a 3×4 row-major affine transform (the last row is implicitly 0 0 0 1). It
// composes block-insert placements so nested INSERTs accumulate correctly, where
// separately tracking rotation+scale would not (a rotation after a non-uniform scale is a
// shear). Block geometry is expanded by applying the composed transform to each entity's
// coordinates (see TransformEntity).
type Affine [3][4]float64

// IdentityAffine is the no-op transform.
func IdentityAffine() Affine {
	return Affine{{1, 0, 0, 0}, {0, 1, 0, 0}, {0, 0, 1, 0}}
}

// InsertAffine builds an INSERT's transform: scale along the block axes, then rotate about
// Z, then translate to the insertion point (the block-reference order).
func InsertAffine(in *Insert) Affine {
	c, s := math.Cos(in.Rotation), math.Sin(in.Rotation)
	sx, sy, sz := in.Scale[0], in.Scale[1], in.Scale[2]
	return Affine{
		{c * sx, -s * sy, 0, in.Insertion[0]},
		{s * sx, c * sy, 0, in.Insertion[1]},
		{0, 0, sz, in.Insertion[2]},
	}
}

// Mul returns a∘b (apply b, then a), so a parent transform composes with a child insert as
// parent.Mul(child).
func (a Affine) Mul(b Affine) Affine {
	var m Affine
	for i := 0; i < 3; i++ {
		for j := 0; j < 4; j++ {
			m[i][j] = a[i][0]*b[0][j] + a[i][1]*b[1][j] + a[i][2]*b[2][j]
			if j == 3 {
				m[i][j] += a[i][3] // homogeneous translation column
			}
		}
	}
	return m
}

// point applies the transform to a 3D point.
func (a Affine) point(p [3]float64) [3]float64 {
	return [3]float64{
		a[0][0]*p[0] + a[0][1]*p[1] + a[0][2]*p[2] + a[0][3],
		a[1][0]*p[0] + a[1][1]*p[1] + a[1][2]*p[2] + a[1][3],
		a[2][0]*p[0] + a[2][1]*p[1] + a[2][2]*p[2] + a[2][3],
	}
}

// vector applies only the linear part (no translation), for direction/axis vectors.
func (a Affine) vector(v [3]float64) [3]float64 {
	return [3]float64{
		a[0][0]*v[0] + a[0][1]*v[1] + a[0][2]*v[2],
		a[1][0]*v[0] + a[1][1]*v[1] + a[1][2]*v[2],
		a[2][0]*v[0] + a[2][1]*v[1] + a[2][2]*v[2],
	}
}

// xyScale is the average in-plane scale factor (geometric mean of the 2×2 linear part's
// row norms), used for radii. Exact for a similarity; an approximation under a non-uniform
// scale, where a circle technically becomes an ellipse — rare for blocks.
func (a Affine) xyScale() float64 {
	rx := math.Hypot(a[0][0], a[0][1])
	ry := math.Hypot(a[1][0], a[1][1])
	return math.Sqrt(rx * ry)
}

// xyMirrored reports whether the in-plane linear part flips orientation (a negative
// determinant, e.g. a mirrored block), which reverses an arc's CCW sweep.
func (a Affine) xyMirrored() bool {
	return a[0][0]*a[1][1]-a[0][1]*a[1][0] < 0
}

// ScaleEntities returns the entities uniformly scaled about the origin by factor — used by
// the importer to convert drawing units to the document's units (see MetersPerUnit). The
// per-entity transform keeps radii and axes consistent with the scaled positions.
func ScaleEntities(entities []Entity, factor float64) []Entity {
	if factor == 1 {
		return entities
	}
	m := Affine{{factor, 0, 0, 0}, {0, factor, 0, 0}, {0, 0, factor, 0}}
	return mapEntities(entities, m)
}

// TranslateEntities returns the entities shifted by (dx,dy,dz). The importer uses it to
// recenter a drawing whose coordinates sit far from the origin (georeferenced survey data
// in the tens of millions) back toward it, so the single-precision GPU vertex buffer keeps
// sub-unit accuracy. A pure translation leaves radii, axes and angles unchanged, so the
// shape is preserved exactly. A zero shift returns the input unchanged.
func TranslateEntities(entities []Entity, dx, dy, dz float64) []Entity {
	if dx == 0 && dy == 0 && dz == 0 {
		return entities
	}
	return mapEntities(entities, Affine{{1, 0, 0, dx}, {0, 1, 0, dy}, {0, 0, 1, dz}})
}

// mapEntities applies one affine to every entity, returning a new slice.
func mapEntities(entities []Entity, m Affine) []Entity {
	out := make([]Entity, len(entities))
	for i, e := range entities {
		out[i] = TransformEntity(e, m)
	}
	return out
}

// TransformEntity returns a copy of e with its coordinates mapped through m. The handle is
// preserved so the instance still traces to its source object.
func TransformEntity(e Entity, m Affine) Entity {
	switch g := e.(type) {
	case *Line:
		return &Line{Handle: g.Handle, Start: m.point(g.Start), End: m.point(g.End)}
	case *Point:
		return &Point{Handle: g.Handle, Position: m.point(g.Position)}
	case *Circle:
		return &Circle{Handle: g.Handle, Center: m.point(g.Center), Radius: g.Radius * m.xyScale(), Normal: g.Normal}
	case *Arc:
		return transformArc(g, m)
	case *Ellipse:
		return transformEllipse(g, m)
	case *LwPolyline:
		return transformLwPolyline(g, m)
	case *Spline:
		return transformSpline(g, m)
	default:
		return e
	}
}

// transformArc maps an arc by transforming its centre and its two angular endpoints, then
// recovering the radius and angles — robust to rotation and mirroring (which swaps the CCW
// start/end).
func transformArc(a *Arc, m Affine) *Arc {
	c := m.point(a.Center)
	sp := m.point(arcPoint(a.Center, a.Radius, a.StartAngle))
	ep := m.point(arcPoint(a.Center, a.Radius, a.EndAngle))
	start := math.Atan2(sp[1]-c[1], sp[0]-c[0])
	end := math.Atan2(ep[1]-c[1], ep[0]-c[0])
	r := math.Hypot(sp[0]-c[0], sp[1]-c[1])
	if m.xyMirrored() {
		start, end = end, start // a mirror reverses the sweep so CCW still holds
	}
	return &Arc{Handle: a.Handle, Center: c, Radius: r, StartAngle: start, EndAngle: end, Normal: a.Normal}
}

// arcPoint is the point at parametric angle ang on a circle of the given centre/radius.
func arcPoint(center [3]float64, r, ang float64) [3]float64 {
	return [3]float64{center[0] + r*math.Cos(ang), center[1] + r*math.Sin(ang), center[2]}
}

// transformEllipse maps an ellipse by transforming its centre and major-axis vector; the
// axis ratio and parametric angles are preserved (exact under a similarity).
func transformEllipse(e *Ellipse, m Affine) *Ellipse {
	return &Ellipse{
		Handle:     e.Handle,
		Center:     m.point(e.Center),
		MajorAxis:  m.vector(e.MajorAxis),
		AxisRatio:  e.AxisRatio,
		StartAngle: e.StartAngle,
		EndAngle:   e.EndAngle,
		Normal:     e.Normal,
	}
}

// transformLwPolyline maps each vertex; bulges are kept (preserved under a similarity,
// approximate under a non-uniform scale).
func transformLwPolyline(p *LwPolyline, m Affine) *LwPolyline {
	pts := make([][2]float64, len(p.Points))
	for i, v := range p.Points {
		t := m.point([3]float64{v[0], v[1], p.Elevation})
		pts[i] = [2]float64{t[0], t[1]}
	}
	elev := m.point([3]float64{0, 0, p.Elevation})[2]
	return &LwPolyline{Handle: p.Handle, Closed: p.Closed, Elevation: elev, Points: pts, Bulges: p.Bulges, Normal: p.Normal}
}

// transformSpline maps control and fit points; degree, knots and weights are invariant
// under the affine.
func transformSpline(s *Spline, m Affine) *Spline {
	cp := make([][3]float64, len(s.ControlPoints))
	for i, p := range s.ControlPoints {
		cp[i] = m.point(p)
	}
	fp := make([][3]float64, len(s.FitPoints))
	for i, p := range s.FitPoints {
		fp[i] = m.point(p)
	}
	return &Spline{
		Handle: s.Handle, Degree: s.Degree, Closed: s.Closed, Rational: s.Rational,
		Knots: s.Knots, ControlPoints: cp, Weights: s.Weights, FitPoints: fp,
	}
}

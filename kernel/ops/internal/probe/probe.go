// SPDX-License-Identifier: GPL-2.0-only

// Package probe holds the read-only geometric questions the operation families all
// ask: a face's outward normal at a point, whether a triangle's winding opposes its
// vertex normals, the overlap of two boxes, and how far along a curve to sample.
//
// None of them changes anything, and none needs the operation layer. They sat in
// kernel/ops only because that is where their first caller was, which made every
// package that wanted to ask one of these questions — validate, query, blend — depend
// on the whole operation layer.
//
//	n := probe.OutwardFaceNormalAt(f, p)
package probe

import (
	stdmath "math"
	"slices"

	"oblikovati.org/kernel/geom"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
)

// CurveTrimSamples is the interior-sample count along an intersection curve (a count, not a tolerance):
// enough that a short in-trim crossing arc of a curve clipped to the whole shared face box is not
// stepped over.
const CurveTrimSamples = 16

// BoxOverlap returns the intersection box of a and b (their per-axis overlap). Its callers reach it
// only after Box.Intersects reported true, so every axis has Min ≤ Max.
func BoxOverlap(a, b math.Box) math.Box {
	return math.Box{
		Min: math.P3(stdmath.Max(a.Min.X, b.Min.X), stdmath.Max(a.Min.Y, b.Min.Y), stdmath.Max(a.Min.Z, b.Min.Z)),
		Max: math.P3(stdmath.Min(a.Max.X, b.Max.X), stdmath.Min(a.Max.Y, b.Max.Y), stdmath.Min(a.Max.Z, b.Max.Z)),
	}
}

// SampleRange returns a finite parameter interval of curve c to sample, bounded to the box overlap. A
// bounded intersection curve (a closed loop from two curved faces) uses its own finite domain (ok=true).
// An UNBOUNDED curve — the infinite line a closed-form plane∩plane returns, domain [-Inf,+Inf] — is
// bounded by projecting overlap's eight corners onto the line: the line is affine, so the parameter of a
// point q is (q−P0)·d / |d|² with d = P(1)−P(0), and the corner projections bracket the box. An unbounded
// curve that is NOT a straight line (a cone section — parabola/hyperbola) cannot be bracketed this way,
// so ok=false and the caller conservatively treats the pair as crossing rather than risk a missed hit.
func SampleRange(c geom.Curve3, overlap math.Box) (lo, hi float64, ok bool) {
	dlo, dhi := c.Domain()
	if !stdmath.IsInf(dlo, 0) && !stdmath.IsInf(dhi, 0) {
		return dlo, dhi, true
	}
	p0, p1, pmid := c.PointAt(0), c.PointAt(1), c.PointAt(0.5)
	d := p0.VectorTo(p1)
	dd := d.Dot(d)
	if dd == 0 || !IsColinearMidpoint(p0, p1, pmid) {
		return 0, 0, false
	}
	lo, hi = stdmath.Inf(1), stdmath.Inf(-1)
	for _, q := range overlap.Corners() {
		t := p0.VectorTo(q).Dot(d) / dd
		lo, hi = stdmath.Min(lo, t), stdmath.Max(hi, t)
	}
	return lo, hi, true
}

// OutwardFaceNormalAt is a face's outward unit normal at point p on it (the surface normal,
// flipped when the face is reversed).
func OutwardFaceNormalAt(f *topo.Face, p math.Point3) math.Vector3 {
	u, v := f.Geometry().ParamAt(p)
	n := f.Geometry().NormalAt(u, v)
	if f.Reversed() {
		n = n.Scale(-1)
	}
	return UnitOr(n)
}

// WindingOpposesNormals reports whether triangle abc's geometric (cross-product) normal opposes
// the triangle's per-vertex shading normals (their sum). Used to wind a patch consistently with
// the normals each vertex actually carries — robust on a curved patch where the flat triangle's
// centroid is off the surface (unlike triangleFlipped's centroid sample).
func WindingOpposesNormals(a, b, c math.Point3, na, nb, nc math.Vector3) bool {
	gn := a.VectorTo(b).Cross(a.VectorTo(c))
	return gn.Dot(na.Add(nb).Add(nc)) < 0
}

// IsColinearMidpoint reports whether c's midpoint sample lies on the chord P0→P1 — i.e. the curve is a
// straight line over [0,1]. An intersection conic that is colinear at three parameters is degenerate to
// a line, so this cleanly separates the affine plane∩plane line (bracketable by corner projection) from
// a curved unbounded section that is not.
func IsColinearMidpoint(p0, p1, pmid math.Point3) bool {
	chord := p0.VectorTo(p1)
	mid := math.P3((p0.X+p1.X)/2, (p0.Y+p1.Y)/2, (p0.Z+p1.Z)/2)
	return pmid.VectorTo(mid).Length() <= colinearRelTol*chord.Length()
}

// UnitOr normalizes v, returning it unchanged when degenerate (zero length).
func UnitOr(v math.Vector3) math.Vector3 {
	l := v.Length()
	if l < math.Scalar(stdmath.Sqrt(float64(math.DefaultTolerance))) {
		return v
	}
	return v.Scale(1 / l)
}

// colinearRelTol is the chord-relative straightness bound for an unbounded section curve: a true line's
// midpoint deviates only by float rounding, far below this, while any conic bows well above it.
const colinearRelTol = 1e-9 // tol:relative — dimensionless straightness fraction of the chord length

// centroid returns the average of a polygon's vertices.
func Centroid(poly []math.Point3) math.Point3 {
	var sx, sy, sz float64
	for _, p := range poly {
		sx, sy, sz = sx+p.X, sy+p.Y, sz+p.Z
	}
	n := float64(len(poly))
	return math.P3(sx/n, sy/n, sz/n)
}

// creaseAngle is the acute angle between the two normals' LINES — min(θ, π−θ) — so an
// orientation-flipped (antiparallel) but tangent-continuous seam reads as 0 crease, not π.
func CreaseAngle(a, b math.Vector3) float64 {
	t := vectorAngle(a, b)
	return stdmath.Min(t, stdmath.Pi-t)
}

// vectorAngle is the angle (radians) between two vectors; a degenerate (zero-length) input returns π
// (worst case) so it cannot mask a fold as a passing seam.
func vectorAngle(a, b math.Vector3) float64 {
	la, lb := a.Length(), b.Length()
	if la == 0 || lb == 0 {
		return stdmath.Pi
	}
	c := a.Dot(b) / (la * lb)
	return stdmath.Acos(stdmath.Max(-1, stdmath.Min(1, c)))
}

// isClosedCircularEdge reports whether e is a full circular rim: closed (its start and end vertex are
// the SAME vertex) AND its geometry is a geom.Circle, or a geom.Arc3d sweeping ~2π (within
// zoneFullCircleTol) back to that shared vertex. The STEP importer never emits geom.Circle — every
// imported full circle round-trips as a full-sweep Arc3d (0 geom.Circle construction sites in
// kernel/exchange/**) — so both forms must count as "a closed circular rim". This is the SOLE such
// predicate in the package: the rim-fillet pick gate (loneRimPick, resolveRim below) and the
// sphere-zone cap fan's fullCircleRimGeom (sphere_zone_mesh.go) both call it, so a full-sweep Arc3d is
// recognized identically everywhere a closed circular edge is classified.
func IsClosedCircularEdge(e *topo.Edge) bool {
	if e.StartVertex() != e.EndVertex() {
		return false
	}
	switch c := e.Geometry().(type) {
	case geom.Circle:
		return true
	case geom.Arc3d:
		return stdmath.Abs(stdmath.Abs(c.SweepAngle)-2*stdmath.Pi) < zoneFullCircleTol
	}
	return false
}

// zoneFullCircleTol: a swept edge is a full circle when its sweep is within this of 2π (radians;
// scale-free, an angle).
const zoneFullCircleTol = 1e-6

// RayTriangleDist returns the positive distance along the ray to triangle abc, via
// Möller–Trumbore, or ok=false if there is no forward hit.
func RayTriangleDist(orig math.Point3, dir math.Vector3, a, b, c math.Point3) (float64, bool) {
	const eps = 1e-9
	e1 := a.VectorTo(b)
	e2 := a.VectorTo(c)
	pv := dir.Cross(e2)
	det := e1.Dot(pv)
	if det > -eps && det < eps {
		return 0, false
	}
	inv := 1 / det
	tv := a.VectorTo(orig)
	u := tv.Dot(pv) * inv
	if u < 0 || u > 1 {
		return 0, false
	}
	qv := tv.Cross(e1)
	v := dir.Dot(qv) * inv
	if v < 0 || u+v > 1 {
		return 0, false
	}
	t := e2.Dot(qv) * inv
	return t, t > eps
}

// ReversedPoints returns a reversed COPY of pts, leaving the input alone — the shape a
// caller needs when it must keep the original orientation as well.
func ReversedPoints(pts []math.Point3) []math.Point3 {
	out := slices.Clone(pts)
	slices.Reverse(out)
	return out
}

// XY converts a Point2 to the [2]float64 form the shared geometry predicates use.
func XY(p math.Point2) [2]float64 { return [2]float64{p.X, p.Y} }

// WrapPi folds an angle into (−π, π] — the signed shortest step between two tube parameters.
func WrapPi(a float64) float64 {
	const twoPi = 2 * stdmath.Pi
	for a > stdmath.Pi {
		a -= twoPi
	}
	for a <= -stdmath.Pi {
		a += twoPi
	}
	return a
}

// NewellUnit returns a loop's unit normal by Newell's method (robust for non-planar loops).
func NewellUnit(loop []math.Point3) math.Vector3 {
	var nx, ny, nz float64
	n := len(loop)
	for i := range n {
		c, d := loop[i], loop[(i+1)%n]
		nx += (c.Y - d.Y) * (c.Z + d.Z)
		ny += (c.Z - d.Z) * (c.X + d.X)
		nz += (c.X - d.X) * (c.Y + d.Y)
	}
	u, err := math.UnitVector3FromVector(math.V3(nx, ny, nz))
	if err != nil {
		return math.V3(0, 0, 1)
	}
	return u.AsVector()
}

// PointInLoop2D is a ray-cast point-in-polygon test for a closed loop.
func PointInLoop2D(p math.Point2, loop []math.Point2) bool {
	in := false
	for i, j := 0, len(loop)-1; i < len(loop); j, i = i, i+1 {
		yi, yj := loop[i].Y, loop[j].Y
		if (yi > p.Y) != (yj > p.Y) {
			x := loop[i].X + (p.Y-yi)/(yj-yi)*(loop[j].X-loop[i].X)
			if p.X < x {
				in = !in
			}
		}
	}
	return in
}

// Sign returns -1, 0 or +1 for a negative, zero or positive value. It compares against
// exact zero on purpose: callers pass an already-computed orientation determinant, and
// widening that to a tolerance here would hide the decision from the predicate that made it.
func Sign(x float64) int {
	if x > 0 {
		return 1
	}
	if x < 0 {
		return -1
	}
	return 0
}

// meetOfPlanes returns the least-squares meeting point of ≥3 planes (exact for 3).
func MeetOfPlanes(planes []geom.Plane) (math.Point3, bool) {
	var a [3][3]float64
	var b [3]float64
	for _, pl := range planes {
		n := pl.Normal()
		d := n.Dot(pl.Origin.AsVector())
		nv := [3]float64{n.X, n.Y, n.Z}
		for i := range 3 {
			for j := range 3 {
				a[i][j] += nv[i] * nv[j]
			}
			b[i] += nv[i] * d
		}
	}
	x, ok := Solve3(a, b)
	return math.P3(x[0], x[1], x[2]), ok
}

// TwoPlaneLine returns a point and direction of the intersection line of two planes, or
// ok=false when they are parallel.
func TwoPlaneLine(a, b geom.Plane) (math.Point3, math.Vector3, bool) {
	na, nb := a.Normal(), b.Normal()
	dir := na.Cross(nb)
	if dir.LengthSquared() < 1e-18 {
		return math.Point3{}, math.Vector3{}, false
	}
	da, db := na.Dot(a.Origin.AsVector()), nb.Dot(b.Origin.AsVector())
	num := nb.Cross(dir).Scale(da).Add(dir.Cross(na).Scale(db))
	return math.P3(0, 0, 0).TranslateBy(num.Scale(1 / dir.LengthSquared())), dir, true
}

// Canon2 orders an undirected vertex-index pair.
func Canon2(a, b int) [2]int {
	if a < b {
		return [2]int{a, b}
	}
	return [2]int{b, a}
}

// CentroidPts averages a point set (a point on the face for its plane origin).
func CentroidPts(pts []math.Point3) math.Point3 {
	var sx, sy, sz float64
	for _, p := range pts {
		sx, sy, sz = sx+p.X, sy+p.Y, sz+p.Z
	}
	n := float64(len(pts))
	return math.P3(sx/n, sy/n, sz/n)
}

// SrcIDAt returns ids[i] or 0 when the point has no carried identity.
func SrcIDAt(ids []uint64, i int) uint64 {
	if i < len(ids) {
		return ids[i]
	}
	return 0
}

// FaceCentroid averages a face's outer-loop edge start points — the face's position, valid for any
// surface type (a planar face's surface.PointAt does not track the trimmed face's location).
func FaceCentroid(f *topo.Face) math.Point3 {
	var sum math.Vector3
	n := 0
	for _, l := range f.Loops() {
		for _, u := range l.EdgeUses() {
			c := u.Edge().Geometry()
			lo, _ := c.Domain()
			sum = sum.Add(c.PointAt(lo).AsVector())
			n++
		}
	}
	if n == 0 {
		return math.P3(0, 0, 0)
	}
	return sum.Scale(1 / float64(n)).AsPoint()
}

// Unit returns v normalized, or v unchanged if it is degenerate.
func Unit(v math.Vector3) math.Vector3 {
	if u, err := math.UnitVector3FromVector(v); err == nil {
		return u.AsVector()
	}
	return v
}

func AbsFloat(x float64) float64 {
	if x < 0 {
		return -x
	}
	return x
}

// perpLeft rotates a direction 90° counterclockwise — the positive-offset side.
func PerpLeft(v math.Vector2) math.Vector2 { return math.V2(-v.Y, v.X) }

func Unit2(v math.Vector2) math.Vector2 {
	l := float64(v.Length())
	if l == 0 {
		return v
	}
	return v.Scale(math.Scalar(1 / l))
}

// FirstNurbsFace returns a body's first B-spline-surface face and its geometry — the face a
// NURBS-only operation acts on, and the guard that refuses a body that has none.
//
// Example: f, s, ok := probe.FirstNurbsFace(body)
func FirstNurbsFace(b *topo.Body) (*topo.Face, geom.BSplineSurface, bool) {
	for _, f := range b.Faces() {
		if s, ok := f.Geometry().(geom.BSplineSurface); ok {
			return f, s, true
		}
	}
	return nil, geom.BSplineSurface{}, false
}

// Solve3 solves the 3×3 system a·x = b by Cramer's rule, ok=false when a is singular.
func Solve3(a [3][3]float64, b [3]float64) ([3]float64, bool) {
	det := Det3(a)
	if det < SingularSolveTol && det > -SingularSolveTol {
		return [3]float64{}, false
	}
	var x [3]float64
	for c := range 3 {
		m := a
		for r := range 3 {
			m[r][c] = b[r]
		}
		x[c] = Det3(m) / det
	}
	return x, true
}

// Det3 returns the determinant of a 3×3 matrix.
func Det3(m [3][3]float64) float64 {
	return m[0][0]*(m[1][1]*m[2][2]-m[1][2]*m[2][1]) -
		m[0][1]*(m[1][0]*m[2][2]-m[1][2]*m[2][0]) +
		m[0][2]*(m[1][0]*m[2][1]-m[1][1]*m[2][0])
}

// SingularSolveTol is the magnitude below which a determinant or a ray/line·plane denominator
// is treated as zero — the linear solve is singular or the line is parallel to the plane. It
// is below the linear DefaultTolerance because it bounds a product of (roughly unit) direction
// terms, not a length.
const SingularSolveTol = 1e-12

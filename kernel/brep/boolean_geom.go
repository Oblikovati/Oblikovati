// SPDX-License-Identifier: GPL-2.0-only

package brep

import (
	stdmath "math"

	"oblikovati/kernel/geom"
	"oblikovati/kernel/topo"
	"oblikovati/math"
)

// planarFace is a planar face flattened for the boolean: its plane, outward normal,
// boundary loops as 3D point rings (loops[0] = outer, the rest = holes), and the source
// face's lineage so a result face can carry it forward (reference-key survival, K1a).
type planarFace struct {
	plane   geom.Plane
	normal  math.Vector3
	loops   [][]math.Point3
	lineage topo.Lineage
}

// facesOf flattens a body's planar faces. A non-planar face makes it return ok=false (the
// planar B-rep boolean handles planar-faceted solids; curved faces need NURBS work).
func facesOf(b *topo.Body) ([]planarFace, bool) {
	var out []planarFace
	for _, f := range b.Faces() {
		pl, ok := f.Geometry().(geom.Plane)
		if !ok {
			return nil, false
		}
		out = append(out, planarFace{plane: pl, normal: f.Geometry().NormalAt(0, 0), loops: loopRings(f), lineage: f.Lineage()})
	}
	return out, true
}

// loopRings returns a face's loops as ordered 3D point rings (outer loop first).
func loopRings(f *topo.Face) [][]math.Point3 {
	var outer [][]math.Point3
	var holes [][]math.Point3
	for _, l := range f.Loops() {
		ring := loopPoints(l)
		if l.IsOuter() {
			outer = append(outer, ring)
		} else {
			holes = append(holes, ring)
		}
	}
	return append(outer, holes...)
}

// loopPoints returns a loop's ordered vertices (honoring each edge use's direction).
func loopPoints(l *topo.Loop) []math.Point3 {
	uses := l.EdgeUses()
	pts := make([]math.Point3, 0, len(uses))
	for _, u := range uses {
		v := u.Edge().StartVertex()
		if u.Reversed() {
			v = u.Edge().EndVertex()
		}
		pts = append(pts, v.Point())
	}
	return pts
}

// to2D / to3D convert between model space and a plane's local (u,v) coordinates.
func to2D(pl geom.Plane, p math.Point3) math.Point2 {
	d := pl.Origin.VectorTo(p)
	return math.P2(d.Dot(pl.UAxis.AsVector()), d.Dot(pl.VAxis.AsVector()))
}

func to3D(pl geom.Plane, q math.Point2) math.Point3 {
	return pl.Origin.TranslateBy(pl.UAxis.AsVector().Scale(q.X).Add(pl.VAxis.AsVector().Scale(q.Y)))
}

// planeLine returns a point and unit direction of the intersection line of two planes,
// or ok=false when they are parallel. n·x = d form is taken from each plane's normal and
// origin.
func planeLine(a, b geom.Plane) (p0 math.Point3, dir math.Vector3, ok bool) {
	na, nb := unit(a.Normal()), unit(b.Normal())
	d := na.Cross(nb)
	if d.LengthSquared() < 1e-18 {
		return p0, dir, false
	}
	da, db := na.Dot(a.Origin.AsVector()), nb.Dot(b.Origin.AsVector())
	// Point on both planes nearest the origin: p0 = (da(nb×d) + db(d×na)) / |d|².
	num := nb.Cross(d).Scale(da).Add(d.Cross(na).Scale(db))
	return math.P3(0, 0, 0).TranslateBy(num.Scale(1 / d.LengthSquared())), unit(d), true
}

func unit(v math.Vector3) math.Vector3 {
	u, err := math.UnitVector3FromVector(v)
	if err != nil {
		return math.V3(0, 0, 0)
	}
	return u.AsVector()
}

// faceLineIntervals returns the parameter intervals (along p0+t·dir) where the line lies
// inside the face (inside the outer ring and outside every hole), via even–odd crossings
// projected into the face plane.
func faceLineIntervals(f planarFace, p0 math.Point3, dir math.Vector3) [][2]float64 {
	o2, d2 := to2D(f.plane, p0), to2Dvec(f.plane, dir)
	if d2.LengthSquared() < 1e-18 {
		return nil // the line is perpendicular to this plane (no in-plane extent)
	}
	var ts []float64
	for _, ring := range f.loops {
		ts = append(ts, ringCrossings(o2, d2, ring, f.plane)...)
	}
	stdSort(ts)
	var out [][2]float64
	for i := 0; i+1 < len(ts); i++ {
		mid := (ts[i] + ts[i+1]) / 2
		if pointInFace2D(o2.TranslateBy(d2.Scale(mid)), f) {
			out = append(out, [2]float64{ts[i], ts[i+1]})
		}
	}
	return out
}

func to2Dvec(pl geom.Plane, v math.Vector3) math.Vector2 {
	return math.V2(v.Dot(pl.UAxis.AsVector()), v.Dot(pl.VAxis.AsVector()))
}

// ringCrossings returns the line parameters t where the 2D line (o2 + t·d2) crosses the
// ring's edges.
func ringCrossings(o2 math.Point2, d2 math.Vector2, ring []math.Point3, pl geom.Plane) []float64 {
	var ts []float64
	n := len(ring)
	for i := 0; i < n; i++ {
		a, b := to2D(pl, ring[i]), to2D(pl, ring[(i+1)%n])
		if t, ok := lineSegCross(o2, d2, a, b); ok {
			ts = append(ts, t)
		}
	}
	return ts
}

// lineSegCross intersects the infinite line (o + t·d) with segment [a,b], returning the
// line parameter t when they cross within the segment.
func lineSegCross(o math.Point2, d math.Vector2, a, b math.Point2) (float64, bool) {
	e := a.VectorTo(b)
	den := d.Cross(e)
	if stdmath.Abs(den) < parallelDenomTol {
		return 0, false // parallel
	}
	ao := o.VectorTo(a)
	t := ao.Cross(e) / den
	s := ao.Cross(d) / den
	if s < -1e-9 || s > 1+1e-9 {
		return 0, false
	}
	return t, true
}

// pointInFace2D reports whether a 2D point is inside the face (in the outer ring, outside
// holes).
func pointInFace2D(q math.Point2, f planarFace) bool {
	if len(f.loops) == 0 || !pointInPolygon2D(q, ring2D(f.plane, f.loops[0])) {
		return false
	}
	for _, h := range f.loops[1:] {
		if pointInPolygon2D(q, ring2D(f.plane, h)) {
			return false
		}
	}
	return true
}

func ring2D(pl geom.Plane, ring []math.Point3) []math.Point2 {
	out := make([]math.Point2, len(ring))
	for i, p := range ring {
		out[i] = to2D(pl, p)
	}
	return out
}

func stdSort(ts []float64) {
	for i := 1; i < len(ts); i++ {
		for j := i; j > 0 && ts[j-1] > ts[j]; j-- {
			ts[j-1], ts[j] = ts[j], ts[j-1]
		}
	}
}

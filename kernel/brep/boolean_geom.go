// SPDX-License-Identifier: GPL-2.0-only

package brep

import (
	stdmath "math"

	"oblikovati.org/kernel/geom"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
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

// facesOf flattens a body's planar faces, DERIVED from the unified curvedFace flatten (facesOfAny) so
// there is one loop-extraction path (ADR-0058). A non-planar face makes it return ok=false (the planar
// B-rep boolean handles planar-faceted solids; curved faces take the curved pipeline). The polygonal 3D
// point rings the planar arrangement needs come from each loop edge's EXACT loop-oriented start vertex
// (loopEdge.v0, stored by orientedLoopEdge from the topo vertex) — NOT PointAt(t0), which only
// approximates a vertex on a curved edge (arc round-trip) and would drift the arrangement's welds. So this
// is byte-identical to the retired loopRings/loopPoints on every face facesOf accepts, arc-bounded caps
// included.
func facesOf(b *topo.Body) ([]planarFace, bool) {
	out := make([]planarFace, 0, len(b.Faces()))
	for _, cf := range facesOfAny(b) {
		pl, ok := cf.surface.(geom.Plane)
		if !ok {
			return nil, false
		}
		out = append(out, planarFace{plane: pl, normal: pl.NormalAt(0, 0), loops: ringsOfLoops(cf.loops), lineage: cf.lineage})
	}
	return out, true
}

// ringsOfLoops projects a curved face's parametric loops onto the polygonal boolean's 3D point rings:
// each loop's ordered edge EXACT start vertices (v0), outer loop already first (loopsOf order).
func ringsOfLoops(loops []curvedLoop) [][]math.Point3 {
	out := make([][]math.Point3, len(loops))
	for i, l := range loops {
		ring := make([]math.Point3, len(l.edges))
		for j, e := range l.edges {
			ring[j] = e.v0
		}
		out[i] = ring
	}
	return out
}

// to2D / to3D convert between model space and a plane's local (u,v) coordinates. These are the planar
// arrangement's OWN exact maps: to2D equals geom.Plane.ParamAt, but to3D keeps its single-translate lift
// (Origin + (uU+vV), one rounding) rather than delegating to geom.Plane.PointAt (which does two). The
// difference is ~1 ULP, but the planar arrangement's welds/coincidence tests depend on this exact lift,
// and — critically — the planar and curved splits keep SEPARATE domain maps by design (exact vs sampled),
// so they need not agree bit-for-bit. Aligning geom.Plane.PointAt to this form instead drifts every
// tessellation/coincidence that uses PointAt (byte-identity-fingerprint regressions, ADR-0058 blocker-1
// finding): the domain maps are correctly NOT unified.
func to2D(pl geom.Plane, p math.Point3) math.Point2 {
	d := pl.Origin.VectorTo(p)
	return math.P2(d.Dot(pl.UAxis.AsVector()), d.Dot(pl.VAxis.AsVector()))
}

func to3D(pl geom.Plane, q math.Point2) math.Point3 {
	return pl.Origin.TranslateBy(pl.UAxis.AsVector().Scale(q.X).Add(pl.VAxis.AsVector().Scale(q.Y)))
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
	if d2.LengthSquared() < 1e-18 { // tol:numeric — degenerate-direction guard (squared length)
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
	// A line that runs exactly ALONG a boundary edge grazes the face: the even–odd midpoint
	// test above lands on the boundary and rejects it non-deterministically (a float tie-break
	// that drops one wall of a boundary-straddle cut, #860). Add the span of every collinear
	// boundary edge explicitly so such an imprint is captured on whichever face owns the edge
	// — for that face interiorSegments later drops it as on-boundary; for the other face (where
	// the same line is interior) the overlap keeps it.
	return append(out, collinearEdgeSpans(o2, d2, f)...)
}

// collinearEdgeSpans returns the line parameters spanned by each boundary edge of f that lies
// along the line (o2 + t·d2) — both endpoints within [arrTol] of the line. The span is the
// edge endpoints projected onto the line; an edge meeting the line at a single point is not
// collinear and is excluded.
func collinearEdgeSpans(o2 math.Point2, d2 math.Vector2, f planarFace) [][2]float64 {
	lenSq := d2.LengthSquared()
	var out [][2]float64
	for _, ring := range f.loops {
		n := len(ring)
		for i := range n {
			a, b := to2D(f.plane, ring[i]), to2D(f.plane, ring[(i+1)%n])
			if distPointLine2D(a, o2, d2) > arrTol || distPointLine2D(b, o2, d2) > arrTol {
				continue
			}
			ta := o2.VectorTo(a).Dot(d2) / lenSq
			tb := o2.VectorTo(b).Dot(d2) / lenSq
			out = append(out, [2]float64{min(ta, tb), max(ta, tb)})
		}
	}
	return out
}

// distPointLine2D is the perpendicular distance from p to the line through o with direction d.
func distPointLine2D(p, o math.Point2, d math.Vector2) float64 {
	op := o.VectorTo(p)
	cross := op.Cross(d)
	return stdmath.Abs(cross) / stdmath.Sqrt(d.LengthSquared())
}

func to2Dvec(pl geom.Plane, v math.Vector3) math.Vector2 {
	return math.V2(v.Dot(pl.UAxis.AsVector()), v.Dot(pl.VAxis.AsVector()))
}

// ringCrossings returns the line parameters t where the 2D line (o2 + t·d2) crosses the
// ring's edges.
func ringCrossings(o2 math.Point2, d2 math.Vector2, ring []math.Point3, pl geom.Plane) []float64 {
	var ts []float64
	n := len(ring)
	for i := range n {
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
	if s < -1e-9 || s > 1+1e-9 { // tol:parametric — segment parameter s in [0,1]
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

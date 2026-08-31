// SPDX-License-Identifier: GPL-2.0-only

package brep

import (
	stdmath "math"

	"oblikovati.org/kernel/geom"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
)

// The planar B-rep boolean operates on the unified curvedFace model (ADR-0058): a planar face is a
// curvedFace whose surface is a geom.Plane and whose loop edges are straight (their exact endpoints carried
// in loopEdge.v0). The former separate planarFace{plane, normal, 3D rings} type is retired; facePlane,
// faceNormal and faceRings read the same three things off a curvedFace, so the exact planar arrangement is
// unchanged while the type is now shared with the curved boolean.

// facesOf flattens a body's faces as curvedFaces and reports ok=false the moment one is non-planar (the
// planar B-rep boolean handles planar-faceted solids; curved faces take the curved pipeline).
func facesOf(b *topo.Body) ([]curvedFace, bool) {
	faces := facesOfAny(b)
	for i := range faces {
		if _, ok := faces[i].surface.(geom.Plane); !ok {
			return nil, false
		}
	}
	return faces, true
}

// facePlane is a planar face's plane (the planar boolean only ever holds curvedFaces whose surface is a
// geom.Plane — facesOf guarantees it).
func facePlane(f curvedFace) geom.Plane { return f.surface.(geom.Plane) }

// faceNormal is a planar face's outward surface normal (constant over a plane).
func faceNormal(f curvedFace) math.Vector3 { return f.surface.NormalAt(0, 0) }

// planarRings projects a planar face's parametric loops onto the polygonal arrangement's 3D point rings: each
// loop's ordered edge EXACT start vertices (loopEdge.v0, bit-exact even for an arc-bounded cap), outer loop
// first (loopsOf order). It is the arrangement's on-demand view of the shared curvedFace loop model.
func planarRings(f curvedFace) [][]math.Point3 {
	out := make([][]math.Point3, len(f.loops))
	for i, l := range f.loops {
		ring := make([]math.Point3, len(l.edges))
		for j, e := range l.edges {
			ring[j] = e.v0
		}
		out[i] = ring
	}
	return out
}

// planarFaceFromRings builds a curvedFace for a planar face given its plane and 3D point rings (outer
// first) — the drill/curved-on-planar handlers synthesize their planar operands this way. Each ring edge
// becomes a straight loopEdge whose exact endpoints (v0/v1) are the ring vertices, so faceRings round-trips
// them bit-for-bit.
func planarFaceFromRings(pl geom.Plane, rings [][]math.Point3, lineage topo.Lineage) curvedFace {
	loops := make([]curvedLoop, len(rings))
	for i, ring := range rings {
		edges := make([]loopEdge, len(ring))
		for j := range ring {
			a, b := ring[j], ring[(j+1)%len(ring)]
			edges[j] = loopEdge{curve: geom.NewLineSegment(a, b), t0: 0, t1: 1, v0: a, v1: b}
		}
		loops[i] = curvedLoop{edges: edges}
	}
	return curvedFace{surface: pl, loops: loops, lineage: lineage}
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
func faceLineIntervals(f curvedFace, p0 math.Point3, dir math.Vector3) [][2]float64 {
	o2, d2 := to2D(facePlane(f), p0), to2Dvec(facePlane(f), dir)
	if d2.LengthSquared() < 1e-18 { // tol:numeric — degenerate-direction guard (squared length)
		return nil // the line is perpendicular to this plane (no in-plane extent)
	}
	var ts []float64
	for _, ring := range planarRings(f) {
		ts = append(ts, ringCrossings(o2, d2, ring, facePlane(f))...)
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
func collinearEdgeSpans(o2 math.Point2, d2 math.Vector2, f curvedFace) [][2]float64 {
	lenSq := d2.LengthSquared()
	var out [][2]float64
	for _, ring := range planarRings(f) {
		n := len(ring)
		for i := range n {
			a, b := to2D(facePlane(f), ring[i]), to2D(facePlane(f), ring[(i+1)%n])
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
func pointInFace2D(q math.Point2, f curvedFace) bool {
	rings := planarRings(f)
	if len(rings) == 0 || !pointInPolygon2D(q, ring2D(facePlane(f), rings[0])) {
		return false
	}
	for _, h := range rings[1:] {
		if pointInPolygon2D(q, ring2D(facePlane(f), h)) {
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

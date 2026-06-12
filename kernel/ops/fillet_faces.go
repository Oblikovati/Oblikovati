// SPDX-License-Identifier: GPL-2.0-only

package ops

import (
	"oblikovati.org/kernel/geom"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
)

// filletResultFaces builds the faces of the filleted body: every original face transformed
// for the fillets touching it (its A/B corners pulled back to the tangent points, its simple
// end corners replaced by an arc or chord fan), one cylinder face per constant filleted edge
// (or a planar ruling strip per variable one), and one sphere patch per corner blend.
func filletResultFaces(body *topo.Body, fils []edgeFillet, blends map[uint64]*cornerBlend) []filletFace {
	abSubst, endCorner := filletMaps(fils)
	var out []filletFace
	for _, f := range body.Faces() {
		out = append(out, transformFace(f, abSubst[f], endCorner[f]))
	}
	for _, ef := range fils {
		if ef.varying {
			out = append(out, rulingStripFaces(ef)...)
			continue
		}
		out = append(out, cylinderFace(ef))
	}
	for _, cb := range blends {
		out = append(out, spherePatchFace(cb))
	}
	return out
}

// rulingStripFaces builds a variable fillet's blend as planar trapezoids between successive
// rulings: chord j of corner 0 pairs with chord j of corner 1 (the corners are sampled at the
// same angular stations). Adjacent rulings of the blend meet at its cone apex on the edge
// line, so each strip face is exactly planar; winding is fixed outward (away from the centre
// line) per quad.
func rulingStripFaces(ef edgeFillet) []filletFace {
	cmid := ef.c0.cen.Midpoint(ef.c1.cen)
	out := make([]filletFace, 0, len(ef.c0.chords)-1)
	for j := 0; j+1 < len(ef.c0.chords); j++ {
		quad := [4]math.Point3{ef.c0.chords[j], ef.c1.chords[j], ef.c1.chords[j+1], ef.c0.chords[j+1]}
		out = append(out, planarQuadFace(quad, cmid))
	}
	return out
}

// planarQuadFace builds one planar face over the quad, wound so its normal points away from
// the blend centre line (approximated by cmid — the wedge spans < π, so the sign is robust).
func planarQuadFace(quad [4]math.Point3, cmid math.Point3) filletFace {
	n := quad[0].VectorTo(quad[1]).Cross(quad[0].VectorTo(quad[3]))
	qc := centroidPts(quad[:])
	if n.Dot(cmid.VectorTo(qc)) < 0 {
		quad = [4]math.Point3{quad[0], quad[3], quad[2], quad[1]}
		n = n.Scale(-1)
	}
	pl, _ := geom.NewPlane(quad[0], n)
	return filletFace{surface: pl, loops: []filletLoop{{
		pts:    quad[:],
		curves: make([]geom.Curve3, 4),
	}}}
}

// filletMaps indexes, per face: the corner-vertex → tangent-point pullbacks (where the face
// is a fillet's A or B face — every corner, simple or blended), and the corner-vertex →
// corner arc (only for SIMPLE end corners; a blend corner's arc lives on the sphere patch,
// and all its faces just pull back to the sphere tangent points).
func filletMaps(fils []edgeFillet) (map[*topo.Face]map[uint64]math.Point3, map[*topo.Face]map[uint64]corner) {
	ab := map[*topo.Face]map[uint64]math.Point3{}
	ends := map[*topo.Face]map[uint64]corner{}
	put := func(m map[*topo.Face]map[uint64]math.Point3, f *topo.Face, id uint64, p math.Point3) {
		if m[f] == nil {
			m[f] = map[uint64]math.Point3{}
		}
		m[f][id] = p
	}
	for _, ef := range fils {
		for _, c := range []corner{ef.c0, ef.c1} {
			put(ab, c.a, c.vertex.ID(), c.ta)
			put(ab, c.b, c.vertex.ID(), c.tb)
			if c.blend {
				continue // the rounded corner is the sphere patch, not an end-face arc
			}
			if ends[c.endFace] == nil {
				ends[c.endFace] = map[uint64]corner{}
			}
			ends[c.endFace][c.vertex.ID()] = c
		}
	}
	return ab, ends
}

// transformFace rebuilds a face's loops, pulling A/B corners to their tangent points and
// expanding each end corner into a tangent-point-to-tangent-point arc. A face untouched by
// any fillet is copied unchanged.
func transformFace(f *topo.Face, subs map[uint64]math.Point3, ends map[uint64]corner) filletFace {
	ff := filletFace{surface: f.Geometry()}
	for _, l := range f.Loops() {
		ff.loops = append(ff.loops, transformLoop(f, l, subs, ends))
	}
	return ff
}

// transformLoop walks a loop's edge uses and applies the per-vertex fillet substitutions.
func transformLoop(f *topo.Face, l *topo.Loop, subs map[uint64]math.Point3, ends map[uint64]corner) filletLoop {
	uses := l.EdgeUses()
	n := len(uses)
	var fl filletLoop
	for i, u := range uses {
		v := useFromVertex(u)
		switch {
		case ends != nil && hasCorner(ends, v):
			c := ends[v.ID()]
			tIn := c.tOf(otherFace(uses[(i-1+n)%n].Edge(), f))
			tOut := c.tOf(otherFace(u.Edge(), f))
			addCornerRound(&fl, c, tIn, tOut)
		case subs != nil && hasSubst(subs, v):
			fl.add(subs[v.ID()], nil)
		default:
			fl.add(v.Point(), nil)
		}
	}
	return fl
}

// addCornerRound expands a simple end corner into its rounded boundary: one true arc for a
// constant fillet, or the chord fan (shared with the ruling strips) for a variable one.
func addCornerRound(fl *filletLoop, c corner, tIn, tOut math.Point3) {
	if len(c.chords) == 0 {
		arc, _ := geom.Arc3dByThreePoints(tIn, c.mid, tOut)
		fl.add(tIn, arc)
		fl.add(tOut, nil)
		return
	}
	for _, p := range orientedChords(c.chords, tIn) {
		fl.add(p, nil)
	}
}

// orientedChords returns the corner's chord samples starting from tIn (the chords run ta→tb;
// a loop entering from the b side walks them reversed).
func orientedChords(chords []math.Point3, tIn math.Point3) []math.Point3 {
	if chords[0].DistanceTo(tIn) < 1e-9 {
		return chords
	}
	out := make([]math.Point3, len(chords))
	for i, p := range chords {
		out[len(chords)-1-i] = p
	}
	return out
}

func hasCorner(ends map[uint64]corner, v *topo.Vertex) bool { _, ok := ends[v.ID()]; return ok }
func hasSubst(subs map[uint64]math.Point3, v *topo.Vertex) bool {
	_, ok := subs[v.ID()]
	return ok
}

// add appends a point and the curve of the segment leaving it (nil ⇒ straight).
func (l *filletLoop) add(p math.Point3, curve geom.Curve3) {
	l.pts = append(l.pts, p)
	l.curves = append(l.curves, curve)
}

// useFromVertex returns the from-vertex of an edge use (honouring reversal).
func useFromVertex(u *topo.EdgeUse) *topo.Vertex {
	if u.Reversed() {
		return u.Edge().EndVertex()
	}
	return u.Edge().StartVertex()
}

// otherFace returns the face sharing edge e that is not f.
func otherFace(e *topo.Edge, f *topo.Face) *topo.Face {
	for _, g := range e.Faces() {
		if g != f {
			return g
		}
	}
	return nil
}

// cylinderFace builds the rolling-ball cylinder face: tangent line on A, end arc, tangent
// line on B, end arc — wound so its geometric normal matches the cylinder's outward radial.
func cylinderFace(ef edgeFillet) filletFace {
	c0, c1 := ef.c0, ef.c1
	arc1, _ := geom.Arc3dByThreePoints(c1.ta, c1.mid, c1.tb) // TA1 → TB1 at end 1
	arc0, _ := geom.Arc3dByThreePoints(c0.tb, c0.mid, c0.ta) // TB0 → TA0 at end 0
	loop := filletLoop{
		pts:    []math.Point3{c0.ta, c1.ta, c1.tb, c0.tb},
		curves: []geom.Curve3{nil, arc1, nil, arc0},
	}
	if cylinderLoopFlipped(ef.cyl, loop) {
		loop = reverseFilletLoop(loop, ef)
	}
	return filletFace{surface: ef.cyl, loops: []filletLoop{loop}}
}

// cylinderLoopFlipped reports whether the loop winds against the cylinder's outward normal at
// the first end arc's midpoint (so it should be reversed for a consistent, outward face).
func cylinderLoopFlipped(cyl geom.Cylinder, loop filletLoop) bool {
	a, b, c := loop.pts[0], loop.pts[1], loop.pts[2]
	n := a.VectorTo(b).Cross(a.VectorTo(c))
	u, v := cyl.ParamAt(centroidPts(loop.pts))
	return n.Dot(cyl.NormalAt(u, v)) < 0
}

// reverseFilletLoop reverses the cylinder loop (and rebuilds its arcs in the new direction).
func reverseFilletLoop(_ filletLoop, ef edgeFillet) filletLoop {
	c0, c1 := ef.c0, ef.c1
	arc0, _ := geom.Arc3dByThreePoints(c0.ta, c0.mid, c0.tb) // TA0 → TB0
	arc1, _ := geom.Arc3dByThreePoints(c1.tb, c1.mid, c1.ta) // TB1 → TA1
	return filletLoop{
		pts:    []math.Point3{c0.ta, c0.tb, c1.tb, c1.ta},
		curves: []geom.Curve3{arc0, nil, arc1, nil},
	}
}

// spherePatchFace builds the corner sphere patch: a spherical triangle bounded by the blend's
// three arcs (each shared with a cylinder fillet), wound so its normal points outward (away
// from the sphere centre).
func spherePatchFace(cb *cornerBlend) filletFace {
	loop := chainArcs(cb.arcs)
	if spherePatchFlipped(cb, loop) {
		loop = reverseArcLoop(loop)
	}
	return filletFace{surface: cb.sphere, loops: []filletLoop{loop}}
}

// chainArcs links the blend's arcs head-to-tail into a closed loop (each tangent point is an
// endpoint of exactly two arcs), with the arc curve on each segment.
func chainArcs(arcs []blendArc) filletLoop {
	used := make([]bool, len(arcs))
	var fl filletLoop
	cur := arcs[0].ta
	for range arcs {
		for j, a := range arcs {
			if used[j] {
				continue
			}
			from, to, ok := arcEndpoints(a, cur)
			if !ok {
				continue
			}
			used[j] = true
			arc, _ := geom.Arc3dByThreePoints(from, a.mid, to)
			fl.pts = append(fl.pts, from)
			fl.curves = append(fl.curves, arc)
			cur = to
			break
		}
	}
	return fl
}

// arcEndpoints orients an arc so it starts at cur (returning from=cur, to=other), or ok=false
// when neither end is cur.
func arcEndpoints(a blendArc, cur math.Point3) (from, to math.Point3, ok bool) {
	switch {
	case a.ta.DistanceTo(cur) < 1e-7:
		return a.ta, a.tb, true
	case a.tb.DistanceTo(cur) < 1e-7:
		return a.tb, a.ta, true
	}
	return math.Point3{}, math.Point3{}, false
}

// spherePatchFlipped reports whether the loop winds against the sphere's outward normal at the
// patch centroid (so it should be reversed to face outward).
func spherePatchFlipped(cb *cornerBlend, loop filletLoop) bool {
	c := centroidPts(loop.pts)
	n := loop.pts[0].VectorTo(loop.pts[1]).Cross(loop.pts[0].VectorTo(loop.pts[2]))
	return n.Dot(cb.center.VectorTo(c)) < 0
}

// reverseArcLoop reverses a closed arc loop, re-deriving each segment's arc in the new
// direction (the arc midpoints are recovered from the original arcs).
func reverseArcLoop(loop filletLoop) filletLoop {
	n := len(loop.pts)
	mids := arcMidpoints(loop)
	var out filletLoop
	for i := 0; i < n; i++ {
		from := loop.pts[(n-i)%n]
		to := loop.pts[(n-i-1+n)%n]
		arc, _ := geom.Arc3dByThreePoints(from, mids[(n-i-1+n)%n], to)
		out.pts = append(out.pts, from)
		out.curves = append(out.curves, arc)
	}
	return out
}

// arcMidpoints samples each segment's arc curve at its midpoint (for re-orienting the loop).
func arcMidpoints(loop filletLoop) []math.Point3 {
	mids := make([]math.Point3, len(loop.curves))
	for i, c := range loop.curves {
		if c != nil {
			mids[i] = c.PointAt(0.5)
		}
	}
	return mids
}

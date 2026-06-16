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
// line) per quad. When one end is a RUN-OUT (radius 0, all its chords are the apex vertex), the
// strips collapse to a triangle fan to that apex so the cone closes manifold (not degenerate quads).
func rulingStripFaces(ef edgeFillet) []filletFace {
	cmid := ef.c0.cen.Midpoint(ef.c1.cen)
	out := make([]filletFace, 0, len(ef.c0.chords)-1)
	for j := 0; j+1 < len(ef.c0.chords); j++ {
		switch {
		case ef.c1.runout: // fan from corner-0's arc to corner-1's apex
			out = append(out, planarTriangleFace(ef.c0.chords[j], ef.c1.cen, ef.c0.chords[j+1], cmid))
		case ef.c0.runout: // fan from corner-1's arc to corner-0's apex
			out = append(out, planarTriangleFace(ef.c1.chords[j], ef.c0.cen, ef.c1.chords[j+1], cmid))
		default:
			quad := [4]math.Point3{ef.c0.chords[j], ef.c1.chords[j], ef.c1.chords[j+1], ef.c0.chords[j+1]}
			out = append(out, planarQuadFace(quad, cmid))
		}
	}
	return out
}

// planarTriangleFace builds one planar triangle, wound so its normal points away from cmid.
func planarTriangleFace(p0, p1, p2, cmid math.Point3) filletFace {
	n := p0.VectorTo(p1).Cross(p0.VectorTo(p2))
	tri := [3]math.Point3{p0, p1, p2}
	if n.Dot(cmid.VectorTo(centroidPts(tri[:]))) < 0 {
		tri = [3]math.Point3{p0, p2, p1}
	}
	pl, _ := geom.NewPlane(tri[0], p0.VectorTo(tri[1]).Cross(p0.VectorTo(tri[2])))
	return filletFace{surface: pl, loops: []filletLoop{{pts: tri[:], curves: make([]geom.Curve3, 3)}}}
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
			if c.blend || c.miter || c.runout {
				continue // the rounded corner is a sphere patch / miter seam / run-out apex, not an end-face arc
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

// cylinderFace builds the rolling-ball cylinder face: tangent line on A, the rounded end at
// corner 1, tangent line on B, the rounded end at corner 0. Each end is a single arc for a
// simple/blend corner, or the seam chords for a miter corner. The loop is wound so its normal
// matches the cylinder's outward radial.
func cylinderFace(ef edgeFillet) filletFace {
	segs := []endSeg{{from: ef.c0.ta, to: ef.c1.ta}}             // A-tangent line c0.ta → c1.ta
	segs = append(segs, cornerEndSegs(ef.c1)...)                 // rounded end 1: c1.ta → c1.tb
	segs = append(segs, endSeg{from: ef.c1.tb, to: ef.c0.tb})    // B-tangent line c1.tb → c0.tb
	segs = append(segs, reverseEndSegs(cornerEndSegs(ef.c0))...) // rounded end 0: c0.tb → c0.ta
	if cylinderSegsFlipped(ef, segs) {
		segs = reverseEndSegs(segs)
	}
	return filletFace{surface: ef.cyl, loops: []filletLoop{loopFromSegs(segs)}}
}

// endSeg is one boundary segment from→to of a cylinder loop, with the arc curve (and its
// midpoint, for re-deriving the arc when the loop is reversed) when the end is rounded by an
// arc; a miter seam's chords are straight (curve nil, arc false).
type endSeg struct {
	from, to math.Point3
	curve    geom.Curve3
	mid      math.Point3
	arc      bool
}

// cornerEndSegs returns the segments rounding corner c from its ta to its tb: the seam chords
// for a miter corner, otherwise a single arc through the corner's mid.
func cornerEndSegs(c corner) []endSeg {
	if c.miter {
		segs := make([]endSeg, 0, len(c.seam)-1)
		for i := 0; i+1 < len(c.seam); i++ {
			segs = append(segs, endSeg{from: c.seam[i], to: c.seam[i+1]})
		}
		return segs
	}
	arc, _ := geom.Arc3dByThreePoints(c.ta, c.mid, c.tb)
	return []endSeg{{from: c.ta, to: c.tb, curve: arc, mid: c.mid, arc: true}}
}

// reverseEndSegs reverses a chain of segments (swapping each from/to and re-deriving any arc in
// the new direction), so a forward end-rounding ta→tb can run tb→ta.
func reverseEndSegs(segs []endSeg) []endSeg {
	out := make([]endSeg, len(segs))
	for i, s := range segs {
		r := endSeg{from: s.to, to: s.from, mid: s.mid, arc: s.arc}
		if s.arc {
			r.curve, _ = geom.Arc3dByThreePoints(s.to, s.mid, s.from)
		}
		out[len(segs)-1-i] = r
	}
	return out
}

// loopFromSegs flattens a closed chain of segments (each seg's to is the next seg's from) into a
// filletLoop: each point with the curve of the segment leaving it.
func loopFromSegs(segs []endSeg) filletLoop {
	var fl filletLoop
	for _, s := range segs {
		fl.pts = append(fl.pts, s.from)
		fl.curves = append(fl.curves, s.curve)
	}
	return fl
}

// cylinderSegsFlipped reports whether the loop winds against the cylinder's outward normal. It
// builds the test triangle from the four tangent corners (always well separated, unlike a
// miter seam's near-collinear chords), so the sign is robust for both arc and seam ends.
func cylinderSegsFlipped(ef edgeFillet, segs []endSeg) bool {
	a, b, c := ef.c0.ta, ef.c1.ta, ef.c1.tb
	n := a.VectorTo(b).Cross(a.VectorTo(c))
	u, v := ef.cyl.ParamAt(centroidPts(loopFromSegs(segs).pts))
	return n.Dot(ef.cyl.NormalAt(u, v)) < 0
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
			appendBlendArc(&fl, a, from, to)
			cur = to
			break
		}
	}
	return fl
}

// appendBlendArc appends one blend arc oriented from→to: a faceted chord polyline (straight segments)
// when the arc is shared with a variable cone, otherwise a single Arc3d through its midpoint. Only the
// segment's start points are added (its end is the next arc's start), so the loop stays closed.
func appendBlendArc(fl *filletLoop, a blendArc, from, to math.Point3) {
	if a.chords == nil {
		arc, _ := geom.Arc3dByThreePoints(from, a.mid, to)
		fl.pts = append(fl.pts, from)
		fl.curves = append(fl.curves, arc)
		return
	}
	pts := orientedChords(a.chords, from) // ta…tb, reversed when entering from tb
	for i := 0; i+1 < len(pts); i++ {
		fl.pts = append(fl.pts, pts[i])
		fl.curves = append(fl.curves, nil) // straight chord matching the cone's ruling end
	}
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

// reverseArcLoop reverses a closed arc loop, re-deriving each segment's arc in the new direction
// (the arc midpoints are recovered from the original arcs). Straight chord segments (nil curve, from a
// faceted variable-cone arc) stay straight rather than being mis-fitted to an arc through the origin.
func reverseArcLoop(loop filletLoop) filletLoop {
	n := len(loop.pts)
	mids := arcMidpoints(loop)
	var out filletLoop
	for i := 0; i < n; i++ {
		from := loop.pts[(n-i)%n]
		to := loop.pts[(n-i-1+n)%n]
		src := (n - i - 1 + n) % n
		var curve geom.Curve3
		if loop.curves[src] != nil {
			curve, _ = geom.Arc3dByThreePoints(from, mids[src], to)
		}
		out.pts = append(out.pts, from)
		out.curves = append(out.curves, curve)
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

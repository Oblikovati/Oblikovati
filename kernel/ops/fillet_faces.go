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
	abSubst, endCorner, edgeInserts := filletMaps(fils)
	var out []filletFace
	for _, f := range body.Faces() {
		out = append(out, transformFace(f, abSubst[f], endCorner[f], edgeInserts[f]))
	}
	for _, ef := range fils {
		switch {
		case ef.varying && ef.exact:
			out = append(out, ruledBlendFaces(ef)...)
		case ef.varying:
			out = append(out, rulingStripFaces(ef)...)
		default:
			out = append(out, cylinderFace(ef))
		}
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
	profiles := append([]corner{ef.c0}, ef.mids...) // c0, intermediate profiles (#695), c1
	profiles = append(profiles, ef.c1)
	var out []filletFace
	for i := 0; i+1 < len(profiles); i++ {
		out = append(out, ruleStripBetween(profiles[i], profiles[i+1], cmid)...)
	}
	return out
}

// ruleStripBetween rules one strip of planar faces between two corner profiles' chord arrays. Only an
// end corner can be a run-out (collapse to an apex); intermediate profiles are always full arcs.
func ruleStripBetween(a, b corner, cmid math.Point3) []filletFace {
	out := make([]filletFace, 0, len(a.chords))
	for j := 0; j+1 < len(a.chords) && j+1 < len(b.chords); j++ {
		switch {
		case b.runout: // fan from a's arc to b's apex
			out = append(out, planarFaceFromRing([]math.Point3{a.chords[j], b.cen, a.chords[j+1]}, cmid))
		case a.runout: // fan from b's arc to a's apex
			out = append(out, planarFaceFromRing([]math.Point3{b.chords[j], a.cen, b.chords[j+1]}, cmid))
		default:
			out = append(out, planarFaceFromRing([]math.Point3{a.chords[j], b.chords[j], b.chords[j+1], a.chords[j+1]}, cmid))
		}
	}
	return out
}

// planarFaceFromRing builds one planar face over a closed point ring (a quad strip or a triangle fan
// of the cone), wound so its normal points away from the blend centre line (cmid; the wedge spans < π,
// so the sign is robust).
func planarFaceFromRing(ring []math.Point3, cmid math.Point3) filletFace {
	n := ring[0].VectorTo(ring[1]).Cross(ring[0].VectorTo(ring[len(ring)-1]))
	if n.Dot(cmid.VectorTo(centroidPts(ring))) < 0 {
		ring = reversedRing(ring)
		n = n.Scale(-1)
	}
	pl, _ := geom.NewPlane(ring[0], n)
	return filletFace{surface: pl, loops: []filletLoop{{pts: ring, curves: make([]geom.Curve3, len(ring))}}}
}

// reversedRing reverses a closed point ring while keeping its first point first (so the winding flips
// but the loop still starts at ring[0]): [p0,p1,…,pn] → [p0,pn,…,p1].
func reversedRing(ring []math.Point3) []math.Point3 {
	out := make([]math.Point3, len(ring))
	out[0] = ring[0]
	for i := 1; i < len(ring); i++ {
		out[i] = ring[len(ring)-i]
	}
	return out
}

// filletMaps indexes, per face: the corner-vertex → tangent-point pullbacks (where the face
// is a fillet's A or B face — every corner, simple or blended), and the corner-vertex →
// corner arc (only for SIMPLE end corners; a blend corner's arc lives on the sphere patch,
// and all its faces just pull back to the sphere tangent points).
func filletMaps(fils []edgeFillet) (map[*topo.Face]map[uint64]math.Point3, map[*topo.Face]map[uint64]corner, map[*topo.Face]map[uint64][]math.Point3) {
	ab := map[*topo.Face]map[uint64]math.Point3{}
	ends := map[*topo.Face]map[uint64]corner{}
	inserts := map[*topo.Face]map[uint64][]math.Point3{}
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
		putEdgeInserts(inserts, ef) // variable fillets split the A/B tangent lines at each mid station (#695)
	}
	return ab, ends, inserts
}

// putEdgeInserts records, for a variable fillet, the intermediate tangent points that must be inserted
// into the two adjacent faces' filleted-edge boundary so they weld to the subdivided ruling strips: the
// A face gets each mid profile's ta, the B face each mid's tb, ordered from the edge's start vertex (#695).
func putEdgeInserts(inserts map[*topo.Face]map[uint64][]math.Point3, ef edgeFillet) {
	if len(ef.mids) == 0 {
		return
	}
	eid := ef.edge.ID()
	ta := make([]math.Point3, len(ef.mids))
	tb := make([]math.Point3, len(ef.mids))
	for i, m := range ef.mids {
		ta[i], tb[i] = m.ta, m.tb
	}
	for f, pts := range map[*topo.Face][]math.Point3{ef.a: ta, ef.b: tb} {
		if inserts[f] == nil {
			inserts[f] = map[uint64][]math.Point3{}
		}
		inserts[f][eid] = pts
	}
}

// transformFace rebuilds a face's loops, pulling A/B corners to their tangent points and
// expanding each end corner into a tangent-point-to-tangent-point arc. A face untouched by
// any fillet is copied unchanged.
func transformFace(f *topo.Face, subs map[uint64]math.Point3, ends map[uint64]corner, inserts map[uint64][]math.Point3) filletFace {
	ff := filletFace{surface: f.Geometry(), parent: f.Lineage()} // provenance: the original face (ADR-0043)
	for _, l := range f.Loops() {
		ff.loops = append(ff.loops, transformLoop(f, l, subs, ends, inserts))
	}
	return ff
}

// transformLoop walks a loop's edge uses and applies the per-vertex fillet substitutions, then
// subdivides the filleted edge at any intermediate tangent points (variable fillets, #695).
func transformLoop(f *topo.Face, l *topo.Loop, subs map[uint64]math.Point3, ends map[uint64]corner, inserts map[uint64][]math.Point3) filletLoop {
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
		addEdgeInserts(&fl, inserts, u)
	}
	return fl
}

// addEdgeInserts appends the mid tangent points along edge use u (oriented to the traversal direction),
// subdividing a variable fillet's tangent line so the adjacent face welds to the ruling strips (#695).
func addEdgeInserts(fl *filletLoop, inserts map[uint64][]math.Point3, u *topo.EdgeUse) {
	if inserts == nil {
		return
	}
	pts, ok := inserts[u.Edge().ID()]
	if !ok {
		return
	}
	for _, p := range orientedInserts(pts, u.Reversed()) {
		fl.add(p, nil)
	}
}

// orientedInserts returns the start→end-ordered insert points, reversed when the edge is traversed
// from its end vertex.
func orientedInserts(pts []math.Point3, reversed bool) []math.Point3 {
	if !reversed {
		return pts
	}
	out := make([]math.Point3, len(pts))
	for i, p := range pts {
		out[len(pts)-1-i] = p
	}
	return out
}

// addCornerRound expands a simple end corner into its rounded boundary: one true arc for a
// constant fillet, or the chord fan (shared with the ruling strips) for a variable one.
func addCornerRound(fl *filletLoop, c corner, tIn, tOut math.Point3) {
	if len(c.chords) == 0 {
		fl.add(tIn, cornerSectionCurve(c, tIn, tOut))
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
	// A concave fillet's surface faces the cylinder CENTRE (material is on the far side), so its loop
	// must wind against the cylinder's outward radial — invert the convex sense.
	if cylinderSegsFlipped(ef, segs) != ef.flip {
		segs = reverseEndSegs(segs)
	}
	return filletFace{surface: ef.cyl, loops: []filletLoop{loopFromSegs(segs)}, parent: filletEdgeProvenance(ef.edge)}
}

// filletEdgeProvenance is the provenance lineage of a blend face generated by a filleted edge: the
// edge's own lineage with a "fillet:cyl" marker (ADR-0043). It derives the blend's name — and,
// through it, the names of the tangent edges where the blend meets the adjacent faces — from the
// stable identity of the filleted edge, not a build-order counter.
func filletEdgeProvenance(e *topo.Edge) topo.Lineage {
	return topo.NewLineage(append(e.Lineage().Tokens(), topo.Tok("fillet", "cyl", 0))...)
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

// cornerSectionCurve is the exact cross-section trim tIn→tOut at a corner: the circular arc for
// an arc profile, or the rational conic (shoulder weight crossW) for a conic profile — the same
// geometry as the blend surface's end isoline, so the seam welds curve-for-curve (#1606).
func cornerSectionCurve(c corner, tIn, tOut math.Point3) geom.Curve3 {
	if c.crossW > 0 {
		if conic, err := geom.NewConicSectionCurve(tIn, c.sh, tOut, c.crossW); err == nil {
			return conic
		}
	}
	arc, _ := geom.Arc3dByThreePoints(tIn, c.mid, tOut)
	return arc
}

// ruledBlendFaces emits a variable (or conic) fillet's blend as EXACT rational ruled faces —
// one degree (2,1) NURBS span between each pair of adjacent cross-section profiles — instead of
// the C0 chord strips (#1606, audit A10). Each span's boundary reuses the profiles' exact
// section curves and the straight tangent rulings, so adjacent faces and end caps weld exactly.
func ruledBlendFaces(ef edgeFillet) []filletFace {
	profiles := append([]corner{ef.c0}, ef.mids...)
	profiles = append(profiles, ef.c1)
	out := make([]filletFace, 0, len(profiles)-1)
	for i := 0; i+1 < len(profiles); i++ {
		out = append(out, ruledBlendSpan(ef, profiles[i], profiles[i+1], i))
	}
	return out
}

// ruledBlendSpan builds one exact blend span between profiles a and b, falling back to the
// planar strip for the span only if the surface constructor rejects it (degenerate frame).
func ruledBlendSpan(ef edgeFillet, a, b corner, i int) filletFace {
	surf, err := geom.NewRuledSectionBlend(
		[3]math.Point3{a.ta, a.sh, a.tb}, [3]math.Point3{b.ta, b.sh, b.tb}, ef.secW)
	if err != nil {
		return planarFaceFromRing([]math.Point3{a.ta, b.ta, b.tb, a.tb}, a.cen.Midpoint(b.cen))
	}
	segs := blendSpanSegs(a, b)
	if blendSegsFlipped(surf, a, b) != ef.flip {
		segs = reverseEndSegs(segs)
	}
	ff := filletFace{surface: surf, loops: []filletLoop{loopFromSegs(segs)}, parent: blendSpanProvenance(ef.edge, i)}
	return ff
}

// blendSpanSegs is the span's boundary chain: A-tangent ruling, profile section at b, B-tangent
// ruling back, profile section at a reversed. A run-out profile contributes no section (its
// three points coincide at the apex), so the chain closes through the apex vertex.
func blendSpanSegs(a, b corner) []endSeg {
	segs := []endSeg{{from: a.ta, to: b.ta}}
	if !b.runout {
		segs = append(segs, endSeg{from: b.ta, to: b.tb, curve: cornerSectionCurve(b, b.ta, b.tb), mid: b.mid, arc: true})
	}
	segs = append(segs, endSeg{from: b.tb, to: a.tb})
	if !a.runout {
		segs = append(segs, endSeg{from: a.tb, to: a.ta, curve: cornerSectionCurve(a, a.tb, a.ta), mid: a.mid, arc: true})
	}
	return segs
}

// blendSegsFlipped reports whether the natural loop winding (as emitted by blendSpanSegs) runs
// against the blend's OUTWARD normal — the winding probe cylinderSegsFlipped performs for the
// constant blend, evaluated on the exact surface: outward is the surface normal at the span
// centre oriented away from the rolling-ball centre line, and the loop's winding is its Newell
// normal over the boundary corners (the wedge spans < π, so both signs are robust).
func blendSegsFlipped(surf geom.BSplineSurface, a, b corner) bool {
	n := surf.NormalAt(0.5, 0.5)
	if a.cen.Midpoint(b.cen).VectorTo(surf.PointAt(0.5, 0.5)).Dot(n) < 0 {
		n = n.Scale(-1) // outward: away from the ball centre line
	}
	ring := []math.Point3{a.ta, b.ta}
	if !b.runout {
		ring = append(ring, b.mid, b.tb)
	}
	ring = append(ring, a.tb)
	if !a.runout {
		ring = append(ring, a.mid)
	}
	return newellNormal(ring).Dot(n) < 0
}

// newellNormal is the Newell winding normal of a 3D point ring.
func newellNormal(ring []math.Point3) math.Vector3 {
	var nx, ny, nz float64
	for i := range ring {
		c, d := ring[i], ring[(i+1)%len(ring)]
		nx += float64((c.Y - d.Y) * (c.Z + d.Z))
		ny += float64((c.Z - d.Z) * (c.X + d.X))
		nz += float64((c.X - d.X) * (c.Y + d.Y))
	}
	return math.V3(math.Scalar(nx), math.Scalar(ny), math.Scalar(nz))
}

// blendSpanProvenance names an exact blend span by its generating edge and span ordinal
// (ADR-0043), mirroring filletEdgeProvenance.
func blendSpanProvenance(e *topo.Edge, i int) topo.Lineage {
	return topo.NewLineage(append(e.Lineage().Tokens(), topo.Tok("fillet", "blend", i))...)
}

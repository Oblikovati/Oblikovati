// SPDX-License-Identifier: GPL-2.0-only

package blend

import (
	"oblikovati.org/kernel/geom"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
)

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

// faceFilletInputs bundles the per-face lookups transformFace applies to ONE face — the A/B
// corner-vertex substitutions, the simple end corners, the edge inserts and their leaving-curve
// chains, and the spread cap pieces — plus subArcScale, so a call site reads as one projection of
// the shared filletRebuildMaps (forFace) instead of six aligned map indexings. The zero value
// means "face untouched by any fillet" (the canal/far-runout pass-through callers).
type faceFilletInputs struct {
	subs         map[uint64]math.Point3
	ends         map[uint64]corner
	inserts      map[uint64][]math.Point3
	insertCurves map[uint64][]geom.Curve3
	spread       map[uint64]facePiece
	subArcScale  float64
}

// forFace projects the shared rebuild maps onto one face's transform inputs.
// subArcScale is the model scale (body bounding-box diagonal) the subs-branch survivor-arc carry gates on;
// 0 DISABLES the carry (the specialized obstacle/runout/canal rebuild callers pass 0, staying byte-identical
// to the pre-carry planar path — only the main transformedBodyFaces path activates the I3 carry).
func (m filletRebuildMaps) forFace(f *topo.Face, subArcScale float64) faceFilletInputs {
	return faceFilletInputs{subs: m.abSubst[f], ends: m.endCorner[f], inserts: m.edgeInserts[f],
		insertCurves: m.insertCurves[f], spread: m.spreads[f], subArcScale: subArcScale}
}

// transformFace rebuilds a face's loops, pulling A/B corners to their tangent points and
// expanding each end corner into a tangent-point-to-tangent-point arc. A face untouched by
// any fillet is copied unchanged.
func transformFace(f *topo.Face, in faceFilletInputs) filletFace {
	ff := filletFace{surface: f.Geometry(), parent: f.Lineage()} // provenance: the original face (ADR-0043)
	for _, l := range f.Loops() {
		ff.loops = append(ff.loops, transformLoop(f, l, in))
	}
	return ff
}

// transformLoop walks a loop's edge uses and applies the per-vertex fillet substitutions, then
// subdivides the filleted edge at any intermediate tangent points (variable fillets, #695).
func transformLoop(f *topo.Face, l *topo.Loop, in faceFilletInputs) filletLoop {
	uses := l.EdgeUses()
	n := len(uses)
	var fl filletLoop
	var rimCarries, subCarries []int // segments carrying a curved survivor's parent arc, trimmed to the retained sub-arc post-loop
	for i, u := range uses {
		v := useFromVertex(u)
		switch {
		case in.spread != nil && hasFacePiece(in.spread, v):
			// A valence>3 runout apex: this far face carries one arc piece of the spread cap,
			// oriented to the loop's traversal by the edges arriving at / leaving the apex.
			addRunoutApex(&fl, in.spread[v.ID()], uses[(i-1+n)%n].Edge().ID(), u.Edge().ID())
		case in.ends != nil && hasCorner(in.ends, v):
			if idx := addEndCorner(&fl, f, in.ends, uses, i); idx >= 0 {
				rimCarries = append(rimCarries, idx) // tOut's leaving segment is a curved rim: trim it post-loop
			}
		case in.subs != nil && hasSubst(in.subs, v):
			if idx := addSubstVertex(&fl, in.subs[v.ID()], u); idx >= 0 {
				subCarries = append(subCarries, idx) // the tangent point's leaving edge is a curved survivor rim (I3)
			}
		default:
			// unchanged survivor: carry its vertex id AND the edge leaving it, so a coincident
			// tangent seam (two edges on one line sharing endpoints) stays two edges (#1600).
			// Preserve a CURVED survivor edge's geometry: both faces sharing it are transformed,
			// so if both dropped the curve (nil) the shared edge would collapse to a straight
			// LineSegment (ec.use), bulging a planar face that borders a cylinder — the Q1 area
			// defect. A straight edge stays nil (a LineSegment curve is identical to nil there).
			fl.addID(v.Point(), ringOrSurvivorCurve(u, in.inserts, in.insertCurves), v.ID(), u.Edge().ID())
		}
		addEdgeInserts(&fl, in.inserts, in.insertCurves, u)
	}
	trimCarriedArcs(&fl, rimCarries, subCarries, in.subArcScale) // both survivor-arc branches: parent arc → the retained sub-arc
	return fl
}

// ringOrSurvivorCurve returns the curve of the segment LEAVING a survivor vertex. When the edge's
// subdivided footprint rim carries a leaving-curve chain (subdivideBossWall's insertCurves), the seam
// takes its own chain entry — the exact sub-span of the footprint conic, oriented to the traversal —
// so the intact wall and (through the edge catalog's value agreement) the re-clipped host bound the
// TRUE rim. Without a chain it falls through to subdividedSurvivorCurve's nil/carry rule unchanged.
func ringOrSurvivorCurve(u *topo.EdgeUse, inserts map[uint64][]math.Point3, insertCurves map[uint64][]geom.Curve3) geom.Curve3 {
	if chain, ok := insertCurves[u.Edge().ID()]; ok {
		return ringLeavingCurve(chain, 0, u.Reversed())
	}
	return subdividedSurvivorCurve(u, inserts)
}

// ringLeavingCurve returns the leaving curve of the k-th VISITED point of a subdivided rim ring under
// the use's traversal direction: forward, chain[k]; reversed, the reverse of the curve that ARRIVES at
// that point in forward order (rev(chain[(n-1-k) mod n]) — the one-slot shift orientedInserts' point
// reversal implies). nil chain entries (band chords) stay nil.
func ringLeavingCurve(chain []geom.Curve3, k int, rev bool) geom.Curve3 {
	n := len(chain)
	if n == 0 {
		return nil
	}
	if !rev {
		return chain[k%n]
	}
	if c := chain[((n-1-k)%n+n)%n]; c != nil {
		return geom.ReverseCurve3(c)
	}
	return nil
}

// subdividedSurvivorCurve returns the survivor's carried curve, EXCEPT it drops a CLOSED-conic rim edge
// (start vertex == end vertex, e.g. an intact runout boss wall's footprint circle) to nil when that edge
// has inserts: the inserts (Task 4, subdivideBossWall) re-trace the rim as straight chords that weld to
// the setback patches/re-clipped host, so carrying the full-circle curve on the first chord would make
// that one edge tessellate the WHOLE circle and self-cross the loop. (When the rim carries a leaving-
// curve chain, ringOrSurvivorCurve takes the exact sub-span instead and never reaches here.)
// Corpus-neutral: variable-fillet inserts (the only inserts today) live on OPEN straight tangent edges,
// so this branch never fires there.
func subdividedSurvivorCurve(u *topo.EdgeUse, inserts map[uint64][]math.Point3) geom.Curve3 {
	if inserts != nil && u.Edge().StartVertex() == u.Edge().EndVertex() {
		if _, ok := inserts[u.Edge().ID()]; ok {
			return nil
		}
	}
	return survivorCurve(u)
}

// survivorCurve returns a survivor edge's curve, oriented to the loop's TRAVERSAL direction, to carry
// into the rebuilt loop. Both faces sharing a curved survivor edge are transformed, so if both dropped
// it (nil) the shared edge collapsed to a straight LineSegment (ec.use), inflating a planar face that
// borders a cylinder — the Q1 defect. The edge catalog builds the edge from whichever face provides the
// curve first, keyed on the loop's (from→to) vertices, so an open arc MUST run from→to: a reversed use
// needs the reversed arc, else the two symmetric end caps come out different (one right, one bulged).
//
// A closed conic rim edge (a full-circle or full-ellipse seam, e.g. the top/bottom rim of an imported
// oblique elliptical cylinder — SURFACE_OF_LINEAR_EXTRUSION) welds both endpoints to ONE vertex. Its
// type is not Arc3d, so the old code dropped it to nil and the coincident-endpoint rim collapsed to a
// zero-length stub, degenerating the wall + cap faces (T6/T7/U4 "inconsistent orientation" / open
// shell). It is carried through here UNCHANGED: the edge catalog recognises it as a closed seam
// (isClosedSeam) and forces the second co-edge's parity, and the periodic curved-face mesher rebuilds
// from the surface (u,v) rather than reading the rim's intrinsic direction — so no reversal is needed.
// Straight edges stay nil (a LineSegment curve is identical to nil in ec.use), a no-op for planar faces.
func survivorCurve(u *topo.EdgeUse) geom.Curve3 {
	switch c := u.Edge().Geometry().(type) {
	case geom.Arc3d:
		if u.Reversed() {
			c.StartAngle += c.SweepAngle
			c.SweepAngle = -c.SweepAngle
		}
		return c
	case geom.LineSegment:
		return nil // straight survivor: nil is identical and keeps the all-planar path a no-op
	default:
		return orientedOpenSurvivor(c, u) // a non-arc curved survivor (B-spline, ellipse rim): carry it, don't drop it
	}
}

// orientedOpenSurvivor returns a non-arc curved survivor's curve oriented to the loop's TRAVERSAL: reversed
// for a reversed use of an OPEN edge, carried unchanged for a CLOSED rim seam (both endpoints on one vertex).
//
// The Arc3d arm above has always flipped its sweep for a reversed use; this arm returned the curve as-is,
// so a REVERSED use of an open B-spline survivor handed the edge catalog a curve running end→start. The
// catalog stores it against the first registering face's from→to vertices (assemble_curved.go), and
// discretizeEdge then FORCES that polyline's ends onto those vertices — producing a boundary that leaps to
// the far end, walks back and leaps again. complex/F2 shipped two such edges (28.52 / 28.18 of endpoint
// mismatch), self-crossing two of its walls.
//
// A CLOSED rim seam must NOT be reversed: it has no distinct endpoints to orient by, the catalog forces the
// second co-edge's parity itself, and the periodic mesher rebuilds from the surface (u,v) — reversing one
// collapsed T6/T7/U4's oblique elliptical rim.
//
// A geom.EllipticalArc is deliberately left ALONE, so the elliptic paths stay byte-identical. It is the one
// non-arc family the retrim dispatches on CONCRETELY (endSegFromUse recognises geom.Arc3d and
// geom.EllipticalArc and drops anything else to a straight chord), and BOTH ways of orienting it were
// measured harmful: wrapping it cost simple/F4's elliptic-vein cap arcs their curve and DECLINED its host
// retrim outright, while flipping its own sweep in place left F4 shipping a boundary 95.8015 (rel 0.2819 of
// its diagonal) off its own EllipticalCylinder — the ellipse-aware sub-span algebra (segParam/subSeg,
// retainedEllipticRimCurve) is written for a forward span. No elliptic survivor edge on the corpus
// currently violates the orientation invariant, so this arm has nothing to repair and is left for whoever
// makes that algebra sign-agnostic.
func orientedOpenSurvivor(c geom.Curve3, u *topo.EdgeUse) geom.Curve3 {
	if !u.Reversed() || u.Edge().StartVertex() == u.Edge().EndVertex() {
		return c
	}
	if _, elliptic := c.(geom.EllipticalArc); elliptic {
		return c
	}
	return geom.ReverseCurve3(c)
}

// addEdgeInserts appends the mid tangent points along edge use u (oriented to the traversal direction),
// subdividing a variable fillet's tangent line so the adjacent face welds to the ruling strips (#695).
// A subdivided footprint rim's inserts also take their own leaving sub-arcs from the insertCurves chain
// (insert j is the ring's visited point j+1 — the seam is visited first); every other insert stays a
// straight chord (nil chain ⇒ nil curve, byte-identical to the pre-chain path).
func addEdgeInserts(fl *filletLoop, inserts map[uint64][]math.Point3, insertCurves map[uint64][]geom.Curve3, u *topo.EdgeUse) {
	if inserts == nil {
		return
	}
	pts, ok := inserts[u.Edge().ID()]
	if !ok {
		return
	}
	chain := insertCurves[u.Edge().ID()]
	for j, p := range orientedInserts(pts, u.Reversed()) {
		fl.add(p, ringLeavingCurve(chain, j+1, u.Reversed()))
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
// constant fillet, or the chord fan (shared with the ruling strips) for a variable one. outgoing is
// the survivor curve of the edge LEAVING tOut (survivorCurve(u)); when it is a curved (Arc3d) wall
// rim the parent arc is carried on the tOut segment and addCornerRound reports true so the caller can
// trim it to the retained sub-arc (curved-host-collapse-rootcause.md). A straight survivor stays nil,
// byte-identical to the pre-fix planar path, so no all-planar green (nor a fingerprint pin) shifts.
func addCornerRound(fl *filletLoop, c corner, tIn, tOut math.Point3, outgoing geom.Curve3) bool {
	if len(c.chords) == 0 {
		fl.add(tIn, cornerSectionCurve(c, tIn, tOut))
		if rim, curved := carriableRim(outgoing); curved {
			fl.add(tOut, rim) // carry the curved wall's parent rim; trimCarriedRimArcs cuts it to the retained span
			return true
		}
		fl.add(tOut, nil)
		return false
	}
	for _, p := range orientedChords(c.chords, tIn) {
		fl.add(p, nil)
	}
	return false
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

// add appends an op-generated point (no carried identity) and the curve of the segment leaving it
// (nil ⇒ straight).
func (l *filletLoop) add(p math.Point3, curve geom.Curve3) {
	l.addID(p, curve, 0, 0)
}

// addID appends a point carrying its source topo-vertex id and the source topo-edge id of the
// segment leaving it (0 = op-generated), so the re-weld preserves the boolean's tangent-contact
// vertex AND edge identity (#1600).
func (l *filletLoop) addID(p math.Point3, curve geom.Curve3, srcV, srcE uint64) {
	l.pts = append(l.pts, p)
	l.curves = append(l.curves, curve)
	l.srcV = append(l.srcV, srcV)
	l.srcE = append(l.srcE, srcE)
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

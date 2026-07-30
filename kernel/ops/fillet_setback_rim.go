// SPDX-License-Identifier: GPL-2.0-only

package ops

import (
	stdmath "math"

	"oblikovati.org/kernel/geom"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
)

// The SETBACK-CLOSE boss footprint RIM: how an intact boss wall's footprint conic is partitioned into
// host-side and band-side sub-arcs, how densely each is chorded, and how those points are recorded for
// transformFace to insert into the wall. Split out of fillet_setback_close.go (which had grown past the
// 500-line limit) — that file now owns the CLOSER (wings, patch faces, host notch detours), this one
// owns the rim those detours and the setback patches both tile. They meet at exactly one contract: the
// wall rim and every host notch read their chord counts and their sub-arcs through here, so they stay
// weld-identical point-for-point by construction.

// subdivideBossWall records, on maps.edgeInserts, the sampled footprint-rim points that transformFace
// must insert into the INTACT boss wall's footprint edge so its rim welds to the re-clipped host (host-
// side arc) and the setback patches (band-side arcs). The wall stays ONE face at full area — only its
// rim gains vertices. Each rim segment's own sub-span of the footprint conic goes to maps.insertCurves
// (the leaving-curve chain, seam entry first): a segment that keeps its exact sub-arc bounds the TRUE
// rim, where the chords this used to drop to left the re-clipped host the whole inscribed-polygon
// surplus — T3's plane read 1208.987870 against the closed form 1204.602895, and the missing 4.38498
// is EXACTLY Σ (r²/2)(θ−sinθ) over these chords (t3-plane-sliver-report.md §1). ok=false when the
// wall or a rim sub-arc is malformed.
func subdivideBossWall(maps filletRebuildMaps, boss crossingBoss, cyl geom.Cylinder, cross1, cross2 math.Point3, bandInner []math.Point3) bool {
	wall := otherFace(boss.footEdge, boss.host)
	if wall == nil {
		return false
	}
	seam := boss.footEdge.StartVertex().Point()
	ring, leaving, ok := bossRimRing(boss, cyl, seam, cross1, cross2, bandInner)
	if !ok || len(ring) < 2 {
		return false
	}
	if maps.edgeInserts[wall] == nil {
		maps.edgeInserts[wall] = map[uint64][]math.Point3{}
	}
	maps.edgeInserts[wall][boss.footEdge.ID()] = ring[1:] // transformFace adds seam (survivor); these follow
	recordRingChain(maps, wall, boss.footEdge.ID(), leaving)
	return true
}

// recordRingChain stores a subdivided footprint rim's leaving-curve chain (aligned to the ring: entry 0
// is the seam's leaving segment) for transformFace/addEdgeInserts to hang on the rim's sub-edges.
func recordRingChain(maps filletRebuildMaps, wall *topo.Face, edgeID uint64, leaving []geom.Curve3) {
	if maps.insertCurves[wall] == nil {
		maps.insertCurves[wall] = map[uint64][]geom.Curve3{}
	}
	maps.insertCurves[wall][edgeID] = leaving
}

// bossRimRing samples the full footprint rim, ordered from the wall seam all the way around: the host-side
// sub-arc seam→cross1, the band-side sub-arcs cross1→…→cross2 (shared with the patches), the host-side
// sub-arc cross2→seam. Each sub-arc is sampled open (sampleCurveNTrimmed), so consecutive sub-arcs
// concatenate without a duplicate vertex and every point matches the neighbour that tiles the same curve.
// The second return is the per-point LEAVING-curve chain: point i's segment to point i+1 (mod n) carries
// its exact sub-span of the footprint conic, so the wall's rim — and, through the edge catalog's value
// agreement, the host notch and patch rails welded to the same edges — bounds the true rim, not chords.
func bossRimRing(boss crossingBoss, cyl geom.Cylinder, seam, cross1, cross2 math.Point3, bandInner []math.Point3) ([]math.Point3, []geom.Curve3, bool) {
	subs, ok := bossRimSubArcs(boss, cyl, seam, cross1, cross2, bandInner)
	if !ok {
		return nil, nil, false
	}
	var ring []math.Point3
	var leaving []geom.Curve3
	for i, a := range subs {
		pts, curves := sampleCurveNTrimmed(a, rimSubArcChordCount(a, i, len(subs)), false)
		ring = append(ring, pts...)
		leaving = append(leaving, curves...)
	}
	ring, leaving = orientRingChainToEdge(ring, leaving, boss.footEdge)
	return ring, leaving, len(ring) > 0
}

// torusRimChordAngle is the maximum swept angle per straight chord (≈7.5°) a TORUS boss's host rim arc is
// sampled at — fine enough that its doubly-curved band (which bulges in the tube parameter) lofts within
// the T1/T4 area gate (measured: 1143.99/2825.96, exact), yet far coarser than the surface tessellation
// (so the shared host-notch boundary stays cheap). A dimensionless angle — scale-free by construction.
const torusRimChordAngle = 2 * stdmath.Pi / 48

// rimSubArcChordCount is the chord count for one footprint rim sub-arc at position i of total. The BAND
// sub-arcs (interior, 0<i<total-1) and every RULED boss's arcs stay at ringSegSamples: a cylinder/cone/
// elliptical-cylinder band lofts as flat quads that match any chord density (S1/S4/T7 byte-identical), and
// the band sub-arcs weld to the setback patches which also tile at ringSegSamples. Only the two HOST sub-
// arcs (i==0, i==total-1) of a TORUS wall densify — the host arcs weld to the flat host notch (never a
// patch), and a coarse host arc on the doubly-curved torus band over-covers the chord-to-arc segment
// (the 241.6° major arc at 6 chords lofted T1 +8.6%, m4-spike §CRITICAL). See hostArcChordCount.
func rimSubArcChordCount(arc geom.Curve3, i, total int) int {
	if i != 0 && i != total-1 {
		return ringSegSamples // band sub-arc: welds to the patches, keep the shared granularity
	}
	return hostArcChordCount(arc)
}

// hostArcChordCount is the chord count for a boss's host-side footprint arc: span-proportional,
// ≈torusRimChordAngle per chord, floored at ringSegSamples. Both the wall rim and the host notch sample
// the SAME host arc through here, so they stay weld-identical at whatever count it returns.
//
// EVERY wall kind densifies — the boss-kind gate this used to consult was retired (it had become an
// unconditional `return true`, leaving a dead branch here). The host-side arc is the LONG way round the
// footprint (240°–315° on the corpus bosses), and chording it at the coarse ringSegSamples leaves the
// re-clipped host plane with the whole inscribed-polygon deficit — Σ (r²/2)(θ−sinθ) over the chords — as
// SURPLUS face area, because the notch cuts inside the true rim. That is exactly the "host plane recedes
// too little" defect coons4-audit.md §C.4 isolated and could not attribute: S7 +10.95% (a 6-chord 315°
// sphere rim), S4 +8.87%, T7 +6.18%, S1 +6.07%. It was never a geometry error — the wall rim and the
// host notch trace the same arcs — it is the polygonal approximation of those arcs, and the fix is to
// chord them at the span-proportional angle the torus wall already used.
func hostArcChordCount(arc geom.Curve3) int {
	n := int(stdmath.Ceil(arcSweepAbs(arc) / torusRimChordAngle))
	if n < ringSegSamples {
		return ringSegSamples
	}
	return n
}

// arcSweepAbs is the absolute swept angle of a footprint sub-arc (a geom.Arc3d circle arc or a
// geom.EllipticalArc), 0 for any other curve (a straight flank chord) — the ruler the torus host-arc chord
// count is derived from.
func arcSweepAbs(c geom.Curve3) float64 {
	switch a := c.(type) {
	case geom.Arc3d:
		return stdmath.Abs(a.SweepAngle)
	case geom.EllipticalArc:
		return stdmath.Abs(a.SweepAngle)
	default:
		return 0
	}
}

// orientRingChainToEdge reverses the rim ring (keeping the seam point first) when it winds AGAINST the
// footprint edge's own native parametrization. transformFace applies the edge use's Reversed() flag to
// the inserts (orientedInserts) assuming they follow the edge's native direction; a ring built the other
// way then winds the wall loop's footprint rim the SAME sense as its top rim (a figure-8 in cylinder u),
// which assembleBody's shell-orient corrupts into a zero-curve top rim (the r8 cap→0 defect). Aligning to
// the edge's native direction keeps the two rims opposite, so the loop is a simple band. The leaving-curve
// chain is re-indexed alongside (reverseLeavingChain), so each point still leaves along its own segment.
func orientRingChainToEdge(ring []math.Point3, leaving []geom.Curve3, footEdge *topo.Edge) ([]math.Point3, []geom.Curve3) {
	fwd, ok := footEdgeNativeForward(footEdge)
	if !ok || len(ring) < 3 {
		return ring, leaving
	}
	if ring[1].DistanceTo(fwd) <= ring[len(ring)-1].DistanceTo(fwd) {
		return ring, leaving
	}
	out := append([]math.Point3{ring[0]}, reversePts(ring[1:])...)
	return out, reverseLeavingChain(leaving)
}

// reverseLeavingChain re-indexes a ring's leaving-curve chain for the reversed traversal: the point
// visited k-th then leaves along the REVERSE of the curve that ARRIVED at it in forward order,
// leaving'[k] = rev(leaving[(n-1-k) mod n]) — the same one-slot shift orientedInserts' point
// reversal implies, kept here so store-time orientation and consume-time reversal stay one rule.
func reverseLeavingChain(leaving []geom.Curve3) []geom.Curve3 {
	n := len(leaving)
	out := make([]geom.Curve3, n)
	for k := 0; k < n; k++ {
		if c := leaving[(n-1-k)%n]; c != nil {
			out[k] = geom.ReverseCurve3(c)
		}
	}
	return out
}

// footEdgeNativeForward is a point a hair (2% of the domain) past the footprint edge's seam in its own
// native parametrization — orientRingToEdge's winding reference. It covers both footprint curve kinds a
// setback boss can carry: a geom.Arc3d (circle boss) and a geom.EllipseFull (oblique elliptical-cylinder
// boss, T7). A curve kind it does not recognise leaves the ring unoriented (ok=false).
func footEdgeNativeForward(footEdge *topo.Edge) (math.Point3, bool) {
	switch g := footEdge.Geometry().(type) {
	case geom.Arc3d:
		lo, hi := g.Domain()
		return g.PointAt(lo + 0.02*(hi-lo)), true
	case geom.EllipseFull:
		lo, hi := g.Domain()
		return g.PointAt(lo + 0.02*(hi-lo)), true
	default:
		return math.Point3{}, false
	}
}

// footprintCenter returns the footprint conic's center for either boss footprint kind: a circle/arc
// (geom.Circle/geom.Arc3d via footprintConic) or an ellipse (geom.EllipseFull, the oblique elliptical-
// cylinder boss of T7). ok=false for any non-conic footprint.
func footprintCenter(edge *topo.Edge) (math.Point3, bool) {
	if e, ok := edge.Geometry().(geom.EllipseFull); ok {
		return e.Center, true
	}
	c, _, ok := footprintConic(edge)
	return c, ok
}

// bossRimSubArcs is the ordered curve chain of a boss footprint rim from the seam: host-side seam→cross1,
// the band-side chain cross1→bandInner…→cross2, host-side cross2→seam — a partition of the FULL conic
// (Σ directed span = 2π). The two host sub-arcs come from the scale-invariant σ-partition
// (partitionFootprintRim, rule (b) §D2 of m4-rim-partition-derivation.md): their minor-vs-major sense is
// DERIVED from the exact native spans of the host complement F∖band, not guessed by a local midpoint test
// (which dropped 242° of the large torus rim, m4-spike.md §CRITICAL). It is the shared source of truth for
// the wall-rim subdivision (all sub-arcs) and the host detour (the two host-side ones).
func bossRimSubArcs(boss crossingBoss, cyl geom.Cylinder, seam, cross1, cross2 math.Point3, bandInner []math.Point3) ([]geom.Curve3, bool) {
	part, ok := partitionFootprintRim(boss, cyl, seam, cross1, cross2)
	if !ok {
		return nil, false
	}
	hostA, ok0 := footprintArcBySpan(boss.footEdge, seam, cross1, part.hostA)
	band, ok1 := bandFootArcs(boss.footEdge, cross1, cross2, bandInner)
	hostB, ok2 := footprintArcBySpan(boss.footEdge, cross2, seam, part.hostB)
	if !ok0 || !ok1 || !ok2 {
		return nil, false
	}
	out := append([]geom.Curve3{hostA}, band...)
	return append(out, hostB), true
}

// bandFootArcs is the band-side sub-arc chain cross1→bandInner…→cross2 on the footprint conic, one
// geom.Arc3d per adjacent pair — the exact same footprintSubArc calls the flank/central patches tile
// (order-independent under sampleCurve3Open), so the wall rim welds to every patch footprint rail.
func bandFootArcs(footEdge *topo.Edge, cross1, cross2 math.Point3, bandInner []math.Point3) ([]geom.Curve3, bool) {
	pts := append([]math.Point3{cross1}, bandInner...)
	pts = append(pts, cross2)
	out := make([]geom.Curve3, 0, len(pts)-1)
	for i := 0; i+1 < len(pts); i++ {
		a, ok := footprintSubArc(footEdge, pts[i], pts[i+1])
		if !ok {
			return nil, false
		}
		out = append(out, a)
	}
	return out, true
}

// footEdgeward returns the fillet contact point at the footprint's own centre station AND the in-plane
// unit vector pointing toward the fillet band (centre→that contact). The host-side test is whether a
// candidate point sits BEHIND the contact line (the band boundary) — `(p−contact)·edgeward < 0` — not
// merely behind the footprint centre: the setback crossings lie ON the contact line, so the correct
// short host-side arc between two crossings dips a hair band-ward of the centre and a centre test
// mis-rejects it (r8 hostB picked the 270° major arc). ok=false for a non-conic footprint / non-planar
// host.
func footEdgeward(boss crossingBoss, cyl geom.Cylinder) (math.Point3, math.Vector3, bool) {
	center, ok0 := footprintCenter(boss.footEdge)
	plane, ok1 := boss.host.Geometry().(geom.Plane)
	if !ok0 || !ok1 {
		return math.Point3{}, math.Vector3{}, false
	}
	contact := filletContact(cyl, plane, spineParam(center, cyl))
	ev, err := math.UnitVector3FromVector(center.VectorTo(contact))
	if err != nil {
		return math.Point3{}, math.Vector3{}, false
	}
	return contact, ev.AsVector(), true
}

// footprintMajorArc is the MAJOR (>180°) footprint sub-arc from→to on a crossingBoss footprint conic,
// through the point antipodal to footprintSubArc's bisector midpoint — the piece that wraps the far side
// of the boss (reading the footprint conic from footEdge).
func footprintMajorArc(footEdge *topo.Edge, from, to math.Point3) (geom.Curve3, bool) {
	if e, ok := footEdge.Geometry().(geom.EllipseFull); ok {
		return ellipseSubArc(e, from, to, true) // oblique elliptical-cylinder boss (T7): the >180° ellipse arc
	}
	c, r, ok := footprintConic(footEdge)
	if !ok {
		return geom.Arc3d{}, false
	}
	bis := c.VectorTo(from).Add(c.VectorTo(to))
	l := bis.Length()
	if l < arcBisectorTiny*r {
		return geom.Arc3d{}, false // endpoints near-antipodal: the major/minor split is ill-defined
	}
	mid := c.TranslateBy(bis.Scale(-r / l))
	arc, err := geom.Arc3dByThreePoints(from, mid, to)
	return arc, err == nil
}

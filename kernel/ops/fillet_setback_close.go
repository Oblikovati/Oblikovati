// SPDX-License-Identifier: GPL-2.0-only

package ops

import (
	"oblikovati.org/kernel/geom"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
)

// buildSetbackFaces closes the intact-boss runout (setback-patch-derivation.md, forensics §3): every
// crossing boss wall is left INTACT (kept byte-area-preserved by transformedBodyFaces, its footprint
// rim merely SUBDIVIDED via maps.edgeInserts so the neighbours weld — never split into sub-faces, never
// emitted here), the two host planes are RE-CLIPPED to a single loop (the footprint opens into the
// fillet cut), and the two plain cyl-R wings (terminating at b.cutLo/b.cutHi) plus the resolved setback
// patches fill the freed span. It appends the wings + patches to set.extra and the re-clipped hosts to
// set.replace, and records the boss-wall footprint subdivisions on maps for transformedBodyFaces to
// apply. ok=false honest-rejects the WHOLE edge (do-no-harm baseline) — never a partial fill.
//
// UNWIRED: no caller in runoutFacesFor yet (Task 5 wires it); the corpus stays byte-identical.
func buildSetbackFaces(set *runoutSet, ef edgeFillet, b setbackBands, loops []RailLoop, res Resolution, maps filletRebuildMaps) bool {
	t, ok := resolveSetbackTiling(b, ef)
	if !ok {
		return false
	}
	if !appendSetbackWings(set, ef, t) {
		return false
	}
	if !appendSetbackPatchFaces(set, ef, loops, res) {
		return false
	}
	return reclipSetbackHosts(set, ef, t, maps)
}

// appendSetbackWings builds the two plain cyl-R wings flanking the freed span: the left wing runs from
// corner c0 to the low setback station b.cutLo, the right from b.cutHi to c1. Each cut cross-section is
// the flank patch's arm arc (armSectionArc, the SAME curve leftFlank/rightFlank tile from) sampled into
// ringSegSamples chords, so the wing and its flank patch share those vertices point-for-point (no
// T-junction). The arm-arc plane order mirrors extractSetbackPatches' leftFlank/rightFlank exactly.
func appendSetbackWings(set *runoutSet, ef edgeFillet, t setbackTiling) bool {
	leftArc, ok0 := armSectionArc(ef.cyl, t.pInner, t.pOuter, t.cutLo)
	rightArc, ok1 := armSectionArc(ef.cyl, t.pOuter, t.pInner, t.cutHi)
	if !ok0 || !ok1 {
		return false
	}
	leftCut, rightCut := wingCutAtSpine(ef, t.cutLo), wingCutAtSpine(ef, t.cutHi)
	left := buildWingFaceCut(ef, leftCut, true, sampledArcSegs(leftArc, leftCut.nodeTa, leftCut.nodeTb))
	right := buildWingFaceCut(ef, rightCut, false, sampledArcSegs(rightArc, rightCut.nodeTa, rightCut.nodeTb))
	set.extra = append(set.extra, left, right)
	return true
}

// appendSetbackPatchFaces resolves each setback RailLoop (extractSetbackPatches' flank/central loops)
// into a certified corner-blend patch and appends it as a filletFace. Its boundary is sampled from the
// loop's own side curves (railLoopToFilletLoops → sampleCurve3Open), the SAME footprint sub-arcs the
// re-clipped hosts and the subdivided boss walls tile, so all three weld. ok=false when any loop fails
// to resolve (honest-reject the whole edge).
func appendSetbackPatchFaces(set *runoutSet, ef edgeFillet, loops []RailLoop, res Resolution) bool {
	parent := filletEdgeProvenance(ef.edge)
	for _, loop := range loops {
		patch, ok := resolveBlend(loop, res)
		if !ok {
			return false
		}
		set.extra = append(set.extra, patchToFilletFace(patch, parent))
	}
	return true
}

// reclipSetbackHosts re-clips both host planes to a single loop and subdivides both boss-wall footprint
// rims (into maps, for transformedBodyFaces). The OUTER boss's host is the simple case (the footprint
// opens directly into the cut); the INNER boss's host also carries the two flank plain-contact seams.
func reclipSetbackHosts(set *runoutSet, ef edgeFillet, t setbackTiling, maps filletRebuildMaps) bool {
	if !reclipOuterHost(set, ef, t, maps) {
		return false
	}
	return reclipInnerHost(set, ef, t, maps)
}

// reclipOuterHost re-clips the outer boss's host plane and subdivides its wall footprint. The outer
// footprint band-side (aCutLo→aSeamLo→aSeamHi→aCutHi) is owned by the three patches; the host detour
// only re-traces the host-side arc (aCutHi→seam→aCutLo) into the outer loop, dropping the footprint hole.
func reclipOuterHost(set *runoutSet, ef edgeFillet, t setbackTiling, maps filletRebuildMaps) bool {
	if !subdivideBossWall(maps, t.outer, ef.cyl, t.aCutLo, t.aCutHi, []math.Point3{t.aSeamLo, t.aSeamHi}) {
		return false
	}
	hostIsA := t.outer.host == ef.a
	tanA, tanB := hostTangent(ef.c0, hostIsA), hostTangent(ef.c1, hostIsA)
	notch, ok := buildHostNotch(t.outer.host, maps, tanA, tanB, outerHostDetour(t.outer, ef.cyl, t))
	if !ok {
		return false
	}
	set.replace[t.outer.host.ID()] = notch
	return true
}

// reclipInnerHost re-clips the inner boss's host plane and subdivides its wall footprint. The inner
// footprint band-side (bSeamLo→bSeamHi) is owned by the central patch; the host detour re-traces the two
// flank plain-contact seams (bCut→bSeam, sampled to match the patch) and the host-side arc between them.
func reclipInnerHost(set *runoutSet, ef edgeFillet, t setbackTiling, maps filletRebuildMaps) bool {
	if !subdivideBossWall(maps, t.inner, ef.cyl, t.bSeamLo, t.bSeamHi, nil) {
		return false
	}
	hostIsA := t.inner.host == ef.a
	tanA, tanB := hostTangent(ef.c0, hostIsA), hostTangent(ef.c1, hostIsA)
	notch, ok := buildHostNotch(t.inner.host, maps, tanA, tanB, innerHostDetour(t.inner, ef.cyl, t))
	if !ok {
		return false
	}
	set.replace[t.inner.host.ID()] = notch
	return true
}

// outerHostDetour is the outer host's notch builder: from the receded tangent corner (from) a straight
// survivor to the near setback crossing, the boss host-side footprint arc split at the wall seam point
// (near→seam→far), and a straight survivor to the other corner. Sampled identically to the wall rim's
// host sub-arcs (hostSideFootArc/appendArcSegs) so the two weld point-for-point.
func outerHostDetour(boss crossingBoss, cyl geom.Cylinder, t setbackTiling) func(from, to math.Point3) ([]notchSeg, bool) {
	return func(from, to math.Point3) ([]notchSeg, bool) {
		near, far := orderByNearer(from, t.aCutLo, t.aCutHi)
		return hostArcDetour(boss, cyl, from, near, far)
	}
}

// hostArcDetour builds a notch that runs from→near (straight), the host-side footprint arc near→seam→far,
// then far→(to) (straight). The arc is split at the boss wall's own footprint seam so both halves weld to
// the subdivided wall rim.
func hostArcDetour(boss crossingBoss, cyl geom.Cylinder, from, near, far math.Point3) ([]notchSeg, bool) {
	seam := boss.footEdge.StartVertex().Point()
	arc1, ok1 := hostSideFootArc(boss, cyl, near, seam)
	arc2, ok2 := hostSideFootArc(boss, cyl, seam, far)
	if !ok1 || !ok2 {
		return nil, false
	}
	segs := appendArcSegs([]notchSeg{{pt: from}}, arc1)
	segs = appendSeamArc(segs, boss, arc2)
	return append(segs, notchSeg{pt: far}), true
}

// appendSeamArc appends the boss footprint SEAM (pinned to its intact-wall vertex id via notchSeg.srcV,
// so spliceNotch welds the notch to the kept wall's seam vertex) followed by arc's samples EXCLUDING its
// first — arc STARTS at the seam (hostSideFootArc(seam, …)), so its point[0] IS the seam and would
// double it under id 0. Emitting the seam once, id-pinned, is exactly what closes the wall↔host weld
// that addID's distinct-id rule (#1600) otherwise splits (S4 cone/cyl runout).
func appendSeamArc(segs []notchSeg, boss crossingBoss, arc geom.Curve3) []notchSeg {
	seamV := boss.footEdge.StartVertex()
	segs = append(segs, notchSeg{pt: seamV.Point(), srcV: seamV.ID()})
	tail := sampleCurve3Open(arc, false)
	for _, p := range tail[1:] {
		segs = append(segs, notchSeg{pt: p})
	}
	return segs
}

// innerHostDetour is the inner host's notch builder: from the receded tangent corner a straight survivor
// to the near flank cut station, the near plain-contact seam (bCut→bSeam, sampled to match the flank
// patch), the boss host-side footprint arc (bSeam→seam→bSeam), the far plain seam, and a straight
// survivor home. Every sub-curve is sampled identically to its patch/wall neighbour so all weld.
func innerHostDetour(boss crossingBoss, cyl geom.Cylinder, t setbackTiling) func(from, to math.Point3) ([]notchSeg, bool) {
	return func(from, to math.Point3) ([]notchSeg, bool) {
		nearCut, farCut := orderByNearer(from, t.bCutLo, t.bCutHi)
		nearSeam, farSeam := orderByNearer(from, t.bSeamLo, t.bSeamHi)
		return innerHostSegs(boss, cyl, from, nearCut, nearSeam, farSeam, farCut)
	}
}

// innerHostSegs assembles the inner host detour's notch segment chain (split from innerHostDetour to
// keep the closure short): from→nearCut, the near plain seam, the host-side arc through the wall seam,
// the far plain seam, farCut→to.
func innerHostSegs(boss crossingBoss, cyl geom.Cylinder, from, nearCut, nearSeam, farSeam, farCut math.Point3) ([]notchSeg, bool) {
	seam := boss.footEdge.StartVertex().Point()
	arc1, ok1 := hostSideFootArc(boss, cyl, nearSeam, seam)
	arc2, ok2 := hostSideFootArc(boss, cyl, seam, farSeam)
	if !ok1 || !ok2 {
		return nil, false
	}
	segs := appendArcSegs([]notchSeg{{pt: from}}, geom.NewLineSegment(nearCut, nearSeam))
	segs = appendArcSegs(segs, arc1)
	segs = appendSeamArc(segs, boss, arc2)
	segs = appendArcSegs(segs, geom.NewLineSegment(farSeam, farCut))
	return append(segs, notchSeg{pt: farCut}), true
}

// orderByNearer returns (a,b) so a is the one nearer to ref — the near/far split every host detour uses
// to orient its traversal from the receded corner it enters from.
func orderByNearer(ref, p, q math.Point3) (near, far math.Point3) {
	if ref.DistanceTo(p) <= ref.DistanceTo(q) {
		return p, q
	}
	return q, p
}

// subdivideBossWall records, on maps.edgeInserts, the sampled footprint-rim points that transformFace
// must insert into the INTACT boss wall's footprint edge so its rim welds to the re-clipped host (host-
// side arc) and the setback patches (band-side arcs). The wall stays ONE face at full area — only its
// rim gains vertices (the closed-conic curve is dropped to straight chords by transformFace, matching
// the neighbours' chords). ok=false when the wall or a rim sub-arc is malformed.
func subdivideBossWall(maps filletRebuildMaps, boss crossingBoss, cyl geom.Cylinder, cross1, cross2 math.Point3, bandInner []math.Point3) bool {
	wall := otherFace(boss.footEdge, boss.host)
	if wall == nil {
		return false
	}
	seam := boss.footEdge.StartVertex().Point()
	ring, ok := bossRimRing(boss, cyl, seam, cross1, cross2, bandInner)
	if !ok || len(ring) < 2 {
		return false
	}
	if maps.edgeInserts[wall] == nil {
		maps.edgeInserts[wall] = map[uint64][]math.Point3{}
	}
	maps.edgeInserts[wall][boss.footEdge.ID()] = ring[1:] // transformFace adds seam (survivor); these follow
	return true
}

// bossRimRing samples the full footprint rim, ordered from the wall seam all the way around: the host-side
// sub-arc seam→cross1, the band-side sub-arcs cross1→…→cross2 (shared with the patches), the host-side
// sub-arc cross2→seam. Each sub-arc is sampled open (sampleCurve3Open), so consecutive sub-arcs
// concatenate without a duplicate vertex and every point matches the neighbour that tiles the same curve.
func bossRimRing(boss crossingBoss, cyl geom.Cylinder, seam, cross1, cross2 math.Point3, bandInner []math.Point3) ([]math.Point3, bool) {
	subs, ok := bossRimSubArcs(boss, cyl, seam, cross1, cross2, bandInner)
	if !ok {
		return nil, false
	}
	var ring []math.Point3
	for _, a := range subs {
		ring = append(ring, sampleCurve3Open(a, false)...)
	}
	return orientRingToEdge(ring, boss.footEdge), len(ring) > 0
}

// orientRingToEdge reverses the rim ring (keeping the seam point first) when it winds AGAINST the
// footprint edge's own native parametrization. transformFace applies the edge use's Reversed() flag to
// the inserts (orientedInserts) assuming they follow the edge's native direction; a ring built the other
// way then winds the wall loop's footprint rim the SAME sense as its top rim (a figure-8 in cylinder u),
// which assembleBody's shell-orient corrupts into a zero-curve top rim (the r8 cap→0 defect). Aligning to
// the edge's native direction keeps the two rims opposite, so the loop is a simple band.
func orientRingToEdge(ring []math.Point3, footEdge *topo.Edge) []math.Point3 {
	fwd, ok := footEdgeNativeForward(footEdge)
	if !ok || len(ring) < 3 {
		return ring
	}
	if ring[1].DistanceTo(fwd) <= ring[len(ring)-1].DistanceTo(fwd) {
		return ring
	}
	out := append([]math.Point3{ring[0]}, reversePts(ring[1:])...)
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
// the band-side chain cross1→bandInner…→cross2, host-side cross2→seam. It is the shared source of truth
// for both the wall-rim subdivision (all sub-arcs) and the host detour (the two host-side ones), so the
// two are sampled from the SAME curves.
func bossRimSubArcs(boss crossingBoss, cyl geom.Cylinder, seam, cross1, cross2 math.Point3, bandInner []math.Point3) ([]geom.Curve3, bool) {
	hostA, ok0 := hostSideFootArc(boss, cyl, seam, cross1)
	hostB, ok1 := hostSideFootArc(boss, cyl, cross2, seam)
	band, ok2 := bandFootArcs(boss.footEdge, cross1, cross2, bandInner)
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

// hostSideFootArc is the footprint sub-arc from→to that stays on the boss's HOST side (away from the
// fillet band): the minor arc when its midpoint is host-side, else the major one, read from the
// crossingBoss (footEdge conic + host plane). The host detour and the wall rim both route their
// host-side pieces through here, so the two trace the identical curve and weld.
func hostSideFootArc(boss crossingBoss, cyl geom.Cylinder, from, to math.Point3) (geom.Curve3, bool) {
	contact, edgeward, ok := footEdgeward(boss, cyl)
	if !ok {
		return nil, false
	}
	minor, ok0 := footprintSubArc(boss.footEdge, from, to)
	if ok0 && contact.VectorTo(minor.PointAt(0.5)).Dot(edgeward) < 0 {
		return minor, true // midpoint is behind the fillet contact line (host side), not inside the band
	}
	return footprintMajorArc(boss.footEdge, from, to)
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

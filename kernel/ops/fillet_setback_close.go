// SPDX-License-Identifier: GPL-2.0-only

package ops

import (
	stdmath "math"

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
// WIRED: runoutFacesFor (fillet_runout_faces.go) calls this after extractSetbackPatches resolves the
// loops. Builds the setback-faithful runout bodies for the two-boss corpus cases (S1/S4/T1/T4/T7/S7) and,
// via the len(b.bosses)==1 branch below (reclipSingleHost), the single-boss path added for #2007
// (S6/S9/T3) — those greened only once this closed the watertight/HolesContained gap the do-no-harm
// baseline left open.
func buildSetbackFaces(set *runoutSet, ef edgeFillet, b setbackBands, loops []RailLoop, res Resolution, maps filletRebuildMaps) bool {
	t, ok := resolveSetbackTiling(b, ef, res)
	if !ok {
		return false
	}
	if !appendSetbackWings(set, ef, t) {
		return false
	}
	if !appendSetbackPatchFaces(set, ef, loops, res) {
		return false
	}
	if len(b.bosses) == 1 {
		return reclipSingleHost(set, ef, t, maps) // one host to re-clip; pInner is the plain non-boss face
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
	return reclipHostNotch(set, ef, t.outer, maps, t.aCutLo, t.aCutHi,
		[]math.Point3{t.aSeamLo, t.aSeamHi}, outerHostDetour(t.outer, ef.cyl, t))
}

// reclipSingleHost re-clips BOTH faces of the ONE-boss edge (#2007): the boss host plane (the footprint
// opens into the cut) and the plain non-boss face (its receded fillet-contact edge is subdivided to match
// the wings + central patch). Unlike the 2-boss case — where both faces carry a boss and both take a
// footprint notch — here the non-boss face has no hole, only a contact edge that must be split at the
// wing/patch stations or it stays a single segment the sub-chorded fill cannot weld to (the 9-open-edge
// failure the plain-transformFace path leaves).
func reclipSingleHost(set *runoutSet, ef edgeFillet, t setbackTiling, maps filletRebuildMaps) bool {
	if !reclipHostNotch(set, ef, t.outer, maps, t.aCutLo, t.aCutHi, nil, outerHostDetour(t.outer, ef.cyl, t)) {
		return false
	}
	return reclipPlainFace(set, ef, t, maps)
}

// reclipPlainFace re-clips the ONE-boss edge's plain (non-boss) face: it carries no footprint hole, so its
// notch simply replaces the receded tangent segment with the subdivided fillet-contact detour
// (plainContactDetour) — the two wing B-tangent segments plus the central patch's plain seam, sampled at
// ringSegSamples so the fill welds point-for-point. buildHostNotch re-traces the rest of the outer loop.
func reclipPlainFace(set *runoutSet, ef edgeFillet, t setbackTiling, maps filletRebuildMaps) bool {
	plain := ef.b
	if t.outer.host == ef.b {
		plain = ef.a
	}
	hostIsA := plain == ef.a
	tanA, tanB := hostTangent(ef.c0, hostIsA), hostTangent(ef.c1, hostIsA)
	notch, ok := buildHostNotch(plain, maps, tanA, tanB, plainContactDetour(t))
	if !ok {
		return false
	}
	set.replace[plain.ID()] = notch
	return true
}

// plainContactDetour is the ONE-boss plain face's notch detour: from the receded tangent corner it enters
// (from), a straight wing B-tangent survivor to the near cut station, the central patch's OWN tangency
// CONTACT LOCUS (near→far) sampled at ringSegSamples, and a straight wing survivor to the far corner
// (to). No boss arc — the plain face has no footprint. The locus replaces the straight segment this used
// to draw at the PLAIN fillet's contact line: the run-out ball recedes from that line (up to 11% of this
// face's area, coons4-audit.md §C.4's separable under-recession), so a straight seam left the host face
// the wrong size AND left the patch boundary off its own surface. orientedLocus makes the polyline
// direction-safe, so the notch and the patch still share identical interior points from either corner.
func plainContactDetour(t setbackTiling) func(from, to math.Point3) ([]notchSeg, bool) {
	return func(from, to math.Point3) ([]notchSeg, bool) {
		if t.mid == nil {
			return nil, false
		}
		near, far := orderByNearer(from, t.bCutLo, t.bCutHi)
		segs := appendArcSegs([]notchSeg{{pt: from}}, orientedLocus(t.mid.railB, near, t.weld), ringSegSamples)
		return append(segs, notchSeg{pt: far}), true
	}
}

// reclipHostNotch subdivides a boss wall's footprint rim and re-clips its host plane to a single-loop
// notch (dropping the footprint hole), keyed into set.replace by host ID. The band-side crossings
// cross1/cross2 plus the interior bandInner seams (nil for the one-boss host and the two-boss inner host)
// bound the opened notch; detour re-traces the host-side arc into the outer loop. Shared by the outer,
// inner and single-boss host re-clips so all three trace the σ-partition host arc identically.
func reclipHostNotch(set *runoutSet, ef edgeFillet, boss crossingBoss, maps filletRebuildMaps,
	cross1, cross2 math.Point3, bandInner []math.Point3, detour func(from, to math.Point3) ([]notchSeg, bool)) bool {
	if !subdivideBossWall(maps, boss, ef.cyl, cross1, cross2, bandInner) {
		return false
	}
	hostIsA := boss.host == ef.a
	tanA, tanB := hostTangent(ef.c0, hostIsA), hostTangent(ef.c1, hostIsA)
	notch, ok := buildHostNotch(boss.host, maps, tanA, tanB, detour)
	if !ok {
		return false
	}
	set.replace[boss.host.ID()] = notch
	return true
}

// reclipInnerHost re-clips the inner boss's host plane and subdivides its wall footprint. The inner
// footprint band-side (bSeamLo→bSeamHi) is owned by the central patch; the host detour re-traces the two
// flank plain-contact seams (bCut→bSeam, sampled to match the patch) and the host-side arc between them.
func reclipInnerHost(set *runoutSet, ef edgeFillet, t setbackTiling, maps filletRebuildMaps) bool {
	return reclipHostNotch(set, ef, t.inner, maps, t.bSeamLo, t.bSeamHi,
		nil, innerHostDetour(t.inner, ef.cyl, t))
}

// outerHostDetour is the outer host's notch builder: from the receded tangent corner (from) a straight
// survivor to the near setback crossing, the boss host-side footprint arc split at the wall seam point
// (near→seam→far), and a straight survivor to the other corner. Its host sub-arcs come from the same
// σ-partition (hostRimArcs) the wall rim uses, so the two weld point-for-point.
func outerHostDetour(boss crossingBoss, cyl geom.Cylinder, t setbackTiling) func(from, to math.Point3) ([]notchSeg, bool) {
	return func(from, to math.Point3) ([]notchSeg, bool) {
		near, far := orderByNearer(from, t.aCutLo, t.aCutHi)
		return hostArcDetour(boss, cyl, from, near, far)
	}
}

// hostArcDetour builds a notch that runs from→near (straight), the host-side footprint arc near→seam→far,
// then far→(to) (straight). The arc is split at the boss wall's own footprint seam so both halves weld to
// the subdivided wall rim. Its two host sub-arcs come from the SAME scale-invariant σ-partition
// (hostRimArcs → partitionFootprintRim / footprintArcBySpan) the wall rim's hostA/hostB derive from
// (bossRimSubArcs), so the host notch and the wall rim trace IDENTICAL host arcs by construction — the M4
// Task-3 fix that welds the LARGE torus rim (host = 241.6° MAJOR arc) where the old local-midpoint
// hostSideFootArc took the 118° MINOR arc and left the notch un-welded (m4-spike.md §CRITICAL).
func hostArcDetour(boss crossingBoss, cyl geom.Cylinder, from, near, far math.Point3) ([]notchSeg, bool) {
	arc1, arc2, ok := hostRimArcs(boss, cyl, near, boss.footEdge.StartVertex().Point(), far)
	if !ok {
		return nil, false
	}
	segs := appendArcSegs([]notchSeg{{pt: from}}, arc1, torusHostArcChordCount(boss, arc1))
	segs = appendSeamArc(segs, boss, arc2)
	return append(segs, notchSeg{pt: far}), true
}

// hostRimArcs returns the two σ-partition host-side footprint sub-arcs of a boss rim, oriented from→seam
// and seam→to (the notch-traversal direction), each with the exact native span the wall-rim subdivision
// derives (partitionFootprintRim: hostA=span(seam↔from), hostB=span(to↔seam)). It is the single source of
// truth that makes every host detour trace the wall rim's host arcs point-for-point — the DRY seam Task 1
// flagged: the wall rim was σ-partitioned but the detours still chose minor-vs-major by a local midpoint,
// which agrees on the small cyl/cone/ellipse footprint yet diverges on the large torus (major vs minor).
func hostRimArcs(boss crossingBoss, cyl geom.Cylinder, from, seam, to math.Point3) (arc1, arc2 geom.Curve3, ok bool) {
	part, ok := partitionFootprintRim(boss, cyl, seam, from, to)
	if !ok {
		return nil, nil, false
	}
	arc1, ok1 := footprintArcBySpan(boss.footEdge, from, seam, part.hostA)
	arc2, ok2 := footprintArcBySpan(boss.footEdge, seam, to, part.hostB)
	if !ok1 || !ok2 {
		return nil, nil, false
	}
	return arc1, arc2, true
}

// appendSeamArc appends the boss footprint SEAM (pinned to its intact-wall vertex id via notchSeg.srcV,
// so spliceNotch welds the notch to the kept wall's seam vertex) followed by arc's samples EXCLUDING its
// first — arc STARTS at the seam (hostRimArcs' seam→to arc), so its point[0] IS the seam and would
// double it under id 0. Emitting the seam once, id-pinned, is exactly what closes the wall↔host weld
// that addID's distinct-id rule (#1600) otherwise splits (S4 cone/cyl runout).
func appendSeamArc(segs []notchSeg, boss crossingBoss, arc geom.Curve3) []notchSeg {
	seamV := boss.footEdge.StartVertex()
	segs = append(segs, notchSeg{pt: seamV.Point(), srcV: seamV.ID()})
	tail := sampleCurveN(arc, torusHostArcChordCount(boss, arc), false)
	for _, p := range tail[1:] {
		segs = append(segs, notchSeg{pt: p})
	}
	return segs
}

// innerHostDetour is the inner host's notch builder: from the receded tangent corner a straight survivor
// to the near flank cut station, the near flank's tangency CONTACT LOCUS (bCut→bSeam, the flank patch's
// own rail object), the boss host-side footprint arc (bSeam→seam→bSeam), the far flank's locus, and a
// straight survivor home. Every sub-curve is the very curve its patch/wall neighbour tiles, so all weld.
func innerHostDetour(boss crossingBoss, cyl geom.Cylinder, t setbackTiling) func(from, to math.Point3) ([]notchSeg, bool) {
	return func(from, to math.Point3) ([]notchSeg, bool) {
		if t.left == nil || t.right == nil {
			return nil, false
		}
		nearCut, farCut := orderByNearer(from, t.bCutLo, t.bCutHi)
		nearSeam, farSeam := orderByNearer(from, t.bSeamLo, t.bSeamHi)
		return innerHostSegs(boss, cyl, from, t.flankLocusFrom(nearCut), t.flankLocusFrom(farSeam),
			nearSeam, farSeam, farCut)
	}
}

// flankLocusFrom returns whichever flank's tangency contact locus starts at p, traced from p — the
// direction-safe accessor the inner host notch needs, since the two flanks own different loci and a
// polyline is not direction-symmetric.
func (t setbackTiling) flankLocusFrom(p math.Point3) geom.Curve3 {
	for _, band := range []*runoutBand{t.left, t.right} {
		for _, end := range []math.Point3{curveStart(band.railB), curveEnd(band.railB)} {
			if float64(end.DistanceTo(p)) <= t.weld {
				return orientedLocus(band.railB, p, t.weld)
			}
		}
	}
	return nil
}

// innerHostSegs assembles the inner host detour's notch segment chain (split from innerHostDetour to
// keep the closure short): from→nearCut via the near flank's contact locus, the host-side arc through
// the wall seam, the far flank's locus back out, farCut→to.
func innerHostSegs(boss crossingBoss, cyl geom.Cylinder, from math.Point3,
	nearLocus, farLocus geom.Curve3, nearSeam, farSeam, farCut math.Point3) ([]notchSeg, bool) {
	arc1, arc2, ok := hostRimArcs(boss, cyl, nearSeam, boss.footEdge.StartVertex().Point(), farSeam)
	if !ok || nearLocus == nil || farLocus == nil {
		return nil, false
	}
	segs := appendArcSegs([]notchSeg{{pt: from}}, nearLocus, ringSegSamples)
	segs = appendArcSegs(segs, arc1, torusHostArcChordCount(boss, arc1))
	segs = appendSeamArc(segs, boss, arc2)
	segs = appendArcSegs(segs, farLocus, ringSegSamples)
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
	for i, a := range subs {
		ring = append(ring, sampleCurveN(a, rimSubArcChordCount(boss, a, i, len(subs)), false)...)
	}
	return orientRingToEdge(ring, boss.footEdge), len(ring) > 0
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
// (the 241.6° major arc at 6 chords lofted T1 +8.6%, m4-spike §CRITICAL). See torusHostArcChordCount.
func rimSubArcChordCount(boss crossingBoss, arc geom.Curve3, i, total int) int {
	if i != 0 && i != total-1 {
		return ringSegSamples // band sub-arc: welds to the patches, keep the shared granularity
	}
	return torusHostArcChordCount(boss, arc)
}

// torusHostArcChordCount is the chord count for a boss's host-side footprint arc: span-proportional,
// ≈torusRimChordAngle per chord, floored at ringSegSamples. Both the wall rim and the host notch sample
// the SAME host arc through here, so they stay weld-identical at whatever count it returns. See
// hostArcDensifies for why every wall kind now densifies.
func torusHostArcChordCount(boss crossingBoss, arc geom.Curve3) int {
	if !hostArcDensifies(boss) {
		return ringSegSamples
	}
	n := int(stdmath.Ceil(arcSweepAbs(arc) / torusRimChordAngle))
	if n < ringSegSamples {
		return ringSegSamples
	}
	return n
}

// hostArcDensifies reports whether a boss's host footprint arc is chorded span-proportionally. It is now
// unconditional. The host-side arc is the LONG way round the footprint (240°–315° on the corpus bosses),
// and chording it at the coarse ringSegSamples leaves the re-clipped host plane with the whole inscribed-
// polygon deficit — Σ (r²/2)(θ−sinθ) over the chords — as SURPLUS face area, because the notch cuts
// inside the true rim. That is exactly the "host plane recedes too little" defect coons4-audit.md §C.4
// isolated and could not attribute: S7 +10.95% (a 6-chord 315° sphere rim), S4 +8.87%, T7 +6.18%,
// S1 +6.07%. It was never a geometry error — the wall rim and the host notch trace the same arcs — it is
// the polygonal approximation of those arcs, and the fix is to chord them at the span-proportional angle
// the torus wall already used. The wall rim and the host notch both read their count through here, so
// they stay weld-identical at whatever count this returns.
func hostArcDensifies(_ crossingBoss) bool { return true }

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

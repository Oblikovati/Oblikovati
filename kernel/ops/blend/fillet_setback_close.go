// SPDX-License-Identifier: GPL-2.0-only

package blend

import (
	"oblikovati.org/kernel/geom"
	"oblikovati.org/kernel/ops/internal/tol"
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
func buildSetbackFaces(set *runoutSet, ef edgeFillet, b setbackBands, loops []RailLoop, res tol.Resolution, maps filletRebuildMaps) bool {
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
func appendSetbackPatchFaces(set *runoutSet, ef edgeFillet, loops []RailLoop, res tol.Resolution) bool {
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
// CONTACT LOCUS (near→far) sampled at ringSegSamples with each segment carrying its own locus sub-span
// (appendTrimmedArcSegs — the same TrimmedCurve3 value the patch's loop offers, so the shared edge
// carries the interpolated locus instead of its chords), and a straight wing survivor to the far corner
// (to). No boss arc — the plain face has no footprint. The locus replaces the straight segment this used
// to draw at the PLAIN fillet's contact line: the run-out ball recedes from that line (up to 11% of this
// face's area, coons4-audit.md §C.4's separable under-recession), so a straight seam left the host face
// the wrong size AND left the patch boundary off its own surface. orientedLocus makes the locus
// direction-safe, so the notch and the patch still share identical interior points from either corner.
func plainContactDetour(t setbackTiling) func(from, to math.Point3) ([]notchSeg, bool) {
	return func(from, to math.Point3) ([]notchSeg, bool) {
		if t.mid == nil {
			return nil, false
		}
		near, far := orderByNearer(from, t.bCutLo, t.bCutHi)
		segs := appendTrimmedArcSegs([]notchSeg{{pt: from}}, orientedLocus(t.mid.railB, near, t.weld), ringSegSamples)
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
	segs := appendTrimmedArcSegs([]notchSeg{{pt: from}}, arc1, hostArcChordCount(arc1))
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
// that addID's distinct-id rule (#1600) otherwise splits (S4 cone/cyl runout). Every segment carries its
// own sub-span of the footprint arc (appendTrimmedArcSegs' rule), so the notch bounds the true rim.
func appendSeamArc(segs []notchSeg, boss crossingBoss, arc geom.Curve3) []notchSeg {
	seamV := boss.footEdge.StartVertex()
	pts, curves := sampleCurveNTrimmed(arc, hostArcChordCount(arc), false)
	segs = append(segs, notchSeg{pt: seamV.Point(), srcV: seamV.ID(), curve: curves[0]})
	for i := 1; i < len(pts); i++ {
		segs = append(segs, notchSeg{pt: pts[i], curve: curves[i]})
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
// the wall seam, the far flank's locus back out, farCut→to. Both loci carry their per-segment
// sub-spans (appendTrimmedArcSegs) — the same TrimmedCurve3 values the flank patches' loops offer —
// so the shared edges carry the interpolated locus instead of its chords.
func innerHostSegs(boss crossingBoss, cyl geom.Cylinder, from math.Point3,
	nearLocus, farLocus geom.Curve3, nearSeam, farSeam, farCut math.Point3) ([]notchSeg, bool) {
	arc1, arc2, ok := hostRimArcs(boss, cyl, nearSeam, boss.footEdge.StartVertex().Point(), farSeam)
	if !ok || nearLocus == nil || farLocus == nil {
		return nil, false
	}
	segs := appendTrimmedArcSegs([]notchSeg{{pt: from}}, nearLocus, ringSegSamples)
	segs = appendTrimmedArcSegs(segs, arc1, hostArcChordCount(arc1))
	segs = appendSeamArc(segs, boss, arc2)
	segs = appendTrimmedArcSegs(segs, farLocus, ringSegSamples)
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

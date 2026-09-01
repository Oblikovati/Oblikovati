// SPDX-License-Identifier: GPL-2.0-only

package blend

import (
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
)

// railWrite is one band-end's rewritten cross-section — a single, order-independent mutation a
// treatment contributes to the shared accumulator (design §2 channel A). It always rewrites ta/tb; a
// dihedral miter additionally carries the shared seam, a torus band-retract / run-off clip the arc
// mid. setMid/setSeam keep each treatment writing EXACTLY the fields its old pass wrote (byte-identity:
// the dihedral pass never touched mid, the torus/run-off passes never touched seam).
type railWrite struct {
	fi      int
	atC1    bool
	ta, tb  math.Point3
	setMid  bool
	mid     math.Point3
	setSeam bool
	seam    []math.Point3
}

// hostEndCorner is a synthetic simple END corner a mixedTorus treatment injects into a host plane's
// re-trim (design §2 channel E), so the shared plane's receded vertex expands into the torus
// top-contact arc. It carries the host face, the corner vertex id, and the end corner to inject.
type hostEndCorner struct {
	face   *topo.Face
	vid    uint64
	corner corner
}

// setbackData is the shared, APPEND-ONLY accumulator every corner treatment mutates instead of
// building + certifying its own body (design §2). Five channels — one per distinct mutation the four
// passes emit — plus fired. No treatment reads a channel another writes (the corner-independence
// guarantee I3), and two distinct corners touch disjoint band-ends, so the composed result is
// order-independent; the phase pipeline (adoptCornerSetback) is the only sequenced dependency.
type setbackData struct {
	railWrites   []railWrite             // (A) direct fils ta/tb/seam/mid rewrites (dihedral, torus, run-off)
	voidSpheres  map[uint64]*cornerBlend // (B) blend replacements ⇒ ONE computeFillets re-solve (concaveSphere)
	dropBlends   map[uint64]bool         // (C) vertices dropped from sphere-patch emission (now a torus)
	extraPatches []filletFace            // (D) extra corner-patch faces (the mixedTorus torus)
	hostEnds     []hostEndCorner         // (E) synthetic host end-corner arcs (mixedTorus top contact)
	fired        bool                    // any non-decline treatment contributed
}

// accumulate classifies every shared corner (sorted by vid) and appends its treatment's channel
// contributions to one setbackData. It runs no assembly and no certify — it is the write-model the
// three-phase pipeline folds once. Example: data := accumulate(setbackCtx{body, fils, blends, miters, ends}).
func accumulate(ctx setbackCtx) setbackData {
	data := setbackData{voidSpheres: map[uint64]*cornerBlend{}, dropBlends: map[uint64]bool{}}
	seenBands := map[int]bool{} // convexRunoff dedups shared bands (mirrors the old band-index union)
	for _, ref := range sortedCornerRefs(ctx) {
		accumulateCorner(ref, ctx, &data, seenBands)
	}
	return data
}

// accumulateCorner dispatches one classified corner to its channel-fill. A declined corner (and a
// signature-matched corner whose geometry solve fails inside the fill) contributes nothing.
func accumulateCorner(ref cornerRef, ctx setbackCtx, data *setbackData, seenBands map[int]bool) {
	switch classifyCorner(ref, ctx) {
	case treatDihedralMiter:
		accumulateDihedral(ref.vid, ctx, data)
	case treatConcaveSphere:
		accumulateConcaveSphere(ref.vid, ctx, data)
	case treatMixedTorus:
		accumulateMixedTorus(ref.vid, ctx, data)
	case treatConvexRunoff:
		accumulateConvexRunoff(ref.vid, ctx, data, seenBands)
	case treatRadiusTorus:
		accumulateRadiusTorus(ref.vid, ctx, data)
	}
}

// accumulateDihedral appends the two miter ends' seam railWrites for a concave-orthogonal dihedral
// corner (P1). It re-samples the setback seam on the arms' own concave-side cylinders (concaveMiterSeam)
// and, when that succeeds, rewrites both ends' ta/tb/seam — the single-source coupling every consumer
// reads. A seam that cannot be sampled leaves the corner untouched (matches resetbackCorner's decline).
func accumulateDihedral(vid uint64, ctx setbackCtx, data *setbackData) {
	pair, cm := ctx.ends[vid], ctx.miters[vid]
	efA, efB := ctx.fils[pair[0].fi], ctx.fils[pair[1].fi]
	seam, ok := concaveMiterSeam(efA, efB, cm)
	if !ok {
		return
	}
	data.railWrites = append(data.railWrites,
		miterEndRailWrite(pair[0], efA, cm.shared, seam),
		miterEndRailWrite(pair[1], efB, cm.shared, seam))
	data.fired = true
}

// miterEndRailWrite builds one miter end's railWrite, matching miterTangents' orientation: the
// shared-face arm carries seam[0]→sBot forward, the outer-face arm carries it reversed so its ta→tb
// still runs the seam the same way the untouched convex path does.
func miterEndRailWrite(end miterEnd, ef edgeFillet, shared *topo.Face, seam []math.Point3) railWrite {
	sTop, sBot := seam[0], seam[len(seam)-1]
	if ef.a == shared {
		return railWrite{fi: end.fi, atC1: end.atC1, ta: sTop, tb: sBot, setSeam: true, seam: seam}
	}
	return railWrite{fi: end.fi, atC1: end.atC1, ta: sBot, tb: sTop, setSeam: true, seam: reversePoints(seam)}
}

// accumulateConcaveSphere records a concave trihedral corner's VOID-side sphere replacement (P2). The
// void sphere forces a single downstream re-solve (Phase B) that retracts each band by r; the treatment
// itself contributes NO railWrite (its retraction is implicit in the re-solve, design §2 note).
func accumulateConcaveSphere(vid uint64, ctx setbackCtx, data *setbackData) {
	cb := ctx.blends[vid]
	faces, ok := concaveTrihedralCornerFaces(vid, cb, ctx.fils)
	if !ok {
		return
	}
	if void, solved := solveVoidCornerSphere(cb.vertex, faces, cb.sphere.Radius); solved {
		data.voidSpheres[vid] = void
		data.fired = true
	}
}

// accumulateMixedTorus records a mixed-sense trihedral corner's torus treatment (P3): three band
// retract railWrites, the vertex dropped from sphere emission, the torus patch face, and the synthetic
// host end-corner. A degenerate torus (pivot radius off 2r) declines inside buildMixedTorusCorner.
func accumulateMixedTorus(vid uint64, ctx setbackCtx, data *setbackData) {
	mc, ok := buildMixedTorusCorner(vid, ctx.blends[vid], ctx.fils)
	if !ok {
		return
	}
	for _, br := range mc.bands {
		data.railWrites = append(data.railWrites, bandRetractRailWrite(ctx.fils, br))
	}
	data.dropBlends[mc.vertexID] = true
	data.extraPatches = append(data.extraPatches, mc.patch)
	data.hostEnds = append(data.hostEnds, hostEndCorner{face: mc.topFace, vid: mc.vertexID, corner: mc.topCorner})
	data.fired = true
}

// bandRetractRailWrite converts one torus band retract into a railWrite: the per-face contact points
// keyed to the band's a/b faces plus the section-arc mid (mirrors applyBandRetract's field writes).
func bandRetractRailWrite(fils []edgeFillet, br bandRetract) railWrite {
	c := fils[br.fi].c0
	if br.atC1 {
		c = fils[br.fi].c1
	}
	return railWrite{fi: br.fi, atC1: br.atC1, ta: br.pt[c.a.ID()], tb: br.pt[c.b.ID()], setMid: true, mid: br.mid}
}

// accumulateConvexRunoff appends the oblique run-off clip railWrites for every band of a convex
// same-sense trihedral corner (P4), deduplicating bands shared between corners (mirrors the old
// band-index union). Only an end whose rail actually moves contributes a railWrite (0–2 per band).
func accumulateConvexRunoff(vid uint64, ctx setbackCtx, data *setbackData, seenBands map[int]bool) {
	bands, ok := convexTrihedralCornerBands(vid, ctx.blends[vid], ctx.fils)
	if !ok {
		return
	}
	for _, b := range bands {
		if seenBands[b.fi] {
			continue
		}
		seenBands[b.fi] = true
		accumulateRunoffBand(ctx.fils[b.fi], b.fi, data)
	}
}

// accumulateRunoffBand applies the shared oblique-runoff setback (setbackObliqueRunoffEnds) to a
// SCRATCH copy of the band and records a railWrite for each end that actually moved. The scratch copy
// isolates the ta/tb/mid writes (corner fields are values), so the caller's fils is untouched.
func accumulateRunoffBand(ef edgeFillet, fi int, data *setbackData) {
	scratch := ef
	if !setbackObliqueRunoffEnds(&scratch) {
		return
	}
	appendIfMoved(data, fi, false, ef.c0, scratch.c0)
	appendIfMoved(data, fi, true, ef.c1, scratch.c1)
	data.fired = true
}

// appendIfMoved records after's ta/tb/mid as a railWrite when it differs from before — so a
// non-overshooting end (left unmoved by setbackObliqueRunoffEnds) contributes nothing.
func appendIfMoved(data *setbackData, fi int, atC1 bool, before, after corner) {
	if before.ta == after.ta && before.tb == after.tb && before.mid == after.mid {
		return
	}
	data.railWrites = append(data.railWrites,
		railWrite{fi: fi, atC1: atC1, ta: after.ta, tb: after.tb, setMid: true, mid: after.mid})
}

// adoptCornerSetback is the ONE composable corner-setback pass: classify + accumulate every corner into
// one setbackData, then fold it in three fixed phases (design §3) — void-sphere blend replacement, ONE
// re-solve iff any void, direct fils railWrites — assemble ONCE (weldSetbackFaces), certify ONCE
// (obstacleImprovedSolid, the do-no-harm floor). It replaces the four early-returning adopt* passes, so
// a body carrying MULTIPLE corner types (e.g. a mixedTorus AND a dihedralMiter) sets back BOTH instead
// of swallowing all but the first-matched. Adoption is gated purely on the composed body certifying a
// watertight hole-contained solid: a config that cannot close falls back to the baseline byte-identical,
// never a worse body. Byte-identity holds for every single-type corpus body because accumulate on a
// pure-type body yields exactly that type's channels and the pipeline reduces to that pass's own
// sequence (design §Strangler reduction proof).
func adoptCornerSetback(body *topo.Body, edges []filletPick, fils []edgeFillet, blends map[uint64]*cornerBlend, miters map[uint64]*cornerMiter, concave ConcaveFill, baseline *topo.Body) *topo.Body {
	data := accumulate(setbackCtx{body: body, fils: fils, blends: blends, miters: miters, ends: miterCornerEnds(fils)})
	if !data.fired {
		return baseline
	}
	workFils, workBlends, ok := resolveSetback(body, edges, fils, blends, miters, concave, data)
	if !ok {
		return baseline // a re-solve failure keeps the material-side baseline (matches P2 today)
	}
	cand := weldSetbackFaces(body, applyRailWrites(workFils, data.railWrites), workBlends, data)
	if obstacleImprovedSolid(cand) {
		return cand
	}
	return baseline
}

// resolveSetback runs Phase A (void-sphere blend substitution) and Phase B (ONE re-solve iff any void
// sphere replaced it) — the re-solve regenerates fils from the mutated blends, so it MUST precede any
// direct railWrite (Phase C). A mixedTorus body (dropBlends but no voidSpheres) never re-solves (R2).
// ok=false on a re-solve failure so the caller floors to the baseline.
func resolveSetback(body *topo.Body, edges []filletPick, fils []edgeFillet, blends map[uint64]*cornerBlend,
	miters map[uint64]*cornerMiter, concave ConcaveFill, data setbackData) ([]edgeFillet, map[uint64]*cornerBlend, bool) {
	if len(data.voidSpheres) == 0 {
		return fils, blends, true
	}
	workBlends := blendsWithVoidSpheres(blends, data.voidSpheres)
	rf, err := computeFillets(body, edges, workBlends, miters, concave, nil)
	if err != nil {
		return nil, nil, false
	}
	return rf, workBlends, true
}

// blendsWithVoidSpheres builds the re-solve's blends map: each concave corner's material-side sphere
// replaced by its void-side sphere, every OTHER blend cloned with arcs reset so the re-run repopulates
// them exactly once (registerBlendArc APPENDS — reusing a struct with arcs would double them). This
// reproduces the old flipConcaveTrihedralBlends output the P2 re-solve consumed.
func blendsWithVoidSpheres(blends, voids map[uint64]*cornerBlend) map[uint64]*cornerBlend {
	out := make(map[uint64]*cornerBlend, len(blends))
	for vid, cb := range blends {
		if void, ok := voids[vid]; ok {
			out[vid] = void
			continue
		}
		out[vid] = cloneBlendResetArcs(cb)
	}
	return out
}

// applyRailWrites returns a fresh fils slice with every railWrite applied; the caller's slice is never
// mutated (edgeFillet corners are values, so the shallow copy fully isolates the writes).
func applyRailWrites(fils []edgeFillet, writes []railWrite) []edgeFillet {
	out := append([]edgeFillet(nil), fils...)
	for _, w := range writes {
		applyRailWrite(&out[w.fi], w)
	}
	return out
}

// applyRailWrite rewrites one band-end's cross-section from a railWrite, touching exactly the fields the
// contributing treatment set (ta/tb always; mid and seam only when flagged).
func applyRailWrite(ef *edgeFillet, w railWrite) {
	c := &ef.c0
	if w.atC1 {
		c = &ef.c1
	}
	c.ta, c.tb = w.ta, w.tb
	if w.setMid {
		c.mid = w.mid
	}
	if w.setSeam {
		c.seam = w.seam
	}
}

// weldSetbackFaces assembles the composed body ONCE, routing by the presence of a torus/hostEnd channel
// (byte-identity R3): a pure dihedral/concaveSphere/run-off body (no torus) takes today's
// assembleFilletBody — preserving its obstacle/runout rebuild compositing — while a mixedTorus body
// takes the DIRECT mixed weld (no rebuild compositing), exactly as each pure-type body assembles today.
func weldSetbackFaces(body *topo.Body, setFils []edgeFillet, workBlends map[uint64]*cornerBlend, data setbackData) *topo.Body {
	if len(data.extraPatches) == 0 && len(data.hostEnds) == 0 {
		return assembleFilletBody(body, setFils, workBlends)
	}
	return weldMixedTorusFaces(body, setFils, workBlends, data)
}

// weldMixedTorusFaces welds the mixed-corner body directly: transformed hosts (+ each synthetic host
// end-corner injected into the re-trim), retracted band cylinders, the remaining sphere patches (every
// blend NOT dropped), and the torus patches. Mirrors filletResultFaces' face list with the torus
// patches as the only extra and the host end-corner as the only host-map injection.
func weldMixedTorusFaces(body *topo.Body, fils []edgeFillet, blends map[uint64]*cornerBlend, data setbackData) *topo.Body {
	maps, caps := filletBuildMaps(body, fils)
	for _, he := range data.hostEnds {
		if maps.endCorner[he.face] == nil {
			maps.endCorner[he.face] = map[uint64]corner{}
		}
		maps.endCorner[he.face][he.vid] = he.corner
	}
	out := transformedBodyFaces(body, maps, map[uint64]filletFace{})
	out = append(out, filletBlendFaces(fils, caps, map[uint64]bool{}, map[uint64][]filletFace{})...)
	for vid, cb := range blends {
		if !data.dropBlends[vid] {
			out = append(out, spherePatchFace(cb))
		}
	}
	out = append(out, data.extraPatches...)
	return assembleBody(out)
}

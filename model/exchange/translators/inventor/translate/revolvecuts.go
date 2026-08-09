// SPDX-License-Identifier: GPL-2.0-only

package translate

import (
	"fmt"

	"oblikovati.org/model/compdef"
	"oblikovati.org/model/exchange/translators/inventor/ipt"
	"oblikovati.org/model/feature"
)

// buildRevolveDispatch builds the revolve-based part. The common case — a plain turned shaft — is
// the incidence/line-set path (buildRevolve), kept verbatim so every currently-solid revolve is
// untouched. A MACHINED revolve part (a turned base with milled cut extrudes and drilled holes: the
// ext,rev,hole cluster — SpoolMotorMachinedHolder, CapstonMotorMachinedHolder) needs its cuts
// applied over the revolve base, and its profile+centreline decoded whole. Only when the primary
// revolve yields no mesh-fitting solid is that second path tried, so the working set can't regress.
func buildRevolveDispatch(def *compdef.PartComponentDefinition, d *ipt.Document, seg []byte, placed []placedSketch) (bool, []string) {
	emitted := emitSketches(def, placed)
	built, notes := buildRevolve(def, seg, placed, emitted)
	// buildRevolve adds the feature but does not always recompute (the RevolveProfile success path
	// leaves that to buildPart); recompute here so firstBodyIsSolid/bodyFitsMesh see the body.
	def.Recompute()
	if built && firstBodyIsSolid(def) && bodyFitsMesh(def, d) {
		emitDroppedCurveSketches(def, d)
		return true, notes
	}
	// The primary revolve gave no gate-passing solid (it didn't build, didn't close, or over-built
	// past Inventor's tessellation). Rebuild from the node graph — which keeps the profile, its
	// centreline, and each cut profile as whole sketches where the incidence line set fragments
	// them — and apply the cut extrudes/holes over the revolve base.
	if ok, gnotes := graphRevolveWithCuts(def, d, seg); ok {
		emitDroppedCurveSketches(def, d)
		return true, gnotes
	}
	// Neither path made a solid: restore the primary attempt so the part keeps its decoded sketches
	// (and any partial body) for buildPart's mesh fallback, exactly as before this dispatch existed.
	clearFeaturesAndSketches(def)
	emitted = emitSketches(def, placed)
	built, notes = buildRevolve(def, seg, placed, emitted)
	emitDroppedCurveSketches(def, d)
	return built, notes
}

// graphRevolveWithCuts rebuilds the part from the node-graph sketches: it revolves the graph's
// profile about its centreline (the kernel arranges a closed profile the line-ring walk can't —
// arcs, or a loop closed only along the axis), then cuts the milled extrudes and drills the hole
// over that base. It commits only when the result closes to a solid that fits the tessellation;
// otherwise it returns false with the def left cleared for the caller to restore the primary state.
// The whole attempt is speculative and self-contained: nothing here reaches a part whose primary
// revolve already produced a good solid.
func graphRevolveWithCuts(def *compdef.PartComponentDefinition, d *ipt.Document, seg []byte) (bool, []string) {
	graph := ipt.GraphSketches(d)
	if !graphRevolveCandidate(graph) {
		return false, nil
	}
	clearFeaturesAndSketches(def) // drop the primary attempt; build fresh from the graph
	placed := placeGraphSketches(graph)
	emitted := emitSketches(def, placed)
	if len(tryKernelRevolve(def, seg, placed, emitted)) == 0 {
		return false, nil
	}
	def.Recompute()
	if !firstBodyIsSolid(def) {
		return false, nil // the revolve base didn't close — not a machined-holder we can rebuild
	}
	notes := applyRevolveCuts(def, d, placed, emitted)
	def.Recompute()
	if firstBodyIsSolid(def) && bodyFitsMesh(def, d) {
		return true, notes
	}
	return false, notes
}

// applyRevolveCuts applies the part's cut/join extrudes and its drilled hole over the revolve base
// already built into def. It mirrors buildExtrudeFeatures' per-extrude region match, but consumes an
// existing base (the revolve) instead of requiring a New-Body extrude to start one. A stage that
// can't resolve its profile or region is skipped with a note; whatever cuts do bind stay.
func applyRevolveCuts(def *compdef.PartComponentDefinition, d *ipt.Document, placed []placedSketch, emitted []emittedSketch) []string {
	var notes []string
	extrudes := ipt.DecodeExtrudes(d)
	profiles := ipt.ExtrudeProfiles(d)
	regions := ipt.ExtrudeRegions(d)
	for i, ex := range extrudes {
		p := profileIndex(profiles, i)
		if p < 0 || p >= len(emitted) || emitted[p].sk == nil {
			notes = append(notes, fmt.Sprintf("revolve cut %d: no profile sketch resolved — skipped", i))
			continue
		}
		region := extrudeRegionAt(regions, i)
		idx := regionProfileIndices(emitted[p].sk, region)
		if len(idx) == 0 {
			notes = append(notes, fmt.Sprintf("revolve cut %d: could not match its region (%d loops) — skipped", i, len(region)))
			continue
		}
		feature.NewExtrudeFeatures(def.Features()).AddExtrude(
			emitted[p].sk, idx, operationOf(ex.Operation), extentOf(ex), 0)
	}
	if h, ok := ipt.DecodeHole(d); ok && len(placed) > 0 && len(emitted) > 0 && emitted[0].sk != nil {
		cx, cy := profileCentroid(placed[0].geom)
		addHole(def, h, placed[0].plane, cx, cy, 0)
	}
	return notes
}

// placeGraphSketches pairs each node-graph sketch with the plane it lives on — the revolve-path
// equivalent of the placement extractSketches does for the incidence/cluster decode.
func placeGraphSketches(graph []ipt.Sketch) []placedSketch {
	out := make([]placedSketch, len(graph))
	for i := range graph {
		out[i] = placedSketch{geom: graph[i], plane: sketchPlaneOf(graph[i])}
	}
	return out
}

// bodyFitsMesh reports whether the rebuilt body stays within Inventor's stored tessellation — the
// non-destructive form of gateBodyAgainstMesh's containment test (it neither drops the body nor
// reports). No stored tessellation ⇒ no oracle, so it does not block. Used to decide whether the
// primary revolve is good enough to keep, and whether a graph rebuild is safe to commit.
func bodyFitsMesh(def *compdef.PartComponentDefinition, d *ipt.Document) bool {
	mesh, ok := meshExtents(d)
	if !ok {
		return true
	}
	body, ok := bodyExtents(def)
	if !ok {
		return false
	}
	_, _, escaped := escapingAxis(body, mesh)
	return !escaped
}

// clearFeaturesAndSketches empties the feature tree and the sketch list, returning the definition to
// its pre-feature state (parameters survive). It lets buildRevolveDispatch abandon one build attempt
// cleanly and start another from a different sketch source.
func clearFeaturesAndSketches(def *compdef.PartComponentDefinition) {
	dropAllFeatures(def)
	sk := def.Sketches()
	for sk.Count() > 0 {
		sk.Remove(sk.Item(sk.Count() - 1).ID())
	}
}

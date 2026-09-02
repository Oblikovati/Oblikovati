// SPDX-License-Identifier: GPL-2.0-only

package feature

import (
	"oblikovati.org/kernel/diag"
	"oblikovati.org/kernel/ops"
	"oblikovati.org/kernel/topo"
)

// Extrude feature — the MERGE / DISSOLVE policy (M48 #2235 split of extrude.go). How an extrude's prism
// tool combines with the running body: the boolean op (join/cut/intersect/new-body), the per-region
// prism combination, the abutting-prism dissolve toggle, and the running-body accumulation. The prism
// geometry builder lives in extrude_prism.go; the feature and collection in extrude.go.

// combinePrisms applies a multi-region selection's prisms to the running bodies.
//
// A Cut or a Join applies each prism SEPARATELY, which is the SAME result by construction —
// A−(B₁∪B₂) = ((A−B₁)−B₂) and A∪(B₁∪B₂) = ((A∪B₁)∪B₂) — but a far better one in practice, because
// the merged multi-lump tool is NOT a bare analytic primitive: it fails combine's
// curvedBooleanWorthTrying test, so BOTH operands get faceted and the whole cut runs through the
// planar path. Per lump the exact curved boolean applies instead, and each boolean is small.
//
// TorquimeterDisk cuts 52 regions at once. Merged, that is an 878-face 52-shell tool, and the planar
// boolean returned a body it had itself just classified invalid (booleanGeneral ships the planar
// result when the operands are over csgFallbackFaceLimit, so no CSG fallback is even attempted):
// 1630 faces, 25 shells, NOT CLOSED. Lump by lump the same part is 461 faces, ONE shell, a closed
// solid within 5% of Inventor. It is also cheaper, not dearer.
//
// Intersect is NOT decomposable this way — A∩(B₁∪B₂) = (A∩B₁)∪(A∩B₂), not a chain — and NewBody
// wants the merged tool as one body, so both keep the merged path. With no running bodies combine
// just adds the tool, and chaining would instead cut/join the lumps into each OTHER, so that case
// keeps the merged path too.
func combinePrisms(in Input, prisms []*topo.Body, merged *topo.Body, op ops.PartFeatureOperation) ([]*topo.Body, error) {
	if len(prisms) < 2 || len(in.Bodies) == 0 || (op != ops.Cut && op != ops.Join) {
		return combine(in, merged, op)
	}
	run, out := in, in.Bodies
	for _, p := range prisms {
		bodies, err := combine(run, p, op)
		if err != nil {
			return nil, err
		}
		run.Bodies, out = bodies, bodies
	}
	return out, nil
}

// SetExtrudeDissolve toggles the abutting-prism dissolve (#38) on every extrude feature in fs and
// returns how many it changed, marking them for the next recompute. The translator's whole-part
// fallback disables the dissolve when the merged tool regressed a working solid — a downstream boolean
// fragility that only surfaces in a LATER feature, so it cannot be caught per-feature — then re-enables
// it if that did not help (the dissolve's mesh is otherwise the better one). 0 ⇒ no extrude to toggle.
func SetExtrudeDissolve(fs *PartFeatures, on bool) int {
	n := 0
	for i := 0; i < fs.Count(); i++ {
		if e, ok := fs.Item(i).feature.(*ExtrudeFeature); ok {
			e.def.NoDissolve = !on
			n++
		}
	}
	if n > 0 {
		fs.MarkAllDirty()
	}
	return n
}

// combine applies an operation between the running bodies and a new body. Phase A
// handles the first body / new-body and the non-overlapping boolean cases. It records on
// in.Diag every degradation of the exact path — an analytic operand faceted for the planar
// boolean, and the planar boolean's own triangle-CSG fallback — so the feature carries the
// quality signal instead of shipping a silently faceted body (#1601).
func combine(in Input, body *topo.Body, op ops.PartFeatureOperation) ([]*topo.Body, error) {
	running := in.Bodies
	// Surface (kSurfaceOperation) and NewBody both skip the boolean: the tool body is added
	// alongside the running bodies untouched. For Surface the tool is already an open sheet
	// (buildTool built it uncapped/non-solid); it joins the model as a surface body. #1858.
	if len(running) == 0 || op == ops.NewBody || op == ops.Surface {
		return append(append([]*topo.Body(nil), running...), body), nil
	}
	target := running[len(running)-1]
	// First try the EXACT curved boolean on the still-analytic operands (see curvedBooleanWorthTrying): the
	// result keeps the curved surface through the M2 curved boolean instead of being re-faceted into a prism.
	// This routes BOTH directions — a curved solid cut by a planar box (#1334/#1335) AND a planar box
	// drilled/joined by a cylinder/cone tool, which previously fell through to faceting and shattered the hole
	// rim into 24 straight segments (#1472). CurvedBoolean takes (target, tool) in feature order; each kernel
	// path checks its own operand roles, so the same call serves both directions, and on no match it returns
	// ok=false and we fall through to the planar path unchanged.
	//
	// This subsumes the old explicit CoaxialEqualCylinders clause for a JOIN of two coaxial equal-radius
	// cylinders (a stepped/stacked shaft, #1831). That clause existed only because the both-cylinder pair
	// failed the then-tighter gate; CoaxialEqualCylinders requires both operands to be BARE cylinder solids,
	// which curvedBooleanWorthTrying now admits on its own, and brep.CoaxialCylinderUnion still recognises
	// the pair from inside curvedExactPaths.
	if curvedBooleanWorthTrying(target, body) {
		if res, ok := ops.CurvedBooleanWithDiagnostics(op, target, body, in.Diag); ok {
			return appendCombined(running, res), nil
		}
	}
	// Otherwise re-facet any analytic curved face into a planar B-rep before the planar boolean — it hangs
	// on a full periodic curved face it cannot consume (#129). A standalone primitive that is never
	// combined keeps its analytic face for thread/chamfer/fillet. Faceting is PERMANENT, so planarizedDiag
	// records it as a defect rather than degrading silently (#1601, audit A5).
	target = planarizedDiag(target, "combine-target", in.Diag)
	body = planarizedDiag(body, "combine-tool", in.Diag)
	res, err := ops.BooleanWithDiagnostics(op, target, body, in.Diag)
	if err != nil {
		return nil, err
	}
	if rescued, ok := facetedBooleanRescue(op, target, body, res, in.Diag); ok {
		res = rescued
	}
	return appendCombined(running, res), nil
}

// facetedBooleanRescue re-runs the boolean on FACETED operands when the analytic ones produced a body
// that is not a valid solid, and reports whether that recovered it.
//
// The boolean cannot consume an analytic face on its planar path, so it falls to triangle CSG once per
// operation; on a part that cuts many regions the soup accumulates and stops stitching. A real
// Inventor disk with 52 cut regions came back as 11780 planar faces in 3 shells with 28 open boundary
// edges, where faceting the operands first gives 461 faces in ONE closed shell.
//
// It runs AFTER the analytic attempt, not before, because faceting up front is what forfeits the
// analytic result a wrapped emboss depends on. The rescue is adopted only when it is genuinely a valid
// solid, so a case the analytic path already handles is untouched, and the faceting is recorded as a
// Defect: it is permanent, and every downstream feature then operates on facets.
func facetedBooleanRescue(op ops.PartFeatureOperation, target, tool, res *topo.Body, rec *diag.Recorder) (*topo.Body, bool) {
	if res == nil || len(res.Faces()) == 0 || ops.Validate(res).ValidSolid() {
		return nil, false
	}
	ft := facetedRescueDiag(target, "combine-target", rec)
	fb := facetedRescueDiag(tool, "combine-tool", rec)
	if ft == target && fb == tool {
		return nil, false // nothing to facet: the operands were already planar
	}
	rescued, err := ops.BooleanWithDiagnostics(op, ft, fb, rec)
	if err != nil || rescued == nil || !ops.Validate(rescued).ValidSolid() {
		return nil, false
	}
	return rescued, true
}

// appendCombined replaces the running target with the boolean result, dropping an empty result.
func appendCombined(running []*topo.Body, res *topo.Body) []*topo.Body {
	out := append([]*topo.Body(nil), running[:len(running)-1]...)
	if res != nil && len(res.Faces()) > 0 {
		out = append(out, res)
	}
	return out
}

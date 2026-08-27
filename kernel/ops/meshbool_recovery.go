// SPDX-License-Identifier: GPL-2.0-only

package ops

import (
	"oblikovati.org/kernel/diag"
	"oblikovati.org/kernel/geom"
	"oblikovati.org/kernel/topo"
)

// Reconstructed curved boolean (ADR-0054 Layer 5). When no exact analytic path in
// curvedExactPaths recognises a curved join — the #2167 cocylindrical arc-wall union is the
// motivating case — the operation would otherwise facet both operands and run the planar
// boolean, shattering the curved walls into mismatched grids (the visible seam). Instead the
// exact mesh-arrangement boolean (ADR-0052) runs on the operands' provenance-tagged
// tessellation and the result is rebuilt on the operands' EXACT surfaces (reconstructBoolean,
// ADR-0054 L1–L4). Wired as the LAST path in curvedExactBoolean, so it fires only where every
// analytic recognizer declined, and both the direct ops.Boolean path and the feature combine
// path (which tries CurvedBoolean on the still-analytic operands before faceting) pick it up.

// CodeBooleanAnalyticReconstruction marks a boolean that no analytic curved path recognised and
// provenance reconstruction (ADR-0054) rebuilt on the exact surfaces — an analytic B-rep in place
// of the faceted planar fallback, recorded so the pipeline can see reconstruction fired.
const CodeBooleanAnalyticReconstruction diag.Code = "boolean.analytic-reconstruction"

// reconstructionCutover gates whether reconstruction is offered as a curved-boolean path — the
// ADR-0054 Layer-5 switch, now ON (the default). Where it fires the reconstruction is exact
// (analytic surfaces from provenance, self-validated below for validity AND the Requicha volume
// bracket, with unweldable oblique-conic SSI declined by weldableSSICurve), upgrading a faceted
// curved join to an analytic B-rep; where reconstruction declines the caller falls back to the
// exact faceted boolean. Kept as a kill-switch during the alpha rollout: flip to false to route
// every curved boolean through the faceted fallback again.
const reconstructionCutover = true

// ReconstructionCutoverEnabled reports whether the ADR-0054 Layer-5 cutover is on, so a
// higher-layer regression that only fires through the reconstruction path (the #2167 piston
// feature test) can guard the seam-wrap layer without hard-coding the gate's state.
func ReconstructionCutoverEnabled() bool { return reconstructionCutover }

// reconstructedCurvedBoolean rebuilds `target op tool` from the exact mesh boolean's provenance and
// returns (body, true) only when reconstruction is enabled, applies (a curved operand, modest size),
// succeeds, and yields a valid solid whose volume sits inside the Requicha bracket — so a case
// reconstruction cannot yet close, or one it closes to the wrong volume, falls through to faceting.
func reconstructedCurvedBoolean(op PartFeatureOperation, target, tool *topo.Body, rec *diag.Recorder) (*topo.Body, bool) {
	if !reconstructionCutover {
		return nil, false
	}
	mop, ok := toMeshboolOp(op)
	if !ok || analyticFaceCount(target)+analyticFaceCount(tool) == 0 {
		return nil, false // no curved operand → the planar B-rep boolean is already exact
	}
	if len(target.Faces())+len(tool.Faces()) > csgFallbackFaceLimit {
		return nil, false // the exact mesh boolean is expensive; gate operand size as the fallbacks do
	}
	recon, ok := reconstructBoolean(target, tool, mop, DefaultQuality())
	if !ok || !validBooleanSolid(recon) {
		return nil, false
	}
	tv, wv, bv := boolVolumes(target, tool, recon)
	if volumeOutOfBracket(op, tv, wv, bv, curvedVolumeGuardFraction*max(tv, wv)) {
		return nil, false // reject a valid-but-wrong-volume reconstruction (the guard the analytic paths get)
	}
	rec.Recordf(CodeBooleanAnalyticReconstruction, diag.Info,
		"%s: no analytic curved path; rebuilt %d analytic faces from the exact boolean's provenance (#2153)", op, analyticFaceCount(recon))
	return recon, true
}

// analyticFaceCount counts a body's non-planar (analytic curved) faces.
func analyticFaceCount(b *topo.Body) int {
	n := 0
	for _, f := range b.Faces() {
		if _, ok := f.Geometry().(geom.Plane); !ok {
			n++
		}
	}
	return n
}

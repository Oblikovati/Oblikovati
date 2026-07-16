// SPDX-License-Identifier: GPL-2.0-only

package ops

// plateProvider is the M6 tier that will port OCCT's variational GeomPlate corner fill
// (docs/superpowers/sdd/plate-p0-brief.md): a tangent-degenerate valence-4 corner (N7's family,
// RailSignatureTangentPlate) has no single concurrent corner ball, so neither analyticSphere nor a
// plain Coons fill is the RIGHT surface — GeomPlate solves a thin-plate energy over the four fixed
// rails instead. THIS FILE IS THE STUB (Task P0): it wires the seam — Fits recognizes the family,
// Build always declines — so the tier slots into blendTiers() with ZERO behavioural change (the
// do-no-harm floor, ADR-0051): every loop still falls through to coons4 exactly as before. The
// plate math itself (P1-P4) lands in kernel/geom/plate_*.go, which this provider will call once it
// exists; per the engine dependency rule this file imports geom+math, NEVER topo (ADR-0051).
type plateProvider struct{}

var _ railProvider = plateProvider{}

// Name reports the provider's telemetry kind (never read by assembly; ADR-2 lineage invariance).
func (plateProvider) Name() CornerBlendKind { return BlendKindPlate }

// Fits claims ONLY a loop the extractor stamped RailSignatureTangentPlate AND that is valence-4 —
// Signature is the sole recognition signal (ADR-0051 M6): the provider never inspects loop shape or
// Side.Adjacent to guess the family, so a mis-shaped valence-4 loop that some future extractor
// forgets to stamp correctly falls through to coons4 instead of being mishandled here.
func (plateProvider) Fits(l RailLoop) bool {
	return l.Signature == RailSignatureTangentPlate && l.Valence() == 4
}

// Build is the P0 STUB: it always declines (ok=false) so resolveBlend's tier walk moves on to
// coons4 — no plate math exists yet (that is P1-P4). This is the do-no-harm floor (ADR-0051): a
// solver that doesn't exist can never fabricate a patch, so N7 stays exactly as coons4 leaves it
// today and the corpus is byte-identical (55) until the real solve lands at P5.
func (plateProvider) Build(RailLoop, Resolution) (CornerBlendPatch, Certificate, bool) {
	return CornerBlendPatch{}, Certificate{}, false
}

// SPDX-License-Identifier: GPL-2.0-only

package ops

// canalProvider is the M6' tier that will build OCCT's N7 corner face result_5 as what two spikes +
// a DRAWEXE pole dump proved it actually is: a rolling-ball CANAL surface (radius-r circular
// cross-sections swept along a ball-center offset-SSI spine), not the variational plate the M6 tier
// this slot replaces once assumed (blend-sweep-spike-report.md, canal-corner-math.md). A
// tangent-degenerate valence-4 corner (N7's family, RailLoop.Canal populated) has no single
// concurrent corner ball, so neither analyticSphere nor a plain Coons fill is the RIGHT surface —
// the canal spine (the two roll hosts' ±r offset intersection) is. THIS FILE IS THE STUB (Task C0):
// it wires the seam — Fits recognizes the family via the Canal payload pointer, Build always
// declines — so the tier slots into blendTiers() with ZERO behavioural change (the do-no-harm
// floor, ADR-0051/ADR-C2): every loop still falls through to coons4 exactly as before. The canal
// math itself (spine SSI, cross-section sweep, BSpline emission) lands in kernel/geom/canal_*.go
// (C1-C2), which this provider will call once it exists; per the engine dependency rule this file
// imports geom+math, NEVER topo (ADR-0051, ADR-C1).
type canalProvider struct{}

var _ railProvider = canalProvider{}

// Name reports the provider's telemetry kind (never read by assembly; ADR-2 lineage invariance).
func (canalProvider) Name() CornerBlendKind { return BlendKindCanal }

// Fits claims ONLY a loop the extractor populated Canal on AND that is valence-4 — the payload
// pointer is the sole recognition signal (ADR-C2, canal-corner-seam-architecture.md §2): the
// provider never inspects loop shape or Side.Adjacent to guess the family, so a mis-shaped
// valence-4 loop that some future extractor forgets to populate falls through to coons4 instead of
// being mishandled here. l.Canal != nil also guarantees the spine payload (Rolls/Radius) is
// present, so a fitting loop can never reach Build without the data it needs.
func (canalProvider) Fits(l RailLoop) bool {
	return l.Canal != nil && l.Valence() == 4
}

// Build is the C0 STUB: it always declines (ok=false) so resolveBlend's tier walk moves on to
// coons4 — no canal math exists yet (that is C1-C3). This is the do-no-harm floor (ADR-0051): a
// solver that doesn't exist can never fabricate a patch, so N7 stays exactly as coons4 leaves it
// today and the corpus is byte-identical (55) until the real solve lands at C3.
func (canalProvider) Build(RailLoop, Resolution) (CornerBlendPatch, Certificate, bool) {
	return CornerBlendPatch{}, Certificate{}, false
}

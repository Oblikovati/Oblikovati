// SPDX-License-Identifier: GPL-2.0-only

package ops

// blendTiers is the ADR-0051 foundation tier order: analytic-known-part first (exact sphere), then
// the M6 plate tier (a tangent-degenerate valence-4 corner GeomPlate can solve better than a plain
// Coons fill — RailSignatureTangentPlate-marked loops only; a stub that always declines until P5),
// then the general fills (4-sided coons4, 3-sided tri3). analyticTorus and nFan are deferred
// promotions (ADR-2) — they slot in ahead of coons4 / at the end when their recognition is
// oracle-grounded.
func blendTiers() []railProvider {
	return []railProvider{analyticSphereProvider{}, plateProvider{}, coons4Provider{}, tri3Provider{}}
}

// resolveBlend fills a RailLoop junction with the first tier whose provider Fits and returns a
// certificate-Valid patch; no such patch ⇒ ok=false and the caller honest-rejects (ADR-0051 ADR-3 /
// #1800). The analytic tiers come first, so a family that IS an exact known part wins over the
// general fill by ordering alone. NOTE: not yet called from computeCorners/solveCorner — the
// extractor-wiring phase does that; this wave only builds+tests the engine (corpus-neutral).
func resolveBlend(loop RailLoop, scale Resolution) (CornerBlendPatch, bool) {
	return resolveBlendWith(loop, scale, blendTiers())
}

// resolveBlendWith is the injectable core (the seam the ordering tests drive with fakes): it walks
// tiers in priority order and returns the first patch that both builds and passes its certificate,
// mirroring resolveCornerBlend's anti-crease guarantee for the RailLoop path.
func resolveBlendWith(loop RailLoop, scale Resolution, tiers []railProvider) (CornerBlendPatch, bool) {
	for _, p := range tiers {
		if !p.Fits(loop) {
			continue
		}
		if patch, cert, ok := p.Build(loop, scale); ok && cert.Valid(scale) {
			return patch, true
		}
	}
	return CornerBlendPatch{}, false
}

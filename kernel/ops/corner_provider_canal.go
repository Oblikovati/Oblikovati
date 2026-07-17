// SPDX-License-Identifier: GPL-2.0-only

package ops

import (
	stdmath "math"

	"oblikovati.org/kernel/geom"
	"oblikovati.org/math"
)

// canalProvider is the M6' tier that builds OCCT's N7 corner face result_5 as what two spikes + a
// DRAWEXE pole dump proved it actually is: a rolling-ball CANAL surface (radius-r circular cross-
// sections swept along a ball-center offset-SSI spine), not the variational plate the M6 tier this slot
// replaces once assumed (blend-sweep-spike-report.md, canal-corner-math.md). A tangent-degenerate
// valence-4 corner (N7's family, RailLoop.Canal populated) has no single concurrent corner ball, so
// neither analyticSphere nor a plain Coons fill is the RIGHT surface — the canal spine (the two roll
// hosts' ±r offset intersection) is. Task C3 wires the real Build: Fits recognizes the family via the
// Canal payload pointer; Build composes the geom canal (kernel/geom/CanalCornerFill: spine SSI + cross-
// section loft), certifies it, and emits it on the received rails, or honest-rejects to coons4 (do-no-
// harm floor, ADR-0051/ADR-C2). Per the engine dependency rule this file imports geom+math, NEVER topo
// (ADR-0051, ADR-C1); the certificate reuses obstacleNoFold/creaseAngle from the shared certify layer.
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

// Build maps the canal payload (roll hosts + radius + reflected-family ends) and the four received
// rails into geom.CanalCornerFill (spine + loft), certifies the emitted surface, and returns it on the
// RECEIVED rails (railLoopToFilletLoops) for a watertight weld — exactly as coons4 does. Any geom
// error (offset self-intersect, non-convergent SSI, B3 point-spine, off-radius station, or the rail
// self-check) OR a mis-shaped loop → honest-reject (ok=false) → resolveBlend falls through to coons4
// (ADR-0051 do-no-harm: N7 can only improve to the canal or fall back to today's coons4, never
// regress). The area 90.194 is an emergent property of the surface, NEVER a cert field (no magic
// scalar in the gate — the seam architecture's load-bearing constraint).
func (canalProvider) Build(l RailLoop, res Resolution) (CornerBlendPatch, Certificate, bool) {
	if l.Canal == nil || l.Valence() != 4 {
		return CornerBlendPatch{}, Certificate{}, false
	}
	rails := [4]geom.Curve3{l.Sides[0].Curve, l.Sides[1].Curve, l.Sides[2].Curve, l.Sides[3].Curve}
	surf, err := geom.CanalCornerFill(l.Canal.Rolls, l.Canal.Radius, rails, l.Canal.Ends, res)
	if err != nil {
		return CornerBlendPatch{}, Certificate{}, false
	}
	patch := CornerBlendPatch{Surface: surf, Loops: railLoopToFilletLoops(l), Kind: BlendKindCanal}
	return patch, certifyCanalPatch(surf, l, res), true
}

// canalCertSamples / canalFootSamples are the per-rail and per-foot-locus scan densities for the canal
// G0/G1 measures (the rails are short radius-r arcs; the foot-loci are gentle on-host curves).
const (
	canalCertSamples = 16
	canalFootSamples = 24
)

// certifyCanalPatch proves the rolling-ball canal patch (ADR-3) through the canal's OWN two guarantees,
// NOT the received coons4 rails: (1) it INTERPOLATES the two fillet-arm END cross-sections — its
// v=0/v=1 boundaries, the rails whose arc centre is a spine End — and welds G1 to those arms; (2) its
// u=0/u=1 FOOT-LOCI lie ON the two roll hosts and are tangent to them (G1). The OTHER two received
// rails — the on-wall bridge E2 and the mid-arm PLANAR cross-section amid — are coons4's approximations
// of the foot-loci, up to 0.28 off the true rolling-ball surface (the spine dips ~0.28 in y that a
// planar rail cannot follow); measuring the canal against them would falsely reject a correct patch
// (seam brief: "only the two end cross-section rails match; the 0.557 residual is interior iso-
// parametrization, not the boundary"). This mirrors the obstacle patch excluding its free G0 rim.
// Closed from the loop; WeldsArms structural; NoFold via the shared column sweep; area 90.194 is a
// test oracle, never a cert field (no magic scalar in the gate).
func certifyCanalPatch(surf geom.BSplineSurface, loop RailLoop, scale Resolution) Certificate {
	devEnds, creaseEnds := canalEndWelds(surf, loop, scale.Weld())
	devFeet, creaseFeet := canalFootLoci(surf, loop.Canal.Rolls)
	return Certificate{
		Closed:      loop.Closed(scale.Weld()),
		WeldsArms:   true,
		NoFold:      obstacleNoFold(surf, scale),
		MaxDev:      stdmath.Max(devEnds, devFeet),
		MaxAngleDev: stdmath.Max(creaseEnds, creaseFeet),
	}
}

// canalEndWelds measures the canal's weld to its fillet arms along the two END cross-section rails (the
// v=0/v=1 boundaries the loft interpolates exactly): the max on-surface G0 residual and the max G1
// crease to each rail's Adjacent arm. Returns +Inf if it cannot find EXACTLY the two end rails (a
// malformed loop → the certificate rejects, honest-reject to coons4).
func canalEndWelds(surf geom.BSplineSurface, loop RailLoop, weld float64) (dev, crease float64) {
	n := 0
	for _, s := range loop.Sides {
		arc, ok := s.Curve.(geom.Arc3d)
		if !ok || !centreIsSpineEnd(arc.Center, loop.Canal.Ends, weld) {
			continue
		}
		n++
		dev = stdmath.Max(dev, canalRailOnSurface(surf, s.Curve))
		crease = stdmath.Max(crease, canalSideCrease(surf, s))
	}
	if n != 2 {
		return stdmath.Inf(1), stdmath.Inf(1)
	}
	return dev, crease
}

// centreIsSpineEnd reports whether an arc centre coincides (within tol) with a spine end — the test
// that picks the two END cross-section rails out of the loop (the mid-arm arc is centred at the mid
// reflected centre, 10 away from either end for N7, so it is excluded).
func centreIsSpineEnd(centre math.Point3, ends [2]math.Point3, tol float64) bool {
	for _, e := range ends {
		if float64(centre.DistanceTo(e)) <= tol {
			return true
		}
	}
	return false
}

// canalRailOnSurface is the max distance from sampled points of one end cross-section rail to the canal
// surface (true nearest point) — ~0 since that rail IS a v-boundary isoparm the loft interpolates.
func canalRailOnSurface(surf geom.BSplineSurface, rail geom.Curve3) float64 {
	m := 0.0
	for _, p := range sampleCurveN(rail, canalCertSamples, false) {
		_, _, d := geom.ProjectPointToSurface(surf, p)
		m = stdmath.Max(m, d)
	}
	return m
}

// canalSideCrease is the max crease angle between the canal normal and side s's Adjacent-arm normal at
// sampled interior points of the shared rail (the shared corner vertex — where the arm tangent planes
// conflict — is excluded, as the obstacle patch excludes its corner band).
func canalSideCrease(surf geom.BSplineSurface, s Side) float64 {
	m := 0.0
	for i, p := range sampleCurveN(s.Curve, canalCertSamples, false) {
		if i == 0 || s.Adjacent == nil {
			continue // skip the shared corner vertex (Adjacent tangent planes conflict there)
		}
		u, v, _ := geom.ProjectPointToSurface(surf, p)
		du, dv := surf.DerivativesAt(u, v)
		au, av := s.Adjacent.ParamAt(p)
		m = stdmath.Max(m, creaseAngle(du.Cross(dv), s.Adjacent.NormalAt(au, av)))
	}
	return m
}

// canalFootLoci measures the canal's u=0 / u=1 iso-boundaries (its FOOT-LOCI on the two roll hosts):
// the max on-host G0 distance and the max G1 crease between the canal normal and the host normal along
// each locus. loftCanal builds every foot ON its host, so both are near-zero — this witnesses the host
// tangency independently (the do-no-harm feet-at-radius guard is upstream; this is the runtime proof).
func canalFootLoci(surf geom.BSplineSurface, rolls []geom.Surface) (dev, crease float64) {
	if len(rolls) != 2 {
		return stdmath.Inf(1), stdmath.Inf(1)
	}
	u0, u1 := surf.UDomain()
	d0, c0 := footLocusOnHost(surf, u0, rolls[0])
	d1, c1 := footLocusOnHost(surf, u1, rolls[1])
	return stdmath.Max(d0, d1), stdmath.Max(c0, c1)
}

// footLocusOnHost scans one u-edge isoparm (v across its domain) and returns the max distance to host
// and the max crease between the canal normal and the host normal there.
func footLocusOnHost(surf geom.BSplineSurface, uEdge float64, host geom.Surface) (dev, crease float64) {
	v0, v1 := surf.VDomain()
	for j := 0; j <= canalFootSamples; j++ {
		v := v0 + float64(j)/float64(canalFootSamples)*(v1-v0)
		p := surf.PointAt(uEdge, v)
		hu, hv := host.ParamAt(p)
		dev = stdmath.Max(dev, float64(host.PointAt(hu, hv).DistanceTo(p)))
		du, dv := surf.DerivativesAt(uEdge, v)
		crease = stdmath.Max(crease, creaseAngle(du.Cross(dv), host.NormalAt(hu, hv)))
	}
	return dev, crease
}

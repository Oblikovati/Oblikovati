// SPDX-License-Identifier: GPL-2.0-only

package blend

import (
	"fmt"
	stdmath "math"

	"oblikovati.org/kernel/geom"
	"oblikovati.org/kernel/ops/internal/probe"
	opstol "oblikovati.org/kernel/ops/internal/tol"
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
// section loft), certifies it, and emits it on the canal's OWN boundary isoparms (canalPatchLoops), or
// honest-rejects to coons4 (do-no-harm floor, ADR-0051/ADR-C2). Per the engine dependency rule this
// file imports geom+math, NEVER topo
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
// canal's OWN four boundary isoparms (canalPatchLoops) for a watertight weld. It CANNOT reuse the
// received rails as coons4 does: one received rail, the mid-arm cross-section amid, sits ~0.28 off the
// canal surface (its centre C′ is not a canal boundary — the canal spans C→C″), so emitting it would
// produce a MALFORMED face (a trim edge 0.28 off its own surface). The canal's own boundary isocurves
// lie ON the surface by construction, and assertLoopsOnCanal GATES that before returning (the check the
// C3 certificate was silent on). Any geom error (offset self-intersect, non-convergent SSI, B3 point-
// spine, off-radius station, the rail self-check) OR a loop off the surface → honest-reject (ok=false)
// → resolveBlend falls through to coons4 (ADR-0051 do-no-harm: N7 can only improve to the canal or
// fall back to today's coons4, never regress). The area 90.194 is an emergent property of the surface,
// NEVER a cert field (no magic scalar in the gate — the seam architecture's load-bearing constraint).
func (canalProvider) Build(l RailLoop, res opstol.Resolution) (CornerBlendPatch, Certificate, bool) {
	if l.Canal == nil || l.Valence() != 4 {
		return CornerBlendPatch{}, Certificate{}, false
	}
	rails := [4]geom.Curve3{l.Sides[0].Curve, l.Sides[1].Curve, l.Sides[2].Curve, l.Sides[3].Curve}
	surf, err := geom.CanalCornerFill(l.Canal.Rolls, l.Canal.Radius, rails, l.Canal.Ends, res)
	if err != nil {
		return CornerBlendPatch{}, Certificate{}, false
	}
	loops, err := canalPatchLoops(surf)
	if err != nil {
		return CornerBlendPatch{}, Certificate{}, false
	}
	if err := assertLoopsOnCanal(surf, loops, res.Weld()); err != nil {
		return CornerBlendPatch{}, Certificate{}, false
	}
	patch := CornerBlendPatch{Surface: surf, Loops: loops, Kind: BlendKindCanal}
	return patch, certifyCanalPatch(surf, l, res), true
}

// canalRingSide is one boundary isocurve of the canal patch plus whether the assembly ring traverses
// it reversed, so the four sides chain end-to-end into a single non-self-crossing closed loop.
type canalRingSide struct {
	curve geom.Curve3
	rev   bool
}

// canalPatchLoops traces the canal surface's OWN four boundary isocurves into ONE closed assembly-ready
// loop, mirroring railLoopToFilletLoops' open-sample-then-concatenate convention (each side sampled
// OPEN so consecutive sides share a corner without duplication). This is the C3-review fix: the
// received rails include amid, the mid-arm cross-section centred at C′, which sits ~0.28 off the canal
// surface (C′ is NOT a canal boundary — the canal spans C→C″), so emitting the rails yields a malformed
// face. The four isocurves lie on the surface by construction. The two v-boundaries (the end cross-
// section arcs) ARE the received a0/a1 rails already, so only the two u-boundaries switch from the
// wrong received [amid, E2] to the true on-host foot-loci — the whole point of the fix.
func canalPatchLoops(surf geom.BSplineSurface) ([]filletLoop, error) {
	sides, err := canalBoundaryIsocurves(surf)
	if err != nil {
		return nil, err
	}
	var pts []math.Point3
	var curves []geom.Curve3
	for _, s := range sides {
		// Each boundary sub-edge carries its curve TRIMMED to its own sub-span (not the whole
		// isocurve), so the patch loop is a simple polygon the NURBS mesher tiles fold-free instead
		// of a self-overlapping ring (N7 fold cure; n7-tessellation-diagnosis.md §3).
		p, cv := sampleCurve3OpenTrimmed(s.curve, s.rev)
		pts = append(pts, p...)
		curves = append(curves, cv...)
	}
	return []filletLoop{{pts: pts, curves: curves}}, nil
}

// canalBoundaryIsocurves extracts the canal patch's four boundary isocurves in CLOSED-ring order:
// v=v0 (forward) → u=u1 (forward) → v=v1 (reversed) → u=u0 (reversed). geom.SurfaceIsoCurve with
// uDirection=false fixes v and runs the curve along u (an end cross-section arc); uDirection=true fixes
// u and runs it along v (a foot-locus). The reversal flags chain the sides: v=v0 ends at (u1,v0) where
// u=u1 begins; u=u1 ends at (u1,v1) where v=v1-reversed begins; v=v1-reversed ends at (u0,v1) where
// u=u0-reversed begins and closes back to (u0,v0). Any degenerate-parameter iso extraction errors →
// the caller honest-rejects to coons4.
func canalBoundaryIsocurves(surf geom.BSplineSurface) ([]canalRingSide, error) {
	u0, u1 := surf.UDomain()
	v0, v1 := surf.VDomain()
	specs := [4]struct {
		uDirection bool
		param      float64
		rev        bool
	}{
		{false, v0, false}, // v=v0 end cross-section arc, forward along u
		{true, u1, false},  // u=u1 foot-locus, forward along v
		{false, v1, true},  // v=v1 end cross-section arc, reversed along u
		{true, u0, true},   // u=u0 foot-locus, reversed along v
	}
	sides := make([]canalRingSide, len(specs))
	for i, s := range specs {
		c, err := geom.SurfaceIsoCurve(surf, s.uDirection, s.param)
		if err != nil {
			return nil, fmt.Errorf("canalPatchLoops: boundary isocurve (uDirection=%v, param=%g): %w", s.uDirection, s.param, err)
		}
		sides[i] = canalRingSide{curve: c, rev: s.rev}
	}
	return sides, nil
}

// assertLoopsOnCanal is the watertight SELF-CHECK the C3 certificate was silent on: every emitted loop
// point must lie ON the canal surface within weld (nearest-point distance via ClosestPointOnSurface).
// For the canal's OWN boundary isocurves this holds BY CONSTRUCTION, so it passes — but making it a
// Build-time GATE means a future regression (or re-introducing the received rails, whose amid rail sits
// ~0.28 off-surface) is an honest reject, not a silently malformed face. It errors carrying the max
// off-surface distance and the weld bound so the reject is diagnosable.
func assertLoopsOnCanal(surf geom.BSplineSurface, loops []filletLoop, weld float64) error {
	if maxDev := maxLoopSurfaceDev(surf, loops); maxDev > weld {
		return fmt.Errorf("canalPatchLoops: max loop-to-surface distance %g exceeds weld %g (loops not on the canal surface)", maxDev, weld)
	}
	return nil
}

// maxLoopSurfaceDev is the max nearest-point distance from any emitted loop vertex to surf — the G0
// on-surface residual. Shared by assertLoopsOnCanal (the Build-time gate) and canalStationProvider's
// certificate (its MaxDev field), so the measured deviation is computed once, not duplicated.
func maxLoopSurfaceDev(surf geom.BSplineSurface, loops []filletLoop) float64 {
	maxDev := 0.0
	for _, l := range loops {
		for _, p := range l.pts {
			_, _, foot := geom.ClosestPointOnSurface(surf, p)
			maxDev = stdmath.Max(maxDev, float64(foot.DistanceTo(p)))
		}
	}
	return maxDev
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
// of the foot-loci; the C3-review measurement showed amid sits ~0.28 off the true rolling-ball surface
// (the spine dips ~0.28 in y that amid's planar rail cannot follow) while E2 is ~7.6e-4 off (nearly on
// it) — so measuring the canal against amid would falsely reject a correct patch (seam brief: "only the
// two end cross-section rails match; the 0.557 residual is interior iso-parametrization, not the
// boundary"). Build therefore emits the canal's OWN boundary isoparms (canalPatchLoops), never these
// received foot-locus approximations. This mirrors the obstacle patch excluding its free G0 rim.
// Closed from the loop; WeldsArms structural; NoFold via the shared column sweep; area 90.194 is a
// test oracle, never a cert field (no magic scalar in the gate).
func certifyCanalPatch(surf geom.BSplineSurface, loop RailLoop, scale opstol.Resolution) Certificate {
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
		m = stdmath.Max(m, probe.CreaseAngle(du.Cross(dv), s.Adjacent.NormalAt(au, av)))
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
		crease = stdmath.Max(crease, probe.CreaseAngle(du.Cross(dv), host.NormalAt(hu, hv)))
	}
	return dev, crease
}

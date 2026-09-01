// SPDX-License-Identifier: GPL-2.0-only

package blend

import (
	"oblikovati.org/kernel/geom"
	"oblikovati.org/kernel/ops/internal/tol"
	"oblikovati.org/math"
)

// CanalStationFill carries the EXACT rolling-ball cross-section stations the canalStationProvider skins
// into the faithful dual-host CORE panel (U4-4b, #2007 Group C; derivation §1.3/§2.3). U4's setback
// surface is a rolling-ball CANAL — OCCT's section is an exact radius-5 arc at every z, only the centre
// migrating (U4-2). The plain coons4 core fill missed the oracle area by ~4.5% because all four sides
// are G0: nothing pulls the transfinite Coons blend onto the true canal surface. Skinning the panel
// through the EXACT arc stations (centre + both host feet per z, each from setbackSection's own per-z
// construction) fixes that — geom.LoftCanalStations lofts a constant-radius BSpline through them and
// self-asserts each foot at radius. nil for every non-core loop (the provider then declines), so the
// corpus is unaffected — it mirrors RailLoop.Canal, a nilable provider-scoped payload only its extractor
// (buildCoreLoop) sets and only its provider (canalStationProvider) reads.
type CanalStationFill struct {
	// Centers are the radius-Radius arc centres (the rolling-ball spine) at each z station.
	Centers []math.Point3
	// FeetA / FeetB are the boss-A / boss-B rim feet at each z station — the two ends of the exact
	// cross-section arc, taken from setbackSection's own endpoints so the loft welds bit-identically to
	// the seam rails at the shared stations (z=±6.240 sliver-core weld, z=0 core-core weld).
	FeetA, FeetB []math.Point3
	// Radius is the rolling-ball radius r (=ef.cyl.Radius) — LoftCanalStations asserts every foot sits
	// exactly r from its centre (a built-in fidelity gate), so a mis-supplied station is declined.
	Radius float64
}

// canalStationProvider is the U4-4b tier that skins a dual-host CORE panel as the FAITHFUL rolling-ball
// canal loft through its exact cross-section stations, replacing the coons4 fill's ~4.5% area miss.
// Unlike canalProvider (which MARCHES an offset-SSI spine and projects feet — the N7 corner family),
// this consumes stations whose centre and BOTH feet are closed-form exact (setbackSection per z), so
// there is no SSI/convergence story — the only approximation is the v-interpolation BETWEEN exact
// stations, which converges to <0.01% of the oracle by K=9 (the stations are exact). It sits ahead of
// coons4 in blendTiers; the Stations payload pointer is the sole recognition signal, so every non-core
// loop falls through to coons4 unchanged. Per the engine dependency rule it imports geom+math, never
// topo (ADR-0051).
type canalStationProvider struct{}

var _ railProvider = canalStationProvider{}

// Name reports the provider's telemetry kind (never read by assembly; ADR-2 lineage invariance).
func (canalStationProvider) Name() CornerBlendKind { return BlendKindCanalStation }

// Fits claims ONLY a loop the extractor populated Stations on AND that is valence-4 — the payload
// pointer is the sole recognition signal (mirroring canalProvider.Fits), so a non-core loop is never
// mishandled here: it falls through to coons4. A non-nil Stations also guarantees the centre/feet rows
// are present, so a fitting loop can never reach Build without the data LoftCanalStations needs.
func (canalStationProvider) Fits(l RailLoop) bool {
	return l.Stations != nil && l.Valence() == 4
}

// Build lofts the exact stations into the constant-radius canal BSpline (geom.LoftCanalStations, which
// asserts each foot at radius — the do-no-harm fidelity gate), then emits the surface's OWN four
// boundary isoparms as the patch loops (canalPatchLoops): the two v-boundaries ARE the end
// cross-section arcs (bit-identical corners to setbackSection at those stations, the seam weld) and the
// two u-boundaries are the on-host foot-loci — all lie ON the surface by construction, unlike the coons4
// rails. Any geom error (off-radius station, degenerate arc) OR a loop off the surface → honest-reject
// (ok=false) → resolveBlend falls through to coons4 (ADR-0051 do-no-harm). The core panel is all-G0
// (derivation §1.4: no analytic Adjacent to be G1 against), so the certificate's angular residual is 0.
func (canalStationProvider) Build(l RailLoop, res tol.Resolution) (CornerBlendPatch, Certificate, bool) {
	if l.Stations == nil || l.Valence() != 4 {
		return CornerBlendPatch{}, Certificate{}, false
	}
	sf := l.Stations
	surf, err := geom.LoftCanalStations(sf.Centers, sf.FeetA, sf.FeetB, sf.Radius, res.Weld())
	if err != nil {
		return CornerBlendPatch{}, Certificate{}, false
	}
	loops, err := canalPatchLoops(surf)
	if err != nil {
		return CornerBlendPatch{}, Certificate{}, false
	}
	dev := maxLoopSurfaceDev(surf, loops)
	if dev > res.Weld() {
		return CornerBlendPatch{}, Certificate{}, false
	}
	patch := CornerBlendPatch{Surface: surf, Loops: loops, Kind: BlendKindCanalStation}
	return patch, certifyCanalStationPatch(surf, l, dev, res), true
}

// certifyCanalStationPatch proves the exact-station canal loft: Closed from the received rail loop (the
// four seam+rim rails already weld into a cycle — the corner tests pin it), WeldsArms structural, NoFold
// via the shared column sweep, MaxDev the measured loop-to-surface G0 residual (the v/u boundary
// isoparms lie on the surface by construction, so ~0), and MaxAngleDev 0 — a CORE panel has no analytic
// Adjacent surface to hold G1 against (derivation §1.4 core table: all four sides G0), so there is no
// tangent-plane agreement to measure. The oracle area (30.334 per split half) is an emergent property of
// the surface, verified by the regression test — never a cert field.
func certifyCanalStationPatch(surf geom.BSplineSurface, loop RailLoop, dev float64, res tol.Resolution) Certificate {
	return Certificate{
		Closed:      loop.Closed(res.Weld()),
		WeldsArms:   true,
		NoFold:      obstacleNoFold(surf, res),
		MaxDev:      dev,
		MaxAngleDev: 0,
	}
}

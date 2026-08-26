// SPDX-License-Identifier: GPL-2.0-only

package ops

import (
	"slices"

	"oblikovati.org/kernel/geom"
)

// obstacleCanalProvider skins a MID-SPAN OBSTACLE band as the TRUE rolling-ball envelope — an
// exact-station canal loft (geom.LoftCanalStations) through the closed-form surf-rst stations
// fillet_obstacle_canal.go solved — replacing the single Coons FillSurface bsplineObstacleProvider
// builds over the same four rails.
//
// WHY. A Coons fill enforces the four boundary curves and lets the interior go where the blend of the
// rails puts it; a canal GUARANTEES the spherical envelope. Measured against live DRAWEXE per-face
// `sprops`, the Coons patch over-reads by +7.19 % (R9), +9.12 % (S3), +13.30 % (T6), +9.54 % (U3) and
// +19.72 % (X3), one to two orders larger than any other per-face error in the obstacle class, and its
// straight wall rail leaves the WALL face short by the whole bulge (−0.717 / −4.625 / −9.477 / −2.998 /
// −147.35 absolute). Both defects are the same one: the ball does not stay on the plain fillet axis over
// the dip. See obstacleCanal's doc comment for the derivation and OCCT's own pole-net receipts.
//
// It sits AHEAD of bsplineObstacleProvider and keys on the ObstacleFeature.Canal payload alone, so an
// obstacle whose stations could not be solved falls through to the Coons model untouched.
type obstacleCanalProvider struct{}

var _ CornerBlendProvider = obstacleCanalProvider{}

// Name reports the provider's telemetry kind (never read by assembly; ADR-2 lineage invariance).
func (obstacleCanalProvider) Name() CornerBlendKind { return BlendKindObstacleCanal }

// Fits claims only an obstacle request carrying the surf-rst station payload.
func (obstacleCanalProvider) Fits(req CornerBlendRequest) bool {
	return req.ObstacleFeature != nil && req.ObstacleFeature.Canal != nil
}

// Build lofts the exact stations and certifies the result. Any geom error (an off-radius station, a
// degenerate section arc) or a boundary that does not lie on the lofted surface within weld is an honest
// reject (ok=false) → resolveObstaclePatch drops the payload and the straight-seam Coons model is tried
// instead, wall front included (ADR-3 do-no-harm).
func (obstacleCanalProvider) Build(req CornerBlendRequest) (CornerBlendPatch, Certificate, bool) {
	of := req.ObstacleFeature
	c := of.Canal
	surf, err := geom.LoftCanalStations(c.Centres, c.FeetRim, c.FeetWall, c.Envelope.Radius, req.Setback.Weld())
	if err != nil {
		return CornerBlendPatch{}, Certificate{}, false
	}
	loops := obstacleCanalPatchLoops(of)
	dev := maxLoopSurfaceDev(surf, loops)
	if dev > req.Setback.Weld() {
		return CornerBlendPatch{}, Certificate{}, false
	}
	cert := certifyObstacleCanalPatch(surf, of, dev, req.Setback)
	return CornerBlendPatch{Surface: surf, Loops: loops, Kind: BlendKindObstacleCanal}, cert, true
}

// obstacleCanalPatchLoops traces the patch boundary as a ring of the loft's OWN station rows: the wall
// foot locus A→D, the rim feet reversed P+→P−, and the two end section arcs. It is the provider-side
// placeholder (buildPatchFace rebuilds the assembly loop with the shared curve identities, exactly as it
// already does for the Coons tier); its purpose here is the MaxDev measurement, which is what proves the
// emitted boundary lies on the lofted surface.
func obstacleCanalPatchLoops(of *ObstacleFeature) []filletLoop {
	c := of.Canal
	var loop filletLoop
	for _, p := range c.FeetWall {
		loop.add(p, nil)
	}
	for _, v := range slices.Backward(c.FeetRim) {
		loop.add(v, nil)
	}
	return []filletLoop{loop}
}

// certifyObstacleCanalPatch proves the obstacle canal. What each field does and does NOT prove:
//
//   - MaxDev is the emitted boundary SAMPLE POINTS against the lofted surface. It is ~1e-14 BY
//     CONSTRUCTION (every boundary point is a station foot and the loft interpolates its stations
//     exactly); what it genuinely catches is the boundary and the surface coming APART — a station row
//     that lost an entry, a pinned end that no longer matches the wing.
//   - MaxAngleDev is the crease between the loft and the fillet WALL along the u-boundary isoparm that
//     lies on it — the G1 the wall face actually needs. The RIM side is intentionally G0 (a sharp
//     base-rim crease) and tangentHostCrease measures only declared Tangents, so it is excluded for
//     free rather than by a special case.
//   - MaxBallDev is the only field that says anything about the patch INTERIOR, and it is exactly the
//     condition a Coons fill through these rails cannot pass. Measured on the five corpus cases with the
//     SAME declared envelope, the Coons obstacle patch reads 2.95 / 7.59 / 6.10 / 5.28 / 23.90 model units
//     (R9/S3/T6/U3/X3) — 0.95 r … 1.06 r, a categorical reject rather than a small deviation, because the
//     section-plane centre solve is near-tangential and amplifies the interior displacement — against
//     6.16e-09 … 1.10e-08 for this loft, inside every one of those cases' welds. Note the obstacle patch
//     is a FULL-WIDTH band, so it reads far worse here than the 1.06e-02 U4's Coons SLIVER read.
//     It is a self-consistency measure (see Certificate.MaxBallDev) — the proof that the declared envelope
//     is the right MODEL is the closed-form per-face pin in
//     model/feature/occtparity/obstacle_canal_oracle_test.go, which is grounded on live DRAWEXE.
//   - Closed/NoFold are structural; WeldsArms is vacuous here (an obstacle band has no arms), exactly as
//     in certifyObstaclePatch.
func certifyObstacleCanalPatch(surf geom.BSplineSurface, of *ObstacleFeature, dev float64, res Resolution) Certificate {
	return Certificate{
		Closed:      true,
		WeldsArms:   true,
		NoFold:      obstacleNoFold(surf, res),
		MaxDev:      dev,
		MaxAngleDev: tangentHostCrease(surf, of.Canal.Envelope, res),
		MaxBallDev:  maxBallDev(surf, &of.Canal.Envelope),
	}
}

// SPDX-License-Identifier: GPL-2.0-only

package ops

import (
	stdmath "math"

	"oblikovati.org/kernel/geom"
	"oblikovati.org/math"
)

// runoutCanalProvider skins a SETBACK-CLOSE run-out band as the TRUE rolling-ball envelope: an
// exact-station canal loft (geom.LoftCanalStations) through the closed-form stations
// fillet_runout_band.go resolved, replacing the Coons fill this family used to get.
//
// Why a provider of its own rather than canalStationProvider: that tier emits the lofted surface's
// OWN boundary isoparms as the patch loops, which is right for U4's core panel (its neighbours are
// built from the same stations) but wrong here — a run-out band's neighbours are the INTACT boss wall
// rim, the re-clipped host notch and the plain fillet wing, all of which tile the extractor's
// ANALYTIC rails at ringSegSamples. So this tier emits the received rails (railLoopToFilletLoops,
// exactly as coons4 did, keeping the weld mechanism unchanged) and the station grid is built so every
// one of those rail samples IS a station foot — which turns MaxDev from the tautology
// coons4-audit.md §C.3 indicts into a genuine rail-vs-surface measurement.
//
// It sits ahead of coons4 in blendTiers and keys on the RailLoop.Runout pointer alone, so every loop
// that does not carry the payload falls through unchanged.
type runoutCanalProvider struct{}

var _ railProvider = runoutCanalProvider{}

// Name reports the provider's telemetry kind (never read by assembly; ADR-2 lineage invariance).
func (runoutCanalProvider) Name() CornerBlendKind { return BlendKindRunoutCanal }

// Fits claims ONLY a valence-4 loop the setback extractor populated Runout on — the payload pointer
// is the sole recognition signal, mirroring canalProvider/canalStationProvider.
func (runoutCanalProvider) Fits(l RailLoop) bool { return l.Runout != nil && l.Valence() == 4 }

// Build lofts the exact stations and certifies the result against the RECEIVED rails. Any geom error
// (an off-radius station, a degenerate section arc) or a rail that does not lie on the lofted surface
// within weld is an honest reject (ok=false) → resolveBlend falls through, and the setback closer
// then rejects the whole edge rather than shipping a wrong surface (ADR-3 do-no-harm).
func (runoutCanalProvider) Build(l RailLoop, res Resolution) (CornerBlendPatch, Certificate, bool) {
	if l.Runout == nil || l.Valence() != 4 {
		return CornerBlendPatch{}, Certificate{}, false
	}
	rc := l.Runout
	surf, err := geom.LoftCanalStations(rc.Centers, rc.FeetA, rc.FeetB, rc.Envelope.Radius, res.Weld())
	if err != nil {
		return CornerBlendPatch{}, Certificate{}, false
	}
	loops := runoutPatchLoops(l)
	dev := maxLoopSurfaceDev(surf, loops)
	if dev > res.Weld() {
		return CornerBlendPatch{}, Certificate{}, false
	}
	patch := CornerBlendPatch{Surface: surf, Loops: loops, Kind: BlendKindRunoutCanal}
	return patch, certifyRunoutCanalPatch(surf, l, dev, res), true
}

// runoutPatchLoops traces the RECEIVED rails into one assembly-ready boundary loop whose sample points
// are byte-identical to railLoopToFilletLoops' (so the wall rim / host notch / wing welds are unchanged)
// but where each sub-edge carries its rail curve TRIMMED to its own sub-span. The trim is load-bearing
// on a canal: with the whole rail attached to every sub-edge the shared tessellator (sampleEdgeCurve)
// sweeps the entire rail once per sub-edge, the boundary self-overlaps, and the NURBS patch mesher folds
// — measured here as a 158.14 mesh area over a surface whose true area is 26.5949. This is the same N7
// defect canalPatchLoops already cures (n7-tessellation-diagnosis.md §3); the Coons fill this tier
// replaces happened to survive it, a canal does not.
func runoutPatchLoops(loop RailLoop) []filletLoop {
	var pts []math.Point3
	var curves []geom.Curve3
	for _, side := range loop.Sides {
		p, cv := sampleCurve3OpenTrimmed(side.Curve, false)
		pts = append(pts, p...)
		curves = append(curves, cv...)
	}
	return []filletLoop{{pts: pts, curves: curves}}
}

// certifyRunoutCanalPatch proves the run-out canal. Unlike the coons4 certificate it indicts, every
// field here measures geometry the surface does NOT own:
//   - MaxDev is the extractor's ANALYTIC rails against the lofted surface (the rails are footprint
//     conics and contact loci, not the surface's isoparms), so it can fail;
//   - MaxAngleDev is the crease between the loft and each TANGENT host along the boundary isoparm
//     that lies on it — the G1 the neighbouring host face actually needs;
//   - MaxBallDev is the INTERIOR envelope residual against the extractor's own declared hosts;
//   - Closed/WeldsArms/NoFold are structural, as everywhere else.
func certifyRunoutCanalPatch(surf geom.BSplineSurface, loop RailLoop, dev float64, res Resolution) Certificate {
	return Certificate{
		Closed:      loop.Closed(res.Weld()),
		WeldsArms:   true,
		NoFold:      obstacleNoFold(surf, res),
		MaxDev:      dev,
		MaxAngleDev: tangentHostCrease(surf, loop.Runout.Envelope, res),
		MaxBallDev:  maxBallDev(surf, loop.Envelope),
	}
}

// tangentHostCrease is the max angle between the canal's normal and a TANGENT host's normal, measured
// along whichever u-boundary isoparm lies on that host. A rolling-ball envelope is tangent to its roll
// host by definition, so a non-zero reading here is a real defect — and because the host is supplied
// by the extractor (never guessed), the measurement is honest. 0 when the band has no tangent host
// (the two-boss CENTRAL band passes through two restriction curves and touches no plane).
func tangentHostCrease(surf geom.BSplineSurface, env BallEnvelope, res Resolution) float64 {
	worst := 0.0
	u0, u1 := surf.UDomain()
	for _, host := range env.Tangents {
		for _, u := range []float64{u0, u1} {
			if c, ok := isoCreaseAgainstHost(surf, host, u, res); ok {
				worst = stdmath.Max(worst, c)
			}
		}
	}
	return worst
}

// isoCreaseAgainstHost measures the tangent-plane disagreement along the u=const boundary isoparm, and
// reports ok=false when that isoparm is NOT on the host (the other side of the band) — so each host is
// measured against its own foot locus and never against the opposite rail.
func isoCreaseAgainstHost(surf geom.BSplineSurface, host geom.Surface, u float64, res Resolution) (float64, bool) {
	v0, v1 := surf.VDomain()
	worst := 0.0
	for i := 0; i <= runoutCreaseSamples; i++ {
		v := v0 + (v1-v0)*float64(i)/runoutCreaseSamples
		p := surf.PointAt(u, v)
		hu, hv, foot := geom.ClosestPointOnSurface(host, p)
		if float64(foot.DistanceTo(p)) > res.Weld() {
			return 0, false // this boundary is not the host's foot locus
		}
		a := surf.NormalAt(u, v)
		b := host.NormalAt(hu, hv)
		worst = stdmath.Max(worst, unsignedNormalAngle(a, b))
	}
	return worst, true
}

// unsignedNormalAngle is the angle between two surface normals modulo orientation (a host surface's
// own normal sense is not guaranteed to agree with the patch's), so an anti-parallel pair reads 0.
func unsignedNormalAngle(a, b math.Vector3) float64 {
	la, lb := float64(a.Length()), float64(b.Length())
	if la == 0 || lb == 0 {
		return 0
	}
	c := stdmath.Abs(float64(a.Dot(b))) / (la * lb)
	return stdmath.Acos(stdmath.Min(1, c))
}

// runoutCreaseSamples is how many stations the tangency crease is sampled at along a foot locus —
// dense enough to catch a mid-band kink, cheap next to the loft itself.
const runoutCreaseSamples = 16

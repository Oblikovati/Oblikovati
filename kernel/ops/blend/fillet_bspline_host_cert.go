// SPDX-License-Identifier: GPL-2.0-only

package blend

import (
	"fmt"
	stdmath "math"
	"os"

	"oblikovati.org/kernel/geom"
	"oblikovati.org/kernel/ops/internal/tol"
	"oblikovati.org/math"
)

// Envelope certification of the B-spline-host canal band, anchored on REQUEST geometry
// only: the two host surfaces and the requested radius. Nothing here reads the march's own
// stations as truth — a certificate that measures its own construction is worthless (the
// MaxBallDev lesson). The reconstructed centre at a probe column is derived from the
// band's u=0 point THROUGH the host-A surface (foot + side·r·normal), then falsified
// against host B and the band's own interior arc; a band that is not the true envelope of
// the r-ball rolling on the two REQUEST hosts cannot pass.

// bsplineHostBandDev is the request-anchored deviation report of one band.
type bsplineHostBandDev struct {
	railA  float64 // u=0 rail off host A
	railB  float64 // u=1 rail off host B
	center float64 // reconstructed centre's |dist(hostB) − r|
	arc    float64 // interior band points' |dist(centre) − r|
}

// worst is the report's single gating number.
func (d bsplineHostBandDev) worst() float64 {
	return stdmath.Max(stdmath.Max(d.railA, d.railB), stdmath.Max(d.center, d.arc))
}

// bsplineHostSideOf is the frozen tangency side of one host: the sign of (C−foot)·N_host
// read at the band's mid station (any station serves — the march asserts the sign is a
// constant of the whole band).
func bsplineHostSideOf(host geom.Surface, st geom.CanalEdgeStation, footA bool) float64 {
	f := st.FootA
	if !footA {
		f = st.FootB
	}
	n := host.NormalAt(f.U, f.V)
	return stdmath.Copysign(1, float64(f.P.VectorTo(st.Center).Dot(n)))
}

// bsplineHostEnvelopeError measures the band against the request geometry at every
// station mid-interval — the adaptive loop's a-posteriori bound (interpolation error
// peaks mid-interval; the station columns themselves are exact by the march acceptance).
func bsplineHostEnvelopeError(canal *bsplineHostCanal, spec bsplineHostMarchSpec) float64 {
	vp := bsplineHostStationV(canal.stations)
	mid := canal.stations[len(canal.stations)/2]
	sideA := bsplineHostSideOf(spec.aF.Geometry(), mid, true)
	worst, argJ := 0.0, -1
	var argDev bsplineHostBandDev
	// The gate covers the whole final band: the two-phase cut already bounded it to the
	// retained window, so every interval must certify (interpolation error peaks at the
	// mid-interval columns; the station columns themselves are exact by march acceptance).
	for j := 0; j+1 < len(vp); j++ {
		v := 0.5 * (vp[j] + vp[j+1])
		seedA, seedB := bsplineHostColumnSeeds(canal, vp, v)
		dev := bsplineHostColumnDev(canal, spec.hostA, spec.hostB, sideA, v, seedA, seedB)
		if w := dev.worst(); w > worst {
			worst, argJ, argDev = w, j, dev
		}
	}
	wgBsplineDebug(fmt.Sprintf("envelope argmax interval %d/%d (iEdge0=%d): railA=%.2e railB=%.2e center=%.2e arc=%.2e",
		argJ, len(vp)-1, canal.plan.iEdge0, argDev.railA, argDev.railB, argDev.center, argDev.arc), nil)
	wgBsplineArgmaxStations(canal, vp, argJ)
	return worst
}

// wgBsplineArgmaxStations traces the stations bracketing the worst interval (temporary
// WG_BSPLINE_DEBUG diagnostics — a CONSTANT rail error under doubling is the signature of
// a foot-row branch jump, which this dump makes visible).
func wgBsplineArgmaxStations(canal *bsplineHostCanal, vp []float64, argJ int) {
	if os.Getenv("WG_BSPLINE_DEBUG") != "1" || argJ < 0 {
		return
	}
	for j := max(0, argJ-1); j <= min(len(canal.stations)-1, argJ+2); j++ {
		st := canal.stations[j]
		wgBsplineDebug(fmt.Sprintf("  st %d arc=%.4f v=%.6f C=%v fA(u=%.4f v=%.4f P=%v) fB(u=%.4f v=%.4f P=%v)",
			j, canal.plan.arcs[j], vp[j], st.Center, st.FootA.U, st.FootA.V, st.FootA.P, st.FootB.U, st.FootB.V, st.FootB.P), nil)
	}
}

// bsplineHostRetainedEnvelopeError certifies the RETAINED band region against the request
// geometry after the end trims are known: uniform columns across [vLo, vHi] (the crossing
// window's hull — a conservative superset of what survives the trims). This closes the
// certification gap the band-build gate leaves over the retained overrun slivers.
func bsplineHostRetainedEnvelopeError(canal *bsplineHostCanal, vLo, vHi float64, res tol.Resolution) float64 {
	vp := bsplineHostStationV(canal.stations)
	mid := canal.stations[len(canal.stations)/2]
	sideA := bsplineHostSideOf(canal.hostA.Geometry(), mid, true)
	hostA := bsplineHostMarchHost(canal.hostA, res.Weld())
	hostB := bsplineHostMarchHost(canal.hostB, res.Weld())
	worst := 0.0
	for k := 0; k <= 64; k++ {
		v := vLo + (vHi-vLo)*float64(k)/64
		seedA, seedB := bsplineHostColumnSeeds(canal, vp, v)
		worst = stdmath.Max(worst, bsplineHostColumnDev(canal, hostA, hostB, sideA, v, seedA, seedB).worst())
	}
	return worst
}

// bsplineHostColumnDev measures one v-column of the band against the request geometry.
// The feet come from the UNCLAMPED seeded inversion (geom.SurfaceFootNear): a retained
// sliver column just past a host's parameter boundary must be measured against the host's
// natural extension — the global, clamped closest-point piles on the boundary there and
// reports a spurious rail error (the G5 0.4-plateau artifact). The seeds are the
// bracketing stations' foot parameters — a numerical aid, not a trusted value: the
// measured residual is still |band point − host surface|.
func bsplineHostColumnDev(canal *bsplineHostCanal, hostA, hostB geom.CanalMarchHost, sideA, v float64, seedA, seedB geom.CanalFoot) bsplineHostBandDev {
	qa := canal.surf.PointAt(0, v)
	ua, va, footA, errA := geom.SurfaceFootNear(hostA, seedA.U, seedA.V, qa)
	qb := canal.surf.PointAt(1, v)
	_, _, footB, errB := geom.SurfaceFootNear(hostB, seedB.U, seedB.V, qb)
	if errA != nil || errB != nil {
		return bsplineHostBandDev{railA: stdmath.Inf(1)}
	}
	c, ok := reconstructBallCentre(hostA, ua, va, footA, sideA, canal.r)
	if !ok {
		return bsplineHostBandDev{center: stdmath.Inf(1)}
	}
	dev := bsplineHostBandDev{
		railA:  float64(footA.DistanceTo(qa)),
		railB:  float64(footB.DistanceTo(qb)),
		center: centreOffHost(hostB, seedB, c, canal.r),
	}
	for k := 1; k < bsplineHostArcSamples; k++ {
		q := canal.surf.PointAt(float64(k)/bsplineHostArcSamples, v)
		dev.arc = stdmath.Max(dev.arc, stdmath.Abs(float64(q.DistanceTo(c))-r0(canal)))
	}
	return dev
}

// r0 is the canal's requested radius (a tiny reader so the arc loop stays within funlen).
func r0(canal *bsplineHostCanal) float64 { return canal.r }

// centreOffHost is |dist(centre, host) − r| with the same seeded unclamped foot.
func centreOffHost(host geom.CanalMarchHost, seed geom.CanalFoot, c math.Point3, r float64) float64 {
	_, _, foot, err := geom.SurfaceFootNear(host, seed.U, seed.V, c)
	if err != nil {
		return stdmath.Inf(1)
	}
	return stdmath.Abs(float64(foot.DistanceTo(c)) - r)
}

// bsplineHostColumnSeeds interpolates the bracketing stations' foot parameters at v —
// the inversion seeds for one certificate column.
func bsplineHostColumnSeeds(canal *bsplineHostCanal, vp []float64, v float64) (seedA, seedB geom.CanalFoot) {
	i := 0
	for i+2 < len(vp) && vp[i+1] < v {
		i++
	}
	f := 0.0
	if vp[i+1] > vp[i] {
		f = stdmath.Min(1, stdmath.Max(0, (v-vp[i])/(vp[i+1]-vp[i])))
	}
	a0, a1 := canal.stations[i].FootA, canal.stations[i+1].FootA
	b0, b1 := canal.stations[i].FootB, canal.stations[i+1].FootB
	return lerpFootParams(a0, a1, f), lerpFootParams(b0, b1, f)
}

// lerpFootParams linearly interpolates two feet's LIFTED parameters (positions unused).
func lerpFootParams(a, b geom.CanalFoot, f float64) geom.CanalFoot {
	return geom.CanalFoot{U: a.U + f*(b.U-a.U), V: a.V + f*(b.V-a.V)}
}

// reconstructBallCentre rebuilds the probe column's ball centre from REQUEST geometry:
// the host-A foot pushed side·r along host A's unit normal there (lifted-parameter
// evaluation — a periodic host wraps, an open host extends).
func reconstructBallCentre(hostA geom.CanalMarchHost, u, v float64, foot math.Point3, side, r float64) (math.Point3, bool) {
	n, err := math.UnitVector3FromVector(hostA.NormalAtLifted(u, v))
	if err != nil {
		return math.Point3{}, false
	}
	return foot.TranslateBy(n.AsVector().Scale(math.Scalar(side * r))), true
}

// distToSurface is the distance from p to its closest-point foot on s.
func distToSurface(s geom.Surface, p math.Point3) float64 {
	_, _, foot := geom.ClosestPointOnSurface(s, p)
	return float64(foot.DistanceTo(p))
}

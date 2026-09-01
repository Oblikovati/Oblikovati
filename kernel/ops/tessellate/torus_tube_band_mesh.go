// SPDX-License-Identifier: GPL-2.0-only

package tessellate

import (
	stdmath "math"
	"sort"

	"oblikovati.org/kernel/geom"
	"oblikovati.org/kernel/ops/internal/probe"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
)

// torusTubeBandLoftMesh meshes a torus face that is a TUBE-WRAPPING band: bounded by two closed loops
// that each wrap the tube (v spans 2π) — one a meridian circle, the other the spiric-rim canal's
// non-analytic contact rail (a BSpline) — joined by one open seam edge along u. This is the u/v
// TRANSPOSE of closedBandLoftMesh's axis-wrapping rim band, needed by the spiric closed-rim host
// (fillet_spiric_rim.go, J3/A4): toUVLoops cannot chart a tube-wrapping loop, and the full-domain
// fallback meshes the ENTIRE torus (+34.7% on J3's host — measured before this mesher).
//
// The loft honours the shared-edge discretization invariant: each boundary row IS discretizeEdge of
// its own edge (so it welds to the cap discs and the canal band bit-for-bit), interior rows sweep u
// from the circle's iso-u toward the rail's own per-station u at the RAIL's tube stations, and
// consecutive rows are stitched by the same zipper the axis-wrapping band uses.
func torusTubeBandLoftMesh(f *topo.Face, s geom.Surface, q Quality) (*Mesh, bool) {
	t, isTorus := s.(geom.Torus)
	if !isTorus {
		return nil, false
	}
	rings, seamN, seamMid, ok := tubeBandRingsAndSeam(f, t, q)
	if !ok || seamN < 2 {
		return nil, false
	}
	circ, rail, ok := tubeBandCircleAndRail(t, rings)
	if !ok {
		return nil, false
	}
	return loftTubeRows(t, circ, rail, seamMid, seamN), true
}

// tubeBandRingsAndSeam splits the face's boundary edges into tube-wrapping closed rings (discretized
// through the SHARED discretizeEdge, never the raw sampler — closed_band_loft.go's hard-won rule) and
// the one open seam edge, whose sample count sets the u subdivision and whose midpoint disambiguates
// which way around the torus the band spans. ok=false unless exactly two rings and one seam are found.
func tubeBandRingsAndSeam(f *topo.Face, t geom.Torus, q Quality) (rings [][]math.Point3, seamN int, seamMid math.Point3, ok bool) {
	seen := 0
	for _, e := range f.Edges() {
		pts := DiscretizeEdge(e, q)
		if len(pts) < 2 {
			return nil, 0, math.Point3{}, false
		}
		if e.StartVertex().ID() == e.EndVertex().ID() {
			if !wrapsTube(t, pts) {
				return nil, 0, math.Point3{}, false // a closed loop NOT around the tube — not this band
			}
			rings = append(rings, dropClosingDup(pts))
			continue
		}
		seen++
		if len(pts) > seamN {
			seamN, seamMid = len(pts), pts[len(pts)/2]
		}
	}
	if len(rings) != 2 || seen != 1 {
		return nil, 0, math.Point3{}, false
	}
	return rings, seamN, seamMid, true
}

// wrapsTube reports whether a closed polyline winds once around the torus TUBE: its unwrapped tube
// angle advances by ~2π end to end.
func wrapsTube(t geom.Torus, pts []math.Point3) bool {
	total := 0.0
	_, prev := t.ParamAt(pts[0])
	for _, p := range pts[1:] {
		_, v := t.ParamAt(p)
		total += probe.WrapPi(v - prev)
		prev = v
	}
	return stdmath.Abs(stdmath.Abs(total)-2*stdmath.Pi) < 0.5 // half a radian of slack: winding is 0 or ±2π, nothing between
}

// tubeRing is one tube-wrapping boundary row: its points sorted by tube angle, with each point's
// azimuth u kept alongside (the rail's u varies per station; the circle's is constant).
type tubeRing struct {
	pts []math.Point3
	vs  []float64 // tube angles, ascending
	us  []float64 // azimuth at each point (raw ParamAt u)
}

// tubeBandCircleAndRail orders the two rings by tube angle and identifies the iso-u CIRCLE ring (the
// meridian cap circle — near-zero u spread) vs the wandering RAIL ring. ok=false when both or neither
// look iso-u (an ambiguous band is not this mesher's shape — decline, keep the fallback).
func tubeBandCircleAndRail(t geom.Torus, rings [][]math.Point3) (circ, rail tubeRing, ok bool) {
	a, b := orderTubeRing(t, rings[0]), orderTubeRing(t, rings[1])
	sa, sb := uSpread(a.us), uSpread(b.us)
	const isoTol = 1e-6 // tol:angular — an iso-u meridian circle's u spread is round-off only
	if sa <= isoTol && sb > isoTol {
		return a, b, true
	}
	if sb <= isoTol && sa > isoTol {
		return b, a, true
	}
	if sa <= isoTol && sb <= isoTol {
		return a, b, true // two meridian circles (an exact-sector band): either assignment lofts correctly
	}
	return tubeRing{}, tubeRing{}, false
}

// orderTubeRing sorts a ring's points ascending by tube angle and records each point's (u, v).
func orderTubeRing(t geom.Torus, pts []math.Point3) tubeRing {
	r := tubeRing{pts: append([]math.Point3(nil), pts...)}
	sort.SliceStable(r.pts, func(i, j int) bool { return tubeParamOf(t, r.pts[i]) < tubeParamOf(t, r.pts[j]) })
	r.vs = make([]float64, len(r.pts))
	r.us = make([]float64, len(r.pts))
	for i, p := range r.pts {
		r.us[i], r.vs[i] = t.ParamAt(p)
	}
	return r
}

// tubeParamOf is a point's tube parameter v on the torus (unconditional ParamAt inversion — the
// tolerance-gated sibling tubeAngleOf serves the canal far-cap, which must REJECT off-tube points).
func tubeParamOf(t geom.Torus, p math.Point3) float64 { _, v := t.ParamAt(p); return v }

// uSpread is the angular spread of a set of azimuths about their first value (wrap-safe).
func uSpread(us []float64) float64 {
	lo, hi := 0.0, 0.0
	for _, u := range us[1:] {
		d := probe.WrapPi(u - us[0])
		lo, hi = stdmath.Min(lo, d), stdmath.Max(hi, d)
	}
	return hi - lo
}

// loftTubeRows builds the row stack — the circle row verbatim, interior rows sweeping u at the rail's
// own tube stations, the rail row verbatim — and stitches consecutive rows with the shared zipper.
// The u span from the circle to the rail is unwrapped THROUGH the seam midpoint's azimuth (the
// closedBandLoftMesh trick, transposed): a 267° sector must loft the long way its seam actually runs,
// not the short complement.
func loftTubeRows(t geom.Torus, circ, rail tubeRing, seamMid math.Point3, seamN int) *Mesh {
	m := &Mesh{}
	uc := circ.us[0]
	uMid, _ := t.ParamAt(seamMid)
	duMid := probe.WrapPi(uMid - uc)
	rows := []bandRow{addTubeRow(m, t, circ.pts, circ.vs)}
	for k := 1; k < seamN-1; k++ {
		frac := float64(k) / float64(seamN-1)
		pts := make([]math.Point3, len(rail.pts))
		for i := range rail.pts {
			du := duMid + probe.WrapPi(rail.us[i]-uMid) // circle → seam-mid → rail: monotone unwrap
			pts[i] = t.PointAt(uc+frac*du, rail.vs[i])
		}
		rows = append(rows, addTubeRow(m, t, pts, rail.vs))
	}
	rows = append(rows, addTubeRow(m, t, rail.pts, rail.vs))
	for i := 0; i+1 < len(rows); i++ {
		stitchBandRows(m, rows[i], rows[i+1])
	}
	return m
}

// addTubeRow adds one row's vertices with exact surface normals, keyed by its tube angles so the
// zipper merges rows of differing counts in tube order.
func addTubeRow(m *Mesh, t geom.Torus, pts []math.Point3, vs []float64) bandRow {
	idx := make([]int, len(pts))
	for i, p := range pts {
		u, v := t.ParamAt(p)
		idx[i] = m.AddVertex(p, t.NormalAt(u, v))
	}
	return bandRow{idx: idx, ang: vs}
}

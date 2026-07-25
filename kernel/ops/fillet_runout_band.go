// SPDX-License-Identifier: GPL-2.0-only

package ops

import (
	stdmath "math"
	"sort"

	"oblikovati.org/kernel/geom"
	"oblikovati.org/math"
)

// runoutStationRefine is how many loft stations are placed per RAIL chord. The rail nodes themselves
// are always stations — that is what makes the emitted boundary lie ON the lofted surface exactly, and
// therefore what makes the weld T-junction-free (it does NOT make the certificate's G0 residual a
// measurement; see certifyRunoutCanalPatch). The refinement controls only the v-interpolation error
// BETWEEN the nodes, which is exactly what MaxBallDev measures — so this constant sets that field's
// headroom and cannot be lowered without re-measuring it.
//
// 12 puts 6·12+1 = 73 exact stations under a ringSegSamples (6-chord) rail. Measured on the S1 flank
// against a 5.00e-08 weld, the interior residual runs 1.28e-07 (refine 4) → 6.61e-09 (8) → 3.42e-09
// (12) → 7.46e-10 (16): at 4 the field would FAIL its own gate, at 12 it clears it by ~15x. The
// convergence is the point — a value gated at a tolerance it cannot fail proves nothing.
const runoutStationRefine = 12

// runoutBand is one resolved run-out band of the SETBACK-CLOSE partition: its exact rolling-ball
// stations (centre + both contacts, each algebraically at radius) plus the two long rails the patch
// and its neighbouring host notch / wall rim BOTH tile. The stations feed geom.LoftCanalStations;
// the rails feed the RailLoop. Every rail NODE is a station, by construction (buildRunoutBand).
type runoutBand struct {
	stations []runoutStation
	railA    geom.Curve3 // A-side rail (the boss footprint sub-arc), traced lo→hi
	railB    geom.Curve3 // B-side rail (the tangency contact locus, or the second footprint sub-arc)
}

// endStation returns the band's first (lo) or last (hi) exact station — the shared cross-section a
// neighbouring band or the plain fillet wing welds to.
func (b runoutBand) endStation(hi bool) runoutStation {
	if hi {
		return b.stations[len(b.stations)-1]
	}
	return b.stations[0]
}

// payload packs the band's exact stations into the provider payload. The three parallel rows are what
// geom.LoftCanalStations consumes; it asserts every foot sits at Radius from its centre, so a station
// this solver got wrong is DECLINED rather than lofted (do-no-harm). The envelope these stations were
// solved from is attached ONCE, on the RailLoop (see RailLoop.Envelope) — never duplicated here.
func (b runoutBand) payload() *RunoutCanal {
	rc := &RunoutCanal{
		Centers: make([]math.Point3, len(b.stations)),
		FeetA:   make([]math.Point3, len(b.stations)),
		FeetB:   make([]math.Point3, len(b.stations)),
	}
	for i, st := range b.stations {
		rc.Centers[i], rc.FeetA[i], rc.FeetB[i] = st.centre, st.footA, st.footB
	}
	return rc
}

// runoutStationFeet resolves one station: given the spine coordinate, the A-side contact and (when the
// station is a B-rail node) the B-side contact already fixed by that rail, it returns the ball centre
// and the resolved B contact. surfRstFeet (tangency on host B) and rstRstFeet (both sides
// restrictions) are its two implementations; buildRunoutBand is written once against it.
type runoutStationFeet func(s float64, footA, footB math.Point3, hasFootB bool) (centre, resolved math.Point3, ok bool)

// surfRstFeet is the flank / single-boss-central flavour: the ball is tangent to hostB and passes
// through the A footprint, so footB is the tangency foot (the projection of the centre onto hostB) —
// it is never a node, since this band's B rail is synthesised FROM the solve.
func surfRstFeet(env runoutEnvelope, hostA, hostB geom.Plane, weld float64) runoutStationFeet {
	return func(s float64, footA, _ math.Point3, _ bool) (math.Point3, math.Point3, bool) {
		c, ok := env.surfRstCentre(hostB, hostA, s, footA, weld)
		if !ok {
			return math.Point3{}, math.Point3{}, false
		}
		return c, projectOntoPlane(c, hostB), true
	}
}

// rstRstFeet is the two-boss CENTRAL flavour: the ball passes through both footprints, so footB is
// the second boss's footprint point at the same station — taken from the B rail's own node when the
// station is one, so that rail's samples are exact station feet too.
func rstRstFeet(env runoutEnvelope, inner crossingBoss, weld float64) runoutStationFeet {
	return func(s float64, footA, footB math.Point3, hasFootB bool) (math.Point3, math.Point3, bool) {
		if !hasFootB {
			var ok bool
			if footB, ok = footprintPointAtStation(inner, env.cyl, s); !ok {
				return math.Point3{}, math.Point3{}, false
			}
		}
		c, ok := env.rstRstCentre(s, footA, footB, weld)
		if !ok {
			return math.Point3{}, math.Point3{}, false
		}
		return c, footB, true
	}
}

// buildRunoutBand resolves one band into exact stations plus its two rails. railA is the A-side boss
// footprint sub-arc (the EXACT intact-footprint curve the wall rim also tiles); its ringSegSamples
// nodes fix the station grid, refined runoutStationRefine-fold. railBNodes, when non-nil, is the
// B-side rail's OWN node curve (the second footprint sub-arc of a rst-rst band) whose nodes are added
// to the grid so its samples are station feet too; when nil the B rail is synthesised as the polyline
// through the computed contacts at railA's nodes. ok=false on any station that will not solve — the
// whole band declines, never a partial fill (do-no-harm).
func buildRunoutBand(env runoutEnvelope, railA geom.Curve3, railBNodes geom.Curve3, boss crossingBoss,
	feet runoutStationFeet) (runoutBand, bool) {
	nodesA, ok := railNodePoints(railA)
	if !ok {
		return runoutBand{}, false
	}
	grid, ok := runoutStationGrid(env, nodesA, railBNodes)
	if !ok {
		return runoutBand{}, false
	}
	var nodesB []math.Point3
	if railBNodes != nil {
		if nodesB, ok = railNodePoints(railBNodes); !ok {
			return runoutBand{}, false
		}
	}
	sts, ok := solveRunoutStations(env, grid, nodesA, nodesB, boss, feet)
	if !ok {
		return runoutBand{}, false
	}
	railB := railBNodes
	if railB == nil {
		if railB, ok = contactLocusRail(sts, len(nodesA)-1); !ok {
			return runoutBand{}, false
		}
	}
	return runoutBand{stations: sts, railA: railA, railB: railB}, true
}

// railNodePoints samples a rail at the SHARED ring granularity, endpoint INCLUDED — the exact point
// set every neighbour (wall rim, host notch, adjacent patch) tiles the same curve into. These points
// become loft stations, which is what makes the emitted patch boundary lie on the lofted surface.
func railNodePoints(rail geom.Curve3) ([]math.Point3, bool) {
	if rail == nil {
		return nil, false
	}
	pts := append(sampleCurveN(rail, ringSegSamples, false), curveEnd(rail))
	return pts, len(pts) == ringSegSamples+1
}

// runoutStationGrid is the sorted station set: every rail node's spine coordinate (both rails, so
// BOTH parametrisations are interpolated exactly) plus runoutStationRefine-fold uniform fill between
// consecutive nodes. ok=false when the nodes are not strictly monotone along the spine — a footprint
// sub-arc that doubles back is not a band rail and declines.
func runoutStationGrid(env runoutEnvelope, nodesA []math.Point3, railBNodes geom.Curve3) ([]float64, bool) {
	base := make([]float64, 0, 2*(ringSegSamples+1))
	for _, p := range nodesA {
		base = append(base, spineParam(p, env.cyl))
	}
	if !strictlyMonotone(base) {
		return nil, false
	}
	if railBNodes != nil {
		nodesB, ok := railNodePoints(railBNodes)
		if !ok {
			return nil, false
		}
		for _, p := range nodesB {
			base = append(base, spineParam(p, env.cyl))
		}
	}
	return refineStationGrid(base), true
}

// strictlyMonotone reports whether xs is strictly increasing or strictly decreasing — the guard that a
// band rail advances along the spine (a rail that reverses would fold the loft).
func strictlyMonotone(xs []float64) bool {
	if len(xs) < 2 {
		return false
	}
	up := xs[1] > xs[0]
	for i := 1; i < len(xs); i++ {
		if (xs[i] > xs[i-1]) != up || xs[i] == xs[i-1] {
			return false
		}
	}
	return true
}

// refineStationGrid sorts the node stations ascending, drops duplicates, and inserts
// runoutStationRefine-1 uniform stations in every gap. The node stations survive untouched, so the
// loft still interpolates every rail sample exactly.
func refineStationGrid(base []float64) []float64 {
	sort.Float64s(base)
	span := base[len(base)-1] - base[0]
	out := []float64{base[0]}
	for i := 1; i < len(base); i++ {
		if base[i]-base[i-1] <= 1e-12*span {
			continue // the two rails share this station (a band corner)
		}
		for k := 1; k < runoutStationRefine; k++ {
			out = append(out, base[i-1]+(base[i]-base[i-1])*float64(k)/runoutStationRefine)
		}
		out = append(out, base[i])
	}
	return out
}

// solveRunoutStations evaluates the closed-form station solve on the whole grid. At a grid value that
// IS a railA node the A foot is the node point ITSELF (not a re-derived footprint point), so the
// emitted rail sample and the lofted station foot are bit-identical and the weld is exact.
func solveRunoutStations(env runoutEnvelope, grid []float64, nodesA, nodesB []math.Point3,
	boss crossingBoss, feet runoutStationFeet) ([]runoutStation, bool) {
	atA, atB := gridNodeIndex(env, grid, nodesA), gridNodeIndex(env, grid, nodesB)
	sts := make([]runoutStation, len(grid))
	for i, s := range grid {
		footA, ok := atA[i]
		if !ok {
			if footA, ok = footprintPointAtStation(boss, env.cyl, s); !ok {
				return nil, false
			}
		}
		nodeB, hasB := atB[i]
		centre, footB, ok := feet(s, footA, nodeB, hasB)
		if !ok {
			return nil, false
		}
		sts[i] = runoutStation{s: s, centre: centre, footA: footA, footB: footB}
	}
	return sts, true
}

// gridNodeIndex maps each rail node onto its own grid slot — the grid was BUILT from these nodes'
// spine coordinates, so the match is exact and the node point can be used verbatim as that station's
// foot (no re-derivation, hence a bit-identical rail-sample ↔ station-foot correspondence).
func gridNodeIndex(env runoutEnvelope, grid []float64, nodes []math.Point3) map[int]math.Point3 {
	at := map[int]math.Point3{}
	for _, p := range nodes {
		at[nearestGridIndex(grid, spineParam(p, env.cyl))] = p
	}
	return at
}

// nearestGridIndex returns the index of the grid value closest to s (the grid is built FROM these
// values, so the match is exact up to the refinement's own arithmetic).
func nearestGridIndex(grid []float64, s float64) int {
	best, bestD := 0, stdmath.Inf(1)
	for i, g := range grid {
		if d := stdmath.Abs(g - s); d < bestD {
			best, bestD = i, d
		}
	}
	return best
}

// contactLocusRail is the B-side rail of a surf-rst band: the degree-1 (polyline) B-spline through the
// EXACT tangency contacts at the rail's node stations. Degree 1 with uniform interior knots makes
// PointAt(k/n) return control point k exactly, so sampleCurveN(rail, ringSegSamples) reproduces the
// station feet bit-identically — which is what lets the patch, the re-clipped host notch and the wing
// weld point-for-point while every one of those points sits on the true envelope (the straight
// LineSegment it replaces sat on the host plane but at the PLAIN fillet's contact, up to 11% of the
// host face's area away — the separable under-recession coons4-audit.md §C.4 isolated).
func contactLocusRail(sts []runoutStation, chords int) (geom.BSplineCurve, bool) {
	if chords < 1 || len(sts) < chords+1 {
		return geom.BSplineCurve{}, false
	}
	step := (len(sts) - 1) / chords
	ctrl := make([]math.Point3, 0, chords+1)
	for k := 0; k <= chords; k++ {
		ctrl = append(ctrl, sts[k*step].footB)
	}
	c, err := geom.NewBSplineCurveUniformWeights(1, ctrl, polylineKnots(len(ctrl)))
	return c, err == nil
}

// polylineKnots is the clamped degree-1 knot vector [0,0,1/n,…,(n-1)/n,1,1] for n+1 control points —
// the parametrisation under which PointAt(k/n) is control point k.
func polylineKnots(count int) []float64 {
	n := count - 1
	knots := make([]float64, 0, count+2)
	knots = append(knots, 0, 0)
	for i := 1; i < n; i++ {
		knots = append(knots, float64(i)/float64(n))
	}
	return append(knots, 1, 1)
}

// reverseCurve3 returns a curve tracing c backwards. A polyline (degree-1 B-spline) is reversed
// EXACTLY by reversing its control points — no coordinate arithmetic — so a host notch entering the
// contact locus from the far corner samples the identical point set the patch does. Any other curve
// kind is returned unchanged: every such rail in this path (Arc3d / EllipticalArc) already samples
// order-independently under sampleCurveN.
func reverseCurve3(c geom.Curve3) geom.Curve3 {
	bs, ok := c.(geom.BSplineCurve)
	if !ok || bs.Degree != 1 {
		return c
	}
	ctrl := make([]math.Point3, len(bs.Ctrl))
	for i, p := range bs.Ctrl {
		ctrl[len(ctrl)-1-i] = p
	}
	rev, err := geom.NewBSplineCurveUniformWeights(1, ctrl, polylineKnots(len(ctrl)))
	if err != nil {
		return c
	}
	return rev
}

// orientedLocus returns the contact-locus rail traced from `from` — the direction-safe accessor every
// host detour uses, since a polyline (unlike a straight segment) is NOT direction-symmetric.
func orientedLocus(rail geom.Curve3, from math.Point3, weld float64) geom.Curve3 {
	if float64(curveStart(rail).DistanceTo(from)) <= weld {
		return rail
	}
	return reverseCurve3(rail)
}

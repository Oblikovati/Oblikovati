// SPDX-License-Identifier: GPL-2.0-only

package ops

import (
	"oblikovati.org/kernel/geom"
	"oblikovati.org/math"
)

// obstacleCanal is the EXACT rolling-ball model of a mid-span obstacle band: over the dip the ball can
// no longer touch the notched host (the obstacle's footprint has eaten the contact line there), so it
// stays TANGENT TO THE FILLET WALL and passes THROUGH the obstacle's base rim. That is precisely
// runoutEnvelope's SURF-RST flavour (fillet_runout_envelope.go) with `tangent` = the fillet's wall plane
// and `restrict` = the notched host plane — and it is what OCCT itself builds.
//
// Receipts (DRAWEXE 8.0.0 live: `blend result s <r> s_<i>`, `explode result F`, `sprops`/`dump`).
// R9's patch dumps as `BSplineSurface urational vrational, Degrees 2 9, NbPoles 3 10` — one rational
// QUADRATIC span in u, i.e. a section ARC, lofted over ten stations. Its u-row weights recover
// cos(β/2) = √(2/3) at mid-span (a section arc of radius 3 subtending arccos(1/3)) and its u=0 pole row
// (the WALL foot locus) evaluates to z = 7.17177 at mid-span, where this model gives 7.171573 — the
// straight wall seam the Coons model used gives 7. The closed form reproduces OCCT's own per-face areas
// to its printing precision on every case whose solid yields one:
//
//	              R9          S3          T6          U3           X3
//	patch     31.215583  149.671704  156.364251   85.917424  1471.701005   (closed form)
//	OCCT      31.2156    149.672     156.364      42.9587x2  1471.7
//	wall gain  0.717167    4.624676    9.476511    2.998261   147.353614   (closed form)
//	OCCT       0.717       4.625       9.476       2.998      147.35
//
// (U3 is the one case where OCCT splits the patch into two equal faces and we emit one; the closed form is
// their sum. The "wall gain" is the bulge the straight-seam model was missing ENTIRELY: OCCT's wall faces
// read 340.717 / 214.625 / 569.476 / 302.998 / 7647.35 against a plain 340 / 210 / 560 / 300 / 7500.)
//
// Centres/FeetRim/FeetWall are the triple geom.LoftCanalStations consumes. Every DIP RIM SAMPLE is a
// station (FeetRim carries ObstacleFeature.RimArcPts by value at those entries, so the notch and the split
// obstacle wall keep tiling the rim at exactly the granularity they already do — no T-junction is
// introduced); obstacleCanalSubdiv extra stations sit BETWEEN them, solved on the analytic rim, purely to
// give the loft's v-interpolation enough support to certify (see obstacleCanalSubdiv). Envelope is the
// payload MaxBallDev needs to certify the INTERIOR: one tangency host + one restriction curve, the pair
// maxBallDev already solves.
type obstacleCanal struct {
	Centres  []math.Point3 // ball centre per station
	FeetRim  []math.Point3 // through-point on the obstacle rim (the u=0 boundary)
	FeetWall []math.Point3 // tangency foot on the fillet wall plane (the u=1 boundary)
	Envelope BallEnvelope  // {Tangents: [wall plane], Through: [rim curve], Spine, Radius}
}

// wallFront returns the patch's wall-side boundary polyline A→D — the same POINT VALUES the wall face's
// own front is subdivided at (recordWallInserts hands out a copy, orderedWallInserts), so the two cannot
// disagree.
//
// It uses EVERY station, not just the rim-sample ones, because unlike the rim this front is private to the
// patch/wall PAIR: no third face tiles it, so densifying it cannot introduce a T-junction. It is worth
// densifying — the front is a curve and the loop tiles it with chords, so an inscribed polyline moves area
// off the wall face and onto the patch. Measured on the five corpus cases: at rim-sample granularity the
// shipped bulge read −0.176 % … −0.272 % of its closed form and the patch +0.0082 % … +0.0347 %; at full
// station density the bulge reads −0.0027 % … −0.0041 % and the patch +0.0029 % … +0.0283 % — the ~64x an
// 8x shorter chord predicts.
func (c *obstacleCanal) wallFront() []math.Point3 { return c.FeetWall }

// obstacleCanalSubdiv is how many station INTERVALS each dip rim-sample gap is split into for the loft.
// It is a v-interpolation support count, and it must be chosen on the INTERIOR condition, never on area:
// area is saturated from the first station set upward (the u=0/u=1 rows interpolate their loci exactly at
// every station either way), so it says nothing about what the surface does BETWEEN stations — the trap
// that left U4's core panels wrong at 1.8e-3 while their area looked converged (u4-canal-report.md
// concern 3). Measured on the corpus's five single-host obstacle cases, MaxBallDev vs subdiv:
//
//	subdiv | R9        S3        T6        U3        X3
//	     1 | 3.59e-05  1.32e-05  1.61e-05  3.14e-05  8.95e-05
//	     2 | 2.13e-06  1.01e-06  1.19e-06  1.35e-06  4.66e-06
//	     4 | 5.97e-08  1.21e-07  1.54e-07  6.20e-08  4.39e-07
//	     8 | 6.16e-09  1.99e-09  3.08e-09  7.55e-09  1.10e-08
//	    16 | 2.21e-10  2.06e-10  2.41e-10  4.79e-10  3.34e-10
//	weld   | 4.50e-08  5.50e-08  6.87e-08  5.38e-08  1.85e-07
//
// i.e. the h^4 of the cubic v-interpolant, and 8 is the FIRST value inside the model weld on all five —
// subdiv 4 misses on EVERY one of them, and 8's tightest margin is 7.1x (U3). Their per-face AREA, by
// contrast, is already inside 0.04 % of the closed form at subdiv 1: that is the saturation this constant
// must not be chosen on.
//
// "First value inside the weld" is the claim, so it is what the gate asserts:
// TestObstacleCanalInteriorConvergesLikeTheCubicItIs derives its whole ladder from THIS constant (K/4,
// K/2, K) and requires K inside the weld and K/2 outside it. Changing the constant therefore moves the
// test — 4 fails for being uncertified, 16 for spending 7 extra rim solves per gap the weld never asked
// for — and re-justifying a new value means re-measuring there, not editing this table.
const obstacleCanalSubdiv = 8

// buildObstacleCanal solves the surf-rst station triple for the dip. It returns nil — never a partial or
// clamped payload — when any station declines (surfRstCentre guards a vanishing discriminant and a
// near-parallel host pair), or when the dip's rim samples are not strictly monotone along the spine (a rim
// that re-enters the band: outside this slice's single-dip model). A nil payload makes the WHOLE obstacle
// fall back to the straight-seam Coons model, wall front included, so the patch and the wall can never
// disagree about where their shared front runs. That fallback is exercised end to end by
// TestFilletSlabObliqueColumnFallsBackToCoons, whose fixture's dip genuinely re-enters the band — no
// corpus case does, so without it the ADR-3 tier would have no body-level coverage at all.
func buildObstacleCanal(ef edgeFillet, d obstacleDetection, og obstacleGeom, of *ObstacleFeature, res Resolution) *obstacleCanal {
	wall, isPlane := d.filletWall.Geometry().(geom.Plane)
	if !isPlane {
		return nil
	}
	env := newRunoutEnvelope(ef.cyl)
	feet, ok := obstacleCanalRimFeet(env, of, d, ef)
	if !ok {
		return nil
	}
	rows, ok := obstacleCanalStations(env, wall, of.HostPlane, feet, res.Weld())
	if !ok {
		return nil
	}
	pinObstacleCanalEnds(&rows, og, of)
	return &obstacleCanal{
		Centres: rows.centres, FeetRim: rows.feetRim, FeetWall: rows.feetWall,
		Envelope: BallEnvelope{
			Tangents: []geom.Surface{wall},
			Through:  []geom.Curve3{of.RimCurve},
			Spine:    env.spine,
			Radius:   env.radius,
		},
	}
}

// obstacleCanalRimFeet is the station's rim-foot row: every dip rim SAMPLE by value, plus
// obstacleCanalSubdiv−1 points solved on the ANALYTIC rim between each consecutive pair
// (dipRimPointAtStation — the same closed-form rim∩section-plane solver the dual-host section endpoints
// use, not the 64-chord polyline). Keeping every rim SAMPLE a station is what makes the patch's rim
// boundary — which must stay at the notch's and the obstacle wall's granularity — lie exactly on the
// lofted surface. ok=false — with a NIL row, like every other decline path here — when the dip carries
// too few rim samples to loft (geom.LoftCanalStations needs 4 for its cubic v-interpolant), when the
// samples are not strictly monotone along the spine, or when an interior solve declines. Both
// PRECONDITIONS are checked before any solve runs: a decline must not pay for 7·(n−1) interior solves
// first.
func obstacleCanalRimFeet(env runoutEnvelope, of *ObstacleFeature, d obstacleDetection, ef edgeFillet) ([]math.Point3, bool) {
	if len(of.RimArcPts) < 4 {
		return nil, false // too coarse a dip to loft (a sliver obstacle: the Coons tier's case)
	}
	ss := make([]float64, len(of.RimArcPts))
	for i, q := range of.RimArcPts {
		ss[i] = float64(env.cyl.Origin.VectorTo(q).Dot(env.spine))
	}
	if !strictlyMonotone(ss) {
		return nil, false // the dip re-enters the band: outside the single-dip model
	}
	shift := axisParam(ef, env.cyl.Origin) // dipRimPointAtStation's station origin vs the envelope's
	feet := []math.Point3{of.RimArcPts[0]}
	for i := 1; i < len(of.RimArcPts); i++ {
		mid, ok := obstacleCanalGapFeet(d, ef, ss[i-1], ss[i], shift)
		if !ok {
			return nil, false
		}
		feet = append(feet, mid...)
		feet = append(feet, of.RimArcPts[i])
	}
	return feet, true
}

// obstacleCanalGapFeet solves the interior rim feet of one rim-sample gap, at the spine stations
// (s0,s1) is split into obstacleCanalSubdiv equal intervals by. ok=false if any solve declines.
func obstacleCanalGapFeet(d obstacleDetection, ef edgeFillet, s0, s1, shift float64) ([]math.Point3, bool) {
	out := make([]math.Point3, 0, obstacleCanalSubdiv-1)
	for k := 1; k < obstacleCanalSubdiv; k++ {
		s := s0 + (s1-s0)*float64(k)/float64(obstacleCanalSubdiv)
		q, ok := dipRimPointAtStation(d, ef, s+shift)
		if !ok {
			return nil, false
		}
		out = append(out, q)
	}
	return out, true
}

// obstacleCanalRows is the three parallel station rows under construction.
type obstacleCanalRows struct {
	centres, feetRim, feetWall []math.Point3
}

// obstacleCanalStations solves one surf-rst station per rim foot. The station's spine coordinate is the
// foot's OWN projection on the fillet axis, so the foot lies in that station's section plane by
// construction and IS the station's rim contact — which is what keeps the emitted rim boundary
// point-for-point identical to the notch's and the split obstacle wall's tiling of the same rim.
func obstacleCanalStations(env runoutEnvelope, wall, host geom.Plane, feet []math.Point3, weld float64) (obstacleCanalRows, bool) {
	var out obstacleCanalRows
	for _, q := range feet {
		s := float64(env.cyl.Origin.VectorTo(q).Dot(env.spine))
		c, okC := env.surfRstCentre(wall, host, s, q, weld)
		n, okN := env.hostNormal(wall, s, weld)
		if !okC || !okN {
			return obstacleCanalRows{}, false
		}
		out.centres = append(out.centres, c)
		out.feetRim = append(out.feetRim, q)
		out.feetWall = append(out.feetWall, c.TranslateBy(n.Scale(math.Scalar(-env.radius))))
	}
	return out, len(out.centres) >= 4
}

// pinObstacleCanalEnds OVERWRITES the two END stations with the WING SECTIONS' own centre / node / wall
// point. At a node the surf-rst offset t is algebraically 0 (there the rim sample IS the plain host
// contact, so w = r·d and the discriminant is r²), so the closed form already lands on the wing section
// to rounding — but the patch's v=0/v=1 boundary arcs must weld to the wing faces BY VALUE, not merely
// within weld, and og/of carry the very values the wing faces use. The by-value weld is asserted on the
// real pipeline output, on every single-host obstacle case, by
// TestObstacleCanalEndStationsWeldToTheWingsBitForBit.
//
// It takes *obstacleCanalRows because it MUTATES the rows: the by-value form worked only through the
// shared slice backing and read as non-mutating.
func pinObstacleCanalEnds(rows *obstacleCanalRows, og obstacleGeom, of *ObstacleFeature) {
	last := len(rows.centres) - 1
	rows.feetRim[0], rows.feetRim[last] = of.Nodes[0], of.Nodes[1]
	rows.feetWall[0], rows.feetWall[last] = og.wallA, og.wallD
	rows.centres[0], rows.centres[last] = og.startCen, og.endCen
}

// SPDX-License-Identifier: GPL-2.0-only

// Package ops — n-valent runout solver. PURE: imports only geom + math (no topo/diag). It turns an
// endCornerFan into a runoutSpread (per-far-face arc pieces + per-far-edge split points) or an
// error (the n-valent generalisation of the #1800 over-radius reject).
package ops

import (
	"fmt"
	stdmath "math"

	"oblikovati.org/kernel/geom"
	"oblikovati.org/math"
)

// runoutSpread is the solved n-valent runout: an arc piece per far face the cap actually touches,
// plus the single split point on every far edge the fillet cylinder crosses. Tasks 4-5 populate
// pieces/splits; this task only declares the shape they land in.
type runoutSpread struct {
	pieces map[uint64]cornerPiece // far-face id -> the arc piece it carries (absent = no arc)
	splits map[uint64]math.Point3 // far-edge id -> its single split point
}

// cornerPiece is the arc-fit of one far face's elliptical cap section, from its A-side tangency to
// its B-side tangency. A nil curve means Task 5 found the piece degenerates to a straight segment.
type cornerPiece struct {
	curve     geom.Curve3 // arc-fit of the elliptical section (nil ⇒ straight)
	tIn, tOut math.Point3
}

// splitOnFarEdge solves d²(x, axis) = r² for x = from + t·(to-from), returning the crossing whose
// POINT is NEAREST the fan apex among the edge's two cylinder crossings — it does not verify the
// crossing is singular. d²(x,ℓ) = |x-c|² - ((x-c)·û)². The quadratic in t is A t² + 2B t + C with
// A = |w|² - (w·û)², B = (u0·w) - (u0·û)(w·û), C = |u0|² - (u0·û)² - r², where u0 = from-center,
// w = to-from, û = normalized axis. Returns ok=false if neither crossing sits on the edge segment
// — the far edge doesn't graze or miss the fillet tube.
//
// Each interior far edge (a straight line from the apex out into the body) crosses the fillet
// cylinder at TWO points: the near one, ~r from the apex inside the fillet's minor-arc band, and a
// far one deep in the body outside the band. Only the near-apex crossing is OCCT's; the far one
// forces a spurious cap and over-shoots the runout area (V5: +3.24%). The earlier "smallest root in
// (0,1)" rule was orientation-dependent — with far edges stored from→to as outer→apex both roots
// land in (0,1) and the smaller-t root is the WRONG far one — so we select by proximity to the apex
// instead, which is orientation-independent (geometry-math-advisor derivation, V5 valence-6 gap).
func splitOnFarEdge(fan endCornerFan, fe fanEdge) (math.Point3, bool) {
	uhat := unit(fan.axis)
	u0 := fan.center.VectorTo(fe.from)
	w := fe.from.VectorTo(fe.to)
	wu, u0u := w.Dot(uhat), u0.Dot(uhat)
	a := w.LengthSquared() - wu*wu
	b := u0.Dot(w) - u0u*wu
	c := u0.LengthSquared() - u0u*u0u - fan.radius*fan.radius
	roots, ok := edgeCylinderRoots(a, b, c, w.LengthSquared())
	if !ok {
		return math.Point3{}, false
	}
	return nearestApexCrossing(fan, fe, w, roots)
}

// edgeCylinderRoots returns the real roots of A t² + 2B t + C = 0 (one or two), with the linear
// fallback when |A| is tiny relative to the edge scale |w|² (axis parallel to the edge, so the
// quadratic term vanishes) — a relative cutoff so it holds across model scales, not a bare epsilon.
// ok=false when there is no real root (discriminant < 0) or the degenerate branch has no linear term
// either. It does NOT filter by segment membership — that is the caller's near-apex selector — so the
// correct root the old strict-(0,1) filter could drop stays a candidate.
func edgeCylinderRoots(a, b, c, wl2 float64) ([]float64, bool) {
	const rel = 1e-12 // |A|/|w|² = sin²(edge,axis); below this the edge is axis-parallel (A vanishes)
	if stdmath.Abs(a) < rel*wl2 {
		if stdmath.Abs(b) < rel*wl2 {
			return nil, false
		}
		return []float64{-c / (2 * b)}, true
	}
	disc := b*b - a*c
	if disc < 0 {
		return nil, false
	}
	s := stdmath.Sqrt(disc)
	return []float64{(-b - s) / a, (-b + s) / a}, true
}

// nearestApexCrossing maps each root t to its point from + t·w and returns the one NEAREST the fan
// apex among those on the edge segment (t within a model-scaled slack of [0,1]). Both roots are
// cylinder points at radius r by construction, so no separate distance-to-axis guard is needed. The
// near-apex crossing is the fillet's genuine runout split; the far one lies deep in the body outside
// the minor-arc band. ok=false when neither root sits on the segment (the edge misses the tube).
func nearestApexCrossing(fan endCornerFan, fe fanEdge, w math.Vector3, roots []float64) (math.Point3, bool) {
	tol := crossingSegmentTol(fan, fe)
	best, bestD, found := math.Point3{}, stdmath.Inf(1), false
	for _, t := range roots {
		if t < -tol || t > 1+tol {
			continue
		}
		p := fe.from.TranslateBy(w.Scale(t))
		if d := float64(p.DistanceTo(fan.apex)); d < bestD {
			best, bestD, found = p, d, true
		}
	}
	return best, found
}

// crossingSegmentTol is the parametric slack (in t-units) allowed beyond [0,1] when testing whether
// a root lies on the far-edge segment: a small physical length κ·min(r, |edge|) converted to
// parameter units by /|edge|, so a crossing nudged just past an endpoint by float rounding still
// counts while a root clearly off the segment does not. Model-scaled rather than a bare epsilon so
// it holds across model scales (CLAUDE.md param-unit rule).
func crossingSegmentTol(fan endCornerFan, fe fanEdge) float64 {
	const kappa = 1e-6
	edgeLen := float64(fe.from.DistanceTo(fe.to))
	if edgeLen == 0 {
		return kappa // degenerate zero-length far edge (from==to); should not occur, but avoid /0
	}
	return kappa * stdmath.Min(fan.radius, edgeLen) / edgeLen
}

// solveRunoutSpread turns a fan into the per-face arc pieces + per-far-edge split points, or an
// error on a validity-certificate failure (Task 5). Membership is three-tier: a far face bounded by
// two crossings gets an arc; one only touched by a neighbour's split gets a split-pullback; an
// untouched face is omitted (its vertex survives). This slice implements the arc tier only — every
// fan face gets a piece — leaving boundaryPoint as the seam Task 6 uses to add the other two tiers.
// Every interior far edge yields exactly one split shared by its two faces — the weld-twice
// invariant, asserted by TestSolveRunoutSpreadChainCloses.
func solveRunoutSpread(fan endCornerFan) (runoutSpread, error) {
	sp := runoutSpread{pieces: map[uint64]cornerPiece{}, splits: map[uint64]math.Point3{}}
	if err := collectFarEdgeSplits(fan, sp.splits); err != nil {
		return runoutSpread{}, err
	}
	if !monotoneAroundAxis(fan, sp) {
		return runoutSpread{}, filletRunoutError(fan, "runout crossings are not in monotone angular order (self-intersecting)", fan.filletEdge)
	}
	// Certificate for boundaryPoint below: it reads sp.splits[edge] without a comma-ok (its
	// signature can't return an error), so every non-flank entry/exit edge the fan loop is about
	// to ask for must already be solved. In this slice the two always line up (fan.fan's
	// entry/exitEdge and fan.farEdges both come from farEdgesOf(chain)), so this never fires
	// today — it exists to turn a future membership-tier miss (deferred three-tier binding) into
	// an honest reject instead of boundaryPoint silently falling through to a zero Point3.
	if err := verifyFanEdgesSplit(fan, sp); err != nil {
		return runoutSpread{}, err
	}
	for i, ff := range fan.fan {
		tIn := boundaryPoint(sp, ff.entryEdge, i == 0, fan.ta)
		tOut := boundaryPoint(sp, ff.exitEdge, i == len(fan.fan)-1, fan.tb)
		piece, err := arcPiece(fan, ff, tIn, tOut)
		if err != nil {
			return runoutSpread{}, err
		}
		sp.pieces[ff.face] = piece
	}
	return sp, nil
}

// collectFarEdgeSplits solves every far edge's fillet-cylinder crossing into dst, keyed by edge id
// — the population half of sp.splits that verifyFanEdgesSplit later checks completeness against.
func collectFarEdgeSplits(fan endCornerFan, dst map[uint64]math.Point3) error {
	for _, fe := range fan.farEdges {
		p, ok := splitOnFarEdge(fan, fe)
		if !ok {
			return filletRunoutError(fan, "no single crossing on far edge", fe.edge)
		}
		dst[fe.edge] = p
	}
	return nil
}

// verifyFanEdgesSplit is the certificate guarding boundaryPoint's unchecked sp.splits[edge] read: it
// fails loudly if any far face's non-flank entry/exit edge lacks a split point, rather than letting
// boundaryPoint return a silent zero Point3 for a missing key.
func verifyFanEdgesSplit(fan endCornerFan, sp runoutSpread) error {
	for _, ff := range fan.fan {
		for _, edge := range []uint64{ff.entryEdge, ff.exitEdge} {
			if edge == 0 {
				continue // flank sentinel, resolved from fan.ta/fan.tb instead
			}
			if _, ok := sp.splits[edge]; !ok {
				return filletRunoutError(fan, "far face bounding edge has no runout split point", edge)
			}
		}
	}
	return nil
}

// arcPiece fills one far face's cornerPiece with a circular-arc approximation of the true elliptical
// section (cylinder ∩ ff's plane) through (tIn, mid, tOut). The arc passes through tIn/tOut EXACTLY
// (welding the pieces) and bulges through mid, a point placed on the cylinder. A degenerate
// (antipodal) chord or a collinear (tIn,mid,tOut) is honest-rejected — never emitted as a sliver.
func arcPiece(fan endCornerFan, ff fanFace, tIn, tOut math.Point3) (cornerPiece, error) {
	mid, ok := ellipseMidPoint(fan, tIn, tOut)
	if !ok {
		return cornerPiece{}, filletRunoutError(fan, "runout section endpoints are antipodal about the fillet axis (degenerate half-turn section)", ff.face)
	}
	arc, err := geom.Arc3dByThreePoints(tIn, mid, tOut)
	if err != nil {
		return cornerPiece{}, filletRunoutError(fan, "runout section arc-fit failed (collinear section)", ff.face)
	}
	return cornerPiece{curve: arc, tIn: tIn, tOut: tOut}, nil
}

// ellipseMidPoint returns an on-cylinder point at the angular bisector of tIn and tOut about the
// fillet axis — the well-defined mid for the arc-fit of the true elliptical section. tIn and tOut
// are both at radius r from the axis (they are cylinder crossings), so their chord midpoint lies on
// their angle-bisector ray from the axis; projecting it out to r lands mid at the bisector angle,
// strictly between the endpoints for spans < pi. ok=false when the chord midpoint sits ON the axis
// (tIn, tOut ~antipodal about the axis): the bisector direction is undefined and the section spans a
// half-turn — a genuine degeneracy to reject rather than emit as a sliver.
func ellipseMidPoint(fan endCornerFan, tIn, tOut math.Point3) (math.Point3, bool) {
	uhat := unit(fan.axis)
	chordMid := tIn.Midpoint(tOut)
	w := fan.center.VectorTo(chordMid)
	foot := fan.center.TranslateBy(uhat.Scale(w.Dot(uhat)))
	radial := foot.VectorTo(chordMid) // perpendicular to the axis by construction
	if radial.Length() < 1e-9*fan.radius {
		return math.Point3{}, false
	}
	return foot.TranslateBy(unit(radial).Scale(fan.radius)), true
}

// monotoneAroundAxis is the non-self-intersection certificate: the boundary chain
// tA → (ordered far-edge splits) → tB must wind about the fillet axis strictly in one rotational
// sense and by less than a full turn. A fold (a step that reverses the winding) or a full wrap means
// the cap section self-intersects (math advisor invariant 3). ta is the reference (angle 0);
// wraparound is handled by scoring each STEP as a signed delta in (−π,π], not by comparing absolute
// [0,2π) angles, so the test is immune to the 0/2π seam.
func monotoneAroundAxis(fan endCornerFan, sp runoutSpread) bool {
	uhat := unit(fan.axis)
	ref := unit(fan.center.VectorTo(fan.ta))
	return windsMonotone(uhat, ref, fan.center, runoutBoundarySeq(fan, sp))
}

// runoutBoundarySeq is the ordered boundary polyline the certificate checks: tA, each far-edge split
// in fan order, then tB.
func runoutBoundarySeq(fan endCornerFan, sp runoutSpread) []math.Point3 {
	seq := make([]math.Point3, 0, len(fan.farEdges)+2)
	seq = append(seq, fan.ta)
	for _, fe := range fan.farEdges {
		seq = append(seq, sp.splits[fe.edge])
	}
	return append(seq, fan.tb)
}

// windsMonotone reports whether seq advances about û (through c, from ref) strictly in one
// rotational sense and by less than a full turn. Each step's signed delta is wrapped to (−π,π] so a
// near-zero (coincident) or sign-flipping (fold) step, or a cumulative |turn| ≥ 2π (self-overlap),
// fails it.
func windsMonotone(uhat, ref math.Vector3, c math.Point3, seq []math.Point3) bool {
	const eps = 1e-9
	prev := angleAbout(uhat, ref, c, seq[0])
	sign, total := 0.0, 0.0
	for i := 1; i < len(seq); i++ {
		a := angleAbout(uhat, ref, c, seq[i])
		d := wrapPi(a - prev)
		if stdmath.Abs(d) < eps || (sign != 0 && d*sign < 0) {
			return false
		}
		sign, total, prev = signOf(d), total+d, a
	}
	return stdmath.Abs(total) < 2*stdmath.Pi-eps
}

// signOf returns −1 for a negative x and +1 otherwise (the rotational sense of a step).
func signOf(x float64) float64 {
	if x < 0 {
		return -1
	}
	return 1
}

// angleAbout returns the angle (0..2π) of point p about axis û through c, measured from ref.
func angleAbout(uhat, ref math.Vector3, c, p math.Point3) float64 {
	w := c.VectorTo(p)
	inPlane := w.Add(uhat.Scale(-w.Dot(uhat)))
	x := float64(inPlane.Dot(ref))
	y := float64(inPlane.Dot(uhat.Cross(ref)))
	a := stdmath.Atan2(y, x)
	if a < 0 {
		a += 2 * stdmath.Pi
	}
	return a
}

// boundaryPoint resolves one end of a far face's piece: the rail point (ta or tb) at the flank
// (sentinel edge==0), else the split shared with the adjacent far face on the bounding far edge —
// the read that makes the weld-twice invariant hold (both neighbours read the same sp.splits entry).
func boundaryPoint(sp runoutSpread, edge uint64, isFlank bool, rail math.Point3) math.Point3 {
	if isFlank && edge == 0 {
		return rail
	}
	return sp.splits[edge]
}

// filletRunoutError reports an n-valent runout certificate failure with the offending fillet edge,
// vertex valence, apex location, and the far edge that failed, plus the standard remediation — the
// generalisation of the #1800 over-radius reject to N>3 corners. Task 5/7 add more certificate
// checks that funnel through this constructor.
func filletRunoutError(fan endCornerFan, reason string, edge uint64) error {
	return fmt.Errorf("fillet: cannot round edge %d at a %d-valent runout vertex %v — %s (edge %d); reduce the radius or fillet the neighbours first",
		fan.filletEdge, len(fan.fan)+2, fan.apex, reason, edge)
}

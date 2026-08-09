// SPDX-License-Identifier: GPL-2.0-only

package feature

import (
	"fmt"
	stdmath "math"

	"oblikovati.org/api/types"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
)

// CORNER RELIEF (#2072). Where two walls meet, their bends run into each other: each one wants the
// material at the corner, and folded flat there is not enough of it. The corner is relieved by
// removing that shared material before either bend reaches it.
//
// A corner is found from the BENDS, not from the topology: two bend lines that share an endpoint
// and run in different directions are two walls folding away from one corner. That is data the
// flat pattern already records for every wall, so nothing new has to be searched for — and it is
// why the feature building the SECOND wall is where the junction first exists.

// cornerJunctionTol is how close two bend-line ends must be to count as the same corner. Bend
// lines are built from the same picked edges, so a real corner is exact to rounding; the tolerance
// only absorbs that.
const cornerJunctionTol = 1e-6

// CornerReliefSpec is the styled corner relief, resolved for one recompute: the shape and the size
// of the cut. Like [ReliefSpec] it arrives through [Input] because a shape is not a number.
type CornerReliefSpec struct {
	Shape types.CornerReliefShape
	Size  float64
}

// cuts reports whether this relief removes material. A tear corner is the deliberate absence of a
// cut, exactly as it is for a bend relief.
func (c CornerReliefSpec) cuts() bool { return c.Shape != types.CornerTear }

// checkBendTransition reports whether a transition can be built where two walls meet (#1959).
//
// Only "none" can, and that is Inventor's default. The other three shaping forms — intersection,
// straight line and arc — describe the FLAT PATTERN's outline through the transition region, and
// this flat pattern does not model that region at all: it lays each wall out as a tab from its bend
// line and unions them. Trim-to-bend is a folded-model cut, but "perpendicular to the bent feature"
// needs the bent feature's own outline, not just its bend line.
//
// They are refused rather than ignored, and refused only AT a junction — a part whose style names
// a transition it never reaches builds fine, because there was nothing to shape.
func checkBendTransition(t types.BendTransition) error {
	if t == types.NoBendTransition || t == types.DefaultBendTransition {
		return nil
	}
	return fmt.Errorf("sheet-metal bend transition %v is not built yet: the shaping transitions "+
		"describe the FLAT PATTERN's outline through the transition region, which this flat does "+
		"not model, and trim-to-bend needs the adjacent wall's outline rather than its bend line; "+
		"use \"none\"", t)
}

// bendJunction is one corner: where the two bend lines meet, and the two bends that make it.
type bendJunction struct {
	at   math.Point3
	a, b BendPlacement
}

// findBendJunction reports where this bend meets one of the bends already placed. Two ends coincide
// at a corner; two bends running the SAME way are one bend split in two (a flange with a gap in
// it), which shares no corner and needs no relief.
func findBendJunction(bend BendPlacement, prior []BendPlacement) (bendJunction, bool) {
	for _, p := range prior {
		if parallelBends(bend, p) {
			continue
		}
		if at, ok := sharedEnd(bend, p); ok {
			return bendJunction{at: at, a: bend, b: p}, true
		}
	}
	return bendJunction{}, false
}

// parallelBends reports whether two bends fold about the same line direction — two stretches of
// one edge rather than two sides of a corner.
func parallelBends(a, b BendPlacement) bool {
	da, err := math.UnitVector3FromVector(a.AxisStart.VectorTo(a.AxisEnd))
	if err != nil {
		return true // a degenerate bend corners with nothing
	}
	db, err := math.UnitVector3FromVector(b.AxisStart.VectorTo(b.AxisEnd))
	if err != nil {
		return true
	}
	return stdmath.Abs(float64(da.AsVector().Dot(db.AsVector()))) > 1-cornerJunctionTol
}

// sharedEnd returns the point where the two bend lines touch, if they do.
func sharedEnd(a, b BendPlacement) (math.Point3, bool) {
	for _, pa := range []math.Point3{a.AxisStart, a.AxisEnd} {
		for _, pb := range []math.Point3{b.AxisStart, b.AxisEnd} {
			if float64(pa.DistanceTo(pb)) < cornerJunctionTol {
				return pa, true
			}
		}
	}
	return math.Point3{}, false
}

// cutCornerRelief removes the styled corner cut where this bend meets an earlier one, and settles
// the bend transition there. Bodies come back unchanged when there is no junction, or when the
// style leaves the corner to tear.
func cutCornerRelief(bodies []*topo.Body, bend BendPlacement, in Input, transition types.BendTransition,
	feat string) ([]*topo.Body, error) {
	junction, ok := findBendJunction(bend, in.PriorBends)
	if !ok {
		return bodies, nil // no junction: no corner to relieve and no transition to shape
	}
	if err := checkBendTransition(transition); err != nil {
		return nil, err
	}
	if !in.Corner.cuts() {
		return bodies, nil
	}
	reach, err := cornerReach(junction, in.Corner)
	if err != nil {
		return nil, err
	}
	tool, err := cornerReliefTool(junction, in.Corner, reach, feat)
	if err != nil {
		return nil, err
	}
	return cutFrom(bodies, tool)
}

// cornerReach is how far the cut extends along each bend line from the corner.
//
// The size-driven shapes take it from the style. TRIM TO BEND takes it from the bends themselves:
// trimming "to the bend" means back to where each bend's outer surface leaves the flat, which is
// its radius plus the material thickness — a number the style does not carry because it is a
// property of the bends being relieved.
func cornerReach(j bendJunction, spec CornerReliefSpec) (float64, error) {
	switch spec.Shape {
	case types.CornerTrimToBend:
		return stdmath.Max(j.a.Radius+j.a.Thickness, j.b.Radius+j.b.Thickness), nil
	case types.CornerRound, types.CornerSquare:
		if spec.Size <= 0 {
			return 0, fmt.Errorf("sheet-metal corner relief: the %v shape is sized by the style, and "+
				"its size is %g; give a positive corner relief size", spec.Shape, spec.Size)
		}
		return spec.Size, nil
	default:
		return 0, fmt.Errorf("sheet-metal corner relief: the %v shape is not built yet — it needs the "+
			"two walls' own outlines, not just their bend lines; use trimToBend, round or square", spec.Shape)
	}
}

// cornerReliefTool builds the cut at the junction: a prism through the material in the plane of the
// parent face, square or round according to the shape.
func cornerReliefTool(j bendJunction, spec CornerReliefSpec, reach float64, feat string) (*topo.Body, error) {
	u, v, err := cornerAxes(j)
	if err != nil {
		return nil, err
	}
	plane := planeFromFrame(j.at, j.a.Up, u)
	sign := 1.0
	if plane.YAxis().AsVector().Dot(v.AsVector()) < 0 {
		sign = -1
	}
	thickness := stdmath.Max(j.a.Thickness, j.b.Thickness)
	pad := thickness
	poly := cornerProfile(spec.Shape, reach, sign)
	return buildPrism(poly, plane, span{near: -thickness - pad, far: pad}, 0, feat+"/corner"), nil
}

// cornerAxes returns the two in-plane directions the cut spreads along: away from the corner down
// each bend line, into the material the two walls share.
func cornerAxes(j bendJunction) (u, v math.UnitVector3, err error) {
	if u, err = awayFromCorner(j.at, j.a); err != nil {
		return u, v, err
	}
	v, err = awayFromCorner(j.at, j.b)
	return u, v, err
}

// awayFromCorner is the direction along a bend line leading away from the corner point.
func awayFromCorner(at math.Point3, bend BendPlacement) (math.UnitVector3, error) {
	far := bend.AxisEnd
	if float64(at.DistanceTo(far)) < cornerJunctionTol {
		far = bend.AxisStart
	}
	return math.UnitVector3FromVector(at.VectorTo(far))
}

// cornerProfile is the cut's outline in the parent face's plane, with the corner at the origin: a
// square of the reach along both axes, or the quarter-round that replaces its inside corner.
func cornerProfile(shape types.CornerReliefShape, reach, sign float64) []math.Point2 {
	if shape != types.CornerRound {
		return ensureCCW2([]math.Point2{
			math.P2(0, 0), math.P2(reach, 0), math.P2(reach, reach*sign), math.P2(0, reach*sign),
		})
	}
	// A round corner relief is the disc of the styled size seated in the corner: its arc runs from
	// one bend line to the other, so neither bend meets a sharp inside corner.
	pts := []math.Point2{math.P2(0, 0)}
	const arcSteps = 12
	for k := 0; k <= arcSteps; k++ {
		phi := stdmath.Pi / 2 * float64(k) / arcSteps
		pts = append(pts, math.P2(reach*stdmath.Cos(phi), reach*stdmath.Sin(phi)*sign))
	}
	return ensureCCW2(pts)
}

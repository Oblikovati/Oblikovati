// SPDX-License-Identifier: GPL-2.0-only

package feature

import (
	stdmath "math"

	"oblikovati.org/kernel/ops"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
)

// AUTO-MITER (#1961). Two walls folding away from one corner stop at their own bend lines, so the
// corner between them is open: measuring the flat pattern of an L-bracket shows the two tabs
// meeting at a POINT, with the square beyond that corner empty. Auto-mitering fills it — Inventor's
// "automatically extends material between adjacent flange edges" — and then cuts a gap on the
// bisector so the two extensions do not fight for the same space when the part folds.
//
// A wall is rebuilt from its own [BendPlacement]: the bend line, the frame, the angle, the radius
// and the run are exactly the section the wall was extruded from, so the extension is the same band
// over a different span rather than a second guess at the wall's shape.

// mitreCorner fills the corner between this bend and one already placed, then cuts the miter gap.
// Bodies come back untouched when the walls do not corner, or when the caller is not mitering.
func mitreCorner(bodies []*topo.Body, bend BendPlacement, in Input, gap float64,
	feat string) ([]*topo.Body, error) {
	junction, ok := findBendJunction(bend, in.PriorBends)
	if !ok {
		return bodies, nil
	}
	return mitreFillAtJunction(bodies, junction, gap, feat)
}

// mitreFillAtJunction carries both walls of a junction past the corner, joins them, then cuts the
// styled gap on the bisector so the folded part has clearance. It is the shared corner fill used by
// the auto-miter (#1961) and by the corner-seam butt/lap styles (#2085) — the latter own no bend,
// so they discover the junction themselves and hand it here. A gap <= 0 butts the two walls tight.
func mitreFillAtJunction(bodies []*topo.Body, junction bendJunction, gap float64,
	feat string) ([]*topo.Body, error) {
	out := bodies
	for _, e := range mitreExtensions(junction) {
		wall, err := extendWall(e, feat)
		if err != nil {
			return nil, err
		}
		if wall == nil {
			continue
		}
		if out, err = combine(Input{Bodies: out}, wall, ops.Join); err != nil {
			return nil, err
		}
	}
	if gap <= 0 {
		return out, nil
	}
	return cutFrom(out, mitreGapTool(junction, gap, feat))
}

// wallExtension is one wall carried past the corner: the bend it belongs to, and how far beyond
// the corner its section is extruded.
type wallExtension struct {
	bend  BendPlacement
	at    math.Point3 // the corner
	reach float64
}

// mitreExtensions works out how far each wall has to run to reach the other. A wall stands off its
// own bend line by the depth of its bend, so the distance one must travel PAST the corner to meet
// the other is exactly that other wall's stand-off — which is why each extension is sized by its
// neighbour and not by itself.
func mitreExtensions(j bendJunction) []wallExtension {
	offA, offB := wallStandOff(j.a), wallStandOff(j.b)
	return []wallExtension{
		{bend: j.a, at: j.at, reach: offB},
		{bend: j.b, at: j.at, reach: offA},
	}
}

// wallStandOff is how far a wall's material sits from its bend line, measured along the bend's
// outward direction. It is read off the band the wall was built from, so a change to the section
// (a different angle, radius or bend position) moves the miter with it.
func wallStandOff(bend BendPlacement) float64 {
	steps := []bendRun{{Angle: bend.Angle, Radius: bend.Radius, Run: bend.Length}}
	out, up := bend.Outward.AsVector(), bend.Up.AsVector()
	far := 0.0
	for _, p := range bandPolygon(steps, out, up, bend.Thickness, func(w math.Vector3) math.Point2 {
		return math.P2(w.Dot(out), w.Dot(up))
	}) {
		if d := stdmath.Abs(float64(p.X)); d > far {
			far = d
		}
	}
	return far
}

// extendWall rebuilds a wall's section and extrudes it over the span BEYOND the corner, which is
// the material the open corner was missing.
func extendWall(e wallExtension, feat string) (*topo.Body, error) {
	if e.reach <= 0 {
		return nil, nil
	}
	along, err := math.UnitVector3FromVector(e.bend.AxisStart.VectorTo(e.bend.AxisEnd))
	if err != nil {
		return nil, err
	}
	// The corner is one end of the bend line; the extension runs the other way, out past it.
	from := float64(e.bend.AxisStart.VectorTo(e.at).Dot(along.AsVector()))
	sign := 1.0
	if from > float64(e.bend.AxisStart.DistanceTo(e.bend.AxisEnd))/2 {
		sign = -1 // the corner is at the far end, so the extension runs back the other way
	}
	steps := []bendRun{{Angle: e.bend.Angle, Radius: e.bend.Radius, Run: e.bend.Length}}
	plane := planePerp(e.bend.AxisStart, along)
	u, v := plane.XAxis().AsVector(), plane.YAxis().AsVector()
	poly := bandPolygon(steps, e.bend.Outward.AsVector(), e.bend.Up.AsVector(), e.bend.Thickness,
		func(w math.Vector3) math.Point2 { return math.P2(w.Dot(u), w.Dot(v)) })
	lo, hi := stdmath.Min(from, from-sign*e.reach), stdmath.Max(from, from-sign*e.reach)
	return buildPrism(poly, plane, span{near: lo, far: hi}, 0, feat+"/miter"), nil
}

// mitreGapTool is the slot cut on the miter line: a slab of the styled gap width, standing on the
// bisector of the two walls and running through both extensions. Without it the two extensions
// occupy the same corner and the fold has nowhere to go.
func mitreGapTool(j bendJunction, gap float64, feat string) *topo.Body {
	normal, err := mitreNormal(j)
	if err != nil {
		return nil
	}
	plane := planeFromFrame(j.at, normal, j.a.Up)
	reach := 4 * (wallStandOff(j.a) + wallStandOff(j.b) + j.a.Length + j.b.Length + gap)
	poly := []math.Point2{
		math.P2(-reach, -reach), math.P2(reach, -reach), math.P2(reach, reach), math.P2(-reach, reach),
	}
	return buildPrism(ensureCCW2(poly), plane, span{near: -gap / 2, far: gap / 2}, 0, feat+"/miterGap")
}

// mitreNormal is the direction across the miter cut: the bisector of the two walls' outward
// directions, which for a right-angled corner is the 45° line the two extensions meet on.
func mitreNormal(j bendJunction) (math.UnitVector3, error) {
	sum := j.a.Outward.AsVector().Add(j.b.Outward.AsVector())
	return math.UnitVector3FromVector(math.V3(float64(sum.Y), -float64(sum.X), 0))
}

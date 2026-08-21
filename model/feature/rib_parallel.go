// SPDX-License-Identifier: GPL-2.0-only

package feature

import (
	"errors"
	"fmt"
	stdmath "math"

	"oblikovati.org/kernel/ops"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
	"oblikovati.org/model/sketch"
)

// The PARALLEL (lateral) rib — Inventor's RibDefinition.IsRib = True (#2064). Where the normal/web
// rib thickens the path IN the sketch plane and extrudes ALONG the plane normal, the parallel rib
// does the opposite: it thickens the profile ALONG the plane normal (a thin wall of the given
// thickness, centred on the sketch plane) and grows that wall IN the plane, perpendicular to the
// path, until it lands on the part. The result is the same wall rotated 90° — the form used for a
// moulded part whose sketch is a section through its wall. A draft is refused (Inventor allows a
// taper only when the direction is normal to the plane).

// recomputeParallel builds the lateral rib: a footprint that is the path grown perpendicular in the
// plane by the depth, extruded by the thickness ALONG the plane normal (centred, no taper).
func (r *RibFeature) recomputeParallel(in Input, pts []math.Point2, t float64) (Output, error) {
	if r.def.Draft != 0 {
		return Output{}, errors.New("rib: a draft/taper needs the direction NORMAL to the sketch " +
			"plane; Inventor refuses a taper on a parallel (lateral) rib, so this one does too")
	}
	plane := r.def.Sketch.Plane()
	toLeft, depth, err := r.parallelGrowth(in, pts)
	if err != nil {
		return Output{}, err
	}
	footprint := ensureCCW2(parallelFootprint(pts, toLeft, depth))
	// Thickness along the plane NORMAL, centred on the sketch plane; no taper (refused above). The
	// Surface operation builds walls only (no caps), as the normal form does.
	r.tool = buildExtrusionShell(footprint, plane, orderedSpan(-t/2, t/2), 0, "rib", r.def.Operation != ops.Surface)
	bodies, err := combine(in, r.tool, r.def.Operation)
	if err != nil {
		return Output{}, err
	}
	return Output{Bodies: bodies}, nil
}

// parallelGrowth resolves how far and which way the lateral wall grows in the plane: perpendicular
// to the path, toward the side the part lies on. toLeft is the growth side (the path's left normal);
// depth is |Depth| for a finite rib, or the FARTHEST distance to the part for a to-next rib (so the
// whole wall lands, the deepest ray governing exactly as the normal rib's to-next does).
func (r *RibFeature) parallelGrowth(in Input, pts []math.Point2) (toLeft bool, depth float64, err error) {
	plane := r.def.Sketch.Plane()
	perp := avgPathNormal(pts)
	if perp == (math.Vector2{}) {
		return false, 0, errors.New("rib: the parallel profile is degenerate — no direction to grow the wall perpendicular to")
	}
	toLeft, err = r.parallelGrowthSide(in, plane, pts, perp)
	if err != nil {
		return false, 0, err
	}
	if r.def.ToNext {
		depth, err = parallelToNextDepth(in.Bodies, plane, pts, growthVector(perp, toLeft))
		return toLeft, depth, err
	}
	depth = stdmath.Abs(callOrZero(r.def.Depth))
	if depth == 0 {
		return toLeft, 0, errors.New("rib: a parallel rib needs a non-zero depth (or set toNext to grow onto the part)")
	}
	return toLeft, depth, nil
}

// parallelGrowthSide reports which way the wall grows — toward whichever side of the path the
// material lies on, sensed by ray-casting the path's midpoint both ways along the perpendicular. A
// finite rib with no material either way grows to the left by convention; a to-next rib errors.
func (r *RibFeature) parallelGrowthSide(in Input, plane sketch.Plane, pts []math.Point2, perp math.Vector2) (bool, error) {
	mid := plane.ToModel(pts[len(pts)/2])
	leftHit, leftOK := nearestBodyHit(in.Bodies, mid, planeDirectionToModel(plane, perp))
	rightHit, rightOK := nearestBodyHit(in.Bodies, mid, planeDirectionToModel(plane, perp.Scale(-1)))
	switch {
	case leftOK && (!rightOK || leftHit <= rightHit):
		return true, nil
	case rightOK:
		return false, nil
	case !r.def.ToNext:
		return true, nil // no material to sense; a finite rib grows to the left by convention
	default:
		return false, errors.New("rib: parallel to-next found no material on either side of the profile to grow the wall onto")
	}
}

// parallelToNextDepth is the parallel rib's to-next extent: it ray-casts each profile point along
// the in-plane growth direction and returns the FARTHEST first-hit, so growing the wall that far
// lands every point of it on the part (the boolean trims the overshoot at the nearer points).
func parallelToNextDepth(bodies []*topo.Body, plane sketch.Plane, pts []math.Point2, grow math.Vector2) (float64, error) {
	if len(bodies) == 0 {
		return 0, errors.New("rib: parallel to-next needs existing material")
	}
	dir := planeDirectionToModel(plane, grow)
	deepest := 0.0
	for i, p := range pts {
		hit, ok := nearestBodyHit(bodies, plane.ToModel(p), dir)
		if !ok {
			return 0, fmt.Errorf("rib: parallel to-next: profile point %d (%v) has no material ahead to grow the wall onto", i, p)
		}
		deepest = stdmath.Max(deepest, hit)
	}
	if deepest <= 0 {
		return 0, errors.New("rib: parallel to-next found the profile already sitting on the material; nothing to grow")
	}
	return deepest, nil
}

// parallelFootprint is the in-plane region the lateral wall covers: the closed band between the path
// and the path offset by depth on the growth side (thickenPathBetween along the per-vertex normal,
// so it follows a curved path exactly as the normal rib's band does).
func parallelFootprint(pts []math.Point2, toLeft bool, depth float64) []math.Point2 {
	if toLeft {
		return thickenPathBetween(pts, depth, 0)
	}
	return thickenPathBetween(pts, 0, -depth)
}

// avgPathNormal is the path's mean in-plane left normal — the perpendicular the lateral wall grows
// along. Averaging over the vertices gives one stable direction for the side test on a gently
// curved path; thickenPathBetween still offsets per-vertex, so the footprint itself stays exact.
func avgPathNormal(pts []math.Point2) math.Vector2 {
	var sum math.Vector2
	for i := range pts {
		sum = sum.Add(vertexNormal2(pts, i))
	}
	if l := float64(sum.Length()); l > 0 {
		return sum.Scale(math.Scalar(1 / l))
	}
	return math.Vector2{}
}

// growthVector is the perpendicular pointed to the growth side.
func growthVector(perp math.Vector2, toLeft bool) math.Vector2 {
	if toLeft {
		return perp
	}
	return perp.Scale(-1)
}

// SPDX-License-Identifier: GPL-2.0-only

package feature

import (
	"fmt"
	stdmath "math"

	"oblikovati.org/kernel/ops"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
	"oblikovati.org/model/sketch"
)

// Shaping a rib's wall from its open path: which side of the path the thickness lands on, how a
// draft tapers the wall, and extending the path's ends onto the part (#1882).
//
// ROOT means the end of the extrusion that lands on the existing material — the far end from the
// sketch plane, where the rib attaches. Everything below is defined against that end, because
// that is the end a moulding draft has to open away from.

// RibThickenSide is which side of the path a rib's wall grows on (Inventor's
// RibDefinition.ThicknessDirection, whose enum is PartFeatureExtentDirectionEnum).
type RibThickenSide int

const (
	// RibThickenSymmetric puts half the thickness on each side of the path. The default, and the
	// zero value, so a definition written before the option keeps its behaviour (#1882).
	RibThickenSymmetric RibThickenSide = iota
	// RibThickenSide1 grows the whole wall on the path's LEFT side, walking the path as drawn.
	RibThickenSide1
	// RibThickenSide2 grows it entirely on the path's RIGHT side.
	RibThickenSide2
)

// bandOffsets is the pair of signed in-plane offsets (left, right) that place a wall of the given
// width on this side of the path.
func (s RibThickenSide) bandOffsets(width float64) (left, right float64) {
	switch s {
	case RibThickenSide1:
		return width, 0
	case RibThickenSide2:
		return 0, -width
	default:
		return width / 2, -width / 2
	}
}

// ribWallBand builds the wall's cross-section at the extrusion's BASE, plus the taper that
// carries it to the far end. The two come out together because a draft makes the two ends
// different widths, and which end the band describes depends on which way the rib grows.
func ribWallBand(pts []math.Point2, t, depth, draft float64, side RibThickenSide,
	atRoot bool) ([]math.Point2, float64, error) {
	width, taper, err := ribDraftedWall(t, depth, draft, atRoot)
	if err != nil {
		return nil, 0, err
	}
	left, right := side.bandOffsets(width)
	return ensureCCW2(thickenPathBetween(pts, left, right)), taper, nil
}

// ribDraftedWall resolves the wall width at the extrusion's base and the taper that carries it to
// the far end.
//
// A draft opens the wall toward the ROOT, so with a positive depth (the base IS the sketch plane)
// the wall widens along the extrusion; with a negative depth the base is the root and the wall
// must instead NARROW toward the sketch plane — hence the flipped taper. Without that flip the
// draft would open away from the part for one depth sign and toward it for the other.
//
// atRoot (Inventor's kRibThicknessAtRoot) says which end honours the nominal thickness t; with no
// draft both ends are t and the choice has no effect.
func ribDraftedWall(t, depth, draft float64, atRoot bool) (base, taper float64, err error) {
	grow := stdmath.Abs(depth) * stdmath.Tan(draft) // half-width the draft adds at the root
	sketchWidth, rootWidth := t, t+2*grow
	if atRoot {
		sketchWidth, rootWidth = t-2*grow, t
	}
	if sketchWidth <= 0 || rootWidth <= 0 {
		return 0, 0, fmt.Errorf("rib: draft %g rad over depth %g leaves the wall %g wide at the "+
			"sketch plane and %g at the root, from a thickness of %g held at the %s; reduce the "+
			"draft or the depth", draft, depth, sketchWidth, rootWidth, t, holdName(atRoot))
	}
	if depth < 0 {
		return rootWidth, -draft, nil
	}
	return sketchWidth, draft, nil
}

// holdName names the thickness plane in errors.
func holdName(atRoot bool) string {
	if atRoot {
		return "root"
	}
	return "sketch plane"
}

// thickenPathBetween offsets a polyline to a closed band polygon: the left offset forward along
// the path, the right offset back, so the two runs close into one loop. The offsets are signed
// along the path's in-plane LEFT normal, which is what lets the band sit on either side of the
// path or straddle it (see [RibThickenSide.bandOffsets]).
func thickenPathBetween(pts []math.Point2, left, right float64) []math.Point2 {
	n := len(pts)
	band := make([]math.Point2, 0, 2*n)
	for i := range n {
		band = append(band, pts[i].TranslateBy(vertexNormal2(pts, i).Scale(math.Scalar(left))))
	}
	for i := n - 1; i >= 0; i-- {
		band = append(band, pts[i].TranslateBy(vertexNormal2(pts, i).Scale(math.Scalar(right))))
	}
	return band
}

// vertexNormal2 is the unit in-plane normal at vertex i (averaged perpendicular of the
// adjacent segments).
func vertexNormal2(pts []math.Point2, i int) math.Vector2 {
	var sum math.Vector2
	if i > 0 {
		sum = sum.Add(segNormal2(pts[i-1], pts[i]))
	}
	if i < len(pts)-1 {
		sum = sum.Add(segNormal2(pts[i], pts[i+1]))
	}
	if l := float64(sum.Length()); l > 0 {
		return sum.Scale(math.Scalar(1 / l))
	}
	return math.V2(0, 0)
}

// segNormal2 is the unit left normal of segment a→b.
func segNormal2(a, b math.Point2) math.Vector2 {
	d := a.VectorTo(b)
	n := math.V2(-d.Y, d.X)
	if l := float64(n.Length()); l > 0 {
		return n.Scale(math.Scalar(1 / l))
	}
	return math.V2(0, 0)
}

// ribExtendedPath lengthens the open path's two ends along their end tangents until they reach the
// existing material, so a wall sketched short of the part still lands on it (Inventor's
// RibDefinition.ExtendProfile, #1882).
//
// An end with nothing ahead of it is left where it is — there is nothing to extend to, and the
// common case is a profile that already spans the part. An end already INSIDE the material is left
// alone too: a ray from in there would run to the far wall and stretch the rib clean through.
func ribExtendedPath(pts []math.Point2, plane sketch.Plane, bodies []*topo.Body) []math.Point2 {
	if len(pts) < 2 || len(bodies) == 0 {
		return pts
	}
	out := append([]math.Point2(nil), pts...)
	last := len(out) - 1
	out[0] = ribExtendedEnd(out[0], out[1], plane, bodies)
	out[last] = ribExtendedEnd(out[last], out[last-1], plane, bodies)
	return out
}

// ribExtendedEnd walks end directly away from its inner neighbour until it meets material.
func ribExtendedEnd(end, inner math.Point2, plane sketch.Plane, bodies []*topo.Body) math.Point2 {
	dir, err := math.UnitVector2FromVector(inner.VectorTo(end))
	if err != nil {
		return end // a zero-length end segment has no direction to extend along
	}
	origin, ray := plane.ToModel(end), planeDirectionToModel(plane, dir.AsVector())
	for _, b := range bodies {
		if ops.PointInsideBody(b, origin) {
			return end
		}
	}
	hit, ok := nearestBodyHit(bodies, origin, ray)
	if !ok {
		return end
	}
	// plane.ToModel is an isometry, so the model-space hit distance is the in-plane distance.
	return end.TranslateBy(dir.AsVector().Scale(math.Scalar(hit)))
}

// planeDirectionToModel maps an in-plane direction to its model-space vector.
func planeDirectionToModel(plane sketch.Plane, v math.Vector2) math.Vector3 {
	x := plane.XAxis().AsVector().Scale(v.X)
	return x.Add(plane.YAxis().AsVector().Scale(v.Y))
}

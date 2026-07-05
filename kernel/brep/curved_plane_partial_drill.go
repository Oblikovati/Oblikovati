// SPDX-License-Identifier: GPL-2.0-only

package brep

import (
	stdmath "math"

	"oblikovati.org/kernel/geom"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
)

// Partial cylindrical drill "scallop" — the curved-on-planar boolean KIND for a through-hole whose circle
// CLIPS a slab edge (an edge notch), #1591 / ADR-0049. The SUBTRACTIVE counterpart to JoinPartialBoss:
// where DrillThroughHole needs the circle strictly inside both cap faces, this handles the circle crossing
// one shared edge of the two parallel cap faces — the two caps keep {plate minus footprint} (planeUV trims),
// the pierced SIDE face is split in two by the removed strip, and a PARTIAL cylinder wall (the inside-plate
// arc, an iso-(u,v) rectangle) closes the notch with its normal pointing inward into the removed material.
//
// Scope (Slice A', mirrors Slice B's single-edge discipline): the drill axis is perpendicular to two parallel
// planar cap faces it pierces through, and the circle clips exactly ONE shared edge of those faces (two
// crossings on one polygon edge), whose SIDE face is planar and parallel to the axis. Anything else defers.

// scallopCap is one of the two parallel cap faces the drill pierces, with the exact circle in its plane.
type scallopCap struct {
	idx    int
	circle geom.Circle
	plane  geom.Plane
	face   curvedFace
}

// CutEdgeScallop returns target − tool when tool is a cylinder drilled perpendicular THROUGH two parallel
// planar faces of target with its circle CLIPPING one shared edge of those faces (an edge scallop), or
// ok=false to defer (a clean interior hole takes DrillThroughHole; anything else keeps CSG).
func CutEdgeScallop(target, tool *topo.Body) (*topo.Body, bool) {
	cyl, base, height, ok := cylinderSolidParams(facesOfAny(tool))
	if !ok {
		return nil, false
	}
	ua := cyl.AxisDir.AsVector()
	faces := facesOfAny(target)
	caps, ok := partialDrillCaps(faces, base, ua, height, cyl.Radius)
	if !ok {
		return nil, false
	}
	body := assembleScallop(faces, caps, ua, cyl.Radius)
	if body == nil {
		return nil, false
	}
	return body, true
}

// partialDrillCaps finds the two parallel planar faces the drill axis pierces perpendicular, each with the
// drill circle CLIPPING (pierced && !clean) its boundary. Ordered low→high along the axis. ok=false unless
// exactly two such faces exist (a through scallop that clips both caps).
func partialDrillCaps(faces []curvedFace, base math.Point3, ua math.Vector3, height, radius float64) ([2]scallopCap, bool) {
	tol := geom.ResolutionForSize(height).Plane()
	var found []scallopCap
	for i, f := range faces {
		pl, isPlane := f.surface.(geom.Plane)
		if !isPlane || stdmath.Abs(float64(unit(pl.Normal()).Dot(ua))) < 1-1e-7 {
			continue // not perpendicular to the drill axis
		}
		t := pierceParam(base, ua, pl)
		if t < -tol || t > height+tol {
			continue // the cap lies OUTSIDE the cylinder's axial extent — the drill never reaches it (a blind stub)
		}
		center := base.TranslateBy(ua.Scale(math.Scalar(t)))
		circ, err := geom.NewCircle(center, ua, radius)
		if err != nil {
			continue
		}
		if pierced, clean := circleVsCap(center, radius, f, pl); pierced && !clean {
			found = append(found, scallopCap{idx: i, circle: circ, plane: pl, face: f})
		}
	}
	if len(found) != 2 {
		return [2]scallopCap{}, false
	}
	if projectOnAxis(found[1].circle.Center, ua) < projectOnAxis(found[0].circle.Center, ua) {
		found[0], found[1] = found[1], found[0]
	}
	return [2]scallopCap{found[0], found[1]}, true
}

// projectOnAxis returns the signed distance of a point along the axis direction (an ordering key).
func projectOnAxis(p math.Point3, ua math.Vector3) float64 {
	return float64(math.P3(0, 0, 0).VectorTo(p).Dot(ua))
}

// assembleScallop builds the cut: both caps trimmed by the circle (planeMaterial), the pierced side face
// split in two, and the partial cylinder wall closing the notch. nil to defer (out of Slice A' scope).
func assembleScallop(faces []curvedFace, caps [2]scallopCap, ua math.Vector3, radius float64) *topo.Body {
	low, high := caps[0], caps[1]
	ta, tb, cols, ok := scallopCrossingArc(low, high, ua, radius)
	if !ok {
		return nil
	}
	lowOut, ok := trimCap(low, ua, radius)
	highOut, ok2 := trimCap(high, ua, radius)
	if !ok || !ok2 {
		return nil
	}
	sideOut, sideIdx, ok := splitScallopSide(faces, ua, cols, low.idx, high.idx)
	if !ok {
		return nil
	}
	out := copyExceptSet(faces, map[int]bool{low.idx: true, high.idx: true, sideIdx: true})
	out = append(out, lowOut...)
	out = append(out, highOut...)
	out = append(out, sideOut...)
	out = append(out, scallopWall(low.circle.Center, ua, radius, low.circle, high.circle, ta, tb))
	return curvedStitch(out)
}

// scallopCrossingArc computes the drill circle's two seat crossings (via the planeUV arrangement on the low
// cap), the inside-plate arc [ta,tb], and the four shared crossing columns. ok=false unless there are exactly
// two crossings on ONE polygon edge (Slice A' scope: a single clipped edge).
func scallopCrossingArc(low, high scallopCap, ua math.Vector3, radius float64) (ta, tb float64, cols [4]math.Point3, ok bool) {
	c := bossPlaneUV(low.face, low.plane, low.circle.Center, ua, radius)
	crossings := c.planeCrossingsOf(low.circle)
	if len(crossings) != 2 || crossings[0].edge != crossings[1].edge || crossings[0].loop != crossings[1].loop {
		return 0, 0, cols, false
	}
	ta, tb = insideArc(low.circle, crossings[0].tConic, crossings[1].tConic, low.face, low.plane)
	return ta, tb, columnPoints(low.circle, high.circle, ta, tb), true
}

// splitScallopSide finds the breached side face and splits it into the two pieces outside the removed strip,
// returning the pieces and the split face's index (to exclude it from the copied faces). ok=false if no side
// face carries the crossings or the split degenerates.
func splitScallopSide(faces []curvedFace, ua math.Vector3, cols [4]math.Point3, lowIdx, highIdx int) ([]curvedFace, int, bool) {
	sideIdx, _, ok := findScallopSide(faces, ua, cols, map[int]bool{lowIdx: true, highIdx: true})
	if !ok {
		return nil, 0, false
	}
	sideOut, ok := splitSideFace(faces[sideIdx], cols)
	if !ok {
		return nil, 0, false
	}
	return sideOut, sideIdx, true
}

// trimCap trims one cap face by the drill circle, keeping the plate minus the in-tool footprint (planeMaterial).
func trimCap(cap scallopCap, ua math.Vector3, radius float64) ([]curvedFace, bool) {
	c := bossPlaneUV(cap.face, cap.plane, cap.circle.Center, ua, radius)
	out, _, err := trimByImprint(c, cap.face, cap.plane, []geom.Curve3{cap.circle}, planeMaterial(c))
	if err != nil || len(out) == 0 {
		return nil, false
	}
	return out, true
}

// insideArc returns the drill circle's INSIDE-plate arc [ta,tb] (ta<tb, tb possibly >1 wrapping the seam):
// the arc between the two crossings whose midpoint sample lies inside the cap face polygon (the material that
// becomes the scallop wall). The complementary arc is the removed lune.
func insideArc(circ geom.Circle, t1, t2 float64, f curvedFace, pl geom.Plane) (ta, tb float64) {
	if t2 < t1 {
		t1, t2 = t2, t1
	}
	mid := (t1 + t2) / 2
	if pointInFace2D(to2D(pl, circ.PointAt(mid)), curvedCapAsPlanar(f, pl)) {
		return t1, t2
	}
	return t2, t1 + 1
}

// columnPoints returns the four shared crossing-column points [lowA, highA, lowB, highB] — the exact circle
// evaluations both caps, the wall and the side split all weld on (the single-source weld currency, #1591).
func columnPoints(low, high geom.Circle, ta, tb float64) [4]math.Point3 {
	return [4]math.Point3{low.PointAt(ta), high.PointAt(ta), low.PointAt(tb), high.PointAt(tb)}
}

// findScallopSide finds the target planar face parallel to the drill axis whose plane carries all four
// crossing columns — the side face the drill breaches and must be split. ok=false if none (out of scope).
func findScallopSide(faces []curvedFace, ua math.Vector3, cols [4]math.Point3, skip map[int]bool) (int, geom.Plane, bool) {
	for i, f := range faces {
		if skip[i] {
			continue
		}
		pl, isPlane := f.surface.(geom.Plane)
		if !isPlane || stdmath.Abs(float64(unit(pl.Normal()).Dot(ua))) > 1e-7 {
			continue // not parallel to the axis (not a breached side face)
		}
		tol := geom.ResolutionForSize(2 * float64(cols[0].DistanceTo(cols[2]))).Plane()
		onPlane := true
		for _, p := range cols {
			if stdmath.Abs(pointPlaneDistance(p, pl)) > tol {
				onPlane = false
				break
			}
		}
		if onPlane {
			return i, pl, true
		}
	}
	return 0, geom.Plane{}, false
}

// copyExceptSet returns all faces whose index is not in skip (a shallow copy; loops are shared, never mutated).
func copyExceptSet(faces []curvedFace, skip map[int]bool) []curvedFace {
	out := make([]curvedFace, 0, len(faces))
	for i, f := range faces {
		if !skip[i] {
			out = append(out, f)
		}
	}
	return out
}

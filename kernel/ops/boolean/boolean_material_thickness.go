// SPDX-License-Identifier: GPL-2.0-only

package boolean

import (
	"cmp"
	stdmath "math"
	"slices"

	"oblikovati.org/kernel/geom"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
)

// A solid body's own minimum WIDTH: the smallest extent it has in any direction its own boundary
// supplies.
//
// It used to be the smallest side of the axis-aligned bounding box, which measures the coordinate
// frame as much as the body. Measured on the RING corpus pair: the same 1e-10 drill read 2e-10 thick
// along Z, 3.4043798713069418 thick after a 37 degree turn about (1,1,0) and 2.0000046063728405e-10
// after 90 degrees — so the size classification REFUSED it at 0 and 90 degrees and BUILT it at 37
// (#3524). A user who rotates a part and gets a different answer has no reason to trust any answer.
//
// The MEASURE is the one the bounding box made — a width, the body's whole extent along a direction.
// Only the DIRECTIONS change: they come from the body's own faces (a plane's normal, a surface of
// revolution's axis and every direction across it) and, for a body whose boundary supplies fewer than
// three independent ones, from the principal frame of its own support points. All of those turn with
// the body, so the width does too.
//
// It stays a GLOBAL measure on purpose. A local reading — the gap between one pair of opposed faces,
// or the diameter of one closed cylindrical face — makes the answer depend on how the shape is
// modelled: a spool with a 1e-3 neck reads 0.002 through its neck cylinder and the whole flange
// diameter as a prism, and two representations of one shape would then decide differently. The
// operand's width is one number whichever way it was built.

// solidThickness is the minimum, over the directions its own boundary supplies, of the body's extent
// along that direction. The answer is the same for the body in any orientation.
//
// Read it as the operand's WIDTH and not as "the thinnest material anywhere in it": it is an extent,
// so a plate with a pocket in it reports the plate (TestAPocketFloorIsNotAPlateThickness pins the 10,
// not the 0.5 of roof), and a direction set is a sample of all directions, so a body thinnest in a
// direction no face names reports the thinnest one that IS named. Both err upward, which is the safe
// side for a classification that refuses.
//
// ok is false only for a nil, non-solid or faceless body — a sheet has no material to measure. Every
// solid with a face IS measured: a body whose boundary supplies fewer than three independent
// directions falls back to the principal frame of its own support points, so "unmeasured, therefore
// never refused" is not reachable (the silent exit ADR-0061 stage 6 exists to close).
//
// ceiling is the width at which the measurement stops caring: the caller (a size classification) only
// asks whether the material is BELOW the model's weld, so a direction that already reaches the weld
// need not be measured to the end. A truncated direction always reports at least ceiling, so it can
// never become the minimum while a genuinely thinner one exists, and the value a refusal names was
// always measured in full. Pass 0 for the exact minimum.
//
// Example:
//
//	if t, ok := solidThickness(tool, res.Weld()); ok && !res.Resolves(t) { /* refuse */ }
func solidThickness(b *topo.Body, ceiling float64) (float64, bool) {
	if b == nil || !b.IsSolid() {
		return 0, false
	}
	normals, axes, support := boundaryDirections(b)
	if len(support) == 0 {
		return 0, false
	}
	var thin spanFloor
	thin.offer(minExtentAlong(normals, support, ceiling))
	thin.offer(minExtentAlong(axisDirections(axes), support, thin.ceiling(ceiling)))
	thin.offer(minAxialWidth(axes, support, thin.ceiling(ceiling)))
	thin.offer(principalWidth(normals, axes, support, thin.ceiling(ceiling)))
	if !thin.found {
		return 0, true // a solid whose boundary names no direction at all has no extent to report
	}
	return thin.value, thin.found
}

// bodyVertexPoints is the half of the support set the vertices give — everything a planar face is
// bounded by.
func bodyVertexPoints(b *topo.Body) []math.Point3 {
	verts := b.Vertices()
	pts := make([]math.Point3, 0, len(verts))
	for _, v := range verts {
		pts = append(pts, v.Point())
	}
	return pts
}

// axisLine is one axis a body's surfaces of revolution turn about: a point on it and its canonical
// direction. A body carrying such a face is measured both ALONG the axis and ACROSS it.
type axisLine struct {
	origin math.Point3
	dir    math.UnitVector3
}

// boundaryDirections walks the body's faces ONCE and returns everything a width needs: the distinct
// directions its planar faces supply, the distinct axes its surfaces of revolution turn about, and
// the support set — the vertices, plus sampled points on each CURVED face, which the vertices do not
// bound (a cylinder has two, both on its seam).
//
// One pass because asking a face whether it is planar costs a normalisation, and a hundred-thousand
// facet import asks once per face.
func boundaryDirections(b *topo.Body) ([]math.UnitVector3, []axisLine, []math.Point3) {
	var normals []math.UnitVector3
	var axes []axisLine
	support := bodyVertexPoints(b)
	for _, f := range b.Faces() {
		if n, ok := geom.PlanarNormal(f.Geometry()); ok {
			normals = append(normals, n)
			continue
		}
		support = f.AppendSupportPoints(support)
		if o, d, ok := geom.RevolvedAxisLine(f.Geometry()); ok {
			axes = appendDistinctAxis(axes, axisLine{origin: o, dir: d})
		}
	}
	return distinctDirections(normals), axes, support
}

// distinctDirections collapses a body's planar normals to the distinct ones. It SORTS rather than
// comparing each new normal with every one already held: a plate's two faces and a
// hundred-thousand-facet import both go through here, and the pairwise form makes the dedupe alone
// quadratic — measured, a 2004-face prism spent 2.2 ms in it against 0.12 ms sorted.
func distinctDirections(dirs []math.UnitVector3) []math.UnitVector3 {
	slices.SortFunc(dirs, compareDirections)
	out := dirs[:0]
	for i, d := range dirs {
		if i == 0 || !geom.ParallelDirections(out[len(out)-1], d) {
			out = append(out, d)
		}
	}
	return out
}

// compareDirections is the total order the dedupe needs: canonical components, compared exactly, so
// the grouping — and the answer — is the same on every platform. Each key is tested in turn rather
// than through cmp.Or, which evaluates every one of its arguments before choosing between them.
func compareDirections(a, b math.UnitVector3) int {
	if c := cmp.Compare(a.X(), b.X()); c != 0 {
		return c
	}
	if c := cmp.Compare(a.Y(), b.Y()); c != 0 {
		return c
	}
	return cmp.Compare(a.Z(), b.Z())
}

// axisDirections is the direction of each axis line: a body is measured ALONG its axes as well as
// across them, and along is where a torus is thin (its tube diameter) and a disc is thin (its face).
func axisDirections(axes []axisLine) []math.UnitVector3 {
	dirs := make([]math.UnitVector3, 0, len(axes))
	for _, a := range axes {
		dirs = appendDistinctDirection(dirs, a.dir)
	}
	return dirs
}

// appendDistinctDirection adds n unless a parallel direction is already held.
func appendDistinctDirection(dirs []math.UnitVector3, n math.UnitVector3) []math.UnitVector3 {
	for _, d := range dirs {
		if geom.ParallelDirections(d, n) {
			return dirs
		}
	}
	return append(dirs, n)
}

// appendDistinctAxis adds a unless the same axis is already held. The test is STRUCTURAL — the same
// origin and a parallel direction — not metric: two faces of one bore carry the surface's own origin
// verbatim, so this collapses them, while two axes that merely look coaxial each keep their own
// width. Merging those would need a model-relative tolerance to decide "the same line", and a
// duplicate axis costs one more scan that the ceiling cuts short after a point or two.
func appendDistinctAxis(axes []axisLine, a axisLine) []axisLine {
	for _, x := range axes {
		if x.origin == a.origin && geom.ParallelDirections(x.dir, a.dir) {
			return axes
		}
	}
	return append(axes, a)
}

// minExtentAlong is the smallest width the body has along any of the given directions.
func minExtentAlong(dirs []math.UnitVector3, support []math.Point3, ceiling float64) (float64, bool) {
	var thin spanFloor
	for _, d := range dirs {
		thin.offer(extentAlong(d, support, thin.ceiling(ceiling)))
	}
	return thin.value, thin.found
}

// extentAlong is how far the body reaches along dir — the span of its support points projected onto
// it. The scan stops as soon as the span reaches ceiling, because a direction that wide cannot hold
// the minimum the caller is looking for.
func extentAlong(dir math.UnitVector3, support []math.Point3, ceiling float64) (float64, bool) {
	lo, hi := stdmath.Inf(1), stdmath.Inf(-1)
	for _, p := range support {
		d := float64(p.AsVector().Dot(dir.AsVector()))
		lo, hi = stdmath.Min(lo, d), stdmath.Max(hi, d)
		if hi-lo >= ceiling {
			break
		}
	}
	// A ZERO extent is a measurement, and the thinnest one there is: a body flat along one of its own
	// boundary directions has no material left in it. brep.SolidCylinderCone collapses a 2e-9-tall cone
	// into two coincident planar discs, and reporting that as "unmeasured" is how the silent exit
	// returns (#3524).
	return hi - lo, len(support) > 0
}

// minAxialWidth is the smallest width the body has ACROSS any of its axes: twice the furthest its
// support reaches from the axis line. That is the body's extent in every direction perpendicular to
// the axis when the body is a solid of revolution about it, and an upper bound on that extent
// otherwise — never below the body's true width, so it cannot refuse geometry that is not thin.
//
// It is what measures a rod: a cylinder's own caps supply only its axis, along which it is long.
func minAxialWidth(axes []axisLine, support []math.Point3, ceiling float64) (float64, bool) {
	var thin spanFloor
	for _, a := range axes {
		thin.offer(axialWidth(a, support, thin.ceiling(ceiling)))
	}
	return thin.value, thin.found
}

// axialWidth is twice the largest radial distance from one axis line, stopping once it reaches the
// ceiling.
func axialWidth(a axisLine, support []math.Point3, ceiling float64) (float64, bool) {
	widest := 0.0
	for _, p := range support {
		widest = stdmath.Max(widest, radialDistance(p, a))
		if 2*widest >= ceiling {
			break
		}
	}
	return 2 * widest, len(support) > 0
}

// radialDistance is how far p sits from an axis line — a rotation-invariant quantity, which is what
// makes a width taken across an axis independent of the frame.
func radialDistance(p math.Point3, a axisLine) float64 {
	return geom.DistanceToAxis(a.origin, a.dir, p)
}

// principalWidth is the last-resort width, for a body whose own boundary names fewer than three
// independent directions: the smallest extent along its support points' principal frame.
//
// It exists so that a solid is never left UNMEASURED, which is the same silent exit as a wrong
// measurement. It is reached by the shape that produced one: model/feature.combine planarizes a
// 1e-10-radius cylinder into 26 faces whose side normals have all collapsed to the cap normal, so
// the body names one direction and is 12 long in it, while its material is 2e-10 across.
//
// A body with an axis is NOT in that case: an axis names its own direction and the whole plane across
// it, which spans space. Bodies that reach here are near-degenerate by construction, and the frame's
// choice is only unique up to a shared eigenspace (geom.PrincipalDirections), so the width can move
// slightly with rotation where the exact directions cannot.
func principalWidth(normals []math.UnitVector3, axes []axisLine, support []math.Point3, ceiling float64) (float64, bool) {
	if len(axes) > 0 || directionsSpanSpace(normals) {
		return 0, false
	}
	frame, ok := geom.PrincipalDirections(support)
	if !ok {
		return 0, false
	}
	return minExtentAlong(frame[:], support, ceiling)
}

// directionsSpanSpace reports whether a set of directions spans all three dimensions, so that a width
// along each of them bounds the body in every direction. It is a RANK test and not a count: three
// normals that share a plane leave the body unmeasured across it.
func directionsSpanSpace(dirs []math.UnitVector3) bool {
	if len(dirs) < 3 {
		return false
	}
	first := dirs[0].AsVector()
	for i, second := range dirs {
		if geom.ParallelDirections(dirs[0], second) {
			continue
		}
		if spansThirdDimension(first.Cross(second.AsVector()), dirs[i+1:]) {
			return true
		}
	}
	return false
}

// spansThirdDimension reports whether any of the remaining directions leaves the plane the first two
// span — the plane's own normal is what they must not all be perpendicular to.
func spansThirdDimension(planeNormal math.Vector3, rest []math.UnitVector3) bool {
	for _, d := range rest {
		if stdmath.Abs(float64(d.AsVector().Dot(planeNormal))) > planarRankFloor*float64(planeNormal.Length()) {
			return true
		}
	}
	return false
}

// planarRankFloor is how far out of a plane a third direction must point to count as independent of
// it. It is a SINE on unit directions, so it carries no model scale, and it matches the parallelism
// the direction dedupe already uses (geom.ParallelDirections).
const planarRankFloor = 1e-9 // tol:angular — independence of a third boundary direction

// spanFloor accumulates the smallest width offered to it, and remembers whether anything was: an
// unmeasured body must be told apart from a zero-width one.
type spanFloor struct {
	value float64
	found bool
}

// offer keeps v when it is proven and thinner than everything offered so far.
func (f *spanFloor) offer(v float64, ok bool) {
	if ok && (!f.found || v < f.value) {
		f.value, f.found = v, true
	}
}

// ceiling is the width beyond which a further measurement cannot change the answer: the caller's own
// ceiling, or the thinnest width found so far when that is smaller. A ceiling of 0 or less means the
// caller wants the exact minimum and nothing is truncated.
func (f *spanFloor) ceiling(callers float64) float64 {
	if callers <= 0 {
		callers = stdmath.Inf(1)
	}
	if f.found && f.value < callers {
		return f.value
	}
	return callers
}

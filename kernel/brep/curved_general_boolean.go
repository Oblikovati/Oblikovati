// SPDX-License-Identifier: GPL-2.0-only

package brep

import (
	stdmath "math"

	"oblikovati.org/kernel/diag"
	"oblikovati.org/kernel/geom"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
)

// General curved∩curved boolean (EPIC Oblikovati/Oblikovati#1403). The bespoke per-pair handlers build the
// result by hand from the SSI imprint loops (a band + caps stitched case by case). This is the GENERAL
// pipeline the half-space/ruled path already proves in miniature, run on a real pair: trim EACH operand's
// curved side by their shared surface-surface-intersection imprint (the same imprint-general
// trimByImprint, #1405), classify kept cells by 3D SOLID MEMBERSHIP (keep(op, isB, insideSolid(other,·)) —
// the same predicate the planar boolean uses) instead of a plane half-space, then weld both kept sides with
// the shared general curvedStitch. No per-pair loop→body constructor.
//
// First migrated pair: cone ∩ cone (the smallest bespoke handler; both operands ruled, so they reuse the
// existing ruledUV plumbing — only the material predicate is new). The bespoke ConeConeIntersect remains as
// a fallback for the configurations this path declines (a full cone reaching its apex, a non-crossing).

// ruledSolidMaterial wraps a ruled side's 3D solid-membership test as a uvSide materialOf builder: a closure
// (not a bound method) so it reads the receiver AFTER trimByImprint shifts the seam — point3 then maps the
// seam-relative (u,v) to the right surface point (#1403, mirroring ruledMaterial).
func ruledSolidMaterial(c *ruledUV) func() materialPredicate {
	return func() materialPredicate {
		return func(uv math.Point2) bool { return c.keptBySolid(uv.X, uv.Y) }
	}
}

// keptBySolid reports whether the band point at (u,v) survives the operation: keep(op, isB, inside) where
// inside is 3D membership of the surface point in the OTHER solid — the general curved∩curved classification
// (#1403). For A∩B this keeps each side's part that lies inside the other solid.
func (c ruledUV) keptBySolid(u, v float64) bool {
	return keep(c.solidOp, c.solidIsB, c.insideOther(c.point3(u, v)))
}

// anyKeptVSolid reports whether SOME axial-distance v in the band is kept at azimuth u — the general
// (solid-membership) analogue of keptV's non-empty test, used by wrapsAllU to pick the band orientation
// convention. It samples v because solid membership has no closed-form interval like the linear plane case.
func (c ruledUV) anyKeptVSolid(u float64) bool {
	const n = 96 // dense enough to catch a thin lens where the imprint barely bites the band
	for j := 0; j <= n; j++ {
		v := c.band.vMin + (c.band.vMax-c.band.vMin)*float64(j)/float64(n)
		if c.keptBySolid(u, v) {
			return true
		}
	}
	return false
}

// newRuledUVFrame builds the (u,v) frame of a ruled side WITHOUT a cut plane — the general curved∩curved
// path needs only the geometry (base/axis/ref/radius), not the plane signed-distance coefficients
// (p,q,s,t,uN stay 0; the half-space predicate is replaced by keptBySolid). base is the surface point at
// v=0 (cone apex / cylinder bottom centre), rad(v)=radSlope·v+radConst (#1403).
func newRuledUVFrame(base math.Point3, axis, ref math.Vector3, radSlope, radConst float64, band coneSideBand_) ruledUV {
	return ruledUV{
		base: base, axis: axis, ref: ref, binor: axis.Cross(ref),
		radSlope: radSlope, radConst: radConst, band: band,
	}
}

// newConeUVSolid builds a frustum side's (u,v) model for a general cut whose kept side is decided by the
// other solid's membership oracle `inside` under op (isB marks this operand as the boolean's B). It is the
// plane-free, solid-membership counterpart of newConeUV (#1403).
func newConeUVSolid(cone geom.Cone, band coneSideBand_, op Op, isB bool, inside func(math.Point3) bool) ruledUV {
	c := newRuledUVFrame(cone.Apex, cone.AxisDir.AsVector(), cone.Ref.AsVector(), stdmath.Tan(cone.HalfAngle), 0, band)
	c.solidMode, c.solidOp, c.solidIsB, c.insideOther = true, op, isB, inside
	return c
}

// coneSideSolidSplit trims a frustum side by an SSI imprint, keeping the cells whose surface point lies in
// the other solid (per op). The same general arrangement trim as the half-space cone split (coneSideUVSplit)
// — only the imprint is an SSI loop set and the predicate is solid membership (#1403).
func coneSideSolidSplit(f curvedFace, cone geom.Cone, band coneSideBand_, imprint []geom.Curve3, op Op, isB bool, inside func(math.Point3) bool) ([]curvedFace, []loopEdge, error) {
	c := newConeUVSolid(cone, band, op, isB, inside)
	return trimByImprint(&c, f, cone, imprint, ruledSolidMaterial(&c))
}

// curvedSolidMembership returns a point-in-solid oracle for a primitive curved solid (cone/cylinder
// frustum), or ok=false for a shape it does not handle analytically. A curved solid's faces are curved, so
// the planar ray-cast insideSolid does not apply — each primitive has its own closed-form inside test. This
// is the general curved∩curved classifier's membership stage; it grows one case per primitive (#1403).
func curvedSolidMembership(b *topo.Body) (func(math.Point3) bool, bool) {
	faces := facesOfAny(b)
	if cone, vMin, vMax, ok := coneSolidParams(faces); ok {
		return func(p math.Point3) bool { return pointInsideConeSolid(cone, vMin, vMax, p) }, true
	}
	if cyl, base, height, ok := cylinderSolidParams(faces); ok {
		return func(p math.Point3) bool { return pointInsideCylinderSolid(cyl, base, height, p) }, true
	}
	return nil, false
}

// pointInsideConeSolid reports whether p is inside a frustum solid: within the apex-distance band
// [vMin, vMax] (between the caps) AND inside the cone radius v·tan(HalfAngle) at that height. A small
// model-relative margin keeps the imprint loop itself (on the surface) from flickering across the boundary.
func pointInsideConeSolid(cone geom.Cone, vMin, vMax float64, p math.Point3) bool {
	axis := cone.AxisDir.AsVector()
	v := float64(cone.Apex.VectorTo(p).Dot(axis))
	rim := vMax * stdmath.Tan(cone.HalfAngle)
	margin := geom.ResolutionForSize(rim + vMax).Plane() // model-relative inside-solid margin (#1399)
	if v < vMin+margin || v > vMax-margin {
		return false
	}
	axisPt := cone.Apex.TranslateBy(axis.Scale(math.Scalar(v)))
	rho := float64(axisPt.VectorTo(p).Length())
	return rho < v*stdmath.Tan(cone.HalfAngle)-margin
}

// coneSideFace finds a frustum side face of b: its curvedFace, the geom.Cone, and the rim band. ok=false
// unless b has a cone side bounded by two full-circle rims (a full cone reaching its apex is not the band
// case and defers to the bespoke handler).
func coneSideFace(b *topo.Body) (curvedFace, geom.Cone, coneSideBand_, bool) {
	for _, f := range facesOfAny(b) {
		cone, isCone := f.surface.(geom.Cone)
		if !isCone {
			continue
		}
		if band, ok := coneSideBand(f, cone); ok {
			return f, cone, band, true
		}
	}
	return curvedFace{}, geom.Cone{}, coneSideBand_{}, false
}

// cylinderSideFace finds a cylinder side face of b: its curvedFace, the geom.Cylinder, and the rim band
// (the same two-circle band the half-space path uses, with v the axial distance from the bottom rim).
func cylinderSideFace(b *topo.Body) (curvedFace, geom.Cylinder, coneSideBand_, bool) {
	for _, f := range facesOfAny(b) {
		if _, isCyl := f.surface.(geom.Cylinder); !isCyl {
			continue
		}
		if cyl, band, ok := fullCylinderSideBand(f); ok {
			return f, cyl, band, true
		}
	}
	return curvedFace{}, geom.Cylinder{}, coneSideBand_{}, false
}

// newCylinderUVSolid builds a cylinder side's (u,v) model for a general cut decided by `inside` under op.
// A cylinder is the degenerate cone — constant radius R (radSlope 0, radConst R), v the axial distance from
// the bottom rim centre (band.bottom) — so it reuses the whole ruled solid-membership trim (#1403).
func newCylinderUVSolid(cyl geom.Cylinder, band coneSideBand_, op Op, isB bool, inside func(math.Point3) bool) ruledUV {
	c := newRuledUVFrame(band.bottom, cyl.AxisDir.AsVector(), cyl.Ref.AsVector(), 0, cyl.Radius, band)
	c.solidMode, c.solidOp, c.solidIsB, c.insideOther = true, op, isB, inside
	return c
}

// cylinderSideSolidSplit trims a cylinder side by an SSI imprint, keeping the cells inside the other solid
// (per op) — the cylinder counterpart of coneSideSolidSplit (#1403).
func cylinderSideSolidSplit(f curvedFace, cyl geom.Cylinder, band coneSideBand_, imprint []geom.Curve3, op Op, isB bool, inside func(math.Point3) bool) ([]curvedFace, []loopEdge, error) {
	c := newCylinderUVSolid(cyl, band, op, isB, inside)
	return trimByImprint(&c, f, cyl, imprint, ruledSolidMaterial(&c))
}

// polylineCurves adapts SSI imprint loops (closed polylines) to the []geom.Curve3 trimByImprint consumes.
// Each curve is a *geom.Polyline (a POINTER): the arrangement's run-merge compares edge curves by `==` to
// fuse consecutive same-curve edges, and a geom.Polyline value is uncomparable (it holds a slice), so it
// must be carried by identity. The same pointers are handed to BOTH operand sides, so each loop's edges
// merge and the two sides emit the SAME curve for the shared imprint (#1403).
func polylineCurves(loops []geom.Polyline) []geom.Curve3 {
	out := make([]geom.Curve3, len(loops))
	for i := range loops {
		out[i] = &loops[i]
	}
	return out
}

// planarCapFaces returns a body's planar (cap) faces as curvedFaces, kept WHOLE — for a clean side-breach
// cut the breach never reaches a cap, so the target's caps survive untouched in the result (#1403).
func planarCapFaces(b *topo.Body) []curvedFace {
	var caps []curvedFace
	for _, f := range facesOfAny(b) {
		if _, isPlane := f.surface.(geom.Plane); isPlane {
			caps = append(caps, f)
		}
	}
	return caps
}

// reverseCurvedFaces flips each tool wall into the CAVITY a Difference carves: the sense flag (so the normal
// points into the void) AND every loop's winding (so the boundary is walked the OTHER way). The tool keeps the
// part INSIDE the target — a tunnel band whose imprint loop is walked the SAME way as the target's own hole —
// so reversing the loop opposes them, the manifold-orientation a watertight cut needs (curvedStitch orients
// each shared edge by its loop traversal, not the face sense). The tunnel is a ruled LOFT band
// (twoClosedRimBandMesh), which lofts rim-to-rim regardless of winding, so the reversal does not change its
// meshed region — only its orientation (#1403/#1476).
func reverseCurvedFaces(faces []curvedFace) []curvedFace {
	out := make([]curvedFace, len(faces))
	for i, f := range faces {
		f.reversed = !f.reversed
		loops := make([]curvedLoop, len(f.loops))
		for j, lp := range f.loops {
			loops[j] = reverseCurvedLoop(lp)
		}
		f.loops = loops
		out[i] = f
	}
	return out
}

// reverseCurvedLoop reverses a loop's traversal: each edge's direction (t0↔t1) and the edge order both flip,
// so the loop walks the opposite way around the same boundary (#1476).
func reverseCurvedLoop(lp curvedLoop) curvedLoop {
	n := len(lp.edges)
	rev := make([]loopEdge, n)
	for i, e := range lp.edges {
		e.t0, e.t1 = e.t1, e.t0
		rev[n-1-i] = e
	}
	return curvedLoop{edges: rev}
}

// loopsClearOfCaps reports whether every imprint point sits STRICTLY between the side band's two cap levels
// (vMin, vMax) — a clean side-breach where the cut leaves both caps whole. A breach reaching a cap needs the
// planar cap itself trimmed, which this first cut migration defers to the bespoke handler (#1403).
func loopsClearOfCaps(side ruledUV, loops []geom.Polyline) bool {
	margin := geom.ResolutionForSize(side.band.vMax - side.band.vMin).Plane()
	for _, lp := range loops {
		for _, p := range lp.Vertices {
			v := float64(side.base.VectorTo(p).Dot(side.axis))
			if v < side.band.vMin+margin || v > side.band.vMax-margin {
				return false
			}
		}
	}
	return true
}

// crossingSide bundles everything a crossing-cylinder general boolean needs about ONE operand: its cylinder
// side face, the rim band, the geom.Cylinder, and a 3D solid-membership oracle for it. Both the cut and the
// join trim one side against the OTHER side's `inside` oracle (#1403).
type crossingSide struct {
	face   curvedFace
	cyl    geom.Cylinder
	band   coneSideBand_
	inside func(math.Point3) bool
}

// crossingCylinderSides resolves the shared inputs every crossing-cylinder general boolean (cut, join)
// starts from: the SSI imprint (exactly two closed loops for a clean rod-through-fat crossing) and, for each
// operand, its cylinder side face + rim band + solid-membership oracle. ok=false when the pair is not two
// bare cylinders meeting in two closed loops, so each caller keeps its bespoke fallback (#1403).
func crossingCylinderSides(a, b *topo.Body, rec *diag.Recorder) ([]geom.Polyline, crossingSide, crossingSide, bool) {
	loops, ok := crossingCylinderImprint(a, b, rec)
	if !ok || len(loops) != 2 {
		return nil, crossingSide{}, crossingSide{}, false
	}
	fA, cylA, bandA, okA := cylinderSideFace(a)
	fB, cylB, bandB, okB := cylinderSideFace(b)
	insideA, okMA := curvedSolidMembership(a)
	insideB, okMB := curvedSolidMembership(b)
	if !okA || !okB || !okMA || !okMB {
		return nil, crossingSide{}, crossingSide{}, false
	}
	return loops, crossingSide{fA, cylA, bandA, insideA}, crossingSide{fB, cylB, bandB, insideB}, true
}

// CrossingCylinderCutGeneral builds target − tool through the GENERAL curved∩curved pipeline (#1403): the
// target side kept OUTSIDE the tool (the breached wall), the target's caps whole, and the tool side kept
// INSIDE the target and reversed (the tunnel wall).
//
// NOT WIRED — known broken (Oblikovati#1476): the OUTSIDE-keep wall meshes the WRONG region, so the result is
// orientation-inconsistent or wrong-volume and validBooleanSolid rejects it. kernel/ops keeps crossing-cylinder
// subtract on the bespoke CrossingCylinderCut until #1476 fixes the arrangement winding. Retained as the
// scaffolding that fix builds on; the brep test only checks its edge-count/face structure, not correctness.
func CrossingCylinderCutGeneral(target, tool *topo.Body, rec *diag.Recorder) (*topo.Body, bool) {
	loops, tgt, tl, ok := crossingCylinderSides(target, tool, rec)
	if !ok {
		return nil, false
	}
	// Clean side-breach only: the tool must breach the target's SIDE between its caps (so the caps stay
	// whole) and not reach the tool's own caps inside the target (a full crossing, both loops on the side).
	if !loopsClearOfCaps(newCylinderUVSolid(tgt.cyl, tgt.band, Difference, false, tl.inside), loops) {
		return nil, false
	}
	imprint := polylineCurves(loops)
	keptA, okKA := keptOrNone(cylinderSideSolidSplit(tgt.face, tgt.cyl, tgt.band, imprint, Difference, false, tl.inside))
	keptB, okKB := keptOrNone(cylinderSideSolidSplit(tl.face, tl.cyl, tl.band, imprint, Difference, true, tgt.inside))
	if !okKA || !okKB {
		return nil, false
	}
	return curvedStitch(cutFaces(keptA, target, keptB)), true
}

// cutFaces assembles a Difference result's boundary: the target's breached wall (kept outside the tool), the
// target's whole caps (clean side-breach), and the tool's wall inside the target reversed into the cavity
// (the tunnel/cut wall) — #1403.
func cutFaces(targetWall []curvedFace, target *topo.Body, toolWall []curvedFace) []curvedFace {
	faces := make([]curvedFace, 0, len(targetWall)+len(toolWall)+2)
	faces = append(faces, targetWall...)
	faces = append(faces, planarCapFaces(target)...)
	faces = append(faces, reverseCurvedFaces(toolWall)...)
	return faces
}

// CrossingCylinderJoinGeneral builds target ∪ tool through the GENERAL curved∩curved pipeline (#1403): each
// side keeps the part OUTSIDE the other (the Union keep-table — the fat's holed wall plus the two rod stubs),
// and BOTH bodies keep their caps whole. The cut's sibling with NO face reversal.
//
// NOT WIRED — known broken (Oblikovati#1476): with the imprint weld fixed this now passes validBooleanSolid
// but meshes the WRONG region (∪ volume 194 vs 383), so adopting it would be worse than the bespoke result.
// kernel/ops keeps crossing-cylinder JOIN on the bespoke CrossingCylinderJoin until #1476 fixes the OUTSIDE-keep
// arrangement winding. Retained as scaffolding; the brep test only checks its edge-count/face structure.
func CrossingCylinderJoinGeneral(a, b *topo.Body, rec *diag.Recorder) (*topo.Body, bool) {
	loops, sa, sb, ok := crossingCylinderSides(a, b, rec)
	if !ok {
		return nil, false
	}
	// Both bodies keep their caps whole, so the breach must lie strictly between EACH body's caps — a rod
	// passing fully through a fat cylinder and sticking out each side (a breach that reaches a cap would need
	// that planar cap trimmed, which this migration defers to the bespoke handler).
	if !loopsClearOfCaps(newCylinderUVSolid(sa.cyl, sa.band, Union, false, sb.inside), loops) ||
		!loopsClearOfCaps(newCylinderUVSolid(sb.cyl, sb.band, Union, true, sa.inside), loops) {
		return nil, false
	}
	imprint := polylineCurves(loops)
	keptA, okKA := keptOrNone(cylinderSideSolidSplit(sa.face, sa.cyl, sa.band, imprint, Union, false, sb.inside))
	keptB, okKB := keptOrNone(cylinderSideSolidSplit(sb.face, sb.cyl, sb.band, imprint, Union, true, sa.inside))
	if !okKA || !okKB {
		return nil, false
	}
	return curvedStitch(joinFaces(keptA, a, keptB, b)), true
}

// joinFaces assembles a Union result's boundary: each operand's wall kept outside the other (the fat's holed
// wall + the rod's two protruding stubs) plus BOTH operands' whole caps. Unlike the cut neither wall is
// reversed — a union keeps every kept wall facing outward — and both bodies contribute their caps (#1403).
func joinFaces(wallA []curvedFace, a *topo.Body, wallB []curvedFace, b *topo.Body) []curvedFace {
	faces := make([]curvedFace, 0, len(wallA)+len(wallB)+4)
	faces = append(faces, wallA...)
	faces = append(faces, planarCapFaces(a)...)
	faces = append(faces, wallB...)
	faces = append(faces, planarCapFaces(b)...)
	return faces
}

// CrossingCylinderIntersectGeneral is the exported entry kernel/ops routes crossing-cylinder intersect
// through: the GENERAL curved∩curved pipeline (#1403), no bespoke loop→body constructor. ok=false outside
// the wired cylinder-through-cylinder crossing so the caller keeps its fallback.
func CrossingCylinderIntersectGeneral(a, b *topo.Body, rec *diag.Recorder) (*topo.Body, bool) {
	return crossingCylinderIntersectGeneral(a, b, rec)
}

// crossingCylinderIntersectGeneral builds cylinder ∩ cylinder (two crossing cylinders) through the GENERAL
// pipeline (#1403): the SSI imprint, then trimByImprint on each cylinder side keeping the part inside the
// other, then curvedStitch. The simplest pair — both sides are cylinders, so the recipe is fully symmetric
// (no rod/fat split): the rod-wall band inside the fat plus the two fat-wall lens caps fall out of the same
// two-sided trim. ok=false when the pair is not a cylinder-through-cylinder crossing.
func crossingCylinderIntersectGeneral(a, b *topo.Body, rec *diag.Recorder) (*topo.Body, bool) {
	loops, ok := crossingCylinderImprint(a, b, rec)
	if !ok || len(loops) == 0 {
		return nil, false
	}
	fA, cylA, bandA, okA := cylinderSideFace(a)
	fB, cylB, bandB, okB := cylinderSideFace(b)
	insideA, okMA := curvedSolidMembership(a)
	insideB, okMB := curvedSolidMembership(b)
	if !okA || !okB || !okMA || !okMB {
		return nil, false
	}
	imprint := polylineCurves(loops)
	keptA, okKA := keptOrNone(cylinderSideSolidSplit(fA, cylA, bandA, imprint, Intersection, false, insideB))
	keptB, okKB := keptOrNone(cylinderSideSolidSplit(fB, cylB, bandB, imprint, Intersection, true, insideA))
	if !okKA || !okKB {
		return nil, false
	}
	return curvedStitch(append(keptA, keptB...)), true
}

// ConeCylinderIntersectGeneral is the exported entry kernel/ops routes cone∩cylinder intersect through: the
// GENERAL curved∩curved pipeline (#1403), no bespoke loop→body constructor. ok=false outside the wired
// frustum-through-cylinder case so the caller keeps its fallback.
func ConeCylinderIntersectGeneral(a, b *topo.Body, rec *diag.Recorder) (*topo.Body, bool) {
	return coneCylinderIntersectGeneral(a, b, rec)
}

// coneCylinderIntersectGeneral builds cone ∩ cylinder through the GENERAL pipeline (#1403): the SSI imprint,
// then trimByImprint on the cone side and the cylinder side each keeping the part inside the other solid,
// then curvedStitch. Same two-sided recipe as cone∩cone, with the cylinder side standing in for one cone —
// the cone band inside the cylinder plus the two cylinder-wall lens caps. ok=false when the pair is not a
// cone-through-cylinder crossing, so kernel/ops keeps the bespoke ConeCylinderIntersect fallback.
func coneCylinderIntersectGeneral(a, b *topo.Body, rec *diag.Recorder) (*topo.Body, bool) {
	loops, ok := coneCylinderImprint(a, b, rec)
	if !ok || len(loops) == 0 {
		return nil, false
	}
	coneBody, cylBody := a, b // identify which operand is the cone, which the cylinder
	if _, _, _, okCone := coneSideFace(a); !okCone {
		coneBody, cylBody = b, a
	}
	fCone, cone, coneBand, okCone := coneSideFace(coneBody)
	fCyl, cyl, cylBand, okCyl := cylinderSideFace(cylBody)
	insideCone, okMC := curvedSolidMembership(coneBody)
	insideCyl, okMY := curvedSolidMembership(cylBody)
	if !okCone || !okCyl || !okMC || !okMY {
		return nil, false
	}
	imprint := polylineCurves(loops)
	keptCone, okA := keptOrNone(coneSideSolidSplit(fCone, cone, coneBand, imprint, Intersection, coneBody == b, insideCyl))
	keptCyl, okB := keptOrNone(cylinderSideSolidSplit(fCyl, cyl, cylBand, imprint, Intersection, cylBody == b, insideCone))
	if !okA || !okB {
		return nil, false
	}
	return curvedStitch(append(keptCone, keptCyl...)), true
}

// keptOrNone adapts a side split's (faces, lid, err) to (faces, ok): ok is true only when the split
// succeeded and kept some geometry, so the two-sided drivers read as one short condition (#1403).
func keptOrNone(faces []curvedFace, _ []loopEdge, err error) ([]curvedFace, bool) {
	return faces, err == nil && len(faces) > 0
}

// ConeConeIntersectGeneral is the exported entry kernel/ops routes cone∩cone intersect through: the GENERAL
// curved∩curved pipeline (#1403), with no bespoke loop→body constructor. It declines (ok=false) for
// configurations outside the wired frustum-crossing case so the caller keeps its fallback.
func ConeConeIntersectGeneral(a, b *topo.Body, rec *diag.Recorder) (*topo.Body, bool) {
	return coneConeIntersectGeneral(a, b, rec)
}

// coneConeIntersectGeneral builds cone ∩ cone through the GENERAL pipeline (#1403): the SSI imprint, then
// trimByImprint on EACH cone side keeping the part inside the other cone, then curvedStitch. The shared
// imprint loops are emitted by both sides referencing the same polyline, so the welder fuses them into a
// watertight body (the rod-wall band between the two loops + the two fat-wall lens caps). ok=false when the
// pair is outside the wired frustum-crossing case, so kernel/ops keeps the bespoke ConeConeIntersect path.
func coneConeIntersectGeneral(a, b *topo.Body, rec *diag.Recorder) (*topo.Body, bool) {
	loops, ok := coneConeImprint(a, b, rec)
	if !ok || len(loops) == 0 {
		return nil, false
	}
	fa, coneA, bandA, okA := coneSideFace(a)
	fb, coneB, bandB, okB := coneSideFace(b)
	insideA, okMA := curvedSolidMembership(a)
	insideB, okMB := curvedSolidMembership(b)
	if !okA || !okB || !okMA || !okMB {
		return nil, false
	}
	imprint := polylineCurves(loops)
	keptA, _, errA := coneSideSolidSplit(fa, coneA, bandA, imprint, Intersection, false, insideB)
	keptB, _, errB := coneSideSolidSplit(fb, coneB, bandB, imprint, Intersection, true, insideA)
	if errA != nil || errB != nil || len(keptA) == 0 || len(keptB) == 0 {
		return nil, false
	}
	return curvedStitch(append(keptA, keptB...)), true
}

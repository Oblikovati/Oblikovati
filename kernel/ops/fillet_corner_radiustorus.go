// SPDX-License-Identifier: GPL-2.0-only

package ops

import (
	stdmath "math"

	"oblikovati.org/kernel/geom"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
)

// Mixed-radius trihedral TORUS corner (OCCT tests/blend simple/A4 r10/r5/r5, complex/E3 r14/r7/r7 +
// r19/r7/r7): three convex arms at an orthogonal planar corner where ONE arm carries the big radius
// rB on the edge between the two walls, and the two rS arms (equal, rS < rB) share the third face
// (the "top"). The corner closes with a TORUS patch — axis along the rB arm's fillet spine, centre
// on that spine at distance rS inside the top plane, major R = rB − rS, minor r = rS — tangent to
// the top plane along a circle of radius R, to the rB band along its outer equator, and to each rS
// band along a tube circle. Derived from the solid's closed form and verified per-face against
// DRAWEXE on A4: patch = rS·(π/2)·(R·(π/2) + rS) = 100.955 for r10/r5/r5, band trims at
// 10−rS / 100−rB exactly (this file's gate test pins those faces).
//
// Assembly rides the unified setback accumulator (fillet_corner_setback_unified.go) exactly like
// the equal-radius mixed-SENSE torus (fillet_corner_torus.go): three band retract railWrites, the
// transient blend dropped from sphere emission, the torus patch appended, and the top face re-trimmed
// through a synthetic host end-corner carrying the R-arc.

// radiusTorusOrthoTol is the largest RELATIVE certificate defect (contact off its plane, or a tube
// foot off its arm's spine, divided by rS) the solve accepts. Dimensionless and scale-invariant —
// the same regime as mixedTorusRadiusTol; a non-orthogonal corner shifts the contacts off their
// planes by O(angle)·r and is declined to the honest baseline message rather than mis-modelled.
const radiusTorusOrthoTol = 1e-6

// radiusTorusCornerGeom is the fully-solved geometry of one mixed-radius torus corner, stashed on
// the transient cornerBlend so the unified setback pass can retract the three bands and emit the
// patch without re-deriving anything.
type radiusTorusCornerGeom struct {
	rB, rS       float64
	center       math.Point3 // torus centre: on the rB spine, rS inside the top plane
	centerTop    math.Point3 // centre of the top-contact circle (centre dropped onto the top plane)
	axis         math.Vector3
	top          *topo.Face
	wallA, wallB *topo.Face
	eqA, eqB     math.Point3 // outer-equator arc ends — the wall contacts shared with the rS bands
	footA, footB math.Point3 // tube centres: the rS spines' feet nearest the axis
	pTopA, pTopB math.Point3 // tube-circle tops on the top plane
	bigEdge      uint64      // pick edge ids, to match each band retract to its fils slot
	smallEdgeA   uint64
	smallEdgeB   uint64
}

// solveRadiusTorusCorner matches and solves the mixed-radius torus corner at v; ok=false leaves the
// caller's honest decline untouched (unmatched radii pattern, non-orthogonal or non-planar faces, a
// non-convex arm, or a failed certificate). On success the returned transient blend routes the
// corner into the setback pass via its radiusTorus marker. Example:
//
//	cb, ok := solveRadiusTorusCorner(v, vid, ps)
func solveRadiusTorusCorner(v *topo.Vertex, vid uint64, ps []filletPick) (*cornerBlend, bool) {
	big, smA, smB, ok := matchRadiusTorusPattern(vid, ps)
	if !ok {
		return nil, false
	}
	rt, ok := radiusTorusGeomOf(v, vid, big, smA, smB)
	if !ok {
		return nil, false
	}
	return radiusTorusTransientBlend(v, rt)
}

// matchRadiusTorusPattern identifies the [rB, rS, rS] corner: exactly one strictly-largest radius
// whose edge joins the two walls, the two equal smaller radii sharing the remaining (top) face, and
// every arm convex. Returned in (big, smallOnWallA, smallOnWallB) order.
func matchRadiusTorusPattern(vid uint64, ps []filletPick) (big, smA, smB filletPick, ok bool) {
	if len(ps) != 3 || !allConvexPicks(ps) {
		return filletPick{}, filletPick{}, filletPick{}, false
	}
	bi, ok := soleBigRadiusIndex(vid, ps)
	if !ok {
		return filletPick{}, filletPick{}, filletPick{}, false
	}
	big = ps[bi]
	s0, s1 := ps[(bi+1)%3], ps[(bi+2)%3]
	// The two rS edges must each share exactly one wall with the big edge; order them A-wall first.
	walls := big.edge.Faces()
	if len(walls) != 2 {
		return filletPick{}, filletPick{}, filletPick{}, false
	}
	if sharedFace(big.edge, s0.edge) == walls[1] {
		s0, s1 = s1, s0
	}
	if sharedFace(big.edge, s0.edge) != walls[0] || sharedFace(big.edge, s1.edge) != walls[1] {
		return filletPick{}, filletPick{}, filletPick{}, false
	}
	if sharedFace(s0.edge, s1.edge) == nil {
		return filletPick{}, filletPick{}, filletPick{}, false
	}
	return big, s0, s1, true
}

// allConvexPicks reports whether every pick is a constant-radius convex arm (the torus derivation
// assumes the material-side rolling ball on all three).
func allConvexPicks(ps []filletPick) bool {
	for _, p := range ps {
		if p.varying() || ClassifyEdgeConvexity(p.edge) != EdgeConvex {
			return false
		}
	}
	return true
}

// soleBigRadiusIndex returns the index of the strictly-largest corner radius when the other two are
// exactly equal and strictly smaller — the [rB, rS, rS] pattern; ok=false otherwise.
func soleBigRadiusIndex(vid uint64, ps []filletPick) (int, bool) {
	r := [3]float64{radiusAtVertex(ps[0], vid), radiusAtVertex(ps[1], vid), radiusAtVertex(ps[2], vid)}
	for i := 0; i < 3; i++ {
		rB, s1, s2 := r[i], r[(i+1)%3], r[(i+2)%3]
		if s1 == s2 && rB > s1 {
			return i, true
		}
	}
	return 0, false
}

// radiusTorusGeomOf solves the torus frame and every contact point from the closed form, then
// certifies each contact ON its plane and each tube foot ON its arm's spine (relative to rS). Any
// defect beyond radiusTorusOrthoTol — a non-orthogonal or skewed corner — declines.
func radiusTorusGeomOf(v *topo.Vertex, vid uint64, big, smA, smB filletPick) (*radiusTorusCornerGeom, bool) {
	rB, rS := radiusAtVertex(big, vid), radiusAtVertex(smA, vid)
	wallA, wallB := sharedFace(big.edge, smA.edge), sharedFace(big.edge, smB.edge)
	top := sharedFace(smA.edge, smB.edge)
	nA, okA := planeNormal(wallA)
	nB, okB := planeNormal(wallB)
	nC, okC := planeNormal(top)
	if !okA || !okB || !okC {
		return nil, false
	}
	center, axis, ok := radiusTorusFrame(v.Point(), nA, nB, nC, top, rB, rS)
	if !ok {
		return nil, false
	}
	rt := &radiusTorusCornerGeom{
		rB: rB, rS: rS, center: center, axis: axis,
		centerTop: center.TranslateBy(nC.Scale(rS)),
		top:       top, wallA: wallA, wallB: wallB,
		eqA: center.TranslateBy(nA.Scale(rB)), eqB: center.TranslateBy(nB.Scale(rB)),
		footA: center.TranslateBy(nA.Scale(rB - rS)), footB: center.TranslateBy(nB.Scale(rB - rS)),
		bigEdge: big.edge.ID(), smallEdgeA: smA.edge.ID(), smallEdgeB: smB.edge.ID(),
	}
	rt.pTopA, rt.pTopB = rt.footA.TranslateBy(nC.Scale(rS)), rt.footB.TranslateBy(nC.Scale(rS))
	return rt, radiusTorusCertified(rt, v, smA, smB, nA, nB, nC)
}

// radiusTorusFrame places the torus centre and axis: the rB spine is the line through
// v + offDir(nA,nB)·rB along ±(nA×nB); the centre is the spine point at distance rS inside the top
// plane, the axis oriented from the centre toward that plane. ok=false when the spine is parallel
// to the top plane (degenerate corner).
func radiusTorusFrame(v math.Point3, nA, nB, nC math.Vector3, top *topo.Face, rB, rS float64) (math.Point3, math.Vector3, bool) {
	spinePoint := v.TranslateBy(nA.Add(nB).Scale(-rB / (1 + nA.Dot(nB))))
	axis := unit(nA.Cross(nB))
	ca := nC.Dot(axis)
	if stdmath.Abs(ca) < 1e-9 {
		return math.Point3{}, math.Vector3{}, false
	}
	if ca < 0 {
		axis = axis.Scale(-1)
		ca = -ca
	}
	pl := top.Geometry().(geom.Plane)
	t := (nC.Dot(pl.Origin.AsVector()) - rS - nC.Dot(spinePoint.AsVector())) / ca
	return spinePoint.TranslateBy(axis.Scale(t)), axis, true
}

// radiusTorusCertified checks the closed form's tangency contract on the actual faces: each wall
// contact ON its wall, each top contact ON the top plane, and each tube foot ON its rS arm's offset
// spine — all within radiusTorusOrthoTol·rS. A skewed (non-orthogonal) corner fails here and keeps
// the honest decline.
func radiusTorusCertified(rt *radiusTorusCornerGeom, v *topo.Vertex, smA, smB filletPick, nA, nB, nC math.Vector3) bool {
	tol := radiusTorusOrthoTol * rt.rS
	planeDefect := func(p math.Point3, f *topo.Face, n math.Vector3) float64 {
		pl := f.Geometry().(geom.Plane)
		return stdmath.Abs(n.Dot(p.AsVector()) - n.Dot(pl.Origin.AsVector()))
	}
	if planeDefect(rt.eqA, rt.wallA, nA) > tol || planeDefect(rt.eqB, rt.wallB, nB) > tol ||
		planeDefect(rt.pTopA, rt.top, nC) > tol || planeDefect(rt.pTopB, rt.top, nC) > tol {
		return false
	}
	return spineDefect(rt.footA, v.Point(), smA, nA, nC) <= tol && spineDefect(rt.footB, v.Point(), smB, nB, nC) <= tol
}

// spineDefect is the perpendicular distance from a tube foot to its rS arm's offset spine — the
// line through v + offDir(nWall,nTop)·rS along the arm's edge direction.
func spineDefect(foot, v math.Point3, sm filletPick, nWall, nTop math.Vector3) float64 {
	q := v.TranslateBy(nWall.Add(nTop).Scale(-sm.r0 / (1 + nWall.Dot(nTop))))
	d := unit(sm.edge.StartVertex().Point().VectorTo(sm.edge.EndVertex().Point()))
	w := q.VectorTo(foot)
	return w.Sub(d.Scale(w.Dot(d))).Length()
}

// radiusTorusTransientBlend wraps the solved corner as a cornerBlend the ordinary band build can
// consume: centre = torus centre (adopted by the rB arm whose spine it lies on, kept frame-derived
// by the rS arms), and a tangent map whose entries are the true shared wall contacts (the top entry
// is transient — both rS arms' top contacts differ and every band end is rewritten by the setback
// railWrites before assembly; the baseline body is never built for a radius-torus corner).
func radiusTorusTransientBlend(v *topo.Vertex, rt *radiusTorusCornerGeom) (*cornerBlend, bool) {
	sph, err := geom.NewSphere(rt.center, rt.rS)
	if err != nil {
		return nil, false
	}
	tan := map[uint64]math.Point3{
		rt.wallA.ID(): rt.eqA,
		rt.wallB.ID(): rt.eqB,
		rt.top.ID():   rt.pTopA,
	}
	return &cornerBlend{vertex: v, center: rt.center, sphere: sph, tan: tan, radiusTorus: rt}, true
}

// radiusTorusPatch builds the torus corner face: the analytic torus (major R = rB−rS, minor rS)
// bounded by the two tube quarter-arcs, the outer-equator arc shared with the rB band, and the
// top-contact arc shared with the top plane's re-trim.
func radiusTorusPatch(rt *radiusTorusCornerGeom) (filletFace, bool) {
	tor, err := geom.NewTorus(rt.center, rt.axis, rt.rB-rt.rS, rt.rS)
	if err != nil {
		return filletFace{}, false
	}
	arcs := []cornerArcSpec{
		{rt.pTopA, rt.eqA, rt.footA, rt.rS},
		{rt.eqA, rt.eqB, rt.center, rt.rB},
		{rt.eqB, rt.pTopB, rt.footB, rt.rS},
		{rt.pTopB, rt.pTopA, rt.centerTop, rt.rB - rt.rS},
	}
	var fl filletLoop
	for _, a := range arcs {
		arc, err := geom.Arc3dByThreePoints(a.from, arcMidOnCircle(a.center, a.from, a.to, a.radius), a.to)
		if err != nil {
			return filletFace{}, false
		}
		fl.pts = append(fl.pts, a.from)
		fl.curves = append(fl.curves, arc)
	}
	return filletFace{surface: tor, loops: []filletLoop{fl}}, true
}

// radiusTorusHostEnd is the synthetic end corner injected into the top plane's re-trim: the receded
// vertex expands into the top-contact arc (radius rB−rS about the axis), keyed by the two walls so
// addEndCorner's tOf resolves either traversal direction.
func radiusTorusHostEnd(rt *radiusTorusCornerGeom, v *topo.Vertex) hostEndCorner {
	return hostEndCorner{face: rt.top, vid: v.ID(), corner: corner{
		a: rt.wallA, b: rt.wallB,
		ta: rt.pTopA, tb: rt.pTopB,
		mid:     arcMidOnCircle(rt.centerTop, rt.pTopA, rt.pTopB, rt.rB-rt.rS),
		endFace: rt.top, vertex: v,
	}}
}

// accumulateRadiusTorus records one mixed-radius torus corner's channel contributions: three band
// retract railWrites (each matched to its fils slot by pick edge id), the transient blend dropped,
// the torus patch, and the top-plane host end corner. A mismatch between the solved corner and the
// built bands contributes nothing, leaving the (deferred-baseline) decline.
func accumulateRadiusTorus(vid uint64, ctx setbackCtx, data *setbackData) {
	rt := ctx.blends[vid].radiusTorus
	bands := cornerBandsAt(vid, ctx.fils)
	patch, ok := radiusTorusPatch(rt)
	if !ok || len(bands) != 3 {
		return
	}
	writes := make([]railWrite, 0, 3)
	for _, b := range bands {
		br, ok := radiusTorusBandRetract(rt, ctx.fils, b)
		if !ok {
			return
		}
		writes = append(writes, bandRetractRailWrite(ctx.fils, br))
	}
	data.railWrites = append(data.railWrites, writes...)
	data.dropBlends[vid] = true
	data.extraPatches = append(data.extraPatches, patch)
	data.hostEnds = append(data.hostEnds, radiusTorusHostEnd(rt, ctx.blends[vid].vertex))
	data.fired = true
}

// radiusTorusBandRetract maps one converging band to its rewritten cross-section by pick edge id:
// the rB band retracts to the outer-equator arc (wall contacts eqA/eqB), each rS band to its tube
// arc (its wall contact + its top contact).
func radiusTorusBandRetract(rt *radiusTorusCornerGeom, fils []edgeFillet, b cornerBand) (bandRetract, bool) {
	switch fils[b.fi].edge.ID() {
	case rt.bigEdge:
		mid := arcMidOnCircle(rt.center, rt.eqA, rt.eqB, rt.rB)
		return bandRetractFor(b, pointByFace(rt.wallA, rt.eqA, rt.wallB, rt.eqB), mid), true
	case rt.smallEdgeA:
		mid := arcMidOnCircle(rt.footA, rt.eqA, rt.pTopA, rt.rS)
		return bandRetractFor(b, pointByFace(rt.wallA, rt.eqA, rt.top, rt.pTopA), mid), true
	case rt.smallEdgeB:
		mid := arcMidOnCircle(rt.footB, rt.eqB, rt.pTopB, rt.rS)
		return bandRetractFor(b, pointByFace(rt.wallB, rt.eqB, rt.top, rt.pTopB), mid), true
	}
	return bandRetract{}, false
}

// blendsCarryRadiusTorus reports whether any solved corner is a mixed-radius torus corner — the
// gate that defers the baseline assembly (whose transient band ends are not weldable) and turns a
// failed setback adoption into an honest error instead of a garbage fallback.
func blendsCarryRadiusTorus(blends map[uint64]*cornerBlend) bool {
	for _, cb := range blends {
		if cb.radiusTorus != nil {
			return true
		}
	}
	return false
}

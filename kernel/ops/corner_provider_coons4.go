// SPDX-License-Identifier: GPL-2.0-only
package ops

import (
	stdmath "math"

	"oblikovati.org/kernel/geom"
	"oblikovati.org/math"
)

// coons4Provider is the GENERAL 4-sided fill tier of the corner-blend engine (ADR-0051): a single
// Coons FillSurface over a 4-valence RailLoop, made G1-tangent to each side's Adjacent surface by a
// purpose-built cross-tangent RIBBON (adjacentRibbon), then certified. Unlike the obstacle provider
// (which reuses the abutting wing faces by identity), this tier works for ANY analytic adjacent —
// plane, cylinder, cone, torus — because the ribbon samples the adjacent's own normal at each control
// point rather than assuming a single extrusion direction. A side with G0 continuity or a nil Adjacent
// gets no ribbon (position-only). Any failed rail/ribbon/fill ⇒ honest-reject (ADR-3).
//
// Side→Coons mapping (load-bearing; pinned by TestQuarterCylLoopIsClosed + TestCoons4FitsAndBuilds).
// Loop sides [s0,s1,s2,s3] chain A→B→C→D→A with A=curveStart(s0), B=curveStart(s1), C=curveStart(s2),
// D=curveStart(s3). The Coons quad corners are (u0,v0)=A,(u1,v0)=B,(u1,v1)=C,(u0,v1)=D:
//
//	Coons rail | fill edge | loop side | pinned corners | ribbon from | order
//	c0         | VMinEdge  | s0        | A→B            | s0.Adjacent | s0.Cont
//	c1         | VMaxEdge  | s2        | D→C            | s2.Adjacent | s2.Cont
//	d0         | UMinEdge  | s3        | A→D            | s3.Adjacent | s3.Cont
//	d1         | UMaxEdge  | s1        | B→C            | s1.Adjacent | s1.Cont
type coons4Provider struct{}

var _ railProvider = coons4Provider{}

// Name reports the provider's telemetry kind (never read by assembly; ADR-2 lineage invariance).
func (coons4Provider) Name() CornerBlendKind { return BlendKindCoons4 }

// Fits claims any 4-sided loop; Build's certificate is the real admissibility gate.
func (coons4Provider) Fits(loop RailLoop) bool { return loop.Valence() == 4 }

// Build fills the loop and certifies it, or declines (ok=false) so a later tier / honest-reject
// handles it. It is the RailLoop-path sibling of bsplineObstacleProvider.Build.
func (coons4Provider) Build(loop RailLoop, scale Resolution) (CornerBlendPatch, Certificate, bool) {
	fill, rails, sides, ok := coons4Fill(loop, scale)
	if !ok {
		return CornerBlendPatch{}, Certificate{}, false
	}
	cert := certifyCoons4Patch(fill, rails, sides, loop, scale)
	patch := CornerBlendPatch{Surface: fill, Loops: railLoopToFilletLoops(loop), Kind: BlendKindCoons4}
	return patch, cert, true
}

// coons4Fill builds the four compatible+refined boundary rails, the G1 ribbons, and the matched
// FillSurface. It returns the refined rails and the assembled sides so certify measures against the
// exact same geometry (no recomputation/duplication). ok=false on any failure (honest-reject, ADR-3).
func coons4Fill(loop RailLoop, scale Resolution) (geom.BSplineSurface, [4]geom.BSplineCurve, [4]geom.FillSide, bool) {
	var noRails [4]geom.BSplineCurve
	c0, c1, d0, d1, ok := loopRails(loop)
	if !ok {
		return geom.BSplineSurface{}, noRails, [4]geom.FillSide{}, false
	}
	c0, c1, d0, d1, ok = refineForG1(c0, c1, d0, d1)
	if !ok {
		return geom.BSplineSurface{}, noRails, [4]geom.FillSide{}, false
	}
	rails := [4]geom.BSplineCurve{c0, c1, d0, d1}
	return assembleCoons4(loop, rails, scale)
}

// assembleCoons4 turns the refined rails into a matched, boundary-pinned FillSurface. Split from
// coons4Fill to keep both bodies within the function-length budget.
func assembleCoons4(loop RailLoop, rails [4]geom.BSplineCurve, scale Resolution) (geom.BSplineSurface, [4]geom.BSplineCurve, [4]geom.FillSide, bool) {
	base, err := geom.CoonsFill(rails[0], rails[1], rails[2], rails[3])
	if err != nil {
		return geom.BSplineSurface{}, rails, [4]geom.FillSide{}, false
	}
	sides, ok := coons4Sides(loop, rails, base)
	if !ok {
		return geom.BSplineSurface{}, rails, [4]geom.FillSide{}, false
	}
	fill, err := geom.FillSurface(rails[0], rails[1], rails[2], rails[3], sides)
	if err != nil {
		return geom.BSplineSurface{}, rails, [4]geom.FillSide{}, false
	}
	fill, err = pinFillBoundary(fill, rails[0], rails[1], rails[2], rails[3])
	return fill, rails, sides, err == nil
}

// loopRails builds the four FillSurface boundary rails from the loop sides per the Side→Coons mapping,
// orients+pins each to its canonical corners (pinnedRail), and makes the pairs compatible+corner-pinned
// (finishRails, reused from the obstacle path). ok=false on any bad side ⇒ honest-reject.
func loopRails(loop RailLoop) (c0, c1, d0, d1 geom.BSplineCurve, ok bool) {
	if loop.Valence() != 4 {
		return geom.BSplineCurve{}, geom.BSplineCurve{}, geom.BSplineCurve{}, geom.BSplineCurve{}, false
	}
	a, b, c, d := loopCorners(loop)
	tol := ResolutionForPoints([]math.Point3{a, b, c, d}).Weld()
	c0, ok0 := sideRail(loop.Sides[0], a, b, tol) // s0: A→B  (c0, VMinEdge)
	c1, ok1 := sideRail(loop.Sides[2], d, c, tol) // s2: D→C  (c1, VMaxEdge)
	d0, ok2 := sideRail(loop.Sides[3], a, d, tol) // s3: A→D  (d0, UMinEdge)
	d1, ok3 := sideRail(loop.Sides[1], b, c, tol) // s1: B→C  (d1, UMaxEdge)
	if !ok0 || !ok1 || !ok2 || !ok3 {
		return geom.BSplineCurve{}, geom.BSplineCurve{}, geom.BSplineCurve{}, geom.BSplineCurve{}, false
	}
	// finishRails pins (c0:a→d2, c1:pm→pp, d0:a→pm, d1:d2→pp); our corners map a=A,d2=B,pm=D,pp=C.
	return finishRails(c0, c1, d0, d1, a, b, d, c)
}

// loopCorners returns the four loop corners A,B,C,D = the start point of each successive side.
func loopCorners(loop RailLoop) (a, b, c, d math.Point3) {
	return curveStart(loop.Sides[0].Curve), curveStart(loop.Sides[1].Curve),
		curveStart(loop.Sides[2].Curve), curveStart(loop.Sides[3].Curve)
}

// sideRail converts one loop side's Curve to the BSplineCurve rail (asBSplineCurve) and pins it to run
// from→to (pinnedRail auto-reverses a rail traced the other way). ok=false on a non-convertible curve.
func sideRail(s Side, from, to math.Point3, tol float64) (geom.BSplineCurve, bool) {
	raw, ok := asBSplineCurve(s.Curve)
	if !ok {
		return geom.BSplineCurve{}, false
	}
	return pinnedRail(raw, from, to, tol)
}

// coons4Sides builds the four FillSides in Coons order [c0,c1,d0,d1] mapped from loop sides
// [s0,s2,s3,s1]. Each G1 side gets an adjacentRibbon that extrudes AWAY from the patch interior: the
// reference is the OUTWARD cross-derivative (the negated plain-Coons inward cross-derivative). This is
// load-bearing for NoFold — MatchSurface glues the ribbon on the OPPOSITE side of the seam (for a
// VMin↔VMin match, `into` gives the fill's into-patch cross-derivative the NEGATED ribbon derivative,
// see geom/match_surface.go), so a ribbon extruded outward lands the fill's cross-derivative back
// INSIDE the patch; an inward ribbon reverses S_v at the seam and folds the surface one station in.
func coons4Sides(loop RailLoop, rails [4]geom.BSplineCurve, base geom.BSplineSurface) ([4]geom.FillSide, bool) {
	length := coons4RibLen(loop)
	fs0, ok0 := ribbonSide(rails[0], loop.Sides[0], inwardCrossV(base, false).Scale(-1), length) // c0
	fs1, ok1 := ribbonSide(rails[1], loop.Sides[2], inwardCrossV(base, true).Scale(-1), length)  // c1
	fs2, ok2 := ribbonSide(rails[2], loop.Sides[3], inwardCrossU(base, false).Scale(-1), length) // d0
	fs3, ok3 := ribbonSide(rails[3], loop.Sides[1], inwardCrossU(base, true).Scale(-1), length)  // d1
	if !ok0 || !ok1 || !ok2 || !ok3 {
		return [4]geom.FillSide{}, false
	}
	return [4]geom.FillSide{fs0, fs1, fs2, fs3}, true
}

// ribbonSide builds one G1 FillSide (ribbon as Adjacent, its VMinEdge is the shared rail). A G0 side
// (Cont≤0) or a nil Adjacent gets a zero, position-only FillSide{Order:0} — no ribbon.
func ribbonSide(rail geom.BSplineCurve, s Side, awayRef math.Vector3, length float64) (geom.FillSide, bool) {
	if s.Cont <= 0 || s.Adjacent == nil {
		return geom.FillSide{Order: 0}, true
	}
	rib, ok := adjacentRibbon(rail, s.Adjacent, awayRef, length)
	if !ok {
		return geom.FillSide{}, false
	}
	return geom.FillSide{Adjacent: rib, AdjEdge: geom.VMinEdge, Order: int(s.Cont)}, true
}

// coons4RibLen is the model-relative ribbon length: a small fraction of the loop's bounding span
// (ADR-0042). Ribbon length only affects first-order matching, so a modest value suffices.
func coons4RibLen(loop RailLoop) float64 {
	a, b, c, d := loopCorners(loop)
	return ResolutionForPoints([]math.Point3{a, b, c, d}).Size() * ribbonSpanFactor
}

// adjacentRibbon is THE new geometry: a degree-(p,1) ribbon whose VMinEdge IS rail and whose second
// control row offsets each control point by `length` along the adjacent surface's tangent-plane
// direction n×t (surface normal × rail tangent, sampled at the control point's Greville parameter).
// Because the direction is computed PER control point from the adjacent's OWN normal, one builder is
// correct for plane/cylinder/cone/torus uniformly — it reduces to extrudeRibbon's constant dir for a
// plane or a cylinder and tracks the varying normal for a cone/torus. The control-net layout mirrors
// extrudeRibbon (corner_blend_obstacle.go) exactly so FillSurface's MatchSurface sees the same shape.
// awayRef points OUT of the patch interior (see coons4Sides) — each control point's n×t offset is
// oriented to agree with it, so the ribbon extrudes away from the fill and MatchSurface does not fold.
func adjacentRibbon(rail geom.BSplineCurve, adj geom.Surface, awayRef math.Vector3, length float64) (geom.BSplineSurface, bool) {
	n := len(rail.Ctrl)
	ctrl := make([][]math.Point3, n)
	w := make([][]float64, n)
	for i := range rail.Ctrl {
		dir, ok := ribbonDir(rail, adj, i, awayRef, length)
		if !ok {
			return geom.BSplineSurface{}, false
		}
		ctrl[i] = []math.Point3{rail.Ctrl[i], rail.Ctrl[i].TranslateBy(dir)}
		w[i] = []float64{rail.Weights[i], rail.Weights[i]}
	}
	surf, err := geom.NewBSplineSurface(rail.Degree, 1, ctrl, w, rail.Knots, []float64{0, 0, 1, 1})
	return surf, err == nil
}

// ribbonDir is the per-control-point cross-tangent offset n×t: the adjacent surface's normal at the
// rail point crossed with the rail tangent, scaled to `length` and sign-oriented to agree with awayRef
// (orientInward, robust against the adjacent's own normal orientation). It declines (ok=false) when the
// rail is parallel to its own surface normal (dir collapses ⇒ ill-posed ribbon).
func ribbonDir(rail geom.BSplineCurve, adj geom.Surface, i int, awayRef math.Vector3, length float64) (math.Vector3, bool) {
	xi := grevilleParam(rail, i)
	u, v := adj.ParamAt(rail.PointAt(xi))
	dir := adj.NormalAt(u, v).Cross(rail.TangentAt(xi))
	l := dir.Length()
	if l < ribbonDirTiny {
		return math.Vector3{}, false
	}
	return orientInward(dir.Scale(length/l), awayRef), true
}

// ribbonDirTiny floors the n×t magnitude: below it the rail runs along its surface normal and the
// tangent-plane offset direction is undefined, so the ribbon is ill-posed and the provider declines.
const ribbonDirTiny = 1e-12

// grevilleParam returns control point i's Greville abscissa in the curve's OWN parameter domain (the
// mean of its Degree covering knots) — the rail parameter whose PointAt lands nearest control point i,
// so the sampled adjacent normal/tangent belong to that control point. Unlike geom.grevilleAbscissae
// this is NOT normalized to [0,1] (we need the true domain parameter for PointAt/TangentAt).
func grevilleParam(c geom.BSplineCurve, i int) float64 {
	if c.Degree == 0 {
		return c.Knots[i]
	}
	sum := 0.0
	for k := 1; k <= c.Degree; k++ {
		sum += c.Knots[i+k]
	}
	return sum / float64(c.Degree)
}

// certifyCoons4Patch proves the patch (ADR-3), reusing the obstacle certify generics: Closed from the
// loop, WeldsArms structural (four spanned sides), NoFold from the anti-fold sweep, MaxDev the G0 rail
// residual on every edge, MaxAngleDev the G1 crease over ONLY the G1 sides (G0 sides skipped, exactly
// like the obstacle rim).
func certifyCoons4Patch(fill geom.BSplineSurface, rails [4]geom.BSplineCurve, sides [4]geom.FillSide, loop RailLoop, scale Resolution) Certificate {
	return Certificate{
		Closed:      loop.Closed(scale.Weld()),
		WeldsArms:   true,
		NoFold:      obstacleNoFold(fill, scale),
		MaxDev:      coons4MaxDev(fill, rails),
		MaxAngleDev: coons4MaxAngleDev(fill, sides),
	}
}

// coons4MaxDev is the max G0 positional residual from each rail to the fill's matching boundary edge
// (edges in Coons order c0→VMin, c1→VMax, d0→UMin, d1→UMax). ~0 after pinFillBoundary.
func coons4MaxDev(fill geom.BSplineSurface, rails [4]geom.BSplineCurve) float64 {
	edges := coons4Edges()
	m := 0.0
	for i, e := range edges {
		m = stdmath.Max(m, railDev(fill, rails[i], e))
	}
	return m
}

// coons4MaxAngleDev is the max G1 crease angle across ONLY the G1 sides (Order>0). A G0 or ribbon-less
// side is skipped — measuring continuity there would falsely reject a correct patch (obstacle rim rule).
func coons4MaxAngleDev(fill geom.BSplineSurface, sides [4]geom.FillSide) float64 {
	edges := coons4Edges()
	m := 0.0
	for i, e := range edges {
		if sides[i].Order > 0 {
			m = stdmath.Max(m, seamCrease(fill, sides[i].Adjacent, e))
		}
	}
	return m
}

// coons4Edges is the fill-edge order matching the rails/sides arrays [c0,c1,d0,d1].
func coons4Edges() [4]fillEdge {
	return [4]fillEdge{edgeVMin, edgeVMax, edgeUMin, edgeUMax}
}

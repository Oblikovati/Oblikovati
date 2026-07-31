// SPDX-License-Identifier: GPL-2.0-only

package ops

import (
	stdmath "math"

	"oblikovati.org/kernel/geom"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
)

// addEndCorner rounds the simple end corner at loop position i on face f: it looks up the corner, resolves
// its incoming/outgoing tangent points, and appends the rounded boundary. When the outgoing survivor edge
// is a CURVED wall rim (addCornerRound carries its parent arc) it returns the tOut segment index so the
// caller can trim that arc to the retained sub-arc; otherwise it returns -1 (a straight survivor, nothing
// to trim). Splitting this out of transformLoop keeps that walker within the statement budget (funlen).
func addEndCorner(fl *filletLoop, f *topo.Face, ends map[uint64]corner, uses []*topo.EdgeUse, i int) int {
	n := len(uses)
	u := uses[i]
	c := ends[useFromVertex(u).ID()]
	tIn := c.tOf(otherFace(uses[(i-1+n)%n].Edge(), f))
	tOut := c.tOf(otherFace(u.Edge(), f))
	if addCornerRound(fl, c, tIn, tOut, survivorCurve(u)) {
		return len(fl.pts) - 1
	}
	return -1
}

// addSubstVertex adds a fillet A/B-corner vertex that transformLoop's `subs` branch pulled back to its
// tangent point, carrying the curve of the edge LEAVING it. A STRAIGHT leaving edge stays nil (the base
// planar-fillet behaviour, byte-identical to the whole planar corpus + every fingerprint pin). A CURVED
// survivor rim leaving the tangent point — an annular-sector host's outer arc leaving the fillet corner
// (I3, i3-recon-rootcause.md) — is carried as its parent arc and the tOut segment index returned, so
// trimCarriedSubArcs trims that parent to the sub-arc actually retained between the tangent point and the
// next loop vertex. The pre-fix code hard-coded nil here, chording that outer arc to a straight line and
// slicing ~24000 (−63%) off the sector, which then folded the adjacent cone (conformCylConeFaces).
func addSubstVertex(fl *filletLoop, tan math.Point3, u *topo.EdgeUse) int {
	if rim, curved := carriableRim(survivorCurve(u)); curved {
		fl.add(tan, rim) // carry the curved survivor rim; trimCarriedSubArcs cuts it to the retained span
		return len(fl.pts) - 1
	}
	fl.add(tan, nil) // straight survivor leaving edge: nil, byte-identical to the base planar path
	return -1
}

// carriableRim reports whether a survivor rim curve is one the retained-span algebra can re-derive from
// the parent's OWN parameters — a circular geom.Arc3d (subArcOnParent) or an elliptic geom.EllipticalArc
// (retainedEllipticRimCurve, fillet_survivor_rim_ellipse.go). Every other kind (straight, closed conic
// seam, B-spline) returns false and keeps the base straight chord, byte-identical to the pre-carry path.
// This is the ONE place the carriable set is named, so the two carry sites (addSubstVertex and
// addCornerRound) and the two trim passes cannot drift apart.
func carriableRim(c geom.Curve3) (geom.Curve3, bool) {
	switch rim := c.(type) {
	case geom.Arc3d:
		return rim, true
	case geom.EllipticalArc:
		return rim, true
	}
	return nil, false
}

// exactRetainedSpanOnParent is the GATE-FREE core of the carry: the carriable parent's own sub-span
// between two points that already lie ON it, or nil when either point is off the parent (or the parent is
// not carriable at all). No empirical chord-vs-arc argument — neither the ENDS branch's quadrant nor the
// subs branch's erased-segment area — applies to an exact span, because if both endpoints ARE points of
// the parent then the parent's sub-span IS the true boundary and a chord across it is simply off the
// surface. Built from the parent's OWN parameters, so a MAJOR span stays major (the N7 whole-curve
// sub-span lesson) and the result runs from→to in the caller's own order by construction.
//
// It is the piece the rim-fillet HOST REBUILD reuses (retainedHostSeamCurve, fillet_rim_build.go) so the
// re-aimed host seam keeps its meridian instead of chording it; the survivor-rim carry reaches the same
// algebra through its own gated wrappers (retainedSubArc / retainedRimCurve).
func exactRetainedSpanOnParent(parent geom.Curve3, p0, p1 math.Point3) geom.Curve3 {
	switch rim := parent.(type) {
	case geom.Arc3d:
		if !rimSpanIsExact(rim, p0, p1) {
			return nil
		}
		return subArcOnParent(rim, projectOntoArcCircle(rim, p0), projectOntoArcCircle(rim, p1))
	case geom.EllipticalArc:
		return retainedEllipticRimCurve(rim, p0, p1) // already exactness-gated (ellipseSpanIsExact)
	case geom.BSplineCurve:
		// Wave-G additive arm: a B-SPLINE parent (a swept pipe wall's curved seam meridian)
		// carries its own exact sub-span (fillet_bspline_host_rim.go). Exactness-gated inside
		// (both points must lie ON the parent), so any caller whose points are off the curve
		// still gets nil — the byte-identical base chord it always shipped.
		return retainedBsplineSpan(rim, p0, p1)
	}
	return nil
}

// subArcAreaFrac is the minimum fraction of the model's characteristic area (scale², scale = the body
// bounding-box diagonal) that chording a carried survivor arc must ERASE for the arc to be worth carrying.
// The ENDS branch gates on an ANGULAR span (retainedRimCurve's quadrant), which cannot separate the subs
// cases: I3's outer rim is 88° and N5's boss rim is 76° — BOTH ≤ π/2, so an angular gate keeps I3 chorded
// (its −63% host collapse) OR carries N5's harmless rim. The subs branch instead measures the ABSOLUTE
// circular-segment area a chord erases, made model-relative by scale²: I3 erases ~13% of scale² (a real
// −24000 host bite that folds the neighbour cone), N5 only ~0.17% (a faithful chord on a minor boss face).
// The 1% cut mirrors the corpus area deps — the arc matters exactly when chording it risks the 1% gate —
// and sits in the wide 0.17%↔13% gap between the two observed clusters with >6× margin on each side. A rim
// below it keeps its base chord (nil), byte-identical to the whole planar corpus + every fingerprint pin.
const subArcAreaFrac = 0.01

// trimCarriedSubArcs replaces each carried full parent arc (stamped on its tOut segment by addSubstVertex)
// with the sub-arc actually retained between that segment's own endpoints — the tangent point and the next
// loop vertex — UNLESS chording that sub-arc erases only a minor segment (chordErasesMinorSegment), in which
// case the base straight chord is restored (nil, byte-identical to the pre-carry planar path). The `subs`
// branch moves the corner vertex only by the fillet pull-back, so a carried arc is nearly the WHOLE parent
// rim (I3's outer arc trims 90°→~88°). Each endpoint is projected onto the parent's own circle first (the
// tangent point sits ~0.17 OFF the rim by the pull-back), then the sub-arc is built from the parent's
// parameters so its span stays faithful (subArcOnParent).
//
// The area gate is SKIPPED when both endpoints already lie exactly on the rim (rimSpanIsExact): the far-end
// trim has then put the band's terminal section ON the wall (fillet_farend_trim.go), so the retained sub-arc
// IS the wall's true boundary and no minor-segment argument can justify chording it. N5's boss rim is the
// only corpus case this reaches, and it moves N5 the right way on both measures: its worst
// boundary-off-its-own-face residual 4.17 -> 0.024 and its per-face gross error vs DRAWEXE 466.6 -> 348.1.
func trimCarriedSubArcs(fl *filletLoop, idxs []int, modelScale float64) {
	n := len(fl.pts)
	for _, i := range idxs {
		p0, p1 := fl.pts[i], fl.pts[(i+1)%n]
		switch parent := fl.curves[i].(type) {
		case geom.Arc3d:
			fl.curves[i] = retainedSubArc(parent, p0, p1, modelScale)
		case geom.EllipticalArc:
			fl.curves[i] = retainedEllipticRimCurve(parent, p0, p1)
		}
	}
}

// retainedSubArc is the CIRCULAR sub-span to carry between a subs-branch segment's own endpoints, or nil
// (the base straight chord) when chording it erases only a minor circular segment. Extracted verbatim
// from trimCarriedSubArcs when the elliptic arm was added, so the circular path stays byte-identical.
func retainedSubArc(parent geom.Arc3d, p0, p1 math.Point3, modelScale float64) geom.Curve3 {
	from, to := projectOntoArcCircle(parent, p0), projectOntoArcCircle(parent, p1)
	if !rimSpanIsExact(parent, p0, p1) && chordErasesMinorSegment(parent, from, to, modelScale) {
		return nil // faithful chord (or carry disabled, scale 0): keep the base loop byte-identical
	}
	return subArcOnParent(parent, from, to)
}

// chordErasesMinorSegment reports whether chording the retained sub-arc from→to erases only a MINOR circular
// segment — less than subArcAreaFrac of the model's characteristic area (scale²) — so the base straight
// chord is faithful and must be kept. It also returns true when modelScale<=0, the sentinel the specialized
// obstacle/runout/canal rebuild callers pass to DISABLE the carry (their paths stay byte-identical to the
// pre-carry planar retrim). The erased area is the circular segment 0.5·R²·(θ−sinθ).
func chordErasesMinorSegment(parent geom.Arc3d, from, to math.Point3, modelScale float64) bool {
	if modelScale <= 0 {
		return true
	}
	span := retainedSpan(parent, from, to)
	segArea := 0.5 * parent.Radius * parent.Radius * (span - stdmath.Sin(span))
	return segArea <= subArcAreaFrac*modelScale*modelScale
}

// subArcOnParent trims parent to the sub-arc from→to, built from parent's OWN parameters (same
// centre/axis/radius, StartAngle at from's parent-offset, SweepAngle = to.offset − from.offset) so the
// retained span stays faithful for both a minor and a major sub-span — a three-point re-fit silently
// snaps a >π span to its minor complement (the N7 whole-curve-sub-span lesson). Endpoints are assumed
// already projected onto the parent circle (trimCarriedSubArcs does so); if arcFrac still rejects one the
// carry falls back to the base straight chord (nil), never the un-trimmed full parent (the crude blow-up).
func subArcOnParent(parent geom.Arc3d, from, to math.Point3) geom.Curve3 {
	tf, okf := arcFrac(parent, from)
	tt, okt := arcFrac(parent, to)
	if !okf || !okt {
		return nil
	}
	return geom.Arc3d{
		Center: parent.Center, Normal: parent.Normal, RefDir: parent.RefDir, Radius: parent.Radius,
		StartAngle: parent.StartAngle + tf*parent.SweepAngle, SweepAngle: (tt - tf) * parent.SweepAngle,
	}
}

// The curved-survivor rim carry. A planar corner fillet whose END corner lands on a CURVED survivor
// face (a partial cylinder/cone/sphere sector's wall — B5/C4/D7/E1/E2, curved-host-collapse-rootcause.md)
// replaces that face's rim-arc endpoint with the corner tangent point. transformLoop's ENDS branch used
// to drop the OUTGOING rim edge's curve to nil (a straight chord across the wall), collapsing a 270° rim
// to its chord and cutting the wall ~in half. addCornerRound now carries the survivor's parent arc on the
// tOut segment; this file trims that parent to the sub-arc actually retained between the two corner tangent
// points, so the wall keeps its full area. A STRAIGHT survivor edge stays nil (byte-identical to the whole
// planar corpus + the 24 fingerprint pins), so only a genuinely curved wall changes.

// trimCarriedArcs applies the post-loop rim-arc trims for BOTH survivor-arc branches of transformLoop: the
// ENDS branch's quadrant-gated end-corner rim carries (rimCarries, trimCarriedRimArcs) and the subs branch's
// material-gated tangent-point carries (subCarries, trimCarriedSubArcs). Combined into one call so
// transformLoop stays within the statement budget (funlen).
func trimCarriedArcs(fl *filletLoop, rimCarries, subCarries []int, subArcScale float64) {
	trimCarriedRimArcs(fl, rimCarries)
	trimCarriedSubArcs(fl, subCarries, subArcScale)
	alignCarriedArcsToSegments(fl)
}

// alignCarriedArcsToSegments enforces the loop's own consistency invariant: a segment's carried curve must
// run BETWEEN that segment's two points. The `default` survivor branch carries an untouched rim arc whole,
// which is wrong whenever the segment's OTHER end was pulled back to a fillet tangent point — the arc then
// sweeps PAST the loop's own vertex and the face tessellates a boundary that crosses its neighbours
// (E1's meridian plane, 3 fold edges; the same overshoot is why E1/D3/D7/Q5/F6 carry a curve-domain-vs-vertex
// gap). Re-trimming to the segment's own span is the arc-side counterpart of what trimCarriedRimArcs and
// trimCarriedSubArcs already do for the two branches that record their indices. Segments whose arc already
// ends at its points (every correctly-built loop) are left untouched, so this is byte-invisible to them, and
// a CLOSED seam (both points on one vertex) is skipped — its full circle IS the boundary.
func alignCarriedArcsToSegments(fl *filletLoop) {
	n := len(fl.pts)
	for i := range fl.curves {
		p0, p1 := fl.pts[i], fl.pts[(i+1)%n]
		switch parent := fl.curves[i].(type) {
		case geom.Arc3d:
			alignCarriedArc(fl, i, parent, p0, p1)
		case geom.EllipticalArc:
			alignCarriedEllipse(fl, i, parent, p0, p1)
		}
	}
}

// alignCarriedArc re-trims one carried CIRCULAR rim to its segment's own span (extracted from
// alignCarriedArcsToSegments verbatim when the elliptic arm was added, so the circular path is
// byte-identical). A closed seam (both points on one vertex) is skipped — its full circle IS the boundary.
func alignCarriedArc(fl *filletLoop, i int, arc geom.Arc3d, p0, p1 math.Point3) {
	if arcSpansItsSegment(arc, p0, p1) || p0.DistanceTo(p1) <= 1e-9*arc.Radius {
		return
	}
	if sub := subArcOnParent(arc, projectOntoArcCircle(arc, p0), projectOntoArcCircle(arc, p1)); sub != nil {
		fl.curves[i] = sub
	}
}

// alignCarriedEllipse is the same consistency repair for a carried ELLIPTIC rim: the default
// (untouched-survivor) branch carries the parent whole, which overshoots the loop's own vertex once the
// segment's other end has been pulled back to a fillet tangent point. Only an on-parent span is
// re-derivable (retainedEllipticRimCurve), so an inexact one keeps the parent rather than degrading to a
// chord — the parent is at worst too long, a chord is off the surface entirely.
func alignCarriedEllipse(fl *filletLoop, i int, ea geom.EllipticalArc, p0, p1 math.Point3) {
	if ellipseSpansItsSegment(ea, p0, p1) || p0.DistanceTo(p1) <= 1e-9*ea.MajorRadius {
		return
	}
	if sub := retainedEllipticRimCurve(ea, p0, p1); sub != nil {
		fl.curves[i] = sub
	}
}

// arcSpansItsSegment reports whether arc already runs from p0 to p1, within 1e-9 of its own radius (the
// rim's own scale, so the test is scale-invariant without threading a Resolution).
func arcSpansItsSegment(arc geom.Arc3d, p0, p1 math.Point3) bool {
	lo, hi := arc.Domain()
	tol := 1e-9 * arc.Radius
	return arc.PointAt(lo).DistanceTo(p0) <= tol && arc.PointAt(hi).DistanceTo(p1) <= tol
}

// trimCarriedRimArcs replaces each carried full parent arc (the whole rim, stamped on its tOut segment by
// addCornerRound) with the sub-arc actually retained between that segment's own endpoints. When the retained
// span's endpoints are EXACTLY on the rim (the far-end trim put the band's terminal section on the wall,
// fillet_farend_trim.go) the sub-arc is always carried: it is then the true wall boundary. Otherwise the
// endpoints still sit on the fillet's CAP contact circle (radius √(r²+R²), OFF the wall), and the empirical
// quadrant gate decides between two imperfect approximations (retainedRimCurve). Either way each endpoint is
// first projected onto the rim's own circle before the retained span is measured — else subArcMajor/arcFrac
// reject an off-circle point and a major sub-arc silently degrades to its minor complement (a 270° rim would
// collapse back to 90°).
func trimCarriedRimArcs(fl *filletLoop, idxs []int) {
	n := len(fl.pts)
	for _, i := range idxs {
		p0, p1 := fl.pts[i], fl.pts[(i+1)%n]
		switch parent := fl.curves[i].(type) {
		case geom.Arc3d:
			from, to := projectOntoArcCircle(parent, p0), projectOntoArcCircle(parent, p1)
			fl.curves[i] = retainedRimCurve(parent, from, to, rimSpanIsExact(parent, p0, p1))
		case geom.EllipticalArc:
			fl.curves[i] = retainedEllipticRimCurve(parent, p0, p1)
		}
	}
}

// rimSpanIsExact reports whether both retained-span endpoints already lie ON the parent rim's own circle,
// within 1e-9 of its radius (the rim's own scale, so it is scale-invariant without threading a Resolution).
// True exactly when the far-end trim landed this corner's tangent points on the wall; false for a corner
// whose flat section cap is still off it (an unsupported wall type, or a decline).
func rimSpanIsExact(parent geom.Arc3d, a, b math.Point3) bool {
	tol := 1e-9 * parent.Radius
	return a.DistanceTo(projectOntoArcCircle(parent, a)) <= tol &&
		b.DistanceTo(projectOntoArcCircle(parent, b)) <= tol
}

// retainedRimCurve is the sub-arc to carry for a curved survivor rim, or nil (the BASE straight chord) when
// the retained span is at most a QUADRANT (π/2) AND the span's endpoints are not exactly on the rim.
//
// exact=true is the answer once the far-end trim has landed the band's terminal section ON the wall
// (fillet_farend_trim.go): the retained sub-arc is then the wall's TRUE boundary, so it is carried whatever
// its span. That retires the quadrant gate for every trimmed wall — B1/B9's 62–67° sector rims were shipping
// a chord 8.26 / 5.24 off their own host, purely because the gate had been calibrated against off-surface
// endpoints.
//
// exact=false keeps the historical EMPIRICAL CROSSOVER for a corner whose flat cap is still off the wall
// (an unsupported wall type, or a declined trim): the cap-contact tangent points sit on radius √(r²+R²), so
// the re-fit sub-arc leaves a corner notch that is only worth paying when the chord itself is badly wrong.
// A chord across a >π/2 rim deviates by more than R(1−cos45°) ≈ 0.29·R — a collapsed curved face or a large
// lune — so there the arc must still be carried.
func retainedRimCurve(parent geom.Arc3d, from, to math.Point3, exact bool) geom.Curve3 {
	if !exact && retainedSpan(parent, from, to) <= stdmath.Pi/2 {
		return nil // ≤ a quarter turn off an off-surface corner: the chord is the less-wrong approximation
	}
	if sub, _, major := subArcMajor(parent, from, to); major {
		return sub // > π: carry from the parent's own parameters so a >180° rim stays major (B5/C4/D7)
	}
	mid := arcMidBetween(parent.Center, parent.Radius, from, to)
	sub, err := geom.Arc3dByThreePoints(from, mid, to)
	if err != nil {
		return nil // degenerate near-antipodal re-fit: fall back to the base chord
	}
	return sub // (π/2, π]: a large minor rim re-fit through its shorter-arc midpoint (E1/E2)
}

// retainedSpan is the absolute angular span (radians) the sub-arc from→to subtends on parent's own circle;
// 0 when either endpoint is off the circle (arcFrac reject), which the ≤π/2 gate then treats as a chord.
func retainedSpan(parent geom.Arc3d, from, to math.Point3) float64 {
	tf, okf := arcFrac(parent, from)
	tt, okt := arcFrac(parent, to)
	if !okf || !okt {
		return 0
	}
	return stdmath.Abs((tt - tf) * parent.SweepAngle)
}

// projectOntoArcCircle drops p onto the circle of arc (same centre/axis/radius): it removes p's out-of-plane
// component along the arc Normal and rescales the in-plane part to the radius. A p on the arc axis (zero
// in-plane component) is returned unchanged — a degeneracy the rim corners never hit (a tangent point is
// always off-axis on the wall). Used to put the off-surface corner tangent points back onto the rim circle
// before the sub-arc span is measured.
func projectOntoArcCircle(arc geom.Arc3d, p math.Point3) math.Point3 {
	v := arc.Center.VectorTo(p)
	axial := arc.Normal.AsVector().Scale(v.Dot(arc.Normal.AsVector()))
	dir, err := math.UnitVector3FromVector(v.Sub(axial))
	if err != nil {
		return p
	}
	return arc.Center.TranslateBy(dir.AsVector().Scale(math.Scalar(arc.Radius)))
}

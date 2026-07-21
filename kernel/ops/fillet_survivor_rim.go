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

// The curved-survivor rim carry. A planar corner fillet whose END corner lands on a CURVED survivor
// face (a partial cylinder/cone/sphere sector's wall — B5/C4/D7/E1/E2, curved-host-collapse-rootcause.md)
// replaces that face's rim-arc endpoint with the corner tangent point. transformLoop's ENDS branch used
// to drop the OUTGOING rim edge's curve to nil (a straight chord across the wall), collapsing a 270° rim
// to its chord and cutting the wall ~in half. addCornerRound now carries the survivor's parent arc on the
// tOut segment; this file trims that parent to the sub-arc actually retained between the two corner tangent
// points, so the wall keeps its full area. A STRAIGHT survivor edge stays nil (byte-identical to the whole
// planar corpus + the 24 fingerprint pins), so only a genuinely curved wall changes.

// trimCarriedRimArcs replaces each carried full parent arc (the whole rim, stamped on its tOut segment by
// addCornerRound) with the sub-arc actually retained between that segment's own endpoints — but ONLY when
// that sub-arc materially deviates from its chord (retainedRimCurve's quadrant gate); otherwise it restores
// the base straight chord (nil). The corner tangent points sit on the fillet's CAP contact circle (radius
// √(r²+R²), OFF the wall surface by the root-cause receipt), so each endpoint is first projected onto the
// rim's own circle before the retained span is measured — else subArcMajor/arcFrac reject the off-circle
// point and a major sub-arc silently degrades to its minor complement (a 270° rim would collapse back to 90°).
func trimCarriedRimArcs(fl *filletLoop, idxs []int) {
	n := len(fl.pts)
	for _, i := range idxs {
		parent, ok := fl.curves[i].(geom.Arc3d)
		if !ok {
			continue // defensive: only a carried Arc3d parent is trimmable (never hit — addCornerRound stamps only arcs)
		}
		from := projectOntoArcCircle(parent, fl.pts[i])
		to := projectOntoArcCircle(parent, fl.pts[(i+1)%n])
		fl.curves[i] = retainedRimCurve(parent, from, to)
	}
}

// retainedRimCurve is the sub-arc to carry for a curved survivor rim, or nil (the BASE straight chord) when
// the retained span is at most a QUADRANT (π/2). WHY the gate: the fillet cap-contact tangent points are off
// the wall (√(r²+R²)), so the re-fit sub-arc leaves a small corner notch; that notch is only worth paying
// when the chord itself is badly wrong. A chord across a >π/2 rim deviates from the wall by more than
// R(1−cos45°) ≈ 0.29·R — a collapsed curved face (B5/C4/D7's 242–255° rims) or a large lune off the adjacent
// meridian plane (E1/E2's 144–146° sphere rims) — so the arc MUST be carried. For a ≤π/2 rim (B1/B9's
// 62–67° sector rims) the off-surface tangent-point notch makes the re-fit arc LESS accurate than the base
// chord, so keeping the chord is BOTH more faithful AND byte-identical to the planar corpus + pins. π/2 is
// therefore an EMPIRICAL CROSSOVER between two imperfect approximations (chord vs off-surface arc), NOT a
// first-principles law; it sits in the wide 67°→144° gap between the two carried clusters (62–67° vs
// 144–255°). A mere >π (major-only) gate would wrongly drop E1/E2's minor sphere meridians and un-green them.
// FOLLOW-UP: an on-surface tangent-point fix (project onto the wall) would make the arc exact and retire
// this gate. The asymmetry is deliberately safe: a sub-π/2 rim that should carry merely stays red; only a
// super-π/2 rim that should chord could drift — none exists in the corpus (B1/B9 are pinned to lock it).
func retainedRimCurve(parent geom.Arc3d, from, to math.Point3) geom.Curve3 {
	if retainedSpan(parent, from, to) <= stdmath.Pi/2 {
		return nil // ≤ a quarter turn: the chord is faithful — keep the base loop byte-identical (B1/B9)
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

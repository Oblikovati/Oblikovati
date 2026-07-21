// SPDX-License-Identifier: GPL-2.0-only

package ops

import (
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
// addCornerRound) with the sub-arc between that segment's own endpoints. The corner tangent points sit on
// the fillet's CAP contact circle (radius √(r²+R²), OFF the wall surface by the root-cause receipt), so
// each endpoint is first projected onto the rim's own circle before the retained span is measured — else
// subArcMajor/arcFrac reject the off-circle point and the major sub-arc silently degrades to its minor
// complement (a 270° rim would collapse back to 90°).
func trimCarriedRimArcs(fl *filletLoop, idxs []int) {
	n := len(fl.pts)
	for _, i := range idxs {
		parent, ok := fl.curves[i].(geom.Arc3d)
		if !ok {
			continue // defensive: only a carried Arc3d parent is trimmable (never hit — addCornerRound stamps only arcs)
		}
		from := projectOntoArcCircle(parent, fl.pts[i])
		to := projectOntoArcCircle(parent, fl.pts[(i+1)%n])
		fl.curves[i] = survivorSubArc(parent, from, to)
	}
}

// survivorSubArc is the retained rim sub-arc between from and to (both projected onto parent's circle),
// mirroring subSeg (fillet_curved_retrim_loop.go): a MAJOR (>π) retained span is carried from the parent's
// OWN parameters (subArcMajor) so a >180° wall rim stays major rather than snapping to its minor complement
// under an ill-conditioned three-point re-fit; a minor span keeps the shorter-arc re-fit through its midpoint.
// Returns nil (a straight chord — the pre-fix behaviour) only on a degenerate near-antipodal re-fit.
func survivorSubArc(parent geom.Arc3d, from, to math.Point3) geom.Curve3 {
	if sub, _, ok := subArcMajor(parent, from, to); ok {
		return sub
	}
	mid := arcMidBetween(parent.Center, parent.Radius, from, to)
	sub, err := geom.Arc3dByThreePoints(from, mid, to)
	if err != nil {
		return nil
	}
	return sub
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

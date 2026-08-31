// SPDX-License-Identifier: GPL-2.0-only

package brep

import (
	stdmath "math"

	"oblikovati.org/kernel/geom"
	"oblikovati.org/math"
)

// Already-cut cylinder-side recognition for the partial-rim second cut (Oblikovati/Oblikovati#1732).
// fullCylinderSideBand (curved_halfspace_cylinder_side.go) resolves a BARE side — two full-circle rims — for
// cylinderOperand. After a first cut the side carries ONE surviving full-circle rim plus a NOTCHED boundary
// (rim arcs + the first cut's section conic); it fails the two-circle test there and, until now, fell straight
// to CSG. cutCylinderSideBand recognises that cut side so a second curved boolean can compose the prior
// boundary as constraint edges instead. It is the deliberate mirror of fullCylinderSideBand: a bare side (two
// full circles) is declined HERE and resolved by cylinderOperand, so the two partition cleanly.

// priorTrimLoop is the surviving boundary of an earlier cut on a cylinder side: the ordered analytic edges of
// the notched-rim loop (rim arcs + the first cut's section conic). A SECOND cut's (u,v) arrangement ingests
// these as constraint edges so it composes with the first removal instead of re-including the notch (#1732).
// Held apart from coneSideBand_ because the bare-band path has no prior boundary.
type priorTrimLoop struct {
	edges []loopEdge
}

// bandTopPadRel pads the recovered vMax as a fraction of the rim radius, floored well above the weld grid, so
// the arrangement's artificial top edge sits clear of the prior loop's highest point. Without the pad an
// oblique section conic's apex lands EXACTLY on v=vMax — a tangency arrangement vertex, the most fragile case —
// instead of strictly inside the padded rectangle where the top edge bounds no real geometry (#1732).
const bandTopPadRel = 1e-3

// cutCylinderSideBand reports whether f is an ALREADY-CUT cylinder side — a geom.Cylinder bounded by exactly
// ONE full-circle rim (the surviving bottom, the v=0 band anchor) plus a closed prior-trim loop — and returns
// the cylinder, the band anchored on that bottom circle with vMax recovered from the prior loop's highest
// surviving point, and the prior loop. A first cut that also severed the bottom rim leaves no full circle (or
// leaves the surviving circle above the notch) and declines here: the v=0 anchor is load-bearing for
// band-extent recovery (#1732).
func cutCylinderSideBand(f curvedFace) (geom.Cylinder, coneSideBand_, priorTrimLoop, bool) {
	cyl, ok := f.surface.(geom.Cylinder)
	if !ok || len(f.loops) == 0 {
		return geom.Cylinder{}, coneSideBand_{}, priorTrimLoop{}, false
	}
	anchor, anchorReversed, prior, ok := anchorCircleAndPrior(f)
	if !ok {
		return geom.Cylinder{}, coneSideBand_{}, priorTrimLoop{}, false
	}
	axis := cyl.AxisDir.AsVector()
	vMax, ok := recoverBandTop(prior, anchor.Center, axis, cyl.Radius)
	if !ok {
		return geom.Cylinder{}, coneSideBand_{}, priorTrimLoop{}, false
	}
	top := anchor.Center.TranslateBy(axis.Scale(math.Scalar(vMax)))
	band := coneSideBand_{
		bottom: anchor.Center, top: top, bottomCirc: anchor,
		vMin: 0, vMax: vMax, rBot: cyl.Radius, rTop: cyl.Radius,
		botRimReversed: anchorReversed, // the vMax end is SYNTHETIC here, so only the anchor carries a source sense
	}
	return cyl, band, prior, true
}

// anchorCircleAndPrior partitions a cut cylinder side's edges into its single surviving full-circle rim (the
// anchor) and the prior-trim loop (everything else). Exactly one full circle is the cut-side signature: two is
// a BARE side (fullCylinderSideBand's job), zero means the first cut removed the anchor rim — both decline.
func anchorCircleAndPrior(f curvedFace) (anchor geom.Circle, anchorReversed bool, prior priorTrimLoop, ok bool) {
	var circles []geom.Circle
	var revs []bool
	var rest []loopEdge
	for _, lp := range f.loops {
		for _, e := range lp.edges {
			if c, isFull := fullRimCircle(e); isFull {
				circles, revs = append(circles, c), append(revs, e.t1 < e.t0)
				continue
			}
			rest = append(rest, e)
		}
	}
	if len(circles) != 1 || len(rest) == 0 {
		return geom.Circle{}, false, priorTrimLoop{}, false
	}
	return circles[0], revs[0], priorTrimLoop{edges: rest}, true
}

// fullRimCircle reports whether an edge is a full-domain circle (a whole rim), returning that circle.
func fullRimCircle(e loopEdge) (geom.Circle, bool) {
	if c, ok := e.curve.(geom.Circle); ok && isFullDomain(e.t0, e.t1) {
		return c, true
	}
	return geom.Circle{}, false
}

// recoverBandTop returns the padded vMax for a cut side's full-rectangle domain: the prior loop's highest
// axial coordinate above the anchor, plus a model-scaled pad. It declines when the prior loop dips BELOW the
// anchor (the surviving full circle is the top rim, not the bottom — the v=0 anchor is gone) or sits at/below
// it (no band). Axial coordinate is distance along the axis from the anchor centre origin (#1732).
func recoverBandTop(prior priorTrimLoop, origin math.Point3, axis math.Vector3, r float64) (float64, bool) {
	lo, hi := priorAxialRange(prior, origin, axis)
	pad := stdmath.Max(bandTopPadRel*r, 1e3*planarStitchGrid)
	if lo < -pad || hi <= pad {
		return 0, false
	}
	return hi + pad, true
}

// priorAxialRange returns the min and max axial coordinate (distance along axis from origin) over the prior loop,
// sampled per edge. vMax is an artificial, PADDED domain bound — never emitted geometry — so a dense sample
// whose chord error (sub-µm on a cm-scale conic at this density) is dwarfed by the pad is a rigorous upper
// bound; no per-curve extremum solve is needed (#1732).
func priorAxialRange(prior priorTrimLoop, origin math.Point3, axis math.Vector3) (lo, hi float64) {
	lo, hi = stdmath.Inf(1), stdmath.Inf(-1)
	const n = 64
	for _, e := range prior.edges {
		for i := 0; i <= n; i++ {
			t := e.t0 + (e.t1-e.t0)*float64(i)/float64(n)
			a := float64(origin.VectorTo(e.curve.PointAt(t)).Dot(axis))
			lo, hi = stdmath.Min(lo, a), stdmath.Max(hi, a)
		}
	}
	return lo, hi
}

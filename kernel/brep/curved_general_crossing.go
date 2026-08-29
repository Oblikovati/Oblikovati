// SPDX-License-Identifier: GPL-2.0-only

package brep

import (
	"fmt"

	"oblikovati.org/kernel/diag"
	"oblikovati.org/kernel/geom"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
)

// Unified ruled-crossing intersect driver (ADR-0058 phase 3). The former per-pair drivers (cone∩cone,
// cone∩cylinder, cylinder∩cylinder) had the identical skeleton — imprint → each side's face + solid
// membership → trim each side inside the other → curvedStitch — differing only in which per-primitive side
// split each called. With the imprint (curvedImprintLoops), membership (curvedSolidMembership) and stitch
// already general, the last per-pair piece is the side split; curvedSideFace + curvedSideSolidSplit make it
// one. ruledCrossingIntersect below now builds EVERY ruled crossing, the cylinder∩cylinder near-pinch (#1818)
// folded in as one conditioning branch. A sphere/torus side folds in the moment curvedSideFace and
// curvedSideSolidSplit learn it. (The cut/join drivers still keep a ≥1-cone guard — RuledConeCrossingCut/Join
// — because a cylinder∩cylinder near-pinch cut/join is a separate, more involved path.)

// curvedSideFace returns an operand's principal curved side face for the trim: a cone/frustum side or a
// cylinder side, with its surface and rim band. ok=false when the body has neither (the caller declines).
func curvedSideFace(b *topo.Body) (curvedFace, geom.Surface, coneSideBand_, bool) {
	if f, cone, band, ok := coneSideFace(b); ok {
		return f, cone, band, true
	}
	if f, cyl, band, ok := cylinderSideFace(b); ok {
		return f, cyl, band, true
	}
	return curvedFace{}, nil, coneSideBand_{}, false
}

// curvedSideSolidSplit trims a curved side face by the SSI imprint, keeping the cells whose surface point
// lies inside the other solid under op — dispatching a cone side to coneSideSolidSplit and a cylinder side
// to cylinderSideSolidSplit (a cylinder is the degenerate cone; both build a ruledUV and trim by the same
// arrangement). An unsupported side surface is an error naming the type, not a silent decline.
func curvedSideSolidSplit(f curvedFace, surface geom.Surface, band coneSideBand_, imprint []geom.Curve3, op Op, isB bool, inside func(math.Point3) bool) ([]curvedFace, []loopEdge, error) {
	switch s := surface.(type) {
	case geom.Cone:
		return coneSideSolidSplit(f, s, band, imprint, op, isB, inside)
	case geom.Cylinder:
		return cylinderSideSolidSplit(f, s, band, imprint, op, isB, inside)
	default:
		return nil, nil, fmt.Errorf("curvedSideSolidSplit: unsupported side surface %T (want geom.Cone or geom.Cylinder)", surface)
	}
}

// RuledCrossingIntersectGeneral is the ONE exported entry kernel/ops routes EVERY ruled-crossing intersect
// through — cone∩cone, cone∩cylinder AND cylinder∩cylinder (ADR-0058 phase 3): the general curved∩curved
// pipeline, no bespoke loop→body constructor. It replaces the former split between a cone-crossing driver and
// a separate cylinder near-pinch driver; the near-pinch band is now one conditioning branch inside it.
func RuledCrossingIntersectGeneral(a, b *topo.Body, rec *diag.Recorder) (*topo.Body, bool) {
	return ruledCrossingIntersect(a, b, rec)
}

// ruledCrossingIntersect builds any ruled∩ruled crossing through the GENERAL pipeline: the shared SSI imprint,
// then trimByImprint on EACH side keeping the part inside the OTHER solid, then curvedStitch. The two sides
// emit the SAME imprint polylines, so the welder fuses them watertight (the rod-wall band between the loops +
// the two fat-wall lens caps). One conditioning branch: an unequal cylinder∩cylinder near-pinch (#1818) trims
// the FATTER cylinder's two lens caps per loop, since a single (u,v) arrangement would fuse their tip-to-tip
// necks — gated on the imprint's own geometry (nearPinchLoops), not on type.
func ruledCrossingIntersect(a, b *topo.Body, rec *diag.Recorder) (*topo.Body, bool) {
	loops, ok := ruledCrossingImprint(a, b, rec)
	if !ok || len(loops) == 0 {
		return nil, false
	}
	fa, sa, bandA, okA := curvedSideFace(a)
	fb, sb, bandB, okB := curvedSideFace(b)
	insideA, okMA := curvedSolidMembership(a)
	insideB, okMB := curvedSolidMembership(b)
	if !okA || !okB || !okMA || !okMB {
		return nil, false
	}
	nearPinch := nearPinchLoops(loops)
	keptA, okKA := keptOrNone(ruledCrossingSideSplit(fa, sa, bandA, loops, false, insideB, nearPinch && fatterCylinderSide(sa, sb)))
	keptB, okKB := keptOrNone(ruledCrossingSideSplit(fb, sb, bandB, loops, true, insideA, nearPinch && fatterCylinderSide(sb, sa)))
	if !okKA || !okKB {
		return nil, false
	}
	return curvedStitch(append(keptA, keptB...)), true
}

// ruledCrossingImprint traces the shared crossing loops, applying the Steinmetz snap-ceiling gate for a bare
// cylinder∩cylinder pair (crossingCylinderLoops) — where near-equal radii below the ceiling belong to the
// exact bicylinder constructor — and the plain general imprint (curvedImprintLoops) for any cone-involving
// pair, which has no such degenerate.
func ruledCrossingImprint(a, b *topo.Body, rec *diag.Recorder) ([]geom.Polyline, bool) {
	if !hasConeSide(a) && !hasConeSide(b) {
		return crossingCylinderLoops(a, b, rec)
	}
	return curvedImprintLoops(a, b, rec)
}

// ruledCrossingSideSplit trims one side by the imprint. perLoop routes a near-pinch cylinder's fatter wall
// through cylinderLensSplit (each lens cap in its own arrangement so the tip-to-tip necks do not fuse, #1818);
// every other side takes the one-arrangement curvedSideSolidSplit.
func ruledCrossingSideSplit(f curvedFace, surface geom.Surface, band coneSideBand_, loops []geom.Polyline, isB bool, inside func(math.Point3) bool, perLoop bool) ([]curvedFace, []loopEdge, error) {
	if perLoop {
		return cylinderLensSplit(true, f, surface.(geom.Cylinder), band, loops, isB, inside)
	}
	return curvedSideSolidSplit(f, surface, band, polylineCurves(loops), Intersection, isB, inside)
}

// fatterCylinderSide reports whether side a is a cylinder strictly fatter than cylinder side b — the near-pinch
// gate's "trim this wall per loop" test (only the FATTER wall carries the two tip-to-tip lens caps). False
// whenever either side is not a cylinder, so a cone-involving crossing never takes the per-loop branch.
func fatterCylinderSide(a, b geom.Surface) bool {
	ca, oka := a.(geom.Cylinder)
	cb, okb := b.(geom.Cylinder)
	return oka && okb && ca.Radius > cb.Radius
}

// hasConeSide reports whether a body has a cone/frustum side face — the guard that keeps the cone-crossing
// driver from claiming a cylinder∩cylinder pair the dedicated near-pinch driver must handle.
func hasConeSide(b *topo.Body) bool {
	_, _, _, ok := coneSideFace(b)
	return ok
}

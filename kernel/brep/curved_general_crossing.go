// SPDX-License-Identifier: GPL-2.0-only

package brep

import (
	"fmt"

	"oblikovati.org/kernel/diag"
	"oblikovati.org/kernel/geom"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
)

// Unified ruled-crossing intersect driver (ADR-0058 phase 3). coneConeIntersectGeneral and
// coneCylinderIntersectGeneral had the identical skeleton — imprint → each side's face + solid membership
// → trim each side inside the other → curvedStitch — differing only in which per-primitive side split each
// called. With the imprint (curvedImprintLoops), membership (curvedSolidMembership) and stitch already
// general, the last per-pair piece is the side split; curvedSideFace + curvedSideSolidSplit make it one.
// One driver now builds cone∩cone AND cone∩cylinder (a sphere/torus side folds in when curvedSideFace and
// curvedSideSolidSplit learn it). Cylinder∩cylinder keeps its dedicated near-pinch driver
// (crossingCylinderIntersectGeneral, #1818), so this requires at least one CONE side and defers otherwise.

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

// RuledConeCrossingIntersectGeneral is the exported entry kernel/ops routes cone∩cone and cone∩cylinder
// intersect through: the GENERAL curved∩curved pipeline (ADR-0058 phase 3), no bespoke loop→body
// constructor. ok=false unless at least one operand is a cone crossing the other's cone/cylinder side.
func RuledConeCrossingIntersectGeneral(a, b *topo.Body, rec *diag.Recorder) (*topo.Body, bool) {
	return ruledConeCrossingIntersect(a, b, rec)
}

// ruledConeCrossingIntersect builds cone∩cone or cone∩cylinder through the GENERAL pipeline: the shared SSI
// imprint (curvedImprintLoops), then trimByImprint on EACH side keeping the part inside the OTHER solid,
// then curvedStitch. The two sides emit the SAME imprint polylines, so the welder fuses them watertight (the
// rod-wall band between the loops + the two fat-wall lens caps). It requires at least one cone side, so a
// cylinder∩cylinder pair defers to its dedicated near-pinch driver (crossingCylinderIntersectGeneral).
func ruledConeCrossingIntersect(a, b *topo.Body, rec *diag.Recorder) (*topo.Body, bool) {
	if !hasConeSide(a) && !hasConeSide(b) {
		return nil, false // cylinder∩cylinder is the dedicated near-pinch driver's job (#1818)
	}
	loops, ok := curvedImprintLoops(a, b, rec)
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
	imprint := polylineCurves(loops)
	keptA, okKA := keptOrNone(curvedSideSolidSplit(fa, sa, bandA, imprint, Intersection, false, insideB))
	keptB, okKB := keptOrNone(curvedSideSolidSplit(fb, sb, bandB, imprint, Intersection, true, insideA))
	if !okKA || !okKB {
		return nil, false
	}
	return curvedStitch(append(keptA, keptB...)), true
}

// hasConeSide reports whether a body has a cone/frustum side face — the guard that keeps the cone-crossing
// driver from claiming a cylinder∩cylinder pair the dedicated near-pinch driver must handle.
func hasConeSide(b *topo.Body) bool {
	_, _, _, ok := coneSideFace(b)
	return ok
}

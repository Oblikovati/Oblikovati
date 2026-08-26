// SPDX-License-Identifier: GPL-2.0-only

package brep

import (
	"oblikovati.org/kernel/diag"
	"oblikovati.org/kernel/geom"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
)

// Partial-rim second cut recognizer (EPIC Oblikovati/Oblikovati#1724 follow-up, #1732): a SECOND curved cut on
// a target whose cylinder side was already notched by a first cut. The target resolves as a cutCylinderOperand
// (one surviving full bottom rim + a prior-trim loop); the tool is a bare ruled solid. The operand's split
// composes the prior boundary and runs the disjoint gate, so this reuses the general ruled cut (ruledCutGeneral
// → cutFaces → curvedStitch) unchanged — only the target's split strategy, imprint window and membership oracle
// differ. Anything outside the disjoint sub-family declines, and kernel/ops keeps its observable CSG fallback.
//
// The already-cut target is NOT a bare cylinder solid (three planar caps, not two), so the bare-solid helpers
// cylinderSolidParams / curvedSolidMembership reject it. Two cut-aware replacements bridge that: partialRimImprint
// traces the SSI from the cut cylinder's surface + recovered band directly, and cutCylinderSolidMembership tests
// membership against the cylinder radius AND every planar cap's inward half-space.

// cutCylinderSideFace finds a body's ALREADY-CUT cylinder side (cutCylinderSideBand) — the mirror of
// cylinderSideFace, which finds a bare two-rim side.
func cutCylinderSideFace(b *topo.Body) (curvedFace, geom.Cylinder, coneSideBand_, priorTrimLoop, bool) {
	for _, f := range facesOfAny(b) {
		if _, isCyl := f.surface.(geom.Cylinder); !isCyl {
			continue
		}
		if cyl, band, prior, ok := cutCylinderSideBand(f); ok {
			return f, cyl, band, prior, true
		}
	}
	return curvedFace{}, geom.Cylinder{}, coneSideBand_{}, priorTrimLoop{}, false
}

// cutCylinderSolidMembership builds a point-inside oracle for the already-cut cylinder solid: inside the
// cylinder radius AND on the inner side of every planar cap. Each cap's inward normal is oriented by a known
// deep-interior point (the axis at half the recovered band height), which is inside the convex notched solid.
// This replaces curvedSolidMembership (which recognises only a bare two-cap cylinder) for the cut target (#1732).
func cutCylinderSolidMembership(b *topo.Body, cyl geom.Cylinder, band coneSideBand_) func(math.Point3) bool {
	axis := cyl.AxisDir.AsVector()
	interior := band.bottom.TranslateBy(axis.Scale(math.Scalar(band.vMax / 2)))
	caps := inwardCapHalfSpaces(b, interior)
	margin := geom.ResolutionForSize(cyl.Radius + band.vMax).Plane()
	return func(p math.Point3) bool {
		d := cyl.Origin.VectorTo(p)
		v := d.Dot(axis)
		if float64(d.Sub(axis.Scale(v)).Length()) >= cyl.Radius-margin {
			return false
		}
		for _, h := range caps {
			if float64(h.origin.VectorTo(p).Dot(h.normal)) < margin {
				return false
			}
		}
		return true
	}
}

// capHalfSpace is one planar cap oriented so its normal points INTO the solid.
type capHalfSpace struct {
	origin math.Point3
	normal math.Vector3
}

// inwardCapHalfSpaces collects the body's planar caps, each normal flipped to point toward the interior sample.
func inwardCapHalfSpaces(b *topo.Body, interior math.Point3) []capHalfSpace {
	var out []capHalfSpace
	for _, f := range facesOfAny(b) {
		pl, ok := f.surface.(geom.Plane)
		if !ok {
			continue
		}
		n := pl.UAxis.AsVector().Cross(pl.VAxis.AsVector())
		if float64(pl.Origin.VectorTo(interior).Dot(n)) < 0 {
			n = n.Scale(-1)
		}
		out = append(out, capHalfSpace{origin: pl.Origin, normal: n})
	}
	return out
}

// partialRimImprint traces the second cut's SSI on the already-cut target's cylinder surface, using the
// recovered band as the axial window — bypassing cylinderSolidParams, which rejects the three-cap notched body.
func partialRimImprint(target, tool *topo.Body, rec *diag.Recorder) ([]geom.Polyline, bool) {
	_, tgtCyl, band, _, okT := cutCylinderSideFace(target)
	toolCyl, _, _, okB := cylinderSolidParams(facesOfAny(tool))
	if !okT || !okB {
		return nil, false
	}
	res := geom.ResolutionForBox(target.RangeBox().Union(tool.RangeBox()))
	window := cylinderTraceWindow(tgtCyl, band.bottom, band.vMax)
	loops := imprintTraceLoops(tgtCyl, toolCyl, window, res, rec)
	if len(loops) == 0 {
		return nil, false
	}
	return loops, true
}

// cutCylinderOperand resolves an already-cut cylinder body into a ruledOperand whose split composes the prior
// boundary and gates on disjointness (#1732). Its newUV builds the BARE frame the cap-clearance check consumes;
// the prior loop and the disjoint gate live in splitCut. ok=false when the body has no recognisable cut
// cylinder side (e.g. a bare two-rim side — cylinderOperand's job).
func cutCylinderOperand(b *topo.Body) (ruledOperand, bool) {
	f, cyl, band, prior, ok := cutCylinderSideFace(b)
	if !ok {
		return ruledOperand{}, false
	}
	inside := cutCylinderSolidMembership(b, cyl, band)
	o := ruledOperand{body: b, face: f, surface: cyl, inside: inside,
		newUV: func(op Op, isB bool, other func(math.Point3) bool) ruledUV {
			return newCylinderUVSolid(cyl, band, op, isB, other)
		},
		splitCut: func(imprint []geom.Curve3, op Op, isB bool, other func(math.Point3) bool) ([]curvedFace, bool) {
			c := newCutCylinderUVSolid(cyl, band, prior, op, isB, other)
			c.placeSeams(imprint)
			if !c.disjointFromPrior(imprint) {
				return nil, false // outside the disjoint sub-family → decline to CSG
			}
			return keptOrNone(trimByImprint(&c, f, cyl, imprint, cutCylinderMaterial(&c)))
		}}
	return o, true
}

// PartialRimCutGeneral builds target − tool where the target's cylinder side was already notched by a first cut
// (#1732). It resolves the target as a cut operand and the tool as a bare ruled operand, then reuses the general
// ruled cut. ok=false unless the imprint is a clean two-loop crossing, the target is a recognisable cut
// cylinder, and the tool a bare ruled solid — so kernel/ops keeps its CSG fallback everywhere else.
func PartialRimCutGeneral(target, tool *topo.Body, rec *diag.Recorder) (*topo.Body, bool) {
	loops, ok := require2Loops(partialRimImprint(target, tool, rec))
	tgt, okT := cutCylinderOperand(target)
	tl, okL := ruledOperandOf(tool)
	if !ok || !okT || !okL {
		return nil, false
	}
	return ruledCutGeneral(tgt, tl, loops)
}

// SPDX-License-Identifier: GPL-2.0-only

package brep

import (
	"oblikovati.org/kernel/diag"
	"oblikovati.org/kernel/geom"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
)

// General curved CUT and JOIN for two crossing RULED solids — cylinders and cones alike (EPIC #1403). The
// intersect drivers (curved_general_boolean.go) weld only the two trimmed walls; a cut or a join also carries
// PLANAR CAPS and, for a cut, REVERSES the tool wall into the cavity. Every crossing ruled pair (cyl∩cyl,
// cone∩cone, cone∩cyl) runs the SAME two recipes here — only the per-operand side-face / membership / UV-frame
// builders differ, captured behind one ruledOperand so the cut and the join are written once. The OUTSIDE-keep
// walls mesh through the wrapping-band emission (curved_general_wrapping_band.go, Oblikovati#1476).
//
// SCOPE: a CLEAN crossing whose imprint is a set of NON-self-intersecting loops (a thin rod or cone through a
// fatter solid, fully or partially). The one ruled crossing this pipeline does NOT cover is the equal-radius
// Steinmetz bicylinder, whose two ellipses pinch into a self-intersecting imprint the (u,v) arrangement cannot
// split into lobes — it keeps its bespoke analytic constructor (see curved_steinmetz.go). An out-of-scope pair
// declines from these drivers (validBooleanSolid rejects the result), so kernel/ops uses the bespoke handler.

// ruledOperand bundles everything a general cut/join needs about ONE crossing ruled solid: its body, the
// curved side face being trimmed, that face's analytic surface, a 3D solid-membership oracle, and a builder
// that frames the side as a (u,v) solid for a given operation. A cylinder and a (possibly truncated) cone are
// both ruled sides, so both fold into this one shape and the cut/join drivers never branch on which (#1403).
type ruledOperand struct {
	body    *topo.Body
	face    curvedFace
	surface geom.Surface
	inside  func(math.Point3) bool
	// newUV frames this operand's side as a (u,v) solid whose kept cells are decided by `other` under op
	// (isB marks this operand as the boolean's B). It builds the same ruledUV the split and the cap-clearance
	// check consume, so the cone/cylinder distinction lives only here.
	newUV func(op Op, isB bool, other func(math.Point3) bool) ruledUV
}

// cylinderOperand resolves a bare cylinder body into a ruledOperand, or ok=false when it has no cylinder side
// face with two full-circle rims or no membership oracle (#1403).
func cylinderOperand(b *topo.Body) (ruledOperand, bool) {
	f, cyl, band, ok := cylinderSideFace(b)
	inside, okM := curvedSolidMembership(b)
	if !ok || !okM {
		return ruledOperand{}, false
	}
	return ruledOperand{body: b, face: f, surface: cyl, inside: inside,
		newUV: func(op Op, isB bool, other func(math.Point3) bool) ruledUV {
			return newCylinderUVSolid(cyl, band, op, isB, other)
		}}, true
}

// coneOperand resolves a frustum body into a ruledOperand, or ok=false when it has no cone side face bounded
// by two full-circle rims or no membership oracle (#1403).
func coneOperand(b *topo.Body) (ruledOperand, bool) {
	f, cone, band, ok := coneSideFace(b)
	inside, okM := curvedSolidMembership(b)
	if !ok || !okM {
		return ruledOperand{}, false
	}
	return ruledOperand{body: b, face: f, surface: cone, inside: inside,
		newUV: func(op Op, isB bool, other func(math.Point3) bool) ruledUV {
			return newConeUVSolid(cone, band, op, isB, other)
		}}, true
}

// ruledOperandOf resolves a body as whichever ruled side it carries — a cone frustum first, else a cylinder —
// so a mixed cone-through-cylinder pair builds each operand without the caller knowing which is which (#1403).
func ruledOperandOf(b *topo.Body) (ruledOperand, bool) {
	if o, ok := coneOperand(b); ok {
		return o, true
	}
	return cylinderOperand(b)
}

// split trims this operand's side by the shared imprint, keeping the cells `other` selects under op, and
// returns the kept curved faces (ok=false when the split fails or keeps nothing). The cone/cylinder UV frame
// is built by newUV; the predicate binds the seam-shifted receiver via ruledSolidMaterial (#1403).
func (o ruledOperand) split(imprint []geom.Curve3, op Op, isB bool, other func(math.Point3) bool) ([]curvedFace, bool) {
	c := o.newUV(op, isB, other)
	return keptOrNone(trimByImprint(&c, o.face, o.surface, imprint, ruledSolidMaterial(&c)))
}

// require2Loops gates a cut/join on the imprint being exactly two closed loops — a clean rod-through-fat
// crossing (entry + exit). Anything else (a tangency, a partial penetration, an open chain) declines so the
// caller keeps its CSG fallback (#1403).
func require2Loops(loops []geom.Polyline, ok bool) ([]geom.Polyline, bool) {
	return loops, ok && len(loops) == 2
}

// ruledCutGeneral builds target − tool for two crossing ruled solids (#1403): the target wall kept OUTSIDE
// the tool (the breached wall) + the target's whole caps + the tool wall kept INSIDE the target and reversed
// into the cavity (the tunnel). One connected solid when the target is drilled (fat − rod); a two-shell solid
// when the target is the rod (rod − fat: a stub each side). ok=false unless the breach lies strictly between
// the target's caps (a cap-reaching breach defers to the bespoke handler).
func ruledCutGeneral(target, tool ruledOperand, loops []geom.Polyline) (*topo.Body, bool) {
	if !loopsClearOfCaps(target.newUV(Difference, false, tool.inside), loops) {
		return nil, false
	}
	imprint := polylineCurves(loops)
	keptT, okT := target.split(imprint, Difference, false, tool.inside)
	keptL, okL := tool.split(imprint, Difference, true, target.inside)
	if !okT || !okL {
		return nil, false
	}
	return curvedStitch(cutFaces(target, keptT, tool, keptL)), true
}

// ruledJoinGeneral builds a ∪ b for two crossing ruled solids (#1403): each wall kept OUTSIDE the other (the
// fat's holed wall + the two protruding stubs) plus BOTH bodies' whole caps. The cut's mirror — no face
// reversal (a union keeps every wall facing outward) and the tool's caps survive too. ok=false unless BOTH
// breaches lie strictly between that body's caps (so every cap stays whole).
func ruledJoinGeneral(a, b ruledOperand, loops []geom.Polyline) (*topo.Body, bool) {
	if !loopsClearOfCaps(a.newUV(Union, false, b.inside), loops) ||
		!loopsClearOfCaps(b.newUV(Union, true, a.inside), loops) {
		return nil, false
	}
	imprint := polylineCurves(loops)
	keptA, okA := a.split(imprint, Union, false, b.inside)
	keptB, okB := b.split(imprint, Union, true, a.inside)
	if !okA || !okB {
		return nil, false
	}
	return curvedStitch(joinFaces(a, keptA, b, keptB)), true
}

// cutFaces assembles a Difference result's boundary: the target's breached wall (kept outside the tool), the
// target's caps that stay OUTSIDE the tool (whole), the tool's wall inside the target reversed into the cavity
// (the tunnel/cut wall), and any tool cap that lies INSIDE the target reversed into the cavity (a blind-hole
// pocket bottom). For a clean side-breach crossing both target caps are whole and no tool cap is inside, so it
// reduces to the original behaviour; a partial penetration (the tool ending inside) keeps its blind cap (#1403).
func cutFaces(target ruledOperand, targetWall []curvedFace, tool ruledOperand, toolWall []curvedFace) []curvedFace {
	faces := make([]curvedFace, 0, len(targetWall)+len(toolWall)+4)
	faces = append(faces, targetWall...)
	faces = append(faces, capsOutside(target.body, tool.inside)...)
	faces = append(faces, reverseCurvedFaces(toolWall)...)
	faces = append(faces, reverseCurvedFaces(capsInside(tool.body, target.inside))...)
	return faces
}

// joinFaces assembles a Union result's boundary: each operand's wall kept outside the other (the fat's holed
// wall + the rod's protruding stubs) plus each operand's caps that lie OUTSIDE the other solid. Unlike the cut
// neither wall is reversed. For a clean crossing every cap is outside (so both bodies contribute all caps); a
// partial penetration drops the tool's blind cap (inside the other) and keeps its entry cap (#1403).
func joinFaces(a ruledOperand, wallA []curvedFace, b ruledOperand, wallB []curvedFace) []curvedFace {
	faces := make([]curvedFace, 0, len(wallA)+len(wallB)+4)
	faces = append(faces, wallA...)
	faces = append(faces, capsOutside(a.body, b.inside)...)
	faces = append(faces, wallB...)
	faces = append(faces, capsOutside(b.body, a.inside)...)
	return faces
}

// capsOutside returns a body's planar caps whose centre lies OUTSIDE the other solid — the caps a union/cut
// keeps whole (#1403). capsInside returns those whose centre lies inside (a partial penetration's blind cap).
func capsOutside(b *topo.Body, otherInside func(math.Point3) bool) []curvedFace {
	return filterCaps(b, func(c math.Point3) bool { return !otherInside(c) })
}

func capsInside(b *topo.Body, otherInside func(math.Point3) bool) []curvedFace {
	return filterCaps(b, otherInside)
}

// filterCaps keeps the planar caps whose plane origin (the cap's centre) satisfies keep. A crossing-pair cap is
// cleanly inside or outside the other solid, so the centre decides the whole cap (#1403).
func filterCaps(b *topo.Body, keep func(math.Point3) bool) []curvedFace {
	var caps []curvedFace
	for _, f := range planarCapFaces(b) {
		if pl, ok := f.surface.(geom.Plane); ok && keep(pl.Origin) {
			caps = append(caps, f)
		}
	}
	return caps
}

// ruledPairGeneral is the shared body of the full-crossing ruled cut/join exports below: trace the
// imprint as two loops (the entry + exit of a clean side breach), resolve both operands to their ruled
// type, and combine. The three injected functions are all that differ between the pairs — the imprint
// tracer, the operand resolver, and ruledCutGeneral vs ruledJoinGeneral (#1502, collapsing six
// near-identical wrappers). ok=false when the imprint is not a clean two-loop crossing or an operand is
// not the expected ruled surface, so kernel/ops keeps its CSG fallback.
func ruledPairGeneral(a, b *topo.Body, rec *diag.Recorder,
	imprint func(*topo.Body, *topo.Body, *diag.Recorder) ([]geom.Polyline, bool),
	operand func(*topo.Body) (ruledOperand, bool),
	combine func(target, tool ruledOperand, loops []geom.Polyline) (*topo.Body, bool)) (*topo.Body, bool) {
	loops, ok := require2Loops(imprint(a, b, rec))
	oa, okA := operand(a)
	ob, okB := operand(b)
	if !ok || !okA || !okB {
		return nil, false
	}
	return combine(oa, ob, loops)
}

// CrossingCylinderCutGeneral routes crossing-cylinder subtract through the general ruled cut (#1403/#1476):
// a fat cylinder drilled by a rod (one solid) or a rod sliced by a fat (two stubs). ok=false outside the
// wired clean-side-breach crossing so kernel/ops keeps its CSG fallback.
func CrossingCylinderCutGeneral(target, tool *topo.Body, rec *diag.Recorder) (*topo.Body, bool) {
	return ruledPairGeneral(target, tool, rec, crossingCylinderImprint, cylinderOperand, ruledCutGeneral)
}

// CrossingCylinderJoinGeneral routes crossing-cylinder JOIN through the general ruled join (#1403/#1476):
// a fat cylinder side-breached by a rod, welded into one solid (keyhole holed wall + two stubs + whole caps).
func CrossingCylinderJoinGeneral(a, b *topo.Body, rec *diag.Recorder) (*topo.Body, bool) {
	return ruledPairGeneral(a, b, rec, crossingCylinderImprint, cylinderOperand, ruledJoinGeneral)
}

// ConeConeCutGeneral routes cone∩cone subtract through the general ruled cut (#1403): a fat frustum drilled by
// a crossing rod frustum (one solid) or a rod frustum sliced by a fat (two tapered stubs). ok=false outside
// the wired clean-side-breach frustum crossing.
func ConeConeCutGeneral(target, tool *topo.Body, rec *diag.Recorder) (*topo.Body, bool) {
	return ruledPairGeneral(target, tool, rec, coneConeImprint, coneOperand, ruledCutGeneral)
}

// ConeConeJoinGeneral routes cone∩cone JOIN through the general ruled join (#1403): a fat frustum side-breached
// by a crossing rod frustum, welded into one solid (holed fat wall + a tapered stub each side + whole caps).
func ConeConeJoinGeneral(a, b *topo.Body, rec *diag.Recorder) (*topo.Body, bool) {
	return ruledPairGeneral(a, b, rec, coneConeImprint, coneOperand, ruledJoinGeneral)
}

// ConeCylinderCutGeneral routes cone∩cylinder subtract through the general ruled cut (#1403): a fat cylinder
// drilled by a crossing cone (or vice-versa). The operands resolve by type (ruledOperandOf), so target/tool
// may be cone-then-cylinder or the reverse. ok=false outside the wired clean-side-breach crossing.
func ConeCylinderCutGeneral(target, tool *topo.Body, rec *diag.Recorder) (*topo.Body, bool) {
	return ruledPairGeneral(target, tool, rec, coneCylinderImprint, ruledOperandOf, ruledCutGeneral)
}

// ConeCylinderJoinGeneral routes cone∩cylinder JOIN through the general ruled join (#1403): a fat cylinder
// side-breached by a crossing cone (or vice-versa), welded into one solid. ok=false outside the wired
// clean-side-breach crossing.
func ConeCylinderJoinGeneral(a, b *topo.Body, rec *diag.Recorder) (*topo.Body, bool) {
	return ruledPairGeneral(a, b, rec, coneCylinderImprint, ruledOperandOf, ruledJoinGeneral)
}

// partialImprint returns the SINGLE imprint loop of a partial penetration — a thin rod that breaches one wall
// of a fatter solid and ENDS inside it. The rod's short extent clips the would-be exit loop, so the imprint
// must be traced ROD-first (its window leaves one loop); tracing fat-first windows the full crossing and
// returns two loops. Which body is the rod is unknown, so both orderings are tried, accepting the one that
// yields exactly one loop. ok=false for a full crossing (two loops either way) or no intersection (#1403).
func partialImprint(a, b *topo.Body, rec *diag.Recorder) ([]geom.Polyline, bool) {
	if l, ok := crossingCylinderImprint(a, b, rec); ok && len(l) == 1 {
		return l, true
	}
	if l, ok := crossingCylinderImprint(b, a, rec); ok && len(l) == 1 {
		return l, true
	}
	return nil, false
}

// PartialPenetrationCutGeneral routes a partial-penetration subtract through the general ruled cut (#1403):
// fat − rod is a blind hole (the holed fat wall + the rod tunnel reversed + the rod's blind cap as the pocket
// bottom); rod − fat is the single stub lump. The cap generalisation in cutFaces keeps the tool's blind cap
// (inside the target) reversed and drops its entry cap. ok=false unless the pair is a clean single-breach
// partial penetration.
func PartialPenetrationCutGeneral(target, tool *topo.Body, rec *diag.Recorder) (*topo.Body, bool) {
	loops, ok := partialImprint(target, tool, rec)
	tgt, okT := ruledOperandOf(target)
	tl, okL := ruledOperandOf(tool)
	if !ok || !okT || !okL {
		return nil, false
	}
	return ruledCutGeneral(tgt, tl, loops)
}

// PartialPenetrationJoinGeneral routes a partial-penetration JOIN through the general ruled join (#1403): the
// fat with a single rod stub sticking out the entry side (the holed fat wall + both fat caps + the rod stub
// from its entry cap to the lens + the rod's entry cap). cutFaces' sibling joinFaces drops the rod's blind cap
// (inside the fat) via the cap generalisation. ok=false unless the pair is a clean single-breach penetration.
func PartialPenetrationJoinGeneral(a, b *topo.Body, rec *diag.Recorder) (*topo.Body, bool) {
	loops, ok := partialImprint(a, b, rec)
	oa, okA := ruledOperandOf(a)
	ob, okB := ruledOperandOf(b)
	if !ok || !okA || !okB {
		return nil, false
	}
	return ruledJoinGeneral(oa, ob, loops)
}

// PartialPenetrationIntersectGeneral routes a partial-penetration intersect (the rod plug inside the fat)
// through the general pipeline (#1403): the fat-wall lens cap + the rod-wall band from the lens to the rod's
// blind end + the rod's blind end cap (the planar disc inside the fat). Unlike a full crossing's intersect
// (two curved lens caps, no planar cap) the plug carries the rod's interior-ending PLANAR cap, added by
// capsInside. ok=false unless the pair is a clean single-breach penetration.
func PartialPenetrationIntersectGeneral(a, b *topo.Body, rec *diag.Recorder) (*topo.Body, bool) {
	loops, ok := partialImprint(a, b, rec)
	oa, okA := ruledOperandOf(a)
	ob, okB := ruledOperandOf(b)
	if !ok || !okA || !okB {
		return nil, false
	}
	imprint := polylineCurves(loops)
	keptA, okKA := oa.split(imprint, Intersection, false, ob.inside)
	keptB, okKB := ob.split(imprint, Intersection, true, oa.inside)
	if !okKA || !okKB {
		return nil, false
	}
	faces := append([]curvedFace{}, keptA...)
	faces = append(faces, keptB...)
	faces = append(faces, capsInside(a, ob.inside)...) // the rod's blind end cap (inside the other solid)
	faces = append(faces, capsInside(b, oa.inside)...)
	return curvedStitch(faces), true
}

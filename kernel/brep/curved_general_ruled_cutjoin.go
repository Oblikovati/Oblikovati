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
// caller keeps its bespoke fallback (#1403).
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
	return curvedStitch(cutFaces(keptT, target.body, keptL)), true
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
	return curvedStitch(joinFaces(keptA, a.body, keptB, b.body)), true
}

// cutFaces assembles a Difference result's boundary: the target's breached wall (kept outside the tool), the
// target's whole caps (clean side-breach), and the tool's wall inside the target reversed into the cavity
// (the tunnel/cut wall) — #1403.
func cutFaces(targetWall []curvedFace, target *topo.Body, toolWall []curvedFace) []curvedFace {
	faces := make([]curvedFace, 0, len(targetWall)+len(toolWall)+2)
	faces = append(faces, targetWall...)
	faces = append(faces, planarCapFaces(target)...)
	faces = append(faces, reverseCurvedFaces(toolWall)...)
	return faces
}

// joinFaces assembles a Union result's boundary: each operand's wall kept outside the other (the fat's holed
// wall + the rod's two protruding stubs) plus BOTH operands' whole caps. Unlike the cut neither wall is
// reversed — a union keeps every kept wall facing outward — and both bodies contribute their caps (#1403).
func joinFaces(wallA []curvedFace, a *topo.Body, wallB []curvedFace, b *topo.Body) []curvedFace {
	faces := make([]curvedFace, 0, len(wallA)+len(wallB)+4)
	faces = append(faces, wallA...)
	faces = append(faces, planarCapFaces(a)...)
	faces = append(faces, wallB...)
	faces = append(faces, planarCapFaces(b)...)
	return faces
}

// CrossingCylinderCutGeneral routes crossing-cylinder subtract through the general ruled cut (#1403/#1476):
// a fat cylinder drilled by a rod (one solid) or a rod sliced by a fat (two stubs). ok=false outside the
// wired clean-side-breach crossing so kernel/ops keeps its bespoke fallback.
func CrossingCylinderCutGeneral(target, tool *topo.Body, rec *diag.Recorder) (*topo.Body, bool) {
	loops, ok := require2Loops(crossingCylinderImprint(target, tool, rec))
	tgt, okT := cylinderOperand(target)
	tl, okL := cylinderOperand(tool)
	if !ok || !okT || !okL {
		return nil, false
	}
	return ruledCutGeneral(tgt, tl, loops)
}

// CrossingCylinderJoinGeneral routes crossing-cylinder JOIN through the general ruled join (#1403/#1476):
// a fat cylinder side-breached by a rod, welded into one solid (keyhole holed wall + two stubs + whole caps).
func CrossingCylinderJoinGeneral(a, b *topo.Body, rec *diag.Recorder) (*topo.Body, bool) {
	loops, ok := require2Loops(crossingCylinderImprint(a, b, rec))
	oa, okA := cylinderOperand(a)
	ob, okB := cylinderOperand(b)
	if !ok || !okA || !okB {
		return nil, false
	}
	return ruledJoinGeneral(oa, ob, loops)
}

// ConeConeCutGeneral routes cone∩cone subtract through the general ruled cut (#1403): a fat frustum drilled by
// a crossing rod frustum (one solid) or a rod frustum sliced by a fat (two tapered stubs). ok=false outside
// the wired clean-side-breach frustum crossing.
func ConeConeCutGeneral(target, tool *topo.Body, rec *diag.Recorder) (*topo.Body, bool) {
	loops, ok := require2Loops(coneConeImprint(target, tool, rec))
	tgt, okT := coneOperand(target)
	tl, okL := coneOperand(tool)
	if !ok || !okT || !okL {
		return nil, false
	}
	return ruledCutGeneral(tgt, tl, loops)
}

// ConeConeJoinGeneral routes cone∩cone JOIN through the general ruled join (#1403): a fat frustum side-breached
// by a crossing rod frustum, welded into one solid (holed fat wall + a tapered stub each side + whole caps).
func ConeConeJoinGeneral(a, b *topo.Body, rec *diag.Recorder) (*topo.Body, bool) {
	loops, ok := require2Loops(coneConeImprint(a, b, rec))
	oa, okA := coneOperand(a)
	ob, okB := coneOperand(b)
	if !ok || !okA || !okB {
		return nil, false
	}
	return ruledJoinGeneral(oa, ob, loops)
}

// ConeCylinderCutGeneral routes cone∩cylinder subtract through the general ruled cut (#1403): a fat cylinder
// drilled by a crossing cone (or vice-versa). The operands resolve by type (ruledOperandOf), so target/tool
// may be cone-then-cylinder or the reverse. ok=false outside the wired clean-side-breach crossing.
func ConeCylinderCutGeneral(target, tool *topo.Body, rec *diag.Recorder) (*topo.Body, bool) {
	loops, ok := require2Loops(coneCylinderImprint(target, tool, rec))
	tgt, okT := ruledOperandOf(target)
	tl, okL := ruledOperandOf(tool)
	if !ok || !okT || !okL {
		return nil, false
	}
	return ruledCutGeneral(tgt, tl, loops)
}

// ConeCylinderJoinGeneral routes cone∩cylinder JOIN through the general ruled join (#1403): a fat cylinder
// side-breached by a crossing cone (or vice-versa), welded into one solid. ok=false outside the wired
// clean-side-breach crossing.
func ConeCylinderJoinGeneral(a, b *topo.Body, rec *diag.Recorder) (*topo.Body, bool) {
	loops, ok := require2Loops(coneCylinderImprint(a, b, rec))
	oa, okA := ruledOperandOf(a)
	ob, okB := ruledOperandOf(b)
	if !ok || !okA || !okB {
		return nil, false
	}
	return ruledJoinGeneral(oa, ob, loops)
}

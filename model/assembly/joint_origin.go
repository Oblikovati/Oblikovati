// SPDX-License-Identifier: GPL-2.0-only

package assembly

import (
	"oblikovati.org/api/types"
	"oblikovati.org/math"
)

// Joint origin definition (#1973): HOW each joint origin's frame is positioned on its component —
// inferred from the picked geometry, offset from it by X/Y in the frame's plane, or projected to the
// midplane between two referenced faces. The definition is applied to the origin primitive in the
// component's local space (before the occurrence placement), so it re-solves associatively.

// originDef positions one joint origin's frame.
type originDef struct {
	mode         types.AssemblyJointOriginMode
	dx, dy       float64
	faceA, faceB Primitive
	hasFaces     bool
}

// apply returns the primitive repositioned per the origin definition. Offset shifts the point by
// dx/dy along the frame's in-plane axes; between-two-faces moves it to the midpoint of the two faces.
func (od originDef) apply(prim Primitive) Primitive {
	switch od.mode {
	case types.JointOriginOffset:
		u, v := tangentFrame(prim.dir.AsVector())
		prim.point = prim.point.TranslateBy(u.Scale(math.Scalar(od.dx))).TranslateBy(v.Scale(math.Scalar(od.dy)))
	case types.JointOriginBetweenTwoFaces:
		if od.hasFaces {
			prim.point = od.faceA.point.Midpoint(od.faceB.point)
		}
	}
	return prim
}

// effectiveA and effectiveB return the joint origins' primitives with their definitions applied — the
// frames the solver actually uses (#1973).
func (j *jointBase) effectiveA() Primitive { return j.aOrigin.apply(j.a.prim) }
func (j *jointBase) effectiveB() Primitive { return j.bOrigin.apply(j.b.prim) }

// SetOriginOneAsInfer / SetOriginTwoAsInfer clear an origin's definition back to inferred (#1973).
func (j *jointBase) SetOriginOneAsInfer() { j.aOrigin = originDef{} }
func (j *jointBase) SetOriginTwoAsInfer() { j.bOrigin = originDef{} }

// SetOriginOneAsOffset / SetOriginTwoAsOffset offset an origin frame by dx/dy in its plane (#1973).
func (j *jointBase) SetOriginOneAsOffset(dx, dy float64) {
	j.aOrigin = originDef{mode: types.JointOriginOffset, dx: dx, dy: dy}
}

func (j *jointBase) SetOriginTwoAsOffset(dx, dy float64) {
	j.bOrigin = originDef{mode: types.JointOriginOffset, dx: dx, dy: dy}
}

// SetOriginOneAsBetweenTwoFaces / ...Two project an origin to the midplane between two faces (#1973).
func (j *jointBase) SetOriginOneAsBetweenTwoFaces(a, b Ref) {
	j.aOrigin = originDef{mode: types.JointOriginBetweenTwoFaces, faceA: a.Primitive, faceB: b.Primitive, hasFaces: true}
}

func (j *jointBase) SetOriginTwoAsBetweenTwoFaces(a, b Ref) {
	j.bOrigin = originDef{mode: types.JointOriginBetweenTwoFaces, faceA: a.Primitive, faceB: b.Primitive, hasFaces: true}
}

// OriginModes returns how each of the two joint origins is positioned (#1973).
func (j *jointBase) OriginModes() (a, b types.AssemblyJointOriginMode) {
	return j.aOrigin.mode, j.bOrigin.mode
}

// OriginOffsets returns the two origins' X/Y offsets (0 for a non-offset origin) (#1973).
func (j *jointBase) OriginOffsets() (ax, ay, bx, by float64) {
	return j.aOrigin.dx, j.aOrigin.dy, j.bOrigin.dx, j.bOrigin.dy
}

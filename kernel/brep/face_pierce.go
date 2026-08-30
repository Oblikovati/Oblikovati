// SPDX-License-Identifier: GPL-2.0-only

package brep

import (
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
)

// PointInFaceTrim reports whether p — a point already ON the face's surface, such as a ray's
// analytic pierce point or the foot of a perpendicular projection — lies within the face's
// trimmed region: inside its outer loop and outside every hole. It is the exact B-rep trim
// classification (the parameter-space even-odd test the curved boolean uses, OCCT
// BRepClass_FaceClassifier), NOT a read of any face tessellation — the primitive a
// pick → reference-key resolution needs to confirm an analytic surface hit actually lands on
// THIS face rather than some other trimmed region of the same surface (M48/C3,
// Oblikovati/Oblikovati#3470, #3471).
//
// The caller MUST supply a point that lies on f.Geometry(); PointInFaceTrim classifies the
// point's surface parameters, so an off-surface point is projected by ParamAt and may misreport.
//
// Example:
//
//	if brep.PointInFaceTrim(f, hit.Point) { /* the ray truly pierced face f */ }
func PointInFaceTrim(f *topo.Face, p math.Point3) bool {
	return pointInTrimUV(curvedFaceOf(f), p)
}

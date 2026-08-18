// SPDX-License-Identifier: GPL-2.0-only

package drawing

import (
	"fmt"
	stdmath "math"
	"strconv"

	"oblikovati.org/api/types"
	"oblikovati.org/kernel/geom"
	"oblikovati.org/kernel/hlr"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
)

// Chamfer notes (#1995). A chamfer note is a feature note reading "<d> × <angle>°" — the chamfer's
// setback distance and its angle from a reference face. Both are DERIVED from the model: the chamfer
// face is planar, bounded by the two picked edges (where it meets each adjacent face); the angle is
// the dihedral between the chamfer face and edgeA's other (reference) face, and the distance is the
// chamfer face width projected onto that reference face (w·cos angle). So the note re-resolves with
// the model, like every other feature note.

// featureNoteOffsetMM is how far a feature-note callout text sits from the feature (up-right).
const featureNoteOffsetMM = 16.0

// AddChamferNote adds a chamfer note on the named base view from the chamfer's two edge reference
// keys. edgeA's non-chamfer face is the reference the angle is measured from. It errors when the
// edges do not bound a straight (planar) chamfer.
func (as *DrawingAnnotations) AddChamferNote(name, viewName string, edgeA, edgeB []byte) (*DrawingAnnotation, error) {
	if _, _, _, err := as.annotationBasis(viewName); err != nil {
		return nil, err
	}
	a := &DrawingAnnotation{
		name: as.uniqueName(name), kind: types.ChamferNoteAnnotation, viewName: viewName,
		edgeKey: edgeA, edgeKeyB: edgeB,
	}
	as.recomputeChamferNote(a)
	if len(a.labels) == 0 {
		return nil, fmt.Errorf("drawing: edges do not bound a straight chamfer in view %q", viewName)
	}
	as.items = append(as.items, a)
	return a, nil
}

// recomputeChamferNote re-derives the chamfer callout from the two edges; with no resolvable straight
// chamfer it clears the glyph.
func (as *DrawingAnnotations) recomputeChamferNote(a *DrawingAnnotation) {
	a.curves, a.labels = nil, nil
	view, body, basis, err := as.annotationBasis(a.viewName)
	if err != nil {
		return
	}
	distCm, angleDeg, anchor, ok := chamferMetricsFromBody(body, a.edgeKey, a.edgeKeyB)
	if !ok {
		return
	}
	text := fmt.Sprintf("%s × %s°", holeCoord(distCm*cmToMM), trimAngleDeg(angleDeg))
	p := view.place(hlr.ProjectPoint(basis, anchor))
	a.curves, a.labels = featureNoteCallout(float64(p.X), float64(p.Y), text)
}

// chamferMetricsFromBody resolves the two chamfer edges, the chamfer face and the reference face, and
// returns the setback distance (cm), the chamfer angle (deg) and the chamfer's 3D midpoint anchor.
func chamferMetricsFromBody(body *topo.Body, keyA, keyB []byte) (distCm, angleDeg float64, anchor math.Point3, ok bool) {
	eA, okA := body.FindEdgeByKey(keyA)
	eB, okB := body.FindEdgeByKey(keyB)
	if !okA || !okB {
		return
	}
	lineA, okLA := eA.Geometry().(geom.LineSegment)
	lineB, okLB := eB.Geometry().(geom.LineSegment)
	if !okLA || !okLB {
		return
	}
	chamfer, ref, ok := chamferAndReferenceFace(eA, eB)
	if !ok {
		return
	}
	cPlane, okC := chamfer.Geometry().(geom.Plane)
	rPlane, okR := ref.Geometry().(geom.Plane)
	if !okC || !okR {
		return 0, 0, math.Point3{}, false
	}
	angleDeg = acuteAngleDeg(cPlane.Normal(), rPlane.Normal())
	w := parallelSegmentGap(lineA, lineB)
	distCm = w * stdmath.Cos(angleDeg*stdmath.Pi/180)
	return distCm, angleDeg, midOfSegments(lineA, lineB), true
}

// chamferAndReferenceFace returns the face the two edges share (the chamfer face) and the other face
// of edgeA (the reference the angle is measured from), or ok=false when they share no single face.
func chamferAndReferenceFace(eA, eB *topo.Edge) (chamfer, ref *topo.Face, ok bool) {
	chamfer = commonFace(eA.Faces(), eB.Faces())
	if chamfer == nil {
		return nil, nil, false
	}
	for _, f := range eA.Faces() {
		if f != chamfer {
			return chamfer, f, true
		}
	}
	return nil, nil, false
}

// commonFace returns the first face present in both slices, or nil.
func commonFace(a, b []*topo.Face) *topo.Face {
	for _, fa := range a {
		for _, fb := range b {
			if fa == fb {
				return fa
			}
		}
	}
	return nil
}

// parallelSegmentGap is the perpendicular distance between two parallel line segments (the chamfer
// face width): the length of segment b's offset from a's line, dropping the component along a.
func parallelSegmentGap(a, b geom.LineSegment) float64 {
	u, err := math.UnitVector3FromVector(a.StartPoint.VectorTo(a.EndPoint))
	if err != nil {
		return 0
	}
	ap := a.StartPoint.VectorTo(b.StartPoint)
	perp := ap.Sub(u.AsVector().Scale(ap.Dot(u.AsVector())))
	return float64(perp.Length())
}

// acuteAngleDeg is the acute angle (0..90°) between two vectors — the dihedral folded so face-normal
// orientation (inward vs outward) does not change the reported chamfer angle.
func acuteAngleDeg(n1, n2 math.Vector3) float64 {
	u1, e1 := math.UnitVector3FromVector(n1)
	u2, e2 := math.UnitVector3FromVector(n2)
	if e1 != nil || e2 != nil {
		return 0
	}
	dot := stdmath.Abs(float64(u1.AsVector().Dot(u2.AsVector())))
	if dot > 1 {
		dot = 1
	}
	return stdmath.Acos(dot) * 180 / stdmath.Pi
}

// midOfSegments is the midpoint of two segments' midpoints — the chamfer face's rough centre.
func midOfSegments(a, b geom.LineSegment) math.Point3 {
	return a.StartPoint.Midpoint(a.EndPoint).Midpoint(b.StartPoint.Midpoint(b.EndPoint))
}

// trimAngleDeg formats an angle without a trailing ".0" — "45", "30.5".
func trimAngleDeg(deg float64) string {
	if deg == stdmath.Trunc(deg) {
		return strconv.Itoa(int(deg))
	}
	return strconv.FormatFloat(deg, 'f', 1, 64)
}

// featureNoteCallout builds a leadered callout at the feature point (sheet mm): a leader from a text
// anchor up-right of the feature to the feature, with an arrowhead, plus the callout text.
func featureNoteCallout(fx, fy float64, text string) ([]DrawingCurve, []AnnotationLabel) {
	const k = 0.70710678 // cos 45°
	tx, ty := fx+featureNoteOffsetMM*k, fy+featureNoteOffsetMM*k
	curves := []DrawingCurve{dimSegment(tx, ty, fx, fy)}
	curves = append(curves, noteArrowhead(tx, ty, fx, fy)...)
	return curves, []AnnotationLabel{{Text: text, X: tx + 2, Y: ty}}
}

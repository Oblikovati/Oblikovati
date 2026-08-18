// SPDX-License-Identifier: GPL-2.0-only

package feature

import (
	"fmt"
	stdmath "math"

	"oblikovati.org/kernel/geom"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
	"oblikovati.org/model/identity"
	"oblikovati.org/model/sketch"
)

// Wrapping an emboss ONTO a curved face (Inventor's Wrap to Face, #1893): text that follows a
// shaft instead of cutting a chord through it.
//
// The wrap is an isometry of the developable surface — arc length along the face equals distance
// in the sketch — which is why the sketch plane has to be TANGENT to it. That tangency is not
// bureaucracy: it is what fixes the correspondence. Without it there is no distinguished point to
// anchor the profile at, and no direction in the sketch that measures arc length one-for-one.
//
// Only the cylinder is built here. Inventor also wraps to a cone, whose development is a circular
// sector rather than a rectangle — a different frame (the tangent line is a generator, and the
// angle per unit arc varies with slant distance). A cone is refused with that said out loud
// rather than wrapped as if it were a cylinder, which would silently distort the profile.

// wrapAngularStep is the finest angular step a wrapped loop is discretized to, matching the
// revolve's 64-per-turn budget so a wrapped emboss is as round as the shaft it sits on. The
// profile's own polygon says nothing about the curvature it is about to be laid on — a straight
// sketch line has two points and would wrap to a chord — so every segment is resampled.
const wrapAngularStep = 2 * stdmath.Pi / revolveSegments

// embossWrapFrame is the sketch→cylinder correspondence: where the sketch's origin of measurement
// lands on the surface, and which way its two in-plane directions run there.
type embossWrapFrame struct {
	cyl geom.Cylinder
	// tangency is the sketch point that lands on the tangency line, and axisPoint is its foot on
	// the cylinder axis. Everything else is measured from there.
	tangency  math.Point2
	axisPoint math.Point3
	// radial points from the axis to the tangency line; circum is the arc-length direction there
	// (axis × radial). Both are unit and both lie in the sketch plane.
	radial math.Vector3
	circum math.Vector3
}

// wrapOntoFace raises (or cuts) the profiles wrapped onto the referenced curved face.
func (f *EmbossFeature) wrapOntoFace(in Input, profiles []*sketch.Profile, d float64) (Output, error) {
	body, err := runningBody(in)
	if err != nil {
		return Output{}, fmt.Errorf("emboss: wrapToFace needs an existing body: %w", err)
	}
	face, mt, err := bindFace(body, f.def.WrapFaceKey, anchorFor(f.def.WrapFaceKey, f.def.FaceAnchors))
	if err != nil {
		return Output{}, fmt.Errorf("emboss: wrapToFace: %w", err)
	}
	frame, err := embossWrapFrameOn(face, f.def.Sketch.Plane())
	if err != nil {
		return Output{}, err
	}
	f.tool, err = wrappedEmbossTool(profiles, f.def.Sketch.Plane(), frame, d,
		f.def.Type.engraves(), featOr(f.featName, "emboss"), in.Diag)
	if err != nil {
		return Output{}, err
	}
	return f.combineWrapped(in, mt)
}

// combineWrapped applies the wrapped tool with the flavour's boolean and reports the face heal.
func (f *EmbossFeature) combineWrapped(in Input, mt identity.MatchType) (Output, error) {
	bodies, err := combine(in, f.tool, f.Operation())
	if err != nil {
		return Output{}, err
	}
	return Output{Bodies: bodies, Heals: faceHeal(f.def.WrapFaceKey, mt)}, nil
}

// embossWrapFrameOn derives the correspondence between the sketch plane and the wrap face,
// refusing every face the wrap is not defined on (Inventor limits it to one planar or conical
// face; a planar one needs no wrap at all).
func embossWrapFrameOn(face *topo.Face, plane sketch.Plane) (embossWrapFrame, error) {
	switch cyl := face.Geometry().(type) {
	case geom.Cylinder:
		return embossWrapFrameFor(cyl, plane)
	case *geom.Cylinder:
		return embossWrapFrameFor(*cyl, plane)
	default:
		return embossWrapFrame{}, fmt.Errorf("emboss: wrapToFace needs a cylindrical face, got %T; "+
			"a planar face needs no wrap, and a cone (whose development is a sector, not a "+
			"rectangle) is not supported yet", face.Geometry())
	}
}

// embossWrapFrameFor builds the frame for a cylinder, checking the two things the wrap needs: the
// sketch plane must be parallel to the axis, and it must touch the cylinder. Together those say
// "tangent", and tangency is what makes distance in the sketch equal arc length on the face.
func embossWrapFrameFor(cyl geom.Cylinder, plane sketch.Plane) (embossWrapFrame, error) {
	axis := cyl.AxisDir.AsVector()
	normal := plane.Normal().AsVector()
	if along := stdmath.Abs(float64(normal.Dot(axis))); along > wrapTangencyTol {
		return embossWrapFrame{}, fmt.Errorf("emboss: wrapToFace needs the sketch plane PARALLEL to "+
			"the face's axis (its normal ⟂ the axis), but the normal·axis is %g; sketch on a work "+
			"plane tangent to the face", along)
	}
	foot := cyl.Origin.TranslateBy(axis.Scale(cyl.Origin.VectorTo(plane.Origin()).Dot(axis)))
	reach := float64(foot.VectorTo(plane.Origin()).Dot(normal)) // signed distance from the axis to the plane
	if stdmath.Abs(stdmath.Abs(reach)-cyl.Radius) > wrapTangencyTol*cyl.Radius {
		return embossWrapFrame{}, fmt.Errorf("emboss: wrapToFace needs the sketch plane TANGENT to "+
			"the face, but it stands %g from the axis of a radius-%g cylinder; without tangency the "+
			"sketch's distances are not the face's arc lengths", stdmath.Abs(reach), cyl.Radius)
	}
	return newEmbossWrapFrame(cyl, plane, axis, normal, reach), nil
}

// wrapTangencyTol is how far from parallel/tangent the sketch plane may sit, relative to the
// cylinder's radius. It is a geometric admissibility test on an authored work plane, not a fit, so
// it is loose enough to survive the plane having been built from the face itself.
const wrapTangencyTol = 1e-6

// newEmbossWrapFrame assembles the frame once the plane is known to be tangent.
func newEmbossWrapFrame(cyl geom.Cylinder, plane sketch.Plane, axis, normal math.Vector3,
	reach float64) embossWrapFrame {
	radial := normal // the plane lies on the +normal side of the axis...
	if reach < 0 {
		radial = normal.Scale(-1) // ...or the -normal side; radial always points AT the plane
	}
	axisPoint := cyl.Origin.TranslateBy(axis.Scale(cyl.Origin.VectorTo(plane.Origin()).Dot(axis)))
	tangency := axisPoint.TranslateBy(radial.Scale(math.Scalar(cyl.Radius)))
	return embossWrapFrame{
		cyl: cyl, tangency: plane.ToSketch(tangency), axisPoint: axisPoint,
		radial: radial, circum: axis.Cross(radial),
	}
}

// at maps a sketch point onto the cylinder at the given radius: the displacement from the tangency
// is split into an axial part, which slides along the axis, and a circumferential part, which is
// spent as ARC LENGTH — so it becomes an angle of arc/radius.
func (fr embossWrapFrame) at(p math.Point2, plane sketch.Plane, radius float64) math.Point3 {
	v := plane.ToModel(fr.tangency).VectorTo(plane.ToModel(p))
	axial, arc := v.Dot(fr.cyl.AxisDir.AsVector()), v.Dot(fr.circum)
	base := fr.axisPoint.
		TranslateBy(fr.cyl.AxisDir.AsVector().Scale(axial)).
		TranslateBy(fr.radial.Scale(math.Scalar(radius)))
	rot := math.Rotation4(float64(arc)/fr.cyl.Radius, fr.cyl.AxisDir, fr.axisPoint)
	return rot.TransformPoint(base)
}

// angleSpan is the angle a sketch-space distance subtends on the cylinder — how the resampling
// step is chosen.
func (fr embossWrapFrame) angleSpan(a, b math.Point2, plane sketch.Plane) float64 {
	v := plane.ToModel(a).VectorTo(plane.ToModel(b))
	return stdmath.Abs(float64(v.Dot(fr.circum))) / fr.cyl.Radius
}

// wrappedLoop maps a closed sketch polygon onto the cylinder at one radius, resampling each
// segment first so the wrapped edge follows the curvature instead of chording across it.
func wrappedLoop(poly []math.Point2, plane sketch.Plane, fr embossWrapFrame, radius float64) []math.Point3 {
	out := make([]math.Point3, 0, len(poly))
	n := len(poly)
	for i := 0; i < n; i++ {
		a, b := poly[i], poly[(i+1)%n]
		steps := wrapSegmentSteps(fr.angleSpan(a, b, plane))
		for s := 0; s < steps; s++ { // b is emitted as the next segment's a
			t := float64(s) / float64(steps)
			out = append(out, fr.at(lerpPoint2(a, b, t), plane, radius))
		}
	}
	return out
}

// wrapSegmentSteps is how many pieces a segment spanning that angle is cut into.
func wrapSegmentSteps(angle float64) int {
	return int(stdmath.Max(1, stdmath.Ceil(angle/wrapAngularStep)))
}

// lerpPoint2 interpolates in sketch space. The wrap is linear in the (axial, arc) coordinates, so
// a uniformly split segment maps to a uniformly split helix-free arc on the face.
func lerpPoint2(a, b math.Point2, t float64) math.Point2 {
	return math.P2(a.X+(b.X-a.X)*math.Scalar(t), a.Y+(b.Y-a.Y)*math.Scalar(t))
}

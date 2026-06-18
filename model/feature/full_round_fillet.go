// SPDX-License-Identifier: GPL-2.0-only

package feature

import (
	"fmt"
	stdmath "math"

	"oblikovati.org/kernel/brep"
	"oblikovati.org/kernel/geom"
	"oblikovati.org/kernel/ops"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
	"oblikovati.org/model/sketch"
)

// FullRoundFilletDefinition is a full-round fillet (#694, Inventor FilletFullRoundSet): it replaces a
// CENTER face between two parallel SIDE faces with a half-cylinder tangent to both sides, removing the
// center face — the rounded top of a rib/wall. Unlike an edge fillet (which fails to fully consume the
// center face at the round radius), this is built by reconstruction: a tool the size of the center-face
// footprint (a box minus the round cylinder = the two sharp corners) is cut from the body, so it stays
// local to the rib and leaves a clean half-round. The radius is half the side-to-side distance.
type FullRoundFilletDefinition struct {
	Side1Keys  [][]byte
	CenterKeys [][]byte
	Side2Keys  [][]byte
}

// FullRoundFilletFeature rounds a center face into a half-cylinder between two parallel sides.
type FullRoundFilletFeature struct{ def *FullRoundFilletDefinition }

// Definition returns the feature's definition.
func (f *FullRoundFilletFeature) Definition() *FullRoundFilletDefinition { return f.def }

// Kind names the feature type.
func (f *FullRoundFilletFeature) Kind() string { return "full-round-fillet" }

// Recompute replaces the center face with a full round between the two side faces.
func (f *FullRoundFilletFeature) Recompute(in Input) (Output, error) {
	return fullRoundFilletBody(in, f.def, "full-round-fillet")
}

// fullRoundFilletBody reconstructs the full round: it derives the rib frame from the three faces,
// builds a corner tool (the center-face-footprint box minus the round cylinder) and cuts it from the
// body. Non-planar faces, non-parallel sides, or a degenerate frame are clean errors (the feature
// goes Sick) — this slice handles the parallel-sides case (variable-radius rounds are a follow-up).
func fullRoundFilletBody(in Input, def *FullRoundFilletDefinition, feat string) (Output, error) {
	body, err := runningBody(in)
	if err != nil {
		return Output{}, err
	}
	center, err := singlePlanarFace(body, def.CenterKeys, feat, "center")
	if err != nil {
		return Output{}, err
	}
	fr, err := fullRoundFrame(body, def, center, feat)
	if err != nil {
		return Output{}, err
	}
	pc := center.face.RangeBox().Center()
	l := faceExtentAlong(center.face, pc, fr.axis) + 4*fr.radius // generous overhang so the tool cuts cleanly
	tool, err := fullRoundCornerTool(pc, fr.up, fr.sideN, fr.axis, fr.radius, l, feat)
	if err != nil {
		return Output{}, err
	}
	result, err := ops.Boolean(ops.Cut, planarized(body, feat), tool)
	if err != nil {
		return Output{}, err
	}
	return Output{Bodies: replaceBody(in.Bodies, body, result)}, nil
}

// ribFrame is the orthonormal frame + radius of a full round, derived from the three faces.
type ribFrame struct {
	up, sideN, axis math.Vector3
	radius          float64
}

// fullRoundFrame resolves the two side faces and derives the rib frame: up (center normal), sideN
// (side normal), axis (along the rib), and the round radius (half the side-to-side distance). It
// errors when the sides are not parallel, the center is not perpendicular to them, or the radius is 0.
func fullRoundFrame(body *topo.Body, def *FullRoundFilletDefinition, center planarFace, feat string) (ribFrame, error) {
	side1, err := singlePlanarFace(body, def.Side1Keys, feat, "side1")
	if err != nil {
		return ribFrame{}, err
	}
	side2, err := singlePlanarFace(body, def.Side2Keys, feat, "side2")
	if err != nil {
		return ribFrame{}, err
	}
	up, sideN := normalize(center.Normal()), normalize(side1.Normal())
	if d := stdmath.Abs(float64(normalize(side2.Normal()).Dot(sideN))); d < 0.999 {
		return ribFrame{}, fmt.Errorf("%s: the two side faces must be parallel (|n1·n2| = %.3f)", feat, d)
	}
	axis := normalize(up.Cross(sideN))
	if float64(axis.Dot(axis)) < 0.5 {
		return ribFrame{}, fmt.Errorf("%s: the center face must be perpendicular to the sides", feat)
	}
	r := stdmath.Abs(float64(side1.Origin.VectorTo(side2.Origin).Dot(sideN))) / 2
	if r <= 0 {
		return ribFrame{}, fmt.Errorf("%s: the side faces are coincident (zero radius)", feat)
	}
	return ribFrame{up: up, sideN: sideN, axis: axis, radius: r}, nil
}

// fullRoundCornerTool builds the cut tool: a box covering the center-face footprint from the round's
// base plane up to the center face, MINUS the round cylinder — i.e. the two sharp corners the round
// shaves off. pc is the center-face centre; up/sideN/axisDir the orthonormal rib frame; r the radius;
// l the length along the rib.
func fullRoundCornerTool(pc math.Point3, up, sideN, axisDir math.Vector3, r, l float64, feat string) (*topo.Body, error) {
	plane, err := sketch.NewPlane(pc, sideN.AsUnit(), up.AsUnit())
	if err != nil {
		return nil, fmt.Errorf("%s: %w", feat, err)
	}
	// Box (in the sideN×up plane, centred on pc): full width 2r across the sides, depth r below the
	// centre face, extruded ±l/2 along the rib axis (the plane normal).
	rect := []math.Point2{{X: -r, Y: -r}, {X: r, Y: -r}, {X: r, Y: 0}, {X: -r, Y: 0}}
	box := buildPrism(rect, plane, span{near: -l / 2, far: l / 2}, 0, feat)

	axisPt := pc.TranslateBy(up.Scale(-r)) // the cylinder axis sits r below the centre face, on the midline
	base := axisPt.TranslateBy(axisDir.Scale(-l / 2))
	cyl, err := brep.SolidCylinder(base, axisDir, r, l)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", feat, err)
	}
	corner, err := ops.Boolean(ops.Cut, planarized(box, feat), planarized(cyl, feat))
	if err != nil {
		return nil, fmt.Errorf("%s: %w", feat, err)
	}
	return corner, nil
}

// planarFace pairs a face with its plane geometry.
type planarFace struct {
	face *topo.Face
	geom.Plane
}

// singlePlanarFace resolves exactly one planar face from keys (a full round takes one face per set).
func singlePlanarFace(body *topo.Body, keys [][]byte, feat, role string) (planarFace, error) {
	want := keyLookup(keys)
	var found *topo.Face
	for _, f := range body.Faces() {
		if want[string(f.ReferenceKey())] {
			if found != nil {
				return planarFace{}, fmt.Errorf("%s: %s must be a single face", feat, role)
			}
			found = f
		}
	}
	if found == nil {
		return planarFace{}, fmt.Errorf("%s: %s face not found", feat, role)
	}
	pl, ok := found.Geometry().(geom.Plane)
	if !ok {
		return planarFace{}, fmt.Errorf("%s: %s must be a planar face", feat, role)
	}
	return planarFace{face: found, Plane: pl}, nil
}

// faceExtentAlong returns the span of the face's vertices projected onto dir, relative to origin.
func faceExtentAlong(f *topo.Face, origin math.Point3, dir math.Vector3) float64 {
	lo, hi := stdmath.Inf(1), stdmath.Inf(-1)
	for _, v := range f.Vertices() {
		d := float64(origin.VectorTo(v.Point()).Dot(dir))
		lo, hi = stdmath.Min(lo, d), stdmath.Max(hi, d)
	}
	if hi < lo {
		return 0
	}
	return hi - lo
}

// normalize returns the unit vector in v's direction (zero stays zero).
func normalize(v math.Vector3) math.Vector3 {
	if l := float64(v.Length()); l > 1e-12 {
		return v.Scale(math.Scalar(1 / l))
	}
	return math.V3(0, 0, 0)
}

// AddFullRoundFillet replaces the center face with a full round between the two parallel side faces
// (#694). Each set is a single planar face.
func (c *DressUpFeatures) AddFullRoundFillet(side1Keys, centerKeys, side2Keys [][]byte) *PartFeature {
	return c.engine.Add(&FullRoundFilletFeature{def: &FullRoundFilletDefinition{
		Side1Keys: side1Keys, CenterKeys: centerKeys, Side2Keys: side2Keys,
	}})
}

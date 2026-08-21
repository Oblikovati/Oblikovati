// SPDX-License-Identifier: GPL-2.0-only

package feature

import (
	"errors"

	"oblikovati.org/kernel/ops"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/model/sketch"
)

// The from-plane emboss (Inventor's kEmbossEngraveFromPlane, #1893).
//
// The face-anchored flavours work off the part's surface: one raises from it, the other cuts into
// it. This one works off the SKETCH PLANE, which for this flavour runs through the part, and does
// both at once — it raises the region a depth on the plane's front side and cuts it the same depth
// on the back. That is how a boss and its relief pocket, or a raised panel and its rebate, come out
// of one feature referenced to one datum instead of two features referenced to a surface.
//
// Both tools are bounded by the plane and the depth alone. An earlier draft bounded the fill by the
// part's reach along the normal so the union would always overlap existing material; that is wrong
// on any part whose underside is not flat — on a table-and-leg it would fill the space beneath the
// top. The plane cuts through the material, so a tool that starts at the plane already overlaps it.

// levelToPlane raises the region on the front side of the sketch plane and cuts it on the back.
//
// NOTE for pattern/mirror replication: this is the only emboss flavour that applies two booleans,
// and [ToolFeature] carries one tool. f.tool is left holding the RAISE, so a pattern of a
// from-plane emboss repeats the raise and not the relief cut. Refusing to build the feature at all
// would be worse; the gap is tracked as a follow-up on #1893.
func (f *EmbossFeature) levelToPlane(in Input, profiles []*sketch.Profile, d float64) (Output, error) {
	if len(f.def.WrapFaceKey) > 0 {
		return Output{}, errors.New("emboss: the from-plane type works off the SKETCH plane, not off " +
			"a face, so there is no face to wrap onto; drop wrapToFace or use a from-face type")
	}
	if len(in.Bodies) == 0 {
		return Output{}, errors.New("emboss: the from-plane type raises and cuts material about a " +
			"plane running through the part, and this part has no body yet")
	}
	bodies, err := f.raiseFront(in, profiles, d)
	if err != nil {
		return Output{}, err
	}
	return f.cutBack(in, bodies, profiles, d)
}

// raiseFront unions the region from the sketch plane out to the depth along the plane normal.
func (f *EmbossFeature) raiseFront(in Input, profiles []*sketch.Profile, d float64) ([]*topo.Body, error) {
	f.tool = buildProfilePrisms(profiles, f.def.Sketch.Plane(), orderedSpan(0, d), f.def.Taper,
		featOr(f.featName, "emboss"), in.Diag)
	return combine(in, f.tool, ops.Join)
}

// cutBack removes the same region on the other side of the plane. It runs against the bodies the
// raise produced, not the ones the feature started with, so the two halves compose into one result.
func (f *EmbossFeature) cutBack(in Input, bodies []*topo.Body, profiles []*sketch.Profile,
	d float64) (Output, error) {
	relief := buildProfilePrisms(profiles, f.def.Sketch.Plane(), orderedSpan(0, -d), f.def.Taper,
		featOr(f.featName, "emboss")+"/relief", in.Diag)
	f.reliefTool = relief // exposed via ToolApplications so a pattern replicates the cut too (#2066)
	run := in
	run.Bodies = bodies
	out, err := combine(run, relief, ops.Cut)
	if err != nil {
		return Output{}, err
	}
	return Output{Bodies: out}, nil
}

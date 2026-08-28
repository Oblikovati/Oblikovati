// SPDX-License-Identifier: GPL-2.0-only

package feature

import (
	"fmt"

	"oblikovati.org/api/types"
	"oblikovati.org/kernel/geom"
	"oblikovati.org/kernel/ops"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
)

// Dress-up features — the THREAD definition (M48 #2233 split of dressup.go). The cosmetic/cut thread
// feature over a cylindrical face (thread spec, pipe-thread flag, represented diameter) and its
// Recompute. The adder collection stays in dressup.go.

// ThreadDefinition applies thread data to a cylindrical face. Cut=false is a cosmetic thread
// (data + display, solid unchanged); Cut=true models a real thread (a helical groove cut).
// Class, Tapered, and ModelDiameter are the #325 parity fields: the tolerance class recorded
// on the spec, the pipe-thread flag (the reference's TaperedThreadInfo split — data-only, a
// cut tapered thread needs a conical face and errors), and which thread diameter the modeled
// cylindrical face represents (zero value = major, the common case).
type ThreadDefinition struct {
	FaceKey       []byte
	Designation   string
	Cut           bool
	Class         string
	Tapered       bool
	ModelDiameter types.ModelDiameterFromThread
	// Offset and Length limit the thread to a sub-span of the face along its axis (Inventor's
	// ThreadOffset / ThreadDepth): the thread runs from the face's min axial extent + Offset for
	// Length (cm). A nil Offset ⇒ 0; a nil or zero Length ⇒ the full face (Inventor's FullDepth).
	// A double-ended stud threads only its two ends by giving each thread its own Offset+Length.
	Offset func() float64
	Length func() float64
	// LeftHanded reverses the thread's sense (#1892). Named for the LEFT hand, not Inventor's
	// RightHanded, for the same reason as HoleTap.LeftHanded: a definition built as a literal
	// would otherwise default every thread to left-handed. A "-LH" designation says the same
	// thing; either one alone makes the thread left-handed.
	LeftHanded bool
	// FaceAnchors maps FaceKey to its mint-time centroid for the geometric recovery tier
	// (ADR-0043 P6 / #1579); see FilletDefinition.EdgeAnchors.
	FaceAnchors map[string]math.Point3
}

// ThreadFeature tags a cylindrical face with a cosmetic thread (Inventor's ThreadFeature): it
// records the resolved thread data and leaves the solid unchanged. Cut-thread geometry (a real
// helical groove) is a separate modeled feature; the cosmetic thread is the data + display.
type ThreadFeature struct {
	def  *ThreadDefinition
	spec *ThreadSpec // resolved on the last recompute (nil until then)
}

func (t *ThreadFeature) Definition() *ThreadDefinition { return t.def }
func (t *ThreadFeature) Kind() string                  { return "thread" }

// Spec returns the thread data resolved on the last recompute (nil if it never bound).
func (t *ThreadFeature) Spec() *ThreadSpec { return t.spec }

// Recompute parses the designation, binds the cylindrical face, records the thread spec, and
// passes the (unchanged) solid through. A bad designation, a lost face, or a non-cylindrical
// face makes the feature Sick — as does cutting a tapered (pipe) thread, which would need a
// conical face the feature doesn't model yet.
func (t *ThreadFeature) Recompute(in Input) (Output, error) {
	spec, err := ParseThreadDesignation(t.def.Designation)
	if err != nil {
		return Output{}, err
	}
	spec.Class, spec.Tapered = t.def.Class, t.def.Tapered
	// The designation may already carry the handedness as an "-LH" suffix, so the flag can only
	// take handedness AWAY from right — the two spellings agree rather than fight (#1892).
	spec.RightHanded = spec.RightHanded && !t.def.LeftHanded
	spec.ModelDiameter = t.def.ModelDiameter
	if spec.ModelDiameter == 0 {
		spec.ModelDiameter = types.ThreadMajorDiameter
	}
	if t.def.Tapered && t.def.Cut {
		return Output{}, fmt.Errorf("thread %q: a cut tapered (pipe) thread needs a conical face; model it cosmetic", t.def.Designation)
	}
	body, err := runningBody(in)
	if err != nil {
		return Output{}, err
	}
	face, mt, err := bindFace(body, t.def.FaceKey, anchorFor(t.def.FaceKey, t.def.FaceAnchors))
	if err != nil {
		return Output{}, fmt.Errorf("thread: %w", err)
	}
	cyl, ok := face.Geometry().(geom.Cylinder)
	if !ok {
		return Output{}, fmt.Errorf("thread %q: face is not cylindrical (%T)", t.def.Designation, face.Geometry())
	}
	vFaceMin, vFaceMax := axialExtent(face.RangeBox(), cyl)
	vMin, vMax := resolveThreadSpan(vFaceMin, vFaceMax, t.def.Offset, t.def.Length)
	spec.Internal = bodyHasMaterialOutside(body, cyl, (vMin+vMax)/2, (spec.MajorDiameter-spec.MinorDiameter)/2/10)
	t.spec = &spec
	if !t.def.Cut {
		return Output{Bodies: in.Bodies, Heals: faceHeal(t.def.FaceKey, mt)}, nil // cosmetic: solid unchanged
	}
	// Modeled (cut) thread: retype the cylindrical face to a threaded surface — O(1), no
	// boolean — so it tessellates and measures as real threaded geometry. The span honours the
	// thread's offset/length so a partial cut thread grooves only its run.
	threaded := geom.ThreadedCylinder{
		Cylinder: cyl, Pitch: spec.Pitch / 10, Depth: (spec.MajorDiameter - spec.MinorDiameter) / 2 / 10,
		Designation: t.def.Designation, Internal: spec.Internal, RightHanded: spec.RightHanded,
		VMin: vMin, VMax: vMax,
	}
	out := make([]*topo.Body, len(in.Bodies))
	copy(out, in.Bodies)
	// Target the RESOLVED face's current key, not the stored one: a healed thread bound to a
	// recovered sibling whose live key differs from t.def.FaceKey (ADR-0043 P6, mirrors edges).
	threadedBody, err := ops.ReplaceFaceSurface(body, face.ReferenceKey(), threaded)
	if err != nil {
		return Output{}, err
	}
	out[len(out)-1] = threadedBody // runningBody is the last body
	return Output{Bodies: out, Heals: faceHeal(t.def.FaceKey, mt)}, nil
}

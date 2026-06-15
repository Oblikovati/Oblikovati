// SPDX-License-Identifier: GPL-2.0-only

package feature

import (
	"bytes"
	"fmt"

	"oblikovati.org/api/types"
	"oblikovati.org/math"
	"oblikovati.org/model/param"
	"oblikovati.org/model/sketch"
)

// In-place editing of a placed feature's scalar parameters (Inventor's double-click / Edit
// Feature). A feature whose definition implements [Editable] exposes its scalar inputs —
// distance, radius, angle, diameter … — as [EditableParam]s the head renders one field each.
// Editing replaces a parametric closure with a constant (a direct value edit); re-binding to a
// named parameter/expression is a follow-up. After Set, the feature must be marked dirty and
// the part recomputed for the change to flow through the history (see app.CommitFeatureEdit).

// EditableParam is one scalar input of a feature. Get/Set read and write it in DATABASE units;
// Unit is the quantity kind so the UI shows and converts to the document's preferred unit.
// Integer marks a whole-number input (a pattern count) the UI edits with an integer field.
type EditableParam struct {
	Label   string
	Unit    param.Unit
	Integer bool
	Get     func() float64
	Set     func(float64)
}

// Editable is implemented by feature definitions whose scalar parameters can be edited after
// placement. EditableParams returns them in display order (empty ⇒ nothing to edit).
type Editable interface {
	EditableParams() []EditableParam
}

// RefKind is the kind of geometry a reference slot accepts (so the editor sets the right pick
// filter and the caller builds the right [PickedRef]).
type RefKind int

const (
	// RefEdges accepts one or more edges (fillet/chamfer).
	RefEdges RefKind = iota
	// RefFaces accepts one or more faces (shell removed faces, draft faces).
	RefFaces
	// RefFace accepts a single face (a hole's placement face).
	RefFace
	// RefProfile accepts a single sketch profile region (extrude/revolve/coil/rib/emboss).
	RefProfile
	// RefPlane accepts a single plane — a planar face or a work plane (a mirror's plane).
	RefPlane
)

// PickedRef is a single geometry pick the editor feeds to a reference slot's Add. The caller
// (app) fills the fields the slot's Kind needs: Key for edge/face slots; Sketch+Profile for a
// profile slot; Origin+Normal(+PlaneKey) for a plane slot.
type PickedRef struct {
	Key      []byte
	Sketch   *sketch.Sketch
	Profile  int
	Origin   math.Point3
	Normal   math.Vector3
	PlaneKey []byte
}

// EditableRefSlot is one geometric input of a placed feature the UI can re-pick — the edges of
// a fillet, the removed faces of a shell, a hole's face, an extrude's profile, a mirror's
// plane. Add applies a pick (appending for Multi slots, replacing otherwise); Clear empties it
// (nil ⇒ not clearable, e.g. a mirror needs a plane); Count reports how many references are
// set; Keys lists the current edge/face keys for highlight (nil for profile/plane); Snapshot
// captures the current state and returns a closure that restores it (for Cancel). Re-picking
// rebinds the feature to the chosen geometry — Inventor's "redefine feature references".
type EditableRefSlot struct {
	Label    string
	Kind     RefKind
	Multi    bool
	Count    func() int
	Keys     func() [][]byte
	Add      func(PickedRef)
	Clear    func()
	Snapshot func() func()
}

// ReferenceEditable is implemented by feature definitions whose geometric references can be
// re-selected after placement. EditableRefs returns the slots in display order.
type ReferenceEditable interface {
	EditableRefs() []EditableRefSlot
}

// keyRefSlotMulti builds a multi edge/face slot over a [][]byte field (its address).
func keyRefSlotMulti(label string, kind RefKind, field *[][]byte) EditableRefSlot {
	return EditableRefSlot{
		Label: label, Kind: kind, Multi: true,
		Count: func() int { return len(*field) },
		Keys:  func() [][]byte { return *field },
		Add: func(r PickedRef) {
			if !containsRefKey(*field, r.Key) {
				*field = append(*field, r.Key)
			}
		},
		Clear:    func() { *field = nil },
		Snapshot: func() func() { saved := cloneRefKeys(*field); return func() { *field = saved } },
	}
}

// keyRefSlotSingle builds a single face/edge slot over a []byte key field (its address).
func keyRefSlotSingle(label string, kind RefKind, field *[]byte) EditableRefSlot {
	return EditableRefSlot{
		Label: label, Kind: kind, Multi: false,
		Count: func() int {
			if len(*field) == 0 {
				return 0
			}
			return 1
		},
		Keys: func() [][]byte {
			if len(*field) == 0 {
				return nil
			}
			return [][]byte{*field}
		},
		Add:      func(r PickedRef) { *field = r.Key },
		Clear:    func() { *field = nil },
		Snapshot: func() func() { saved := append([]byte(nil), *field...); return func() { *field = saved } },
	}
}

// profileRefSlotIndex builds a single-profile slot over a (sketch, index) pair of fields.
func profileRefSlotIndex(label string, skField **sketch.Sketch, idxField *int) EditableRefSlot {
	return EditableRefSlot{
		Label: label, Kind: RefProfile, Multi: false,
		Count: func() int {
			if *skField == nil {
				return 0
			}
			return 1
		},
		Add: func(r PickedRef) { *skField = r.Sketch; *idxField = r.Profile },
		Snapshot: func() func() {
			sk, i := *skField, *idxField
			return func() { *skField = sk; *idxField = i }
		},
	}
}

// profileRefSlotIndices builds a single-profile slot over a (sketch, indices) pair, replacing
// the region set with the one picked region (re-selecting THE profile).
func profileRefSlotIndices(label string, skField **sketch.Sketch, idxField *[]int) EditableRefSlot {
	return EditableRefSlot{
		Label: label, Kind: RefProfile, Multi: false,
		Count: func() int {
			if *skField == nil {
				return 0
			}
			return len(*idxField)
		},
		Add: func(r PickedRef) { *skField = r.Sketch; *idxField = []int{r.Profile} },
		Snapshot: func() func() {
			sk, idx := *skField, append([]int(nil), *idxField...)
			return func() { *skField = sk; *idxField = idx }
		},
	}
}

// planeRefSlot builds a single-plane slot over a feature's origin/normal(/key) fields.
func planeRefSlot(label string, origin *math.Point3, normal *math.Vector3, key *[]byte) EditableRefSlot {
	return EditableRefSlot{
		Label: label, Kind: RefPlane, Multi: false,
		Count: func() int {
			if normal.LengthSquared() == 0 {
				return 0
			}
			return 1
		},
		Add: func(r PickedRef) {
			*origin, *normal = r.Origin, r.Normal
			if key != nil {
				*key = r.PlaneKey
			}
		},
		Snapshot: func() func() {
			o, n := *origin, *normal
			var k []byte
			if key != nil {
				k = append([]byte(nil), *key...)
			}
			return func() {
				*origin, *normal = o, n
				if key != nil {
					*key = k
				}
			}
		},
	}
}

func containsRefKey(ks [][]byte, key []byte) bool {
	for _, k := range ks {
		if bytes.Equal(k, key) {
			return true
		}
	}
	return false
}

func cloneRefKeys(ks [][]byte) [][]byte {
	if ks == nil {
		return nil
	}
	out := make([][]byte, len(ks))
	for i, k := range ks {
		out[i] = append([]byte(nil), k...)
	}
	return out
}

// EditableRefs exposes the fillet's rounded edges.
func (f *FilletFeature) EditableRefs() []EditableRefSlot {
	return []EditableRefSlot{keyRefSlotMulti("Edges", RefEdges, &f.def.EdgeKeys)}
}

// EditableRefs exposes the chamfer's bevelled edges.
func (c *ChamferFeature) EditableRefs() []EditableRefSlot {
	return []EditableRefSlot{keyRefSlotMulti("Edges", RefEdges, &c.def.EdgeKeys)}
}

// EditableRefs exposes the shell's removed (open) faces.
func (s *ShellFeature) EditableRefs() []EditableRefSlot {
	return []EditableRefSlot{keyRefSlotMulti("Removed faces", RefFaces, &s.def.RemovedFaceKeys)}
}

// EditableRefs exposes the draft's tapered faces.
func (d *FaceDraftFeature) EditableRefs() []EditableRefSlot {
	return []EditableRefSlot{keyRefSlotMulti("Faces", RefFaces, &d.def.FaceKeys)}
}

// EditableRefs exposes the hole's placement face.
func (h *HoleFeature) EditableRefs() []EditableRefSlot {
	return []EditableRefSlot{keyRefSlotSingle("Placement face", RefFace, &h.def.PlacementFaceKey)}
}

// EditableRefs exposes the extrude's profile.
func (e *ExtrudeFeature) EditableRefs() []EditableRefSlot {
	return []EditableRefSlot{profileRefSlotIndices("Profile", &e.def.Sketch, &e.def.ProfileIndices)}
}

// EditableRefs exposes the revolve's profile.
func (r *RevolveFeature) EditableRefs() []EditableRefSlot {
	return []EditableRefSlot{profileRefSlotIndex("Profile", &r.def.Sketch, &r.def.ProfileIndex)}
}

// EditableRefs exposes the coil's profile.
func (c *CoilFeature) EditableRefs() []EditableRefSlot {
	return []EditableRefSlot{profileRefSlotIndex("Profile", &c.def.Sketch, &c.def.ProfileIndex)}
}

// EditableRefs exposes the rib's open profile.
func (r *RibFeature) EditableRefs() []EditableRefSlot {
	return []EditableRefSlot{profileRefSlotIndex("Profile", &r.def.Sketch, &r.def.ProfileIndex)}
}

// EditableRefs exposes the emboss profile.
func (e *EmbossFeature) EditableRefs() []EditableRefSlot {
	return []EditableRefSlot{profileRefSlotIndices("Profile", &e.def.Sketch, &e.def.ProfileIndices)}
}

// EditableRefs exposes the mirror's plane.
func (m *MirrorFeature) EditableRefs() []EditableRefSlot {
	return []EditableRefSlot{planeRefSlot("Mirror plane", &m.def.Origin, &m.def.Normal, &m.def.MirrorPlaneKey)}
}

// scalarParam builds an EditableParam over a func()float64 field: Get calls it (nil ⇒ 0), Set
// replaces it with a constant. Pass the ADDRESS of the definition's closure field.
func scalarParam(label string, u param.Unit, field *func() float64) EditableParam {
	return EditableParam{
		Label: label, Unit: u,
		Get: func() float64 {
			if *field == nil {
				return 0
			}
			return (*field)()
		},
		Set: func(v float64) { *field = func() float64 { return v } },
	}
}

// intParam builds a whole-number param over a func()int field (its address); Set rounds and
// clamps to ≥1 (a pattern needs at least one occurrence) and rewrites the field as a constant.
func intParam(label string, field *func() int) EditableParam {
	return EditableParam{
		Label: label, Unit: param.Unitless, Integer: true,
		Get: func() float64 {
			if *field == nil {
				return 0
			}
			return float64((*field)())
		},
		Set: func(v float64) {
			n := int(v + 0.5)
			if n < 1 {
				n = 1
			}
			*field = func() int { return n }
		},
	}
}

// spacingParam edits a step VECTOR's magnitude (a pattern's spacing), preserving its direction.
// A zero vector (lost direction) falls back to the given axis so editing still has a direction.
func spacingParam(label string, vec *math.Vector3, fallback math.Vector3) EditableParam {
	return EditableParam{
		Label: label, Unit: param.Length,
		Get: func() float64 { return vec.Length() },
		Set: func(v float64) {
			dir := *vec
			if dir.LengthSquared() == 0 {
				dir = fallback
			}
			u, err := math.UnitVector3FromVector(dir)
			if err != nil {
				return
			}
			*vec = u.AsVector().Scale(v)
		},
	}
}

// EditableParams exposes a rectangular pattern's counts and per-direction spacing.
func (r *RectangularPatternFeature) EditableParams() []EditableParam {
	return []EditableParam{
		intParam("Count X", &r.def.CountX),
		intParam("Count Y", &r.def.CountY),
		spacingParam("Spacing X", &r.def.StepX, math.V3(1, 0, 0)),
		spacingParam("Spacing Y", &r.def.StepY, math.V3(0, 1, 0)),
	}
}

// EditableParams exposes a circular pattern's count and total angle.
func (c *CircularPatternFeature) EditableParams() []EditableParam {
	return []EditableParam{
		intParam("Count", &c.def.Count),
		scalarParam("Angle", param.Angle, &c.def.Angle),
	}
}

// EditableParams exposes the extrude's distance (and the asymmetric second distance, if set).
func (e *ExtrudeFeature) EditableParams() []EditableParam {
	ps := []EditableParam{scalarParam("Distance", param.Length, &e.def.Extent.Distance)}
	if e.def.Extent.Distance2 != nil {
		ps = append(ps, scalarParam("Distance 2", param.Length, &e.def.Extent.Distance2))
	}
	return ps
}

// EditableParams exposes the revolve angle (0 ⇒ a full revolution).
func (r *RevolveFeature) EditableParams() []EditableParam {
	return []EditableParam{scalarParam("Angle", param.Angle, &r.def.Angle)}
}

// EditableParams exposes each move operation's own scalars in list order, so a composed
// move ("rotate then slide") is edited per operation (M20-F20). A baked single-transform
// move (no operation list) exposes nothing — its matrix is not a scalar surface.
func (m *MoveFeature) EditableParams() []EditableParam {
	var ps []EditableParam
	for i := range m.def.Ops {
		ps = append(ps, moveOpParams(i+1, &m.def.Ops[i])...)
	}
	return ps
}

// moveOpParams returns the editable scalars of the n-th move operation (1-based label).
func moveOpParams(n int, op *MoveOperation) []EditableParam {
	switch op.Kind {
	case types.MoveFreeDrag:
		return []EditableParam{
			scalarParam(fmt.Sprintf("Move %d X", n), param.Length, &op.X),
			scalarParam(fmt.Sprintf("Move %d Y", n), param.Length, &op.Y),
			scalarParam(fmt.Sprintf("Move %d Z", n), param.Length, &op.Z),
		}
	case types.MoveAlongRay:
		return []EditableParam{scalarParam(fmt.Sprintf("Move %d Distance", n), param.Length, &op.Dist)}
	case types.MoveRotateAboutLine:
		return []EditableParam{scalarParam(fmt.Sprintf("Move %d Angle", n), param.Angle, &op.Angle)}
	default:
		return nil
	}
}

// EditableParams exposes the coil's pitch and revolution count.
func (c *CoilFeature) EditableParams() []EditableParam {
	return []EditableParam{
		scalarParam("Pitch", param.Length, &c.def.Pitch),
		scalarParam("Revolutions", param.Unitless, &c.def.Revolutions),
	}
}

// EditableParams exposes the rib's wall thickness and depth.
func (r *RibFeature) EditableParams() []EditableParam {
	return []EditableParam{
		scalarParam("Thickness", param.Length, &r.def.Thickness),
		scalarParam("Depth", param.Length, &r.def.Depth),
	}
}

// EditableParams exposes the emboss depth.
func (e *EmbossFeature) EditableParams() []EditableParam {
	return []EditableParam{scalarParam("Depth", param.Length, &e.def.Depth)}
}

// EditableParams exposes the fillet radius.
func (f *FilletFeature) EditableParams() []EditableParam {
	return []EditableParam{scalarParam("Radius", param.Length, &f.def.Radius)}
}

// EditableParams exposes the chamfer distance.
func (c *ChamferFeature) EditableParams() []EditableParam {
	ps := []EditableParam{scalarParam("Distance", param.Length, &c.def.Distance)}
	switch c.def.Type {
	case types.ChamferTwoDistances:
		ps = append(ps, scalarParam("Distance 2", param.Length, &c.def.Distance2))
	case types.ChamferDistanceAndAngle:
		ps = append(ps, scalarParam("Angle", param.Angle, &c.def.Angle))
	}
	return ps
}

// EditableParams exposes the shell wall thickness.
func (s *ShellFeature) EditableParams() []EditableParam {
	return []EditableParam{scalarParam("Thickness", param.Length, &s.def.Thickness)}
}

// EditableParams exposes the draft angle.
func (d *FaceDraftFeature) EditableParams() []EditableParam {
	return []EditableParam{scalarParam("Angle", param.Angle, &d.def.Angle)}
}

// EditableParams exposes a hole's diameter, depth (blind only), and recess inputs by type.
func (h *HoleFeature) EditableParams() []EditableParam {
	ps := []EditableParam{scalarParam("Diameter", param.Length, &h.def.Diameter)}
	if !h.def.ThroughAll {
		ps = append(ps, scalarParam("Depth", param.Length, &h.def.Depth))
	}
	switch h.def.Type {
	case CounterboreHole:
		ps = append(ps,
			scalarParam("Counterbore Ø", param.Length, &h.def.CounterDiameter),
			scalarParam("Counterbore Depth", param.Length, &h.def.CounterDepth))
	case CountersinkHole:
		ps = append(ps,
			scalarParam("Countersink Ø", param.Length, &h.def.CounterDiameter),
			scalarParam("Countersink Angle", param.Angle, &h.def.CounterAngle))
	}
	return ps
}

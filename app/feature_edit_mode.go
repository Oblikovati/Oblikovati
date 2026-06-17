// SPDX-License-Identifier: GPL-2.0-only

package app

import (
	"errors"

	"oblikovati.org/model/feature"
)

// Edit mode for the creation tools: double-clicking a feature whose tool can round-trip
// its definition re-opens the SAME tool (and therefore the same property panel) seeded
// from the committed feature — every property available at creation is available while
// editing. The tool's Commit then writes the state back into the definition instead of
// adding a new feature; Cancel restores a definition snapshot. Features whose panel
// cannot round-trip the definition (patterns, mirror, rib, emboss) keep the generic
// FeatureEditTool parameter/reference editor.

// featureEditMode binds a creation tool to the committed feature it re-edits. The zero
// value means the tool is creating a new feature. Embedded by the edit-capable tools.
type featureEditMode struct {
	target     *feature.PartFeature
	restoreDef func() // definition snapshot, applied on Cancel
}

// bindEdit puts the tool into edit mode over f, with restore capturing the definition
// state to reinstate on Cancel.
func (m *featureEditMode) bindEdit(f *feature.PartFeature, restore func()) {
	m.target = f
	m.restoreDef = restore
}

// IsEditing reports whether the tool re-edits a committed feature (vs creating one).
func (m *featureEditMode) IsEditing() bool { return m.target != nil }

// EditingName returns the edited feature's name for the panel breadcrumb, or "".
func (m *featureEditMode) EditingName() string {
	if m.target == nil {
		return ""
	}
	return m.target.Name()
}

// commitFeatureEdit finishes an in-place edit after the caller mutated the definition:
// it restores full history evaluation, recomputes with the new recipe, records the
// transaction, and surfaces a sick feature as an error so the panel stays open.
func commitFeatureEdit(s *Session, f *feature.PartFeature) error {
	part, err := activePart(s)
	if err != nil {
		return err
	}
	s.endEditScope() // restore full evaluation before the final rebuild
	part.Features().MarkDirty(f)
	part.Recompute()
	s.recordEdit(part, "Edit "+f.Name())
	s.Selection().SetFilter(NewSelectionFilter())
	if !f.Health().OK() {
		return errors.New("feature edit: " + f.Health().Reason)
	}
	return nil
}

// cancelFeatureEdit reverts an in-place edit: the caller-provided restore reinstates the
// definition snapshot, the edit scope ends, and the part rebuilds at the original recipe.
func cancelFeatureEdit(s *Session, f *feature.PartFeature, restore func()) {
	if restore != nil {
		restore()
	}
	s.endEditScope()
	if part, err := activePart(s); err == nil {
		part.Features().MarkDirty(f)
		part.Recompute()
	}
	s.Selection().SetFilter(NewSelectionFilter())
}

// editToolFor builds the creation tool re-opened over a committed feature, or false for
// feature kinds the generic editor still serves.
func editToolFor(s *Session, f *feature.PartFeature) (Tool, bool) {
	switch d := f.Definition().(type) {
	case *feature.ExtrudeFeature:
		return editExtrudeTool(f, d), true
	case *feature.RevolveFeature:
		return editRevolveTool(s, f, d), true
	case *feature.CoilFeature:
		return editCoilTool(s, f, d), true
	case *feature.HoleFeature:
		return editHoleTool(f, d), true
	case *feature.FilletFeature:
		return editFilletToolOr(f, d)
	case *feature.ChamferFeature:
		return editChamferTool(f, d), true
	case *feature.ShellFeature:
		return editShellTool(f, d), true
	case *feature.FaceDraftFeature:
		return editDraftTool(f, d), true
	default:
		return nil, false
	}
}

// editExtrudeTool seeds an ExtrudeTool from a committed extrude: profiles, operation,
// extent (type, direction, distances, asymmetry) and taper — the full creation surface.
func editExtrudeTool(f *feature.PartFeature, ext *feature.ExtrudeFeature) *ExtrudeTool {
	def := ext.Definition()
	t := NewExtrudeTool()
	for _, idx := range def.ProfileIndices {
		t.profiles = append(t.profiles, ProfileHandle{Sketch: def.Sketch, ProfileIndex: idx})
	}
	t.operation = def.Operation
	t.taper = def.Taper
	seedExtrudeExtent(t, def.Extent)
	t.bindEdit(f, snapshotExtrudeDef(def))
	return t
}

// seedExtrudeExtent copies a committed extent into the tool's option state.
func seedExtrudeExtent(t *ExtrudeTool, ext feature.Extent) {
	t.extent = ext.Type
	t.direction = ext.Direction
	if ext.Distance != nil {
		t.distance = ext.Distance()
	}
	if ext.Distance2 != nil {
		t.asymmetric = true
		t.distance2 = ext.Distance2()
	}
}

func snapshotExtrudeDef(def *feature.ExtrudeDefinition) func() {
	orig := *def
	orig.ProfileIndices = append([]int(nil), def.ProfileIndices...)
	return func() { *def = orig }
}

// editRevolveTool seeds a RevolveTool from a committed revolve: profile, axis (work
// axis, specific centerline, or the sketch's own), angle and operation.
func editRevolveTool(s *Session, f *feature.PartFeature, rv *feature.RevolveFeature) *RevolveTool {
	def := rv.Definition()
	t := NewRevolveTool()
	t.profile = &ProfileHandle{Sketch: def.Sketch, ProfileIndex: def.ProfileIndex}
	t.angle = callOrZeroFn(def.Angle)
	t.operation = def.Operation
	seedRevolveAxis(s, t, def)
	t.bindEdit(f, snapshotRevolveDef(def))
	return t
}

// seedRevolveAxis maps the definition's axis precedence back onto the tool: a specific
// centerline, an origin work axis (matched by identity), or the sketch's own centerline.
func seedRevolveAxis(s *Session, t *RevolveTool, def *feature.RevolveDefinition) {
	if def.AxisCenterline != nil {
		t.centerline, t.centerlineSk = def.AxisCenterline, def.AxisCenterlineSketch
		return
	}
	if def.Axis == nil {
		t.useCenterln = true
		return
	}
	part, err := activePart(s)
	if err != nil {
		return
	}
	for _, ref := range []feature.WorkRef{feature.OriginXAxis, feature.OriginYAxis, feature.OriginZAxis} {
		if a, ok := part.WorkGeometry().AxisByRef(ref); ok && a == def.Axis {
			t.axis = ref
			return
		}
	}
}

func snapshotRevolveDef(def *feature.RevolveDefinition) func() {
	orig := *def
	return func() { *def = orig }
}

// editCoilTool seeds a CoilTool from a committed coil: profile, axis, pitch,
// revolutions and operation.
func editCoilTool(s *Session, f *feature.PartFeature, c *feature.CoilFeature) *CoilTool {
	def := c.Definition()
	t := NewCoilTool()
	t.profile = &ProfileHandle{Sketch: def.Sketch, ProfileIndex: def.ProfileIndex}
	t.pitch = callOrZeroFn(def.Pitch)
	t.revolutions = callOrZeroFn(def.Revolutions)
	t.operation = def.Operation
	seedCoilAxis(s, t, def.Axis)
	t.bindEdit(f, snapshotCoilDef(def))
	return t
}

// seedCoilAxis matches the definition's work axis back to its origin reference.
func seedCoilAxis(s *Session, t *CoilTool, axis *feature.WorkAxis) {
	part, err := activePart(s)
	if err != nil || axis == nil {
		return
	}
	for _, ref := range []feature.WorkRef{feature.OriginXAxis, feature.OriginYAxis, feature.OriginZAxis} {
		if a, ok := part.WorkGeometry().AxisByRef(ref); ok && a == axis {
			t.axis = ref
			return
		}
	}
}

func snapshotCoilDef(def *feature.CoilDefinition) func() {
	orig := *def
	return func() { *def = orig }
}

// editHoleTool seeds a HoleTool from a committed hole: the placement-face key, the bore
// size/termination, the seat type with its dimensions, and the drill point.
func editHoleTool(f *feature.PartFeature, h *feature.HoleFeature) *HoleTool {
	def := h.Definition()
	t := NewHoleTool()
	t.seededFaceKey = append([]byte(nil), def.PlacementFaceKey...)
	t.diameter = callOrZeroFn(def.Diameter)
	t.depth = callOrZeroFn(def.Depth)
	t.through = def.ThroughAll
	t.pointAngle = callOrZeroFn(def.PointAngle)
	seedHoleSeat(t, def)
	t.bindEdit(f, snapshotHoleDef(def))
	return t
}

// seedHoleSeat copies the hole's seat profile (counterbore/countersink) into the tool.
func seedHoleSeat(t *HoleTool, def *feature.HoleDefinition) {
	t.counterbore = def.Type == feature.CounterboreHole
	t.countersink = def.Type == feature.CountersinkHole
	t.cDiameter = callOrZeroFn(def.CounterDiameter)
	t.cDepth = callOrZeroFn(def.CounterDepth)
	t.sinkAngle = callOrZeroFn(def.CounterAngle)
}

func snapshotHoleDef(def *feature.HoleDefinition) func() {
	orig := *def
	orig.PlacementFaceKey = append([]byte(nil), def.PlacementFaceKey...)
	return func() { *def = orig }
}

// editFilletTool / editChamferTool / editShellTool / editDraftTool seed the dress-up
// tools: the definition's reference keys are retained as the seeded selection (live
// handles cannot exist — the feature consumed that geometry), so the panel shows the
// count, the clear empties it, and new viewport picks extend it.
// editFilletToolOr opens the fillet panel over the feature, or defers to the generic
// editor for shapes the single-radius panel can't show (multi-set fillets).
func editFilletToolOr(f *feature.PartFeature, fl *feature.FilletFeature) (Tool, bool) {
	if t := editFilletTool(f, fl); t != nil {
		return t, true
	}
	return nil, false
}

func editFilletTool(f *feature.PartFeature, fl *feature.FilletFeature) *FilletTool {
	def := fl.Definition()
	if len(def.EdgeSets) > 1 {
		return nil // a multi-set fillet doesn't fit the single-radius panel
	}
	t := NewFilletTool()
	t.cornerType = def.CornerType
	t.concaveStrategy = def.ConcaveStrategy
	if len(def.EdgeSets) == 1 {
		seedFilletSet(t, def.EdgeSets[0])
	} else {
		t.seededEdgeKeys = cloneKeys(def.EdgeKeys)
		t.radius = callOrZeroFn(def.Radius)
	}
	t.bindEdit(f, snapshotFilletDef(def))
	return t
}

// seedFilletSet seeds the panel from a single edge set: constant (Radius) or variable
// (start→end, #323).
func seedFilletSet(t *FilletTool, set feature.FilletEdgeSet) {
	t.seededEdgeKeys = cloneKeys(set.EdgeKeys)
	if set.Radius != nil {
		t.radius = callOrZeroFn(set.Radius)
		return
	}
	t.variable = true
	t.startRadius = callOrZeroFn(set.StartRadius)
	t.endRadius = callOrZeroFn(set.EndRadius)
}

func snapshotFilletDef(def *feature.FilletDefinition) func() {
	orig := *def
	orig.EdgeKeys = cloneKeys(def.EdgeKeys)
	return func() { *def = orig }
}

func editChamferTool(f *feature.PartFeature, c *feature.ChamferFeature) *ChamferTool {
	def := c.Definition()
	t := NewChamferTool()
	t.seededEdgeKeys = cloneKeys(def.EdgeKeys)
	t.distance = callOrZeroFn(def.Distance)
	t.flatCorners = def.FlatCorners
	t.concaveStrategy = def.ConcaveStrategy
	t.bindEdit(f, snapshotChamferDef(def))
	return t
}

func snapshotChamferDef(def *feature.ChamferDefinition) func() {
	orig := *def
	orig.EdgeKeys = cloneKeys(def.EdgeKeys)
	return func() { *def = orig }
}

func editShellTool(f *feature.PartFeature, sh *feature.ShellFeature) *ShellTool {
	def := sh.Definition()
	t := NewShellTool()
	t.seededFaceKeys = cloneKeys(def.RemovedFaceKeys)
	t.thickness = callOrZeroFn(def.Thickness)
	t.bindEdit(f, snapshotShellDef(def))
	return t
}

func snapshotShellDef(def *feature.ShellDefinition) func() {
	orig := *def
	orig.RemovedFaceKeys = cloneKeys(def.RemovedFaceKeys)
	return func() { *def = orig }
}

func editDraftTool(f *feature.PartFeature, d *feature.FaceDraftFeature) *DraftTool {
	def := d.Definition()
	t := NewDraftTool()
	t.seededFaceKeys = cloneKeys(def.FaceKeys)
	t.angleDeg = callOrZeroFn(def.Angle) / degToRad
	t.bindEdit(f, snapshotDraftDef(def))
	return t
}

func snapshotDraftDef(def *feature.FaceDraftDefinition) func() {
	orig := *def
	orig.FaceKeys = cloneKeys(def.FaceKeys)
	return func() { *def = orig }
}

// cloneKeys deep-copies a reference-key list so a snapshot survives later mutation.
func cloneKeys(keys [][]byte) [][]byte {
	out := make([][]byte, len(keys))
	for i, k := range keys {
		out[i] = append([]byte(nil), k...)
	}
	return out
}

// callOrZeroFn evaluates a parameter closure, treating nil as zero.
func callOrZeroFn(f func() float64) float64 {
	if f == nil {
		return 0
	}
	return f()
}

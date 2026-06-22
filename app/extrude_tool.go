// SPDX-License-Identifier: GPL-2.0-only

package app

import (
	"errors"

	"oblikovati.org/kernel/ops"
	"oblikovati.org/math"
	"oblikovati.org/model/compdef"
	"oblikovati.org/model/feature"
	"oblikovati.org/model/sketch"
	"oblikovati.org/renderer"
)

// ExtrudeTool is the interactive Extrude command: activate it, hover a sketch region
// (a closed area) and click to pick it, Ctrl+click more regions to extrude together,
// set a distance, and OK to add an extrude feature to the active part — the full
// Inventor extrude flow, driven entirely by session input so a test exercises it with
// synthetic clicks. It is the worked example proving geometry flows end to end.
type ExtrudeTool struct {
	featureEditMode // set ⇒ this panel re-edits a committed extrude (see editExtrudeTool)
	profiles        []ProfileHandle
	distance        float64 // primary depth, model units
	distance2       float64 // asymmetric second-direction depth, model units
	taper           float64 // draft angle, radians
	extent          feature.ExtentType
	direction       feature.ExtentDirection
	asymmetric      bool
	operation       ops.PartFeatureOperation
	added           *feature.PartFeature
	toPlane         *feature.WorkPlane // "to face" termination target (a work plane or a face's plane)
	toFace          *FaceHandle        // the picked termination face, for the highlight (nil for a work plane)
}

// NewExtrudeTool returns an extrude tool defaulting to a positive distance extrusion that
// creates a new body.
func NewExtrudeTool() *ExtrudeTool {
	return &ExtrudeTool{operation: ops.NewBody, extent: feature.DistanceExtent, direction: feature.PositiveDir}
}

// Name implements [Tool].
func (t *ExtrudeTool) Name() string { return "Extrude" }

// Start is a no-op; the engine installs the filter from AcceptedKinds.
func (t *ExtrudeTool) Start(*Session) {}

// AcceptedKinds declares extrude's selection per step: it picks sketch regions (profiles), and
// once at least one region is picked AND the extent is "to face", it switches to picking the
// termination face (or work plane) the extrude runs up to — the engine re-derives this after each
// pick, so the same tool drives both steps with no manual filter juggling.
func (t *ExtrudeTool) AcceptedKinds() []SelectionKind {
	if t.extent == feature.ToFaceExtent && len(t.profiles) > 0 {
		return []SelectionKind{SelectFace, SelectWorkPlane}
	}
	return []SelectionKind{SelectProfile}
}

// Picks reports the picked regions plus the termination face for the unified highlight.
func (t *ExtrudeTool) Picks() []Selectable {
	return appendPick(profileSelectables(t.profiles), t.toFace)
}

// Pick captures the region the user clicked, or — during the "to face" step — the termination
// face/work-plane the extrude runs up to. A plain region click selects a single region.
func (t *ExtrudeTool) Pick(_ *Session, sel Selectable) {
	switch h := sel.(type) {
	case ProfileHandle:
		t.profiles = []ProfileHandle{h}
	case FaceHandle:
		if pl, ok := sketchPlaneFromFace(h); ok {
			t.toPlane, t.toFace = feature.NewFixedWorkPlane(pl), &h
		}
	case WorkPlaneHandle:
		t.toPlane = h.Plane
	}
}

// PickWithMods extends the selection on Ctrl+click — adding the region, or removing it
// if it was already picked (toggle) — and replaces it otherwise. This is how the user
// gathers several closed paths into one extrusion.
func (t *ExtrudeTool) PickWithMods(s *Session, sel Selectable, mods Modifier) {
	p, ok := sel.(ProfileHandle)
	if !ok {
		t.Pick(s, sel) // a "to face" termination pick (face/work-plane) — not a region
		return
	}
	if !mods.Has(CtrlMod) {
		t.Pick(s, sel)
		return
	}
	if i := indexOfProfile(t.profiles, p); i >= 0 {
		t.profiles = append(t.profiles[:i], t.profiles[i+1:]...)
		return
	}
	t.profiles = append(t.profiles, p)
}

// ClearProfiles empties the picked-profile selection — the property panel's selector
// clear (⊗) affordance, returning the tool to its select-a-region step.
func (t *ExtrudeTool) ClearProfiles() { t.profiles = nil }

// SourceSketchName returns the sketch the picked profiles come from, for the property
// panel's breadcrumb and From row; "" until a profile is picked.
func (t *ExtrudeTool) SourceSketchName() string {
	if len(t.profiles) == 0 {
		return ""
	}
	return t.profiles[0].Sketch.Name()
}

// indexOfProfile returns the position of p in handles, or -1. ProfileHandle is
// comparable (sketch pointer + region index), so equality identifies the same region.
func indexOfProfile(handles []ProfileHandle, p ProfileHandle) int {
	for i, h := range handles {
		if h == p {
			return i
		}
	}
	return -1
}

// SetDistance sets the extrusion distance (the value the in-canvas field / spinner
// would set).
func (t *ExtrudeTool) SetDistance(d float64) { t.distance = d }

// Distance returns the current extrusion distance (database units).
func (t *ExtrudeTool) Distance() float64 { return t.distance }

// The extrude options the dialog drives: extent type, direction, the asymmetric
// second-direction depth, and the draft taper. All values are database units / radians.
func (t *ExtrudeTool) SetExtentType(e feature.ExtentType)     { t.extent = e }
func (t *ExtrudeTool) ExtentType() feature.ExtentType         { return t.extent }
func (t *ExtrudeTool) SetDirection(d feature.ExtentDirection) { t.direction = d }
func (t *ExtrudeTool) Direction() feature.ExtentDirection     { return t.direction }
func (t *ExtrudeTool) SetAsymmetric(on bool)                  { t.asymmetric = on }
func (t *ExtrudeTool) Asymmetric() bool                       { return t.asymmetric }
func (t *ExtrudeTool) SetSecondDistance(d float64)            { t.distance2 = d }
func (t *ExtrudeTool) SecondDistance() float64                { return t.distance2 }
func (t *ExtrudeTool) SetTaper(radians float64)               { t.taper = radians }
func (t *ExtrudeTool) Taper() float64                         { return t.taper }

// needsDistance reports whether the current extent is gauged by the distance field (vs.
// measured from the model, like through-all / to-next).
func (t *ExtrudeTool) needsDistance() bool {
	return t.extent == feature.DistanceExtent || t.extent == feature.DistanceFromFaceExtent
}

// buildExtent assembles the model extent from the tool's option state.
func (t *ExtrudeTool) buildExtent() feature.Extent {
	ext := feature.Extent{Type: t.extent, Direction: t.direction}
	if t.needsDistance() {
		d := t.distance
		ext.Distance = func() float64 { return d }
		if t.asymmetric {
			d2 := t.distance2
			ext.Distance2 = func() float64 { return d2 }
		}
	}
	if t.extent == feature.ToFaceExtent {
		ext.ToPlane = t.toPlane
	}
	return ext
}

// PickedProfiles returns the regions the user has picked (in click order), for the UI
// to highlight. Empty until the first pick.
func (t *ExtrudeTool) PickedProfiles() []ProfileHandle {
	return append([]ProfileHandle(nil), t.profiles...)
}

// PickedProfile returns the first picked region (and true), or false when none has been
// picked yet — a convenience for single-region UI/tests.
func (t *ExtrudeTool) PickedProfile() (ProfileHandle, bool) {
	if len(t.profiles) == 0 {
		return ProfileHandle{}, false
	}
	return t.profiles[0], true
}

// SetOperation chooses join/cut/intersect/new-body; Operation returns the current choice.
func (t *ExtrudeTool) SetOperation(op ops.PartFeatureOperation) { t.operation = op }
func (t *ExtrudeTool) Operation() ops.PartFeatureOperation      { return t.operation }

// CanCommit reports whether enough input is gathered: at least one region, a non-zero distance
// for a distance-gauged extent, and a termination target for a "to face" extent.
func (t *ExtrudeTool) CanCommit() bool {
	if len(t.profiles) == 0 {
		return false
	}
	if t.needsDistance() && t.distance == 0 {
		return false
	}
	if t.extent == feature.ToFaceExtent && t.toPlane == nil {
		return false
	}
	return true
}

// Commit adds the extrude feature to the active part and recomputes; a sick feature
// (e.g. an open profile) keeps the tool open by returning an error. All picked regions
// must lie on one sketch (a single extrude consumes one sketch's regions); picks on
// other sketches are ignored.
func (t *ExtrudeTool) Commit(s *Session) error {
	if t.IsEditing() {
		return t.commitEdit(s)
	}
	part, err := activePart(s)
	if err != nil {
		return err
	}
	def, _ := t.draftDefinition() // CanCommit (checked by the dialog) guarantees ok
	t.added = feature.NewExtrudeFeatures(part.Features()).AddExtrudeFeature(def)
	part.Recompute()
	s.recordEdit(part, "Extrude")
	if !t.added.Health().OK() {
		return errors.New("extrude: " + t.added.Health().Reason)
	}
	return nil
}

// commitEdit writes the panel state back into the committed extrude's definition —
// the same properties the create path passes to AddExtrude.
func (t *ExtrudeTool) commitEdit(s *Session) error {
	skt := t.profiles[0].Sketch
	def := t.target.Definition().(*feature.ExtrudeFeature).Definition()
	def.Sketch, def.ProfileIndices = skt, profileIndicesOn(t.profiles, skt)
	def.Operation, def.Extent, def.Taper = t.operation, t.buildExtent(), t.taper
	return commitFeatureEdit(s, t.target)
}

// profileIndicesOn returns the region indices among handles that belong to sketch skt.
func profileIndicesOn(handles []ProfileHandle, skt *sketch.Sketch) []int {
	var out []int
	for _, h := range handles {
		if h.Sketch == skt {
			out = append(out, h.ProfileIndex)
		}
	}
	return out
}

// AddedFeature returns the feature created on commit (for inspection/tests).
func (t *ExtrudeTool) AddedFeature() *feature.PartFeature { return t.added }

// draftDefinition builds the ExtrudeDefinition from the tool's current state — the single
// source of truth shared by Commit and the live preview, so the previewed solid is exactly
// the geometry OK will create. ok is false until at least one region is picked.
func (t *ExtrudeTool) draftDefinition() (*feature.ExtrudeDefinition, bool) {
	if len(t.profiles) == 0 {
		return nil, false
	}
	skt := t.profiles[0].Sketch
	return &feature.ExtrudeDefinition{
		Sketch: skt, ProfileIndices: profileIndicesOn(t.profiles, skt),
		Operation: t.operation, Extent: t.buildExtent(), Taper: t.taper,
	}, true
}

// DraftFeature returns the unattached extrude feature the viewport previews before commit
// (satisfying DraftPreviewable), and whether enough input is gathered to show it — the same
// gate as CanCommit, so the translucent preview appears exactly when OK becomes available.
func (t *ExtrudeTool) DraftFeature(*Session) (feature.Feature, bool) {
	def, ok := t.draftDefinition()
	if !ok || !t.CanCommit() {
		return nil, false
	}
	return feature.NewExtrudeFeature(def), true
}

// Prompt guides the user through the extrude steps (Inventor's status-bar prompts).
func (t *ExtrudeTool) Prompt(*Session) string {
	switch {
	case len(t.profiles) == 0:
		return "Select a region to extrude (Ctrl+click to add more)"
	case t.needsDistance() && t.distance == 0:
		return "Specify the extrude distance"
	default:
		return "Set the extrude options and click OK"
	}
}

// SubmitToken accepts a typed height value from the command line (M26); the region itself
// is picked in the viewport, and the step prompt is the tool's existing Prompt().
func (t *ExtrudeTool) SubmitToken(_ *Session, tok CommandToken) error {
	if tok.Kind != ValueToken {
		return errors.New("extrude: expected a height value (pick the region in the viewport)")
	}
	t.distance = tok.Value
	return nil
}

// Preview returns a transient wireframe of the prisms the tool will create — each
// picked region's bottom and top loops plus vertical connectors — so the viewport
// shows a live preview before OK (Inventor's in-canvas preview). Empty until a region
// and distance are set.
func (t *ExtrudeTool) Preview(*Session) []renderer.DrawItem {
	// Preview the swept prism only for a distance extent; model-gauged extents
	// (through-all / to-next) have no fixed depth to draw until commit.
	if len(t.profiles) == 0 || !t.needsDistance() || t.distance == 0 {
		return nil
	}
	var items []renderer.DrawItem
	for _, ph := range t.profiles {
		if ph.ProfileIndex >= ph.Sketch.Profiles().Count() {
			continue
		}
		poly := ph.Sketch.Profiles().Item(ph.ProfileIndex).OuterLoop().Polygon()
		items = append(items, prismWireframe(ph.Sketch.Plane(), poly, t.distance))
	}
	return items
}

// prismWireframe builds the line-primitive draw item outlining the prism a region's
// polygon sweeps to distance dist along the sketch plane normal.
func prismWireframe(plane sketch.Plane, poly []math.Point2, dist float64) renderer.DrawItem {
	up := plane.Normal().AsVector().Scale(dist)
	n := len(poly)
	pts := make([]math.Point3, 0, 2*n)
	for _, p := range poly { // bottom ring [0,n)
		pts = append(pts, plane.ToModel(p))
	}
	for i := 0; i < n; i++ { // top ring [n,2n)
		pts = append(pts, pts[i].TranslateBy(up))
	}
	var idx []int
	for i := 0; i < n; i++ {
		j := (i + 1) % n
		idx = append(idx, i, j)     // bottom loop
		idx = append(idx, n+i, n+j) // top loop
		idx = append(idx, i, n+i)   // vertical
	}
	return renderer.DrawItem{Primitive: renderer.Lines, Positions: pts, Indices: idx, Color: [4]float32{1, 0.6, 0, 1}}
}

// Cancel restores the default selection filter.
func (t *ExtrudeTool) Cancel(s *Session) {
	if t.IsEditing() {
		cancelFeatureEdit(s, t.target, t.restoreDef)
		return
	}
}

// activePart returns the active document's part component definition, or an error.
func activePart(s *Session) (*compdef.PartComponentDefinition, error) {
	d := s.ActiveDocument()
	if d == nil {
		return nil, errors.New("app: no active document")
	}
	part, ok := d.Content().(*compdef.PartComponentDefinition)
	if !ok {
		return nil, errors.New("app: active document is not a part")
	}
	return part, nil
}

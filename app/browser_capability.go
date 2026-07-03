// SPDX-License-Identifier: GPL-2.0-only

package app

// Browser selection-handle capabilities (audit I7, #1630). head/ui used to switch on the
// concrete handle type to decide a browser node's editable name, whether it may be renamed,
// and its double-click action — so every new selectable kind had to be hand-added to each
// switch, and a kind missing from one switch lost that behavior silently (the #1521/#1574
// shape). Here each handle self-describes instead: head/ui consumes the capability and never
// names a concrete handle type. This mirrors the house precedent model/sketch.pointDefined
// (each entity declares its own, so the consumer needs no type switch) and the context-menu
// data already returned by BrowserMenu — the rename/activate verbs take the *Session
// explicitly (not a hidden closure), the same app-internal dispatch pattern BrowserMenuItem
// uses, so a handle stays unit-testable against a real session.

// NodeRenameable is a selection handle that carries its own browser display name and, unless
// it is a fixed-name datum (the origin coordinate-system frame), renames itself through the
// session. head/ui renders the in-place rename affordance through this capability rather than
// switching on concrete handle types (#1630, #1264). Handles that are never renamed simply do
// not implement it, so nodeName falls back to the row label.
type NodeRenameable interface {
	Selectable
	// NodeName is the handle's current editable name (the raw name, without any status badge).
	NodeName() string
	// Renameable reports whether the in-place rename targets this node — false for the
	// grounded origin datums, whose names are fixed.
	Renameable() bool
	// Rename applies a new name; the session enforces a non-empty, document-unique name and
	// keeps the stable id.
	Rename(s *Session, name string) error
}

// NodeActivatable is a selection handle that performs the browser's double-click action
// (Inventor's edit-on-double-click): a feature opens its parameter editor, a sketch re-enters
// the sketch environment, an occurrence opens its placed document. head/ui invokes the
// capability instead of switching on concrete handle types (#1630). A handle with no
// double-click action does not implement it.
type NodeActivatable interface {
	Selectable
	// Activate runs the node's double-click action against the session.
	Activate(s *Session)
}

// Compile-time coverage: every browser-renameable / -activatable handle kind must implement
// its capability, so a new kind that forgets the method is a build break here, not a silently
// blank browser row (#1630). The runtime parity checks live in browser_capability_test.go.
var (
	_ NodeRenameable = FeatureHandle{}
	_ NodeRenameable = SketchHandle{}
	_ NodeRenameable = Sketch3DHandle{}
	_ NodeRenameable = WorkPlaneHandle{}
	_ NodeRenameable = WorkAxisHandle{}
	_ NodeRenameable = WorkPointHandle{}

	_ NodeActivatable = FeatureHandle{}
	_ NodeActivatable = AssemblyFeatureHandle{}
	_ NodeActivatable = SketchHandle{}
	_ NodeActivatable = WorkPlaneHandle{}
	_ NodeActivatable = OccurrenceHandle{}
	_ NodeActivatable = RepresentationHandle{}
	_ NodeActivatable = ModelStateHandle{}
	_ NodeActivatable = DrawingViewHandle{}
)

// --- NodeRenameable: features and sketches are always renameable ---

func (h FeatureHandle) NodeName() string                  { return h.Feature.Name() }
func (h FeatureHandle) Renameable() bool                  { return true }
func (h FeatureHandle) Rename(s *Session, n string) error { return s.RenameFeature(h.Feature, n) }

func (h SketchHandle) NodeName() string                  { return h.Sketch.Name() }
func (h SketchHandle) Renameable() bool                  { return true }
func (h SketchHandle) Rename(s *Session, n string) error { return s.RenameSketch(h.Sketch, n) }

func (h Sketch3DHandle) NodeName() string                  { return h.Sketch3D.Name() }
func (h Sketch3DHandle) Renameable() bool                  { return true }
func (h Sketch3DHandle) Rename(s *Session, n string) error { return s.RenameSketch3D(h.Sketch3D, n) }

// --- NodeRenameable: work datums are renameable unless they are origin-frame elements ---

func (h WorkPlaneHandle) NodeName() string { return h.Plane.Name() }
func (h WorkPlaneHandle) Renameable() bool { return !h.Plane.IsCoordinateSystemElement() }
func (h WorkPlaneHandle) Rename(s *Session, n string) error {
	return s.RenameWorkPlane(h.Plane, n)
}

func (h WorkAxisHandle) NodeName() string { return h.Axis.Name() }
func (h WorkAxisHandle) Renameable() bool { return !h.Axis.IsCoordinateSystemElement() }
func (h WorkAxisHandle) Rename(s *Session, n string) error {
	return s.RenameWorkAxis(h.Axis, n)
}

func (h WorkPointHandle) NodeName() string { return h.Point.Name() }
func (h WorkPointHandle) Renameable() bool { return !h.Point.IsCoordinateSystemElement() }
func (h WorkPointHandle) Rename(s *Session, n string) error {
	return s.RenameWorkPoint(h.Point, n)
}

// --- NodeActivatable: each handle knows its own double-click action ---

func (h FeatureHandle) Activate(s *Session)         { s.BeginEditFeature(h) }
func (h AssemblyFeatureHandle) Activate(s *Session) { s.BeginEditAssemblyFeature(h) }
func (h SketchHandle) Activate(s *Session)          { s.BeginEditSketch(h) }
func (h WorkPlaneHandle) Activate(s *Session)       { s.BeginEditWorkPlane(h) }
func (h OccurrenceHandle) Activate(s *Session)      { _ = s.OpenOccurrenceDocument(h.Occurrence) }
func (h RepresentationHandle) Activate(s *Session)  { _ = s.ActivateRepresentation(h) }
func (h ModelStateHandle) Activate(s *Session)      { _ = s.ActivateModelState(h) }
func (h DrawingViewHandle) Activate(s *Session)     { s.BeginEditDrawingView(h) }

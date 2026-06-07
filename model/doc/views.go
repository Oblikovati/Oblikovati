// SPDX-License-Identifier: GPL-2.0-only

package doc

import (
	"errors"
	"fmt"
	stdmath "math"

	"oblikovati/api/types"
	"oblikovati/math"
)

// ErrLastView is returned by [DocumentViews.Close] when asked to remove the only view —
// a document must always retain at least one view.
var ErrLastView = errors.New("doc: cannot close the last view of a document")

// viewIndexError reports an out-of-range view index, naming the operation and bounds.
func viewIndexError(op string, i, n int) error {
	return fmt.Errorf("doc: %s view index %d out of range [0,%d)", op, i, n)
}

// View is one view of a document: a named camera frame (Inventor's View.Camera). A
// document owns a collection of views and each has its own camera, which is why
// switching documents (or views) restores that view's camera rather than resetting it.
//
// The frame is stored as math value types (not the renderer's scene.Camera) so the model
// layer carries no rendering dependency; the session converts to/from scene.Camera and
// supplies the transient viewport pixel size. Eye/Target are points, Up a vector, FOV the
// vertical field of view in radians.
type View struct {
	Name   string
	Eye    math.Point3
	Target math.Point3
	Up     math.Vector3
	FOV    float64
	// Framed reports whether this view's camera has been fitted to the model yet. A
	// brand-new default view starts unframed so the UI Home-fits it the first time it is
	// shown; a saved/loaded or user-navigated view is framed, so switching to it restores
	// its camera instead of resetting the view (the per-document camera fix).
	Framed bool
	// Projection is the view's camera projection (ViewCube projection menu). Per-view, so
	// each tile can be perspective or orthographic independently. Defaults to perspective.
	Projection ProjectionMode
	// Home is this view's custom Home camera (ViewCube "Set Current View as Home"); nil ⇒
	// the default iso Home. Go Home / the Home button restore it.
	Home *ViewHome
}

// ViewHome is a view's saved Home camera. FitToView keeps only the viewing direction,
// re-fitting to the model extents on each Go Home (Inventor's "Fit to View"); otherwise
// the exact framing (Fixed Distance) is restored.
type ViewHome struct {
	Eye       math.Point3
	Target    math.Point3
	Up        math.Vector3
	FOV       float64
	FitToView bool
}

// ProjectionMode is a view's camera projection, matching Inventor's ViewCube projection
// menu (Orthographic / Perspective / Perspective with Ortho Faces).
type ProjectionMode int

const (
	// ProjPerspective is FOV perspective (the default).
	ProjPerspective ProjectionMode = iota
	// ProjOrthographic is a parallel projection (no foreshortening).
	ProjOrthographic
	// ProjPerspectiveOrthoFaces is perspective, switching to orthographic only when the
	// view is aligned to a principal axis (a ViewCube face view).
	ProjPerspectiveOrthoFaces
)

// IsValid reports whether m is a defined projection mode.
func (m ProjectionMode) IsValid() bool {
	return m >= ProjPerspective && m <= ProjPerspectiveOrthoFaces
}

// DefaultView is the framing a brand-new view starts at (matching scene.NewCamera): an
// eye on +Z looking at the origin, Y up, 45° vertical FOV. The session re-frames it to
// the model the first time the view is shown.
func DefaultView(name string) *View {
	return &View{
		Name:   name,
		Eye:    math.P3(0, 0, 10),
		Target: math.P3(0, 0, 0),
		Up:     math.V3(0, 1, 0),
		FOV:    stdmath.Pi / 4,
	}
}

// DocumentViews is a document's view collection (Inventor's Document.Views): the ordered
// views, which one is active, and how they tile in the viewport. A document always has at
// least one view — the zero collection lazily seeds a default via [Document.Views].
type DocumentViews struct {
	views  []*View
	active int
	layout types.ViewLayout
	// splitX/splitY are the divider positions (0..1) for split layouts: splitX is the
	// vertical divider (left|right), splitY the horizontal one (top/bottom). Zero means
	// "use the default 0.5" so a freshly-seeded collection needs no initialization.
	splitX float32
	splitY float32
	front  CubeOrient // ViewCube orientation (Set/Reset Front); zero value ⇒ identity
}

// Front returns the document's ViewCube orientation, defaulting to the identity (the
// un-redefined front) for a zero/unset value.
func (vs *DocumentViews) Front() CubeOrient {
	if vs.front == (CubeOrient{}) {
		return IdentityCubeOrient()
	}
	return vs.front
}

// SetFront redefines (or resets, with IdentityCubeOrient) the document's ViewCube front.
func (vs *DocumentViews) SetFront(o CubeOrient) { vs.front = o }

const minSplit, maxSplit = 0.1, 0.9

// Split returns the divider positions (0..1), defaulting an unset (zero) value to centre.
func (vs *DocumentViews) Split() (x, y float32) {
	x, y = vs.splitX, vs.splitY
	if x == 0 {
		x = 0.5
	}
	if y == 0 {
		y = 0.5
	}
	return x, y
}

// SetSplit sets the divider positions, clamped to [0.1, 0.9] so no tile collapses.
func (vs *DocumentViews) SetSplit(x, y float32) {
	vs.splitX = clampSplit(x)
	vs.splitY = clampSplit(y)
}

func clampSplit(v float32) float32 {
	if v < minSplit {
		return minSplit
	}
	if v > maxSplit {
		return maxSplit
	}
	return v
}

// Views returns the document's view collection, seeding a single default view on first
// use so a document is never viewless (Inventor: a document always has ≥1 view).
func (d *Document) Views() *DocumentViews {
	if d.views == nil {
		d.views = &DocumentViews{views: []*View{DefaultView("View 1")}, layout: types.LayoutSingle}
	}
	return d.views
}

// RestoreViews replaces the document's collection with views loaded from disk (each is
// already framed), the active index, and the layout. An empty set is ignored so the lazy
// default view stands; an out-of-range active index is clamped. Used by persistence on open.
func (d *Document) RestoreViews(views []*View, active int, layout types.ViewLayout) {
	if len(views) == 0 {
		return
	}
	if active < 0 || active >= len(views) {
		active = 0
	}
	if !layout.IsValid() {
		layout = types.LayoutSingle
	}
	d.views = &DocumentViews{views: views, active: active, layout: layout}
}

// All returns the views in order (do not mutate the slice).
func (vs *DocumentViews) All() []*View { return vs.views }

// Count is the number of views.
func (vs *DocumentViews) Count() int { return len(vs.views) }

// ActiveIndex is the index of the active view.
func (vs *DocumentViews) ActiveIndex() int { return vs.active }

// Active returns the active view (never nil — the collection always holds ≥1 view).
func (vs *DocumentViews) Active() *View { return vs.views[vs.active] }

// Layout returns the current tiling layout.
func (vs *DocumentViews) Layout() types.ViewLayout { return vs.layout }

// SetLayout sets the tiling layout, ignoring an undefined value.
func (vs *DocumentViews) SetLayout(l types.ViewLayout) {
	if l.IsValid() {
		vs.layout = l
	}
}

// Add appends v and makes it the active view, returning its index.
func (vs *DocumentViews) Add(v *View) int {
	vs.views = append(vs.views, v)
	vs.active = len(vs.views) - 1
	return vs.active
}

// Activate makes the view at index i active. Out-of-range is an error.
func (vs *DocumentViews) Activate(i int) error {
	if i < 0 || i >= len(vs.views) {
		return viewIndexError("activate", i, len(vs.views))
	}
	vs.active = i
	return nil
}

// Rename sets the name of the view at index i.
func (vs *DocumentViews) Rename(i int, name string) error {
	if i < 0 || i >= len(vs.views) {
		return viewIndexError("rename", i, len(vs.views))
	}
	vs.views[i].Name = name
	return nil
}

// Close removes the view at index i. Closing the last remaining view is refused (a
// document always has ≥1 view); the active index is kept in range.
func (vs *DocumentViews) Close(i int) error {
	if i < 0 || i >= len(vs.views) {
		return viewIndexError("close", i, len(vs.views))
	}
	if len(vs.views) == 1 {
		return ErrLastView
	}
	vs.views = append(vs.views[:i], vs.views[i+1:]...)
	if vs.active >= len(vs.views) {
		vs.active = len(vs.views) - 1
	}
	return nil
}

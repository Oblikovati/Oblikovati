// SPDX-License-Identifier: GPL-2.0-only

package feature

import (
	"fmt"

	"oblikovati.org/kernel/topo"
	"oblikovati.org/model/internal/collview"
)

// WorkSurface construction surfaces (M20-F16, #654… see #650). A surface-output feature
// (a surface extrude/revolve/loft/sweep, a boundary patch, a knit/stitch, a ruled or
// trim/extend result) emits an open sheet body rather than a solid. Those sheet bodies are
// gathered here as named, visibility-controlled WorkSurfaces, so they can be addressed,
// hidden, renamed, and (later) consumed by reference — mirroring how WorkPlanes wraps datum
// planes. WorkSurfaces is a work-feature-like collection (produced by features, no Add of
// its own), not a feature triangle.

// WorkSurface is one named construction surface wrapping a surface (sheet) body.
type WorkSurface struct {
	name        string
	visible     bool
	translucent bool
	body        *topo.Body // current geometry, refreshed each Sync (body identity is not stable)
	source      string     // best-effort name of the producing feature (may be empty)
}

// Name / Visible / Translucent / Body / Source expose the surface's state.
func (w *WorkSurface) Name() string          { return w.name }
func (w *WorkSurface) Visible() bool         { return w.visible }
func (w *WorkSurface) Translucent() bool     { return w.translucent }
func (w *WorkSurface) Body() *topo.Body      { return w.body }
func (w *WorkSurface) Source() string        { return w.source }
func (w *WorkSurface) SetVisible(v bool)     { w.visible = v }
func (w *WorkSurface) SetTranslucent(v bool) { w.translucent = v }

// SetName renames the surface; an empty name is rejected so the browser always has a label.
func (w *WorkSurface) SetName(name string) error {
	if name == "" {
		return fmt.Errorf("work surface name must not be empty")
	}
	w.name = name
	return nil
}

// WorkSurfaces is the part's ordered collection of construction surfaces. It is kept in
// sync with the model's surface bodies; display state (name/visibility/translucency) is
// keyed by position, so it survives a recompute that rebuilds the underlying body objects.
type WorkSurfaces struct {
	items []*WorkSurface
}

// NewWorkSurfaces returns an empty collection.
func NewWorkSurfaces() *WorkSurfaces { return &WorkSurfaces{} }

// Count / All / Item read the collection. Item returns nil for an out-of-range index.
func (c *WorkSurfaces) Count() int              { return len(c.items) }
func (c *WorkSurfaces) All() []*WorkSurface     { return c.items }
func (c *WorkSurfaces) Item(i int) *WorkSurface { return collview.At(c.items, i) }

// Sync reconciles the collection with the part's current result bodies: the surface
// (non-solid) bodies, in result order, become the work surfaces. A surviving position
// keeps its display state and just refreshes its body; a new position is auto-named
// "Surfacei"; trailing positions whose surfaces are gone are dropped.
func (c *WorkSurfaces) Sync(bodies []*topo.Body) {
	surfaces := surfaceBodiesOf(bodies)
	for i, b := range surfaces {
		if i < len(c.items) {
			c.items[i].body = b
			continue
		}
		c.items = append(c.items, &WorkSurface{name: fmt.Sprintf("Surface%d", i+1), visible: true, body: b})
	}
	if len(surfaces) < len(c.items) {
		c.items = c.items[:len(surfaces)]
	}
}

// HasName reports whether another surface (not the one at exclude) already uses name —
// the uniqueness guard a rename consults.
func (c *WorkSurfaces) HasName(name string, exclude int) bool {
	for i, w := range c.items {
		if i != exclude && w.name == name {
			return true
		}
	}
	return false
}

// surfaceBodiesOf returns the open (non-solid) sheet bodies of a result body list.
func surfaceBodiesOf(bodies []*topo.Body) []*topo.Body {
	var out []*topo.Body
	for _, b := range bodies {
		if b != nil && !b.IsSolid() {
			out = append(out, b)
		}
	}
	return out
}

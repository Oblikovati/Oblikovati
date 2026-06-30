// SPDX-License-Identifier: GPL-2.0-only

package app

import (
	"fmt"

	"oblikovati.org/api/wire"
)

// Nesting support for dockable-window control trees (grid/group/tabs, ADR-0019). The wire
// spec became a tree, so two flat-era assumptions are corrected here: edits must find a
// control at any depth, and the host must be guarded against a runaway/deep tree before it
// recurses while rendering.

// maxControlNestDepth bounds how deeply controls may nest. A real layout (tabs ▸ group ▸
// grid ▸ field) is ~4 deep; 16 is generous headroom while still a hard backstop so a buggy
// add-in can't stack-overflow the immediate-mode renderer. Top-level controls are depth 1.
const maxControlNestDepth = 16

// validateControlTree rejects a control tree the renderer could not safely draw: one nested
// past maxControlNestDepth, or one using the reserved RowSpan (ADR-0020 defers row span, so
// the host renders rows as content-height auto-flow and RowSpan must be 1). Messages name the
// offending window and value so the add-in author can fix it.
func validateControlTree(windowID string, controls []wire.PanelControlSpec, depth int) error {
	if depth > maxControlNestDepth {
		return fmt.Errorf("app: dockable window %q nests controls %d deep, max %d", windowID, depth, maxControlNestDepth)
	}
	for _, c := range controls {
		if c.Cell != nil && c.Cell.RowSpan > 1 {
			return fmt.Errorf("app: dockable window %q control %q uses RowSpan=%d (reserved; must be 1)", windowID, c.ID, c.Cell.RowSpan)
		}
		if err := validateControlTree(windowID, c.Children, depth+1); err != nil {
			return err
		}
	}
	return nil
}

// setControlValue updates, in place, the value of the control identified by id anywhere in
// the tree, returning whether it was found. IDs are unique within a window (the add-in's
// contract), so the first match is authoritative. Mutating through the shared slice backing
// arrays updates the stored window without a rebuild.
func setControlValue(controls []wire.PanelControlSpec, id, value string) bool {
	for i := range controls {
		if controls[i].ID == id {
			controls[i].Value = value
			return true
		}
		if setControlValue(controls[i].Children, id, value) {
			return true
		}
	}
	return false
}

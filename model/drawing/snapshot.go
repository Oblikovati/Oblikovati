// SPDX-License-Identifier: GPL-2.0-only

package drawing

import (
	"encoding/json"
	"fmt"
)

// Undo/redo snapshots use a fast internal JSON codec, distinct from the human-readable YAML on-disk
// format (MarshalRecipe / ApplyRecipe, recipe.go). A snapshot is transient and never written to a
// file, so it trades readability for speed — exactly the part/assembly snapshot codec rationale
// (#1147). The drawing recipe types carry no custom MarshalYAML and no interface-typed fields, so
// the plain JSON struct codec round-trips faithfully and deterministically (the no-op-delta byte
// check in the undo stream relies on that determinism). Landing DrawingContent on this codec puts it
// on the central undo seam, so a wire drawing edit records an undo step rather than only replicating
// (#1448, found while wiring #1426): its undo labels (PR #1447) were dead until this.

// MarshalSnapshot encodes the drawing's full state for the undo/redo stream. It shares the recipe
// build with the YAML save (recipe.go) but encodes JSON for speed; see the package note in this file.
func (c *Content) MarshalSnapshot() ([]byte, error) {
	return json.Marshal(c.buildRecipe())
}

// RestoreSnapshot replaces the drawing's entire state with a snapshot produced by
// [Content.MarshalSnapshot] — the undo/redo restore direction. It is a full REPLACE (the sheets are
// rebuilt from the recipe in place, preserving the Content pointer and its injected resolvers), so
// navigating to any snapshot yields exactly that state regardless of what the drawing now holds. The
// snapshot is decoded before the rebuild, so a malformed snapshot leaves the drawing untouched.
//
// The rebuilt sheets carry no projected view curves yet, so the view-projection cache is cleared and
// re-projected immediately when the referenced model resolves; otherwise the head's next SyncViews
// re-projects (the unchanged-body fast path would wrongly skip a rebuild without the cache reset).
func (c *Content) RestoreSnapshot(snapshot []byte) error {
	var r drawingRecipe
	if err := json.Unmarshal(snapshot, &r); err != nil {
		return fmt.Errorf("drawing: parse snapshot: %w", err)
	}
	if err := c.applyRecipeStruct(r); err != nil {
		return err
	}
	c.lastViewBody = nil // the rebuilt views hold no projected curves; force re-projection
	c.SyncViews()        // re-project now if the model resolves; else the head's next frame does
	return nil
}

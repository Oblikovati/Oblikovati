// SPDX-License-Identifier: GPL-2.0-only

package compdef

import (
	"encoding/json"
	"fmt"
)

// Undo/redo snapshots use a fast internal codec, distinct from the human-readable YAML on-disk
// format (MarshalRecipe / ApplyRecipe). A snapshot is transient and never written to a file, so
// it trades readability for speed: yaml.v3 reflection-encoding a large part's recipe was ~7 s /
// 150 MB on import of a dense drawing (#1147), which froze the edit and exceeded the add-in
// host-call timeout. JSON over the same recipe struct is several times faster, deterministic
// (so the no-op-delta byte check in the undo stream stays valid), and still text (so the
// session transaction log / bug report stays readable). The recipe types carry no custom
// MarshalYAML and no interface-typed fields, so a plain struct codec round-trips faithfully —
// the snapshot round-trip tests pin this.

// MarshalSnapshot encodes the part's full parametric recipe for the undo/redo stream. See the
// package note above for why this is JSON rather than the on-disk YAML.
func (d *PartComponentDefinition) MarshalSnapshot() ([]byte, error) {
	r, err := d.buildRecipe()
	if err != nil {
		return nil, err
	}
	return json.Marshal(r)
}

// RestoreSnapshot replaces the part's entire recipe with a snapshot produced by
// [PartComponentDefinition.MarshalSnapshot] and recomputes — the undo/redo restore direction.
// It is a full REPLACE, not a merge: it resets the definition to empty in place (preserving the
// definition pointer, so the document Content and held references stay valid) before applying,
// so navigating to any snapshot yields exactly that state regardless of what the part now holds.
// The recipe is decoded before the reset, so a malformed snapshot leaves the part untouched.
func (d *PartComponentDefinition) RestoreSnapshot(snapshot []byte) error {
	var r partRecipe
	if err := json.Unmarshal(snapshot, &r); err != nil {
		return fmt.Errorf("compdef: parse part snapshot: %w", err)
	}
	d.resetRecipe()
	return d.applyRecipeStruct(r)
}

// MarshalSnapshot encodes the assembly's recipe for the undo/redo stream (JSON; see the package
// note). An assembly snapshot captures occurrences/features/sketches just as MarshalRecipe does.
func (a *AssemblyComponentDefinition) MarshalSnapshot() ([]byte, error) {
	r, err := a.buildRecipe()
	if err != nil {
		return nil, err
	}
	return json.Marshal(r)
}

// RestoreSnapshot replaces the assembly's recipe from a [AssemblyComponentDefinition.MarshalSnapshot]
// snapshot — the undo/redo restore direction. Unlike a part it cannot finish in place: it resets
// the occurrence structure and re-stashes the snapshot as pending, then the owner-aware caller
// pairs it with [AssemblyComponentDefinition.ResolveReferences] to re-bind each occurrence to its
// component document (app/undo.go's rebindReferences does this after every restore, #763).
func (a *AssemblyComponentDefinition) RestoreSnapshot(snapshot []byte) error {
	var r assemblyRecipe
	if err := json.Unmarshal(snapshot, &r); err != nil {
		return fmt.Errorf("compdef: parse assembly snapshot: %w", err)
	}
	a.resetOccurrences()
	return a.applyRecipeStruct(r)
}

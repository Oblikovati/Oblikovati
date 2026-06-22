// SPDX-License-Identifier: GPL-2.0-only

package app

import (
	"errors"

	"oblikovati.org/model/compdef"
	"oblikovati.org/model/feature"
	"oblikovati.org/model/sketch"
)

// Sheet-metal UI access (M13). The Sheet Metal ribbon tab and its tools act on the active
// part only when it is in the sheet-metal environment (created with the sheet-metal subtype,
// which seeds the rule). These mirror [hasActivePart]/[activePart] for that gate.

// hasActiveSheetMetalPart reports whether the active document is a sheet-metal part — the
// enable predicate for the Sheet Metal tab's commands.
func hasActiveSheetMetalPart(s *Session) bool {
	p, err := activePart(s)
	return err == nil && p.IsSheetMetal()
}

// activeSheetMetalPart returns the active sheet-metal part, erroring when the active document
// is not a part or is an ordinary (non-sheet-metal) part.
func activeSheetMetalPart(s *Session) (*compdef.PartComponentDefinition, error) {
	p, err := activePart(s)
	if err != nil {
		return nil, err
	}
	if !p.IsSheetMetal() {
		return nil, errors.New("the active part is not a sheet-metal part")
	}
	return p, nil
}

// commitSheetMetalFeature recomputes the part, records the edit for undo, clears the selection
// filter, and reports a sick feature as a tool error (so the tool stays open) — the shared
// finish path every sheet-metal tool's Commit ends with.
func commitSheetMetalFeature(s *Session, part *compdef.PartComponentDefinition, added *feature.PartFeature, label string) error {
	part.Recompute()
	s.recordEdit(part, label)
	if !added.Health().OK() {
		return errors.New(label + ": " + added.Health().Reason)
	}
	return nil
}

// lineHandleInSketch returns the index of a picked sketch entity among a sketch's lines (the
// bend/axis line a Bend/Fold/Contour-Roll tool needs), ok=false when it is not a line of sk.
func lineHandleInSketch(sk *sketch.Sketch, ent sketch.Entity) (*sketch.Sketch, int, bool) {
	for i := 0; i < sk.Lines().Count(); i++ {
		if sketch.Entity(sk.Lines().Item(i)) == ent {
			return sk, i, true
		}
	}
	return nil, 0, false
}

// lineHandleInPart finds the sketch and line index a picked line entity belongs to, searching
// the part's sketches — the Bend/Fold tools resolve their bend line this way.
func lineHandleInPart(part *compdef.PartComponentDefinition, ent sketch.Entity) (*sketch.Sketch, int, bool) {
	sks := part.Sketches()
	for i := 0; i < sks.Count(); i++ {
		if sk, idx, ok := lineHandleInSketch(sks.Item(i), ent); ok {
			return sk, idx, true
		}
	}
	return nil, 0, false
}

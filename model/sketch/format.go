// SPDX-License-Identifier: GPL-2.0-only

package sketch

import "oblikovati.org/api/types"

// Per-entity format overrides (#2015): the line type, colour and line weight the Format panel's
// three lists set on selected geometry, and the values a DWG import carries in from the file's
// layer table.
//
// They live in a side table keyed by entity id rather than on the entities themselves. Absence
// means Default, which models the semantics with no sentinel values and costs an unstyled sketch
// nothing — and it keeps Point untouched, which matters because Point is arena-allocated to stay
// small and point count dominates a large DWG import.
//
//	sk.SetEntityFormat(line.EntityID(), sketch.EntityFormat{LineType: "dashed"})

// EntityFormat is one entity's format overrides. Each field independently means "inherit" when
// unset, so an entity can override its colour while taking the sketch's line type.
type EntityFormat struct {
	// LineType is the .lin pattern name; "" inherits the sketch's line type.
	LineType string
	// Color overrides the entity's colour; a Color whose Source is types.AutomaticColorSource
	// inherits instead. The zero Color is NOT the marker — its Source is 0, which is not a
	// member of the enum, so it correctly reads as unset.
	Color types.Color
	// LineWeight is the stroke width in millimetres; 0 inherits.
	LineWeight float64
}

// IsDefault reports whether the format overrides nothing, in which case it need not be stored.
func (f EntityFormat) IsDefault() bool {
	return f.LineType == "" && f.LineWeight == 0 && !f.Color.IsOverride()
}

// EntityFormat returns the entity's overrides, or ok=false when it takes the sketch defaults.
func (s *Sketch) EntityFormat(id ID) (EntityFormat, bool) {
	f, ok := s.formats[id]
	return f, ok
}

// SetEntityFormat stores an entity's overrides. A format that overrides nothing clears the entry
// instead of storing an empty one, so "no overrides" has exactly one representation.
func (s *Sketch) SetEntityFormat(id ID, f EntityFormat) {
	if f.IsDefault() {
		s.ClearEntityFormat(id)
		return
	}
	if s.formats == nil {
		s.formats = map[ID]EntityFormat{}
	}
	s.formats[id] = f
}

// ClearEntityFormat returns an entity to the sketch defaults.
func (s *Sketch) ClearEntityFormat(id ID) { delete(s.formats, id) }

// EntityFormatCount reports how many entities carry overrides — what persistence writes and what
// the prune tests assert against.
func (s *Sketch) EntityFormatCount() int { return len(s.formats) }

// CopyEntityFormat carries one entity's overrides onto another, for the pattern, mirror and
// block-instance copies. Copying from an unstyled entity stores nothing.
func (s *Sketch) CopyEntityFormat(from, to ID) {
	f, ok := s.formats[from]
	if !ok {
		return
	}
	s.SetEntityFormat(to, f)
}

// carryEntityFormats copies each cloned entity's format onto its clone, so a pattern, mirror or
// block instance keeps the formatting of the geometry it came from.
//
// from is the sketch that owns the originals, which is not always the target: a cross-sketch copy
// clones source entities into this sketch, and the format is the one piece of an entity's state
// that lives in a side table rather than on the entity itself — so it cannot be read off the
// pointer the way geometry can.
func (target *Sketch) carryEntityFormats(from *Sketch, m map[Entity]Entity) {
	for src, dst := range m {
		f, ok := from.EntityFormat(src.EntityID())
		if !ok {
			continue
		}
		target.SetEntityFormat(dst.EntityID(), f)
	}
}

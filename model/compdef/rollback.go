// SPDX-License-Identifier: GPL-2.0-only

package compdef

// Rollback / end-of-part (EOP) state. The EOP marker defines how far down the
// feature program the model evaluates; moving it up "rolls back" the part so
// features after the marker are suppressed from evaluation. The actual re-evaluate-
// up-to-marker is the feature engine's job (M08); this stores the marker and bumps
// the geometry version so a re-evaluate is triggered.

// EndOfPartPosition returns the feature index the model evaluates up to, or -1 when
// the whole program is evaluated (the marker is at the end).
func (d *PartComponentDefinition) EndOfPartPosition() int { return d.eop }

// IsRolledBack reports whether the EOP marker is before the end of the program, so
// some trailing features are currently suppressed.
func (d *PartComponentDefinition) IsRolledBack() bool { return d.eop != endOfPartAtEnd }

// SetEndOfPart moves the EOP marker to the given feature index, rolling the part
// back to that point, and advances the geometry version to request re-evaluation.
// A negative index restores the marker to the end (see [RollToEnd]).
func (d *PartComponentDefinition) SetEndOfPart(position int) {
	if position < 0 {
		position = endOfPartAtEnd
	}
	if position != d.eop {
		d.eop = position
		d.MarkChanged()
	}
}

// RollToEnd moves the EOP marker back to the end, re-including every feature.
func (d *PartComponentDefinition) RollToEnd() { d.SetEndOfPart(endOfPartAtEnd) }

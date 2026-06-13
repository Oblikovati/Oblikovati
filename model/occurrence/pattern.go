// SPDX-License-Identifier: GPL-2.0-only

package occurrence

import "oblikovati.org/math"

// OccurrencePatternElement is one replicated instance of an occurrence pattern: its
// computed placement plus per-element state — a suppression flag and an optional
// reposition override that wins over the pattern-computed placement. The reference
// API's OccurrencePatternElement (M11-F04).
type OccurrencePatternElement struct {
	placement  math.Matrix4 // arrangement offset · seed base
	suppressed bool
	override   *math.Matrix4 // reposition override; nil ⇒ use placement
}

// Transform returns the element's effective placement: the reposition override if
// set, otherwise the pattern-computed placement.
func (e *OccurrencePatternElement) Transform() math.Matrix4 {
	if e.override != nil {
		return *e.override
	}
	return e.placement
}

// Suppressed reports whether this element is omitted from the pattern.
func (e *OccurrencePatternElement) Suppressed() bool { return e.suppressed }

// SetSuppressed includes or omits this element.
func (e *OccurrencePatternElement) SetSuppressed(suppressed bool) { e.suppressed = suppressed }

// Reposition overrides this element's placement (e.g. nudging one patterned instance
// off the regular grid); the override survives regeneration by index.
func (e *OccurrencePatternElement) Reposition(m math.Matrix4) { e.override = &m }

// ClearReposition drops the override so the element returns to its pattern placement.
func (e *OccurrencePatternElement) ClearReposition() { e.override = nil }

// Repositioned reports whether the element carries a reposition override.
func (e *OccurrencePatternElement) Repositioned() bool { return e.override != nil }

// OccurrencePattern replicates a seed component across an [Arrangement]. Each position
// becomes an [OccurrencePatternElement] placed at the arrangement offset composed with
// the seed's base placement. Editing the arrangement (e.g. a new count) regenerates
// the elements, preserving each surviving element's suppression and reposition
// override by index — the reference API's *OccurrencePattern.
type OccurrencePattern struct {
	seed        Definition
	base        math.Matrix4
	arrangement Arrangement
	elements    []*OccurrencePatternElement
}

// NewOccurrencePattern builds a pattern of seed (placed at base) over arrangement and
// generates its elements.
func NewOccurrencePattern(seed Definition, base math.Matrix4, arrangement Arrangement) *OccurrencePattern {
	p := &OccurrencePattern{seed: seed, base: base, arrangement: arrangement}
	p.Regenerate()
	return p
}

// SetArrangement swaps the arrangement (e.g. to change count or spacing) and
// regenerates the elements, keeping surviving elements' state by index.
func (p *OccurrencePattern) SetArrangement(a Arrangement) {
	p.arrangement = a
	p.Regenerate()
}

// Regenerate rebuilds the elements from the current arrangement. Each surviving
// position (by index) keeps its suppression and reposition override, so a count edit
// leaves the state of the elements that remain untouched.
func (p *OccurrencePattern) Regenerate() {
	placements := p.arrangement.Placements()
	next := make([]*OccurrencePatternElement, len(placements))
	for i, offset := range placements {
		e := &OccurrencePatternElement{placement: offset.Mul(p.base)}
		if i < len(p.elements) {
			e.suppressed = p.elements[i].suppressed
			e.override = p.elements[i].override
		}
		next[i] = e
	}
	p.elements = next
}

// Seed returns the component this pattern replicates.
func (p *OccurrencePattern) Seed() Definition { return p.seed }

// Count returns the number of pattern elements (including suppressed ones).
func (p *OccurrencePattern) Count() int { return len(p.elements) }

// Element returns the i-th pattern element in arrangement order.
func (p *OccurrencePattern) Element(i int) *OccurrencePatternElement { return p.elements[i] }

// Elements returns a snapshot of the pattern's elements in arrangement order.
func (p *OccurrencePattern) Elements() []*OccurrencePatternElement {
	out := make([]*OccurrencePatternElement, len(p.elements))
	copy(out, p.elements)
	return out
}

// RangeBox returns the axis-aligned box enclosing the seed placed at every
// unsuppressed element (empty when all are suppressed). Empty placements are skipped,
// as in [Occurrences.RangeBox].
func (p *OccurrencePattern) RangeBox() math.Box {
	box := math.EmptyBox()
	for _, e := range p.elements {
		if e.suppressed {
			continue
		}
		eb := p.seed.RangeBox().Transform(e.Transform())
		if eb.IsEmpty() {
			continue
		}
		box = box.Union(eb)
	}
	return box
}

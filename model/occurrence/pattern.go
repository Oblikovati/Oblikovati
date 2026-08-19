// SPDX-License-Identifier: GPL-2.0-only

package occurrence

import (
	"fmt"

	"oblikovati.org/api/types"
	"oblikovati.org/math"
)

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

	// Persistent identity + the occurrences each element drives (#1976). occs is aligned to
	// elements: occs[0] is the seed occurrence, occs[i>=1] the generated copy for element i, so
	// suppressing/repositioning an element acts on the real occurrence. Set by BindOccurrences.
	id   uint64
	name string
	occs []*Occurrence
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

// Arrangement returns the layout the pattern replicates the seed across — read for persistence.
func (p *OccurrencePattern) Arrangement() Arrangement { return p.arrangement }

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

// ID returns the pattern's session id, assigned when it is recorded in an
// [OccurrencePatternSet].
func (p *OccurrencePattern) ID() uint64 { return p.id }

// Name returns the pattern's display name; SetName renames it.
func (p *OccurrencePattern) Name() string     { return p.name }
func (p *OccurrencePattern) SetName(n string) { p.name = n }

// Kind reports the arrangement family — "circular", "rectangular", or "" for another
// arrangement — so a client can tell how the pattern was laid out.
func (p *OccurrencePattern) Kind() string {
	switch p.arrangement.(type) {
	case *CircularArrangement, CircularArrangement:
		return "circular"
	case *RectangularArrangement, RectangularArrangement:
		return "rectangular"
	default:
		return ""
	}
}

// BindOccurrences links the pattern's elements to the occurrences they drive: the seed (element
// 0) and one generated occurrence per element beyond it, in element order. Editing an element
// then moves or suppresses the real occurrence.
func (p *OccurrencePattern) BindOccurrences(seed *Occurrence, generated []*Occurrence) {
	p.occs = append([]*Occurrence{seed}, generated...)
}

// Occurrences returns the occurrences the pattern drives, element 0 (the seed) first.
func (p *OccurrencePattern) Occurrences() []*Occurrence {
	out := make([]*Occurrence, len(p.occs))
	copy(out, p.occs)
	return out
}

// SetSuppressed suppresses or unsuppresses the WHOLE pattern, moving every GENERATED element
// together (#1976). Element 0 is the seed — the pre-existing component — so it is left in place; an
// all-suppressed pattern leaves just the seed.
func (p *OccurrencePattern) SetSuppressed(suppressed bool) {
	for i := 1; i < len(p.elements); i++ {
		p.setElementSuppressed(i, suppressed)
	}
}

// SetElementSuppressed suppresses or unsuppresses one generated element by index (1..Count-1),
// erroring on the seed (element 0) or an out-of-range index.
func (p *OccurrencePattern) SetElementSuppressed(i int, suppressed bool) error {
	if err := p.checkGenerated(i); err != nil {
		return err
	}
	p.setElementSuppressed(i, suppressed)
	return nil
}

// checkGenerated rejects the seed (element 0, which the pattern does not own) and out-of-range
// indices, so pattern edits only ever touch the generated instances.
func (p *OccurrencePattern) checkGenerated(i int) error {
	if i < 1 || i >= len(p.elements) {
		return fmt.Errorf("occurrence pattern element %d out of range [1,%d) (element 0 is the seed)", i, len(p.elements))
	}
	return nil
}

// setElementSuppressed sets the element flag and the driven occurrence together.
func (p *OccurrencePattern) setElementSuppressed(i int, suppressed bool) {
	p.elements[i].SetSuppressed(suppressed)
	if i < len(p.occs) && p.occs[i] != nil {
		p.occs[i].SetSuppressed(suppressed)
	}
}

// RepositionElement moves one element (by index) to an explicit placement, off the regular grid,
// erroring when the index is out of range. The driven occurrence moves with it.
func (p *OccurrencePattern) RepositionElement(i int, m math.Matrix4) error {
	if err := p.checkGenerated(i); err != nil {
		return err
	}
	p.elements[i].Reposition(m)
	if i < len(p.occs) && p.occs[i] != nil {
		p.occs[i].SetTransform(m)
	}
	return nil
}

// Suppression reports whether none, some, or all of the pattern's elements are suppressed —
// the pattern-level state Inventor's OccurrencePatternSuppressionEnum reports (#1976).
func (p *OccurrencePattern) Suppression() types.OccurrencePatternSuppression {
	generated := len(p.elements) - 1 // element 0 is the seed, not part of the pattern's suppression
	if generated <= 0 {
		return types.NoneSuppressed
	}
	suppressed := 0
	for i := 1; i < len(p.elements); i++ {
		if p.elements[i].suppressed {
			suppressed++
		}
	}
	switch suppressed {
	case 0:
		return types.NoneSuppressed
	case generated:
		return types.AllElementsSuppressed
	default:
		return types.SomeElementsSuppressed
	}
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

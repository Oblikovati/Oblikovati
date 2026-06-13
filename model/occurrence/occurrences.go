// SPDX-License-Identifier: GPL-2.0-only

package occurrence

import "oblikovati.org/math"

// Occurrences is the collection of component occurrences an assembly owns — the
// reference API's ComponentOccurrences (M11-F01). It mints occurrence ids, tracks a
// revision that advances on every structural or placement change (so the assembly's
// geometry version derives from it), and unions its members' boxes.
type Occurrences struct {
	items    []*Occurrence
	byID     map[uint64]*Occurrence
	nextID   uint64
	revision uint64
}

// NewOccurrences returns an empty occurrence collection.
func NewOccurrences() *Occurrences {
	return &Occurrences{byID: map[uint64]*Occurrence{}}
}

// Add places source in the assembly under name with transform, returning the new
// occurrence. The transform locates the component in the assembly's space; pass
// [math.Identity4] to place it at the origin.
func (c *Occurrences) Add(name string, source RangeBoxSource, transform math.Matrix4) *Occurrence {
	c.nextID++
	o := &Occurrence{id: c.nextID, name: name, transform: transform, source: source, owner: c}
	c.items = append(c.items, o)
	c.byID[o.id] = o
	c.bump()
	return o
}

// Count returns the number of occurrences; Item returns the i-th in placement order.
func (c *Occurrences) Count() int             { return len(c.items) }
func (c *Occurrences) Item(i int) *Occurrence { return c.items[i] }

// ByID returns the occurrence with the given session id.
func (c *Occurrences) ByID(id uint64) (*Occurrence, bool) {
	o, ok := c.byID[id]
	return o, ok
}

// All returns a snapshot of the occurrences in placement order.
func (c *Occurrences) All() []*Occurrence {
	out := make([]*Occurrence, len(c.items))
	copy(out, c.items)
	return out
}

// Remove deletes an occurrence, reporting whether it was present and advancing the
// revision when it was.
func (c *Occurrences) Remove(o *Occurrence) bool {
	if _, ok := c.byID[o.id]; !ok {
		return false
	}
	delete(c.byID, o.id)
	for i, existing := range c.items {
		if existing == o {
			c.items = append(c.items[:i], c.items[i+1:]...)
			c.bump()
			return true
		}
	}
	return false
}

// RangeBox returns the axis-aligned box enclosing every unsuppressed occurrence,
// empty when the assembly has none. Empty contributions (suppressed occurrences, or
// a source with no geometry) are skipped — unioning the empty box would blow the
// result out to infinite extents, since the empty box has ±inf corners.
func (c *Occurrences) RangeBox() math.Box {
	box := math.EmptyBox()
	for _, o := range c.items {
		ob := o.RangeBox()
		if ob.IsEmpty() {
			continue
		}
		box = box.Union(ob)
	}
	return box
}

// Revision returns a counter that advances on every structural change (add/remove)
// and every placement/suppression change, so consumers can detect when the assembly
// geometry changed (the basis for the assembly's ModelGeometryVersion).
func (c *Occurrences) Revision() uint64 { return c.revision }

// bump advances the revision after a mutation.
func (c *Occurrences) bump() { c.revision++ }

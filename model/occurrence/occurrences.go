// SPDX-License-Identifier: GPL-2.0-only

package occurrence

import "oblikovati.org/math"

// Occurrences is the collection of component occurrences an assembly owns — the
// reference API's ComponentOccurrences (M11-F01/F02). It mints occurrence ids, tracks
// a revision that advances on every structural or placement change (so the assembly's
// geometry version derives from it), unions its members' boxes, and notifies a
// [OccurrenceListener] of each mutation so the assembly raises domain events (M11-F07).
type Occurrences struct {
	items    []*Occurrence
	byID     map[uint64]*Occurrence
	nextID   uint64
	revision uint64

	listener OccurrenceListener
	// Drag-batch state: while suspend > 0, per-occurrence transform notifications are
	// withheld and coalesced. dragStart holds each moved occurrence's placement when the
	// batch began; dragOrder preserves first-moved order so the resume flush is
	// deterministic.
	suspend   int
	dragStart map[*Occurrence]math.Matrix4
	dragOrder []*Occurrence
}

// NewOccurrences returns an empty occurrence collection with a silent (no-op) listener.
func NewOccurrences() *Occurrences {
	return &Occurrences{byID: map[uint64]*Occurrence{}, listener: silentListener{}}
}

// SetListener installs the observer notified after each occurrence mutation, replacing
// any previous one; pass nil to detach. The assembly definition installs its event
// source here so occurrence changes raise domain events (M11-F07).
func (c *Occurrences) SetListener(l OccurrenceListener) {
	if l == nil {
		l = silentListener{}
	}
	c.listener = l
}

// AddByComponentDefinition places def in the assembly under name with transform,
// returning the new occurrence. def is shared — placing the same definition twice
// yields two occurrences that both track its edits (the flyweight). The transform
// locates the component in the assembly's space; pass [math.Identity4] for the origin.
func (c *Occurrences) AddByComponentDefinition(name string, def Definition, transform math.Matrix4) *Occurrence {
	c.nextID++
	o := &Occurrence{id: c.nextID, name: name, transform: transform, definition: def, owner: c}
	c.items = append(c.items, o)
	c.byID[o.id] = o
	c.bump()
	c.listener.OccurrenceAdded(o)
	return o
}

// Replace swaps an occurrence's definition in place — the "replace component"
// operation: a different component at the same placement, keeping the occurrence's id,
// name, transform, and state. Reports whether the occurrence belongs to this
// collection.
func (c *Occurrences) Replace(o *Occurrence, def Definition) bool {
	// Verify identity, not just id presence: ids are per-collection, so a foreign
	// occurrence can share an id with one of ours.
	if c.byID[o.id] != o {
		return false
	}
	previous := o.definition
	o.definition = def
	c.bump()
	c.listener.OccurrenceReplaced(o, previous)
	return true
}

// Count returns the number of occurrences; Item returns the i-th in placement order.
func (c *Occurrences) Count() int             { return len(c.items) }
func (c *Occurrences) Item(i int) *Occurrence { return c.items[i] }

// ByID returns the occurrence with the given session id.
func (c *Occurrences) ByID(id uint64) (*Occurrence, bool) {
	o, ok := c.byID[id]
	return o, ok
}

// byName returns the first occurrence with the given instance name, or nil. Instance
// names are expected unique within a collection (the placement layer auto-numbers
// them, e.g. "pin:1"/"pin:2"); path resolution relies on that.
func (c *Occurrences) byName(name string) *Occurrence {
	for _, o := range c.items {
		if o.name == name {
			return o
		}
	}
	return nil
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
	// Verify identity, not just id presence: a foreign occurrence may share an id
	// with one of ours, and deleting on a bare id match would evict the real entry.
	if c.byID[o.id] != o {
		return false
	}
	delete(c.byID, o.id)
	for i, existing := range c.items {
		if existing == o {
			c.items = append(c.items[:i], c.items[i+1:]...)
			c.bump()
			c.listener.OccurrenceRemoved(o)
			return true
		}
	}
	return false
}

// RangeBox returns the axis-aligned box enclosing every unsuppressed occurrence,
// empty when the assembly has none. Empty contributions (suppressed occurrences, or a
// definition with no geometry) are skipped — unioning the empty box would blow the
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

// Revision returns a counter that advances on every structural change (add/remove/
// replace) and every placement/suppression change, so consumers can detect when the
// assembly geometry changed (the basis for the assembly's ModelGeometryVersion).
func (c *Occurrences) Revision() uint64 { return c.revision }

// bump advances the revision after a mutation.
func (c *Occurrences) bump() { c.revision++ }

// transformed records a placement change for o (whose prior placement was previous):
// it advances the revision and notifies the listener immediately, unless a drag batch
// is active, in which case the notification is deferred and coalesced to a single call
// at [Occurrences.ResumeNotifications]. Called by [Occurrence.SetTransform].
func (c *Occurrences) transformed(o *Occurrence, previous math.Matrix4) {
	c.bump()
	if c.suspend == 0 {
		c.listener.OccurrenceTransformed(o, previous)
		return
	}
	if _, seen := c.dragStart[o]; !seen {
		c.dragStart[o] = previous
		c.dragOrder = append(c.dragOrder, o)
	}
}

// suppressed records a suppression toggle for o and notifies the listener. Suppression
// is a low-frequency, meaningful change, so it is never coalesced — it fires even
// inside a drag batch. Called by [Occurrence.SetSuppressed].
func (c *Occurrences) suppressed(o *Occurrence) {
	c.bump()
	c.listener.OccurrenceSuppressionChanged(o)
}

// SuspendNotifications begins a batch that coalesces per-occurrence transform
// notifications until the matching [Occurrences.ResumeNotifications], so a solver drag
// or drive animation raises one OccurrenceTransformed per moved occurrence instead of
// one per step (M11-F07). Calls nest; the batch ends only when the outermost resumes.
// The revision still advances per step — only the event stream is batched.
func (c *Occurrences) SuspendNotifications() {
	if c.suspend == 0 {
		c.dragStart = map[*Occurrence]math.Matrix4{}
		c.dragOrder = nil
	}
	c.suspend++
}

// ResumeNotifications ends a batch started by [Occurrences.SuspendNotifications],
// flushing one OccurrenceTransformed per occurrence that ended the batch at a
// different placement than it began (a net no-op move emits nothing), in first-moved
// order. An unbalanced resume (no batch active) is ignored.
func (c *Occurrences) ResumeNotifications() {
	if c.suspend == 0 {
		return
	}
	c.suspend--
	if c.suspend > 0 {
		return
	}
	for _, o := range c.dragOrder {
		if start := c.dragStart[o]; o.transform != start {
			c.listener.OccurrenceTransformed(o, start)
		}
	}
	c.dragStart, c.dragOrder = nil, nil
}

// SPDX-License-Identifier: GPL-2.0-only

package sketch

import (
	"fmt"

	"oblikovati.org/model/param"
)

// Sketch collection (M48 #2245 split of sketch.go). The Sketches container: creating named/auto-named
// sketches on a plane, shared parameters and block definitions, and the lookup/remove operations. One
// sketch aggregate lives in sketch.go.

// Sketches is the collection of planar sketches owned by a component definition.
type Sketches struct {
	items []*Sketch
	byID  map[ID]*Sketch
	seq   int // running counter behind the Sketch1, Sketch2, … auto-names
	// blockDefs is the part-level block-definition registry every sketch of
	// the part places instances from (M06-F07, #622).
	blockDefs *BlockDefinitions
	// params is the document's parameter DAG, shared into every sketch added to the
	// collection so dimension expressions referencing user parameters resolve. Nil for
	// a bare collection (tests), which leaves each sketch with its own empty set.
	params *param.Parameters
}

// ShareParameters makes the collection hand the document's parameter DAG to every
// sketch it creates (live or on restore), so a dimension expression like "width"
// resolves against user parameters. Wiring at creation matters: the restore path adds a
// sketch's dimensions immediately, and SetParameters must precede them.
func (c *Sketches) ShareParameters(ps *param.Parameters) { c.params = ps }

// NewSketches returns an empty collection.
func NewSketches() *Sketches {
	return &Sketches{byID: map[ID]*Sketch{}, blockDefs: &BlockDefinitions{}}
}

// BlockDefinitions returns the part-level block-definition registry.
func (sc *Sketches) BlockDefinitions() *BlockDefinitions { return sc.blockDefs }

// Add creates a planar sketch on plane and adds it to the collection, giving it the
// next free auto-name (Sketch1, Sketch2, …) like the reference API.
func (c *Sketches) Add(plane Plane) *Sketch {
	return c.AddNamed(c.nextSketchName(), plane)
}

// nextSketchName mints the first unused Sketch{N} name, advancing the counter past
// names already taken (so a restored "Sketch3" doesn't collide with a later Add).
func (c *Sketches) nextSketchName() string {
	for {
		c.seq++
		name := fmt.Sprintf("Sketch%d", c.seq)
		if !c.nameTaken(name) {
			return name
		}
	}
}

// nameTaken reports whether a sketch in the collection already uses name.
func (c *Sketches) nameTaken(name string) bool {
	for _, s := range c.items {
		if s.Name() == name {
			return true
		}
	}
	return false
}

// AddNamed creates a named planar sketch on plane.
func (c *Sketches) AddNamed(name string, plane Plane) *Sketch {
	s := &Sketch{base: newBase(name), plane: plane}
	s.initCollections()
	if c.params != nil {
		s.SetParameters(c.params) // before any dimensions are added (live or on restore)
	}
	c.items = append(c.items, s)
	c.byID[s.id] = s
	return s
}

// restoreSketchID pins a freshly-added sketch's local id to its persisted value so the
// sketch's document-derived persistent reference key (#153) is stable across load, re-keying
// the byID index and raising the id clock past it. A zero saved id (a legacy recipe with no
// persisted sketch id) keeps the minted one.
func (c *Sketches) restoreSketchID(s *Sketch, saved uint64) {
	if saved == 0 {
		return
	}
	delete(c.byID, s.id)
	s.id = ID(saved)
	c.byID[s.id] = s
	raiseIDSeq(saved)
}

// Count returns the number of sketches.
func (c *Sketches) Count() int { return len(c.items) }

// Item returns the sketch at index i (0-based).
func (c *Sketches) Item(i int) *Sketch { return c.items[i] }

// ByID returns the sketch with the given id.
func (c *Sketches) ByID(id ID) (*Sketch, bool) {
	s, ok := c.byID[id]
	return s, ok
}

// Remove deletes the sketch with the given id, reporting whether it was found. The
// auto-name counter is not rewound — Inventor does not reuse a deleted sketch's number.
func (c *Sketches) Remove(id ID) bool {
	if _, ok := c.byID[id]; !ok {
		return false
	}
	delete(c.byID, id)
	for i, s := range c.items {
		if s.id == id {
			c.items = append(c.items[:i], c.items[i+1:]...)
			break
		}
	}
	return true
}

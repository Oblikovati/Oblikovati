// SPDX-License-Identifier: GPL-2.0-only

package sketch

// Sketch3D is a sketch whose points live in model 3D space (no host plane). It is
// used for sweep/pipe paths. The same constraint model applies with 3 variables
// per point; the solver core is dimension-agnostic (modeling/00).
type Sketch3D struct {
	base
	ents []Entity
}

// Entities returns the 3D sketch's geometry in insertion order.
func (s *Sketch3D) Entities() []Entity {
	out := make([]Entity, len(s.ents))
	copy(out, s.ents)
	return out
}

// EntityCount returns the number of entities.
func (s *Sketch3D) EntityCount() int { return len(s.ents) }

// Sketches3D is the collection of 3D sketches.
type Sketches3D struct {
	items []*Sketch3D
	byID  map[ID]*Sketch3D
}

// NewSketches3D returns an empty collection.
func NewSketches3D() *Sketches3D {
	return &Sketches3D{byID: map[ID]*Sketch3D{}}
}

// Add creates a 3D sketch.
func (c *Sketches3D) Add() *Sketch3D { return c.AddNamed("") }

// AddNamed creates a named 3D sketch.
func (c *Sketches3D) AddNamed(name string) *Sketch3D {
	s := &Sketch3D{base: newBase(name)}
	c.items = append(c.items, s)
	c.byID[s.id] = s
	return s
}

// Count returns the number of 3D sketches.
func (c *Sketches3D) Count() int { return len(c.items) }

// Item returns the 3D sketch at index i.
func (c *Sketches3D) Item(i int) *Sketch3D { return c.items[i] }

// ByID returns the 3D sketch with the given id.
func (c *Sketches3D) ByID(id ID) (*Sketch3D, bool) {
	s, ok := c.byID[id]
	return s, ok
}

// DrawingSketch is a 2D sketch on a drawing sheet. Geometrically it is a planar
// sketch in sheet coordinates with no model-space mapping (its "plane" is the
// sheet); annotation use of these sketches is M14.
type DrawingSketch struct {
	base
	ents []Entity
}

// Entities returns the drawing sketch's geometry in insertion order.
func (s *DrawingSketch) Entities() []Entity {
	out := make([]Entity, len(s.ents))
	copy(out, s.ents)
	return out
}

// EntityCount returns the number of entities.
func (s *DrawingSketch) EntityCount() int { return len(s.ents) }

// DrawingSketches is the collection of sketches on a drawing sheet.
type DrawingSketches struct {
	items []*DrawingSketch
	byID  map[ID]*DrawingSketch
}

// NewDrawingSketches returns an empty collection.
func NewDrawingSketches() *DrawingSketches {
	return &DrawingSketches{byID: map[ID]*DrawingSketch{}}
}

// Add creates a drawing-sheet sketch.
func (c *DrawingSketches) Add() *DrawingSketch { return c.AddNamed("") }

// AddNamed creates a named drawing-sheet sketch.
func (c *DrawingSketches) AddNamed(name string) *DrawingSketch {
	s := &DrawingSketch{base: newBase(name)}
	c.items = append(c.items, s)
	c.byID[s.id] = s
	return s
}

// Count returns the number of drawing sketches.
func (c *DrawingSketches) Count() int { return len(c.items) }

// Item returns the drawing sketch at index i.
func (c *DrawingSketches) Item(i int) *DrawingSketch { return c.items[i] }

// ByID returns the drawing sketch with the given id.
func (c *DrawingSketches) ByID(id ID) (*DrawingSketch, bool) {
	s, ok := c.byID[id]
	return s, ok
}

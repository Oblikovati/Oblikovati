// SPDX-License-Identifier: GPL-2.0-only

package sketch

// Sketch3D and its Sketches3D collection live in sketch3d.go (the real, constraint-
// solving 3D sketch). DrawingSketch stays here as the other sketch variant.

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

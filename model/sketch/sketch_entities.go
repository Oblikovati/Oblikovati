// SPDX-License-Identifier: GPL-2.0-only

package sketch

import "oblikovati.org/math"

// Sketch — ENTITY MUTATION (M48 #2245 split of sketch.go). Adding, removing and deleting entities from
// the aggregate (and dropping them from their typed collection), plus the sketch-point allocation arena.
// The entity accessors and collections live in sketch.go.

// add appends an entity to the sketch's geometry list.
func (s *Sketch) add(e Entity) { s.ents = append(s.ents, e) }

// removeEntity drops an entity from the geometry list (used by delete/trim). It does not
// touch constraints; callers handle those. Returns whether it was present.
func (s *Sketch) removeEntity(e Entity) {
	for i, x := range s.ents {
		if x == e {
			s.ents = append(s.ents[:i], s.ents[i+1:]...)
			return
		}
	}
}

// deleteEntity removes e from the entity list AND its typed collection, so an edit that
// drops a curve (e.g. a whole-curve trim) leaves no dangling collection entry that
// Count/Item/serialization would still report.
func (s *Sketch) deleteEntity(e Entity) {
	s.removeEntity(e)
	s.dropFromCollection(e)
	s.ClearEntityFormat(e.EntityID()) // the format dies with its entity (#2015)
}

// dropFromCollection removes e from its typed collection (the deleteEntity
// half that knows every entity family).
func (s *Sketch) dropFromCollection(e Entity) {
	switch t := e.(type) {
	case *Line:
		s.lines.remove(t)
	case *Circle:
		s.circles.remove(t)
	case *Arc:
		s.arcs.remove(t)
	case *TextBox:
		s.texts.remove(t)
		s.deleteTextBoxAnchor(t) // the anchor record dies with its text (M06-F11)
	case *Ellipse:
		s.ellipses.remove(t)
	case *EllipticalArc:
		s.ellArcs.remove(t)
	case *Spline:
		s.splines.remove(t)
	case *Point:
		s.points.remove(t)
	case *BlockInstance:
		s.blocks.remove(t) // also detaches the definition back-reference (M06-F07)
	}
}

// newPoint creates a constrainable point at pos and registers it as a solver
// variable. Curve factories use it for endpoints/centers (not added to Entities).
func (s *Sketch) newPoint(pos math.Point2) *Point {
	p := s.ptArena.alloc()
	p.id, p.X, p.Y = nextID(), pos.X, pos.Y
	s.pts = append(s.pts, p)
	return p
}

// pointArenaBlock is how many Points one arena block holds. Large enough that a dense imported
// polyline (thousands of vertices) costs a handful of allocations, small enough that a sketch
// with a few points wastes little: 1024 * sizeof(Point) ≈ 24 KB per block.
const pointArenaBlock = 1024

// pointArena hands out stable *Point from fixed-size blocks, so bulk authoring — importing a
// drawing's tens of thousands of polyline vertices (#1549) — allocates one block per
// pointArenaBlock points instead of one heap object per point, cutting the import's allocation
// count and steady-state GC scan cost. Blocks are never reallocated, so every handed-out
// pointer stays valid; points removed from a sketch simply leave a stale slot (rare, and freed
// with the whole sketch). Identity is still by pointer, since each &block[i] is unique.
type pointArena struct {
	blocks [][]Point
	used   int // points taken from the current (last) block
}

// alloc returns a pointer to the next free, zeroed Point, starting a fresh block when full.
func (a *pointArena) alloc() *Point {
	if len(a.blocks) == 0 || a.used == pointArenaBlock {
		a.blocks = append(a.blocks, make([]Point, pointArenaBlock))
		a.used = 0
	}
	p := &a.blocks[len(a.blocks)-1][a.used]
	a.used++
	return p
}

// NewPoint creates a sketch vertex point for use as a shared endpoint of lines or
// arcs (it is not a standalone point marker). It lets a bulk authoring caller — the
// DWG importer — connect a polyline's segments through shared vertices instead of
// duplicating endpoints, roughly a third less geometry on dense drawings.
func (s *Sketch) NewPoint(pos math.Point2) *Point { return s.newPoint(pos) }

// removePoint drops a solver point (a deactivated spline-handle end or a
// point moved into a block definition).
func (s *Sketch) removePoint(p *Point) {
	for i, x := range s.pts {
		if x == p {
			s.pts = append(s.pts[:i], s.pts[i+1:]...)
			return
		}
	}
}

// newRefPoint creates a fixed reference point (a projected anchor): a real Point other
// geometry can be constrained to, but excluded from the solver's free variables, so the
// solver holds it in place while other geometry moves to meet it.
func (s *Sketch) newRefPoint(pos math.Point2) *Point {
	p := &Point{id: nextID(), X: pos.X, Y: pos.Y}
	s.refPts = append(s.refPts, p)
	return p
}

// SPDX-License-Identifier: GPL-2.0-only

package sketch

import (
	"fmt"
	"sync/atomic"

	"github.com/Oblikovati/oblikovati/math"
	"github.com/Oblikovati/oblikovati/model/health"
	"github.com/Oblikovati/oblikovati/model/param"
)

// ID is the session-stable handle of a sketch or sketch entity, used by
// constraints and selection. Like document ids it is not persisted across
// sessions (regenerated on load); cross-session identity is via reference keys.
type ID uint64

var idSeq atomic.Uint64

func nextID() ID { return ID(idSeq.Add(1)) }

// Entity is a piece of sketch geometry (line, arc, circle, point, …). The full
// interface — constrainable points, curve evaluation — is filled in by the entity
// types (M06-F02); here it is the minimum the container needs.
type Entity interface {
	// EntityID returns the entity's sketch-local id.
	EntityID() ID
}

// base holds the identity and edit/visibility/display state common to every sketch kind.
type base struct {
	id      ID
	name    string
	editing bool
	visible bool
	health  health.Health

	// Display + solve overrides (Inventor Sketch.Color/LineType/LineWeight/DeferUpdates).
	color        string  // empty ⇒ inherit the document default
	lineType     string  // empty ⇒ inherit (api/types.SketchLineType value)
	lineWeight   float64 // 0 ⇒ inherit
	deferUpdates bool    // true ⇒ batch edits, solve on resume (M21-F08)
}

func newBase(name string) base {
	return base{id: nextID(), name: name, visible: true, health: health.Healthy}
}

// ID returns the sketch's session id.
func (b *base) ID() ID { return b.id }

// Name returns the sketch's display name.
func (b *base) Name() string { return b.name }

// SetName renames the sketch.
func (b *base) SetName(name string) { b.name = name }

// IsEditing reports whether the sketch is in edit mode (open for geometry changes).
func (b *base) IsEditing() bool { return b.editing }

// Edit enters edit mode; ExitEdit leaves it.
func (b *base) Edit()     { b.editing = true }
func (b *base) ExitEdit() { b.editing = false }

// Visible reports whether the sketch is shown; SetVisible toggles it.
func (b *base) Visible() bool     { return b.visible }
func (b *base) SetVisible(v bool) { b.visible = v }

// Health returns the sketch's solve health (set by the solver, M06-F05).
func (b *base) Health() health.Health { return b.health }

// Color returns the sketch's color override ("" ⇒ inherit); SetColor sets it.
func (b *base) Color() string     { return b.color }
func (b *base) SetColor(c string) { b.color = c }

// LineType returns the sketch's line-type override (api/types.SketchLineType value,
// "" ⇒ inherit); SetLineType sets it.
func (b *base) LineType() string     { return b.lineType }
func (b *base) SetLineType(t string) { b.lineType = t }

// LineWeight returns the sketch's line-weight override in cm (0 ⇒ inherit);
// SetLineWeight sets it.
func (b *base) LineWeight() float64     { return b.lineWeight }
func (b *base) SetLineWeight(w float64) { b.lineWeight = w }

// DeferUpdates reports whether the sketch batches edits (solving on resume);
// SetDeferUpdates toggles it (the solve gate is wired in M21-F08).
func (b *base) DeferUpdates() bool     { return b.deferUpdates }
func (b *base) SetDeferUpdates(d bool) { b.deferUpdates = d }

// Sketch is a planar 2D sketch hosted on a [Plane]. It owns its entities and (from
// F03/F04) its constraints, and resolves to profiles/paths via the solver (F05/F06).
type Sketch struct {
	base
	plane Plane
	ents  []Entity
	pts   []*Point // every constrainable point (endpoints, centers, standalone) — the solver's variables

	lines    *Lines
	arcs     *Arcs
	circles  *Circles
	ellipses *Ellipses
	ellArcs  *EllipticalArcs
	splines  *Splines
	points   *Points
	images   *SketchImages
	fills    *FillRegions
	texts    *TextBoxes
	eqCurves *EquationCurves
	fixedSpl *FixedSplines
	offSpl   *OffsetSplines
	blocks   *Blocks
	geomCons *GeometricConstraints
	dimCons  *DimensionConstraints
	params   *param.Parameters
}

// Plane returns the sketch's host plane.
func (s *Sketch) Plane() Plane { return s.plane }

// Entities returns the sketch's geometry in insertion order.
func (s *Sketch) Entities() []Entity {
	out := make([]Entity, len(s.ents))
	copy(out, s.ents)
	return out
}

// EntityCount returns the number of entities.
func (s *Sketch) EntityCount() int { return len(s.ents) }

// EntityByID returns the entity with the given session id, or false if none matches.
func (s *Sketch) EntityByID(id ID) (Entity, bool) {
	for _, e := range s.ents {
		if e.EntityID() == id {
			return e, true
		}
	}
	return nil, false
}

// PointByID returns the constrainable point with the given id — including curve
// endpoints/centers, which are not standalone entities — or false if none matches.
func (s *Sketch) PointByID(id ID) (*Point, bool) {
	for _, p := range s.pts {
		if p.id == id {
			return p, true
		}
	}
	return nil, false
}

// AllPoints returns every constrainable point in the sketch — endpoints, centers,
// and standalone points — which are the solver's position variables.
func (s *Sketch) AllPoints() []*Point {
	out := make([]*Point, len(s.pts))
	copy(out, s.pts)
	return out
}

// Lines/Arcs/Circles/Ellipses/Splines/Points/Blocks return the typed entity
// factories (the Lines etc. collections).
func (s *Sketch) Lines() *Lines       { return s.lines }
func (s *Sketch) Arcs() *Arcs         { return s.arcs }
func (s *Sketch) Circles() *Circles   { return s.circles }
func (s *Sketch) Ellipses() *Ellipses { return s.ellipses }
func (s *Sketch) Splines() *Splines   { return s.splines }

// EllipticalArcs returns the elliptical-arc collection.
func (s *Sketch) EllipticalArcs() *EllipticalArcs { return s.ellArcs }

// Images returns the sketch-image collection.
func (s *Sketch) Images() *SketchImages { return s.images }

// FillRegions returns the fill-region collection; TextBoxes the sketch-text collection.
func (s *Sketch) FillRegions() *FillRegions { return s.fills }
func (s *Sketch) TextBoxes() *TextBoxes     { return s.texts }

// EquationCurves/FixedSplines/OffsetSplines return the derived-curve collections.
func (s *Sketch) EquationCurves() *EquationCurves { return s.eqCurves }
func (s *Sketch) FixedSplines() *FixedSplines     { return s.fixedSpl }
func (s *Sketch) OffsetSplines() *OffsetSplines   { return s.offSpl }
func (s *Sketch) Points() *Points                 { return s.points }
func (s *Sketch) Blocks() *Blocks                 { return s.blocks }

// GeometricConstraints returns the sketch's geometric-constraint collection.
func (s *Sketch) GeometricConstraints() *GeometricConstraints { return s.geomCons }

// DimensionConstraints returns the sketch's dimensional-constraint collection.
func (s *Sketch) DimensionConstraints() *DimensionConstraints { return s.dimCons }

// Parameters returns the parameter store backing this sketch's dimensions. By
// default a sketch owns its own; a component definition swaps in the shared store
// via [Sketch.SetParameters] so dimensions join the document's parameter DAG.
func (s *Sketch) Parameters() *param.Parameters { return s.params }

// SetParameters replaces the parameter store (and re-points the dimension
// collection at it). Call before adding dimensions.
func (s *Sketch) SetParameters(ps *param.Parameters) {
	s.params = ps
	s.dimCons.params = ps
}

// Constraints returns every residual-bearing constraint — all geometric plus the
// driving dimensions — which is exactly what the solver (F05) consumes. Driven
// dimensions are excluded (they report, they do not constrain).
func (s *Sketch) Constraints() []Constraint {
	out := s.geomCons.All()
	for _, d := range s.dimCons.items {
		if !d.driven {
			out = append(out, d)
		}
	}
	return out
}

// add appends an entity to the sketch's geometry list.
func (s *Sketch) add(e Entity) { s.ents = append(s.ents, e) }

// newPoint creates a constrainable point at pos and registers it as a solver
// variable. Curve factories use it for endpoints/centers (not added to Entities).
func (s *Sketch) newPoint(pos math.Point2) *Point {
	p := &Point{id: nextID(), X: pos.X, Y: pos.Y}
	s.pts = append(s.pts, p)
	return p
}

// initCollections wires the typed entity factories to this sketch.
func (s *Sketch) initCollections() {
	s.lines = &Lines{s: s}
	s.arcs = &Arcs{s: s}
	s.circles = &Circles{s: s}
	s.ellipses = &Ellipses{s: s}
	s.ellArcs = &EllipticalArcs{s: s}
	s.splines = &Splines{s: s}
	s.points = &Points{s: s}
	s.images = &SketchImages{s: s}
	s.fills = &FillRegions{s: s}
	s.texts = &TextBoxes{s: s}
	s.eqCurves = &EquationCurves{s: s}
	s.fixedSpl = &FixedSplines{s: s}
	s.offSpl = &OffsetSplines{s: s}
	s.blocks = &Blocks{s: s}
	s.geomCons = &GeometricConstraints{}
	s.params = param.NewParameters()
	s.dimCons = &DimensionConstraints{params: s.params}
}

// ToModel maps a sketch-space point to model space via the host plane.
func (s *Sketch) ToModel(p math.Point2) math.Point3 { return s.plane.ToModel(p) }

// ToSketch maps a model-space point onto the sketch plane.
func (s *Sketch) ToSketch(p math.Point3) math.Point2 { return s.plane.ToSketch(p) }

// Sketches is the collection of planar sketches owned by a component definition.
type Sketches struct {
	items []*Sketch
	byID  map[ID]*Sketch
	seq   int // running counter behind the Sketch1, Sketch2, … auto-names
}

// NewSketches returns an empty collection.
func NewSketches() *Sketches {
	return &Sketches{byID: map[ID]*Sketch{}}
}

// Add creates a planar sketch on plane and adds it to the collection, giving it the
// next free auto-name (Sketch1, Sketch2, …) like Inventor.
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
	c.items = append(c.items, s)
	c.byID[s.id] = s
	return s
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

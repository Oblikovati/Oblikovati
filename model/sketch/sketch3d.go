// SPDX-License-Identifier: GPL-2.0-only

package sketch

import (
	"fmt"

	"github.com/Oblikovati/oblikovati/math"
	"github.com/Oblikovati/oblikovati/model/param"
)

// Sketch3D is a non-planar sketch whose geometry lives directly in model 3D space (no
// host plane). It is the path/profile source for sweeps, lofts, coils and pipes. Its
// points carry three solver variables each; the constraint solver is dimension-agnostic
// (it sees every DOF as a []*math.Scalar), so the same Newton/LM core that solves a 2D
// sketch solves this one unchanged (ADR-0009, modeling/00).
//
// Example:
//
//	s := sketches3D.Add()
//	a := s.AddPoint3D(math.P3(0, 0, 0))
//	b := s.AddPoint3D(math.P3(1, 0, 0))
//	s.GeometricConstraints3D().Add(NewCoincident3D(a, b)) // pin b onto a
//	s.Solve()
type Sketch3D struct {
	base
	ents   []Entity
	pts    []*Point3D // free constrainable points — the solver's position variables
	refPts []*Point3D // fixed reference points (projected/included anchors): constrainable but not solved
	byID   map[ID]Entity

	dimensionsVisible bool

	geomCons *GeometricConstraints
	dimCons  *DimensionConstraints3D
	params   *param.Parameters
}

// scalar3DContributor is implemented by 3D entities that own scalar DOFs beyond their
// points (a circle/arc radius, a helix pitch). The solver universe collects these so
// free, unconstrained scalars are counted in the DOF total.
type scalar3DContributor interface {
	scalarDOFs() []*math.Scalar
}

// initSketch3D wires the empty collections of a fresh 3D sketch.
func (s *Sketch3D) initSketch3D() {
	s.byID = map[ID]Entity{}
	s.dimensionsVisible = true
	s.geomCons = &GeometricConstraints{}
	s.params = param.NewParameters()
	s.dimCons = NewDimensionConstraints3D(s.params)
}

// Entities returns the 3D sketch's geometry in insertion order.
func (s *Sketch3D) Entities() []Entity {
	out := make([]Entity, len(s.ents))
	copy(out, s.ents)
	return out
}

// EntityCount returns the number of entities.
func (s *Sketch3D) EntityCount() int { return len(s.ents) }

// EntityByID returns the entity with the given session id, or false if none matches.
func (s *Sketch3D) EntityByID(id ID) (Entity, bool) {
	e, ok := s.byID[id]
	return e, ok
}

// PointByID returns the constrainable 3D point with the given id — including curve
// endpoints/centers, which are not standalone entities — or false if none matches.
func (s *Sketch3D) PointByID(id ID) (*Point3D, bool) {
	for _, p := range s.pts {
		if p.id == id {
			return p, true
		}
	}
	for _, p := range s.refPts {
		if p.id == id {
			return p, true // a fixed reference anchor can be constrained to
		}
	}
	return nil, false
}

// newRefPoint3D creates a fixed reference point (a projected/included anchor): a real
// Point3D other geometry can be constrained to, but excluded from the solver's free
// variables, so the solver holds it in place while other geometry moves to meet it.
func (s *Sketch3D) newRefPoint3D(pos math.Point3) *Point3D {
	p := NewPoint3D(pos)
	s.refPts = append(s.refPts, p)
	return p
}

// AllPoints3D returns every constrainable 3D point — endpoints, centers, and standalone
// points — which are the solver's position variables.
func (s *Sketch3D) AllPoints3D() []*Point3D {
	out := make([]*Point3D, len(s.pts))
	copy(out, s.pts)
	return out
}

// DimensionsVisible reports whether the sketch's dimensions are shown; SetDimensionsVisible
// toggles it (Inventor Sketch3D.DimensionsVisible).
func (s *Sketch3D) DimensionsVisible() bool     { return s.dimensionsVisible }
func (s *Sketch3D) SetDimensionsVisible(v bool) { s.dimensionsVisible = v }

// GeometricConstraints3D returns the sketch's geometric-constraint collection. It reuses
// the dimension-agnostic [GeometricConstraints] container (the 3D constraint factories
// hang off the returned value in M22-F05).
func (s *Sketch3D) GeometricConstraints3D() *GeometricConstraints { return s.geomCons }

// DimensionConstraints3D returns the sketch's dimensional-constraint collection.
func (s *Sketch3D) DimensionConstraints3D() *DimensionConstraints3D { return s.dimCons }

// Parameters returns the parameter store backing this sketch's dimensions.
func (s *Sketch3D) Parameters() *param.Parameters { return s.params }

// SetParameters repoints the sketch (and its dimension collection) at a shared parameter
// store so its dimensions join the document's parameter DAG. Call before adding dimensions.
func (s *Sketch3D) SetParameters(ps *param.Parameters) {
	s.params = ps
	s.dimCons.params = ps
}

// newPoint3D creates a constrainable 3D point at pos and registers it as a solver
// variable (not as an entity — curve factories use it for endpoints/centers).
func (s *Sketch3D) newPoint3D(pos math.Point3) *Point3D {
	p := NewPoint3D(pos)
	s.pts = append(s.pts, p)
	return p
}

// AddPoint3D adds a standalone 3D sketch point (both a solver variable and an entity).
func (s *Sketch3D) AddPoint3D(pos math.Point3) *Point3D {
	p := s.newPoint3D(pos)
	s.addEntity3D(p)
	return p
}

// addEntity3D registers an entity in the geometry list and the id index.
func (s *Sketch3D) addEntity3D(e Entity) {
	s.ents = append(s.ents, e)
	s.byID[e.EntityID()] = e
}

// Constraints returns every residual-bearing constraint — all geometric plus the driving
// dimensions — which is exactly what the solver consumes. Driven dimensions are excluded.
func (s *Sketch3D) Constraints() []Constraint {
	out := s.geomCons.All()
	for _, d := range s.dimCons.items {
		if !d.driven {
			out = append(out, d)
		}
	}
	return out
}

// variables returns the full DOF universe of the 3D sketch: every point's X,Y,Z plus any
// scalar DOFs owned by entities (radii, pitches). The solver needs the whole universe
// (not just constraint-referenced variables) so free geometry is counted in the DOF total.
func (s *Sketch3D) variables() []*math.Scalar {
	vars := make([]*math.Scalar, 0, len(s.pts)*3)
	for _, p := range s.pts {
		vars = append(vars, &p.X, &p.Y, &p.Z)
	}
	for _, e := range s.ents {
		if c, ok := e.(scalar3DContributor); ok {
			vars = append(vars, c.scalarDOFs()...)
		}
	}
	return vars
}

// Solve resolves the 3D sketch from its constraints, updating geometry in place and
// recording health (sick when non-convergent, warning when over-constrained-but-solvable).
func (s *Sketch3D) Solve() SolveResult {
	result := Solve(s.Constraints(), s.variables(), Options{})
	s.health = healthFromSolve(result)
	return result
}

// AnalyzeConstraints reports the DOF/redundancy structure without moving geometry.
func (s *Sketch3D) AnalyzeConstraints() DOFAnalysis {
	return analyzeDOF(s.Constraints(), s.variables())
}

// DegreesOfFreedom returns the sketch's remaining free degrees of freedom (0 when fully
// constrained).
func (s *Sketch3D) DegreesOfFreedom() int { return s.AnalyzeConstraints().DOF }

// Sketches3D is the collection of 3D sketches owned by a component definition.
type Sketches3D struct {
	items []*Sketch3D
	byID  map[ID]*Sketch3D
	seq   int // running counter behind the "3D Sketch1", "3D Sketch2", … auto-names
}

// NewSketches3D returns an empty collection.
func NewSketches3D() *Sketches3D {
	return &Sketches3D{byID: map[ID]*Sketch3D{}}
}

// Add creates a 3D sketch with the next free auto-name ("3D Sketch1", "3D Sketch2", …).
func (c *Sketches3D) Add() *Sketch3D { return c.AddNamed(c.nextName()) }

// AddNamed creates a named 3D sketch.
func (c *Sketches3D) AddNamed(name string) *Sketch3D {
	s := &Sketch3D{base: newBase(name)}
	s.initSketch3D()
	c.items = append(c.items, s)
	c.byID[s.id] = s
	return s
}

// nextName mints the first unused "3D Sketch{N}" name, advancing past taken numbers.
func (c *Sketches3D) nextName() string {
	for {
		c.seq++
		name := fmt.Sprintf("3D Sketch%d", c.seq)
		if !c.nameTaken(name) {
			return name
		}
	}
}

// nameTaken reports whether a sketch in the collection already uses name.
func (c *Sketches3D) nameTaken(name string) bool {
	for _, s := range c.items {
		if s.Name() == name {
			return true
		}
	}
	return false
}

// Count returns the number of 3D sketches; Item returns the i-th (0-based).
func (c *Sketches3D) Count() int           { return len(c.items) }
func (c *Sketches3D) Item(i int) *Sketch3D { return c.items[i] }

// ByID returns the 3D sketch with the given id.
func (c *Sketches3D) ByID(id ID) (*Sketch3D, bool) {
	s, ok := c.byID[id]
	return s, ok
}

// Remove deletes the 3D sketch with the given id, reporting whether it was found. The
// auto-name counter is not rewound (Inventor does not reuse a deleted sketch's number).
func (c *Sketches3D) Remove(id ID) bool {
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

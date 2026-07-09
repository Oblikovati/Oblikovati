// SPDX-License-Identifier: GPL-2.0-only

package feature

import (
	"fmt"

	"oblikovati.org/kernel/geom"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
)

// Edge-derived datums build on a B-rep edge resolved through the edge resolver (work_edge_ref.go):
// an axis along a straight edge, or a point at an edge's midpoint (#1840, #1842).

// edgeAxisDef is an axis lying along a straight B-rep edge. The two Inventor constructors that land
// here — AddByEdge ("analytic-edge") and AddByLine on an edge ("line-by-entity") — share this
// definition; the kind field preserves which one authored the datum. A non-linear edge goes sick.
type edgeAxisDef struct {
	edge WorkRef
	kind string
}

func (d edgeAxisDef) kindName() string { return d.kind }
func (d edgeAxisDef) refs() []WorkRef  { return []WorkRef{d.edge} }
func (d edgeAxisDef) eval(r workResolver) (math.Point3, math.UnitVector3, error) {
	e, err := r.edge(d.edge)
	if err != nil {
		return math.Point3{}, math.UnitVector3{}, err
	}
	return edgeLine(e)
}

// edgeLine returns the origin and unit direction of a straight edge; a non-linear edge is an error.
func edgeLine(e *topo.Edge) (math.Point3, math.UnitVector3, error) {
	switch c := e.Geometry().(type) {
	case geom.Line:
		return c.Origin, c.Dir, nil
	case geom.LineSegment:
		dir, err := math.UnitVector3FromVector(c.StartPoint.VectorTo(c.EndPoint))
		if err != nil {
			return math.Point3{}, math.UnitVector3{}, err
		}
		return c.StartPoint, dir, nil
	default:
		return math.Point3{}, math.UnitVector3{}, fmt.Errorf("edge is not linear (%T); no axis", c)
	}
}

// AddByAnalyticEdge creates the axis coincident with a straight edge (Inventor's AddByEdge, #1840).
func (c *WorkAxes) AddByAnalyticEdge(edge WorkRef) *WorkAxis {
	return c.addUser(edgeAxisDef{edge: edge, kind: "analytic-edge"})
}

// AddByLineByEntity creates the axis along a linear edge (Inventor's AddByLine on an edge, #1840) —
// same geometry as AddByAnalyticEdge; the distinct kind preserves the authoring constructor.
func (c *WorkAxes) AddByLineByEntity(edge WorkRef) *WorkAxis {
	return c.addUser(edgeAxisDef{edge: edge, kind: "line-by-entity"})
}

// edgeMidpointPointDef is the midpoint of a B-rep edge (Inventor's AddByMidpoint, #1842): the
// average of the edge's trimmed endpoints — exact for a straight edge, the chord midpoint for a
// curved one (the linear case is what this targets).
type edgeMidpointPointDef struct{ edge WorkRef }

func (d edgeMidpointPointDef) kindName() string { return "edge-midpoint" }
func (d edgeMidpointPointDef) refs() []WorkRef  { return []WorkRef{d.edge} }
func (d edgeMidpointPointDef) eval(r workResolver) (math.Point3, error) {
	e, err := r.edge(d.edge)
	if err != nil {
		return math.Point3{}, err
	}
	return edgeChordMidpoint(e), nil
}

// edgeChordMidpoint returns the midpoint of an edge's trimmed span (the average of its two vertices,
// computed via point/vector ops so it works for any coordinate scalar type).
func edgeChordMidpoint(e *topo.Edge) math.Point3 {
	a, b := e.StartVertex().Point(), e.EndVertex().Point()
	return a.TranslateBy(a.VectorTo(b).Scale(0.5))
}

// AddByMidpointOfEdge creates the datum point at an edge's midpoint (#1842).
func (c *WorkPoints) AddByMidpointOfEdge(edge WorkRef) *WorkPoint {
	return c.addUser(edgeMidpointPointDef{edge: edge})
}

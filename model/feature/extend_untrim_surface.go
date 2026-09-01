// SPDX-License-Identifier: GPL-2.0-only

package feature

import (
	"errors"

	"oblikovati.org/kernel/ops/surface"

	"oblikovati.org/kernel/geom"
)

// Surface extend and untrim features (M36-F11). Extend lengthens the running surface body's NURBS
// face past an edge with a linear/G1/G2 continuation; Untrim recovers the face's full surface
// domain. Both act on the most recent surface body and replace it. (Distinct from the planar
// ExtendFeature in surface_edit.go, which grows a flat boundary edge.)

// ExtendSurfaceDefinition is the recipe for a NURBS surface extend: which edge, how far, and the
// continuation order (1=linear/tangent G1, 2=curvature G2, 3=G3).
type ExtendSurfaceDefinition struct {
	Edge     geom.Boundary
	Distance float64
	Order    int
}

// ExtendSurfaceFeature lengthens the running NURBS surface past an edge.
type ExtendSurfaceFeature struct {
	def      *ExtendSurfaceDefinition
	featName string
}

// Definition returns the extend recipe.
func (e *ExtendSurfaceFeature) Definition() *ExtendSurfaceDefinition { return e.def }

// Kind implements [Feature].
func (e *ExtendSurfaceFeature) Kind() string { return "extend-surface" }

// Recompute extends the running surface body's NURBS face and replaces it.
func (e *ExtendSurfaceFeature) Recompute(in Input) (Output, error) {
	target, err := lastBody(in, "extend surface")
	if err != nil {
		return Output{}, err
	}
	out, err := surface.ExtendFaceSurface(target, e.def.Edge, e.def.Distance, e.def.Order)
	if err != nil {
		return Output{}, err
	}
	return Output{Bodies: replaceLast(in.Bodies, out)}, nil
}

// ExtendSurfaceFeatures adds NURBS extend features into the engine.
type ExtendSurfaceFeatures struct{ engine *PartFeatures }

// NewExtendSurfaceFeatures binds the collection to a feature engine.
func NewExtendSurfaceFeatures(engine *PartFeatures) *ExtendSurfaceFeatures {
	return &ExtendSurfaceFeatures{engine: engine}
}

// Add extends the running NURBS surface past edge by distance with the continuation order.
func (c *ExtendSurfaceFeatures) Add(edge geom.Boundary, distance float64, order int) *PartFeature {
	def := &ExtendSurfaceDefinition{Edge: edge, Distance: distance, Order: order}
	ef := &ExtendSurfaceFeature{def: def, featName: "ExtendSurface"}
	pf := c.engine.Add(ef)
	ef.featName = pf.name
	return pf
}

// UntrimFeature recovers the full domain of the running surface body's NURBS face.
type UntrimFeature struct{ featName string }

// Kind implements [Feature].
func (u *UntrimFeature) Kind() string { return "untrim-surface" }

// Recompute rebuilds the running surface body's NURBS face on its full domain.
func (u *UntrimFeature) Recompute(in Input) (Output, error) {
	target, err := lastBody(in, "untrim surface")
	if err != nil {
		return Output{}, err
	}
	for _, f := range target.Faces() {
		if _, ok := f.Geometry().(geom.BSplineSurface); ok {
			out, err := surface.UntrimFace(target, f.ReferenceKey())
			if err != nil {
				return Output{}, err
			}
			return Output{Bodies: replaceLast(in.Bodies, out)}, nil
		}
	}
	return Output{}, errors.New("untrim surface: body has no NURBS surface face")
}

// UntrimFeatures adds untrim features into the engine.
type UntrimFeatures struct{ engine *PartFeatures }

// NewUntrimFeatures binds the collection to a feature engine.
func NewUntrimFeatures(engine *PartFeatures) *UntrimFeatures { return &UntrimFeatures{engine: engine} }

// Add recovers the running NURBS surface's full domain.
func (c *UntrimFeatures) Add() *PartFeature {
	uf := &UntrimFeature{featName: "Untrim"}
	pf := c.engine.Add(uf)
	uf.featName = pf.name
	return pf
}

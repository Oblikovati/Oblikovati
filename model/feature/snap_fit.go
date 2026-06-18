// SPDX-License-Identifier: GPL-2.0-only

package feature

import (
	"fmt"

	"oblikovati.org/kernel/ops"
	"oblikovati.org/math"
	"oblikovati.org/model/sketch"
)

// SnapFitDefinition is a cantilever snap-fit connector (#486, M20-F10 plastic features): a beam of
// Length×Width×Thickness with a catch lip (CatchLength×CatchHeight) on top of its free end. The whole
// hook is one extruded step-profile (so it is a single valid solid, no boolean), built at the origin
// on the XZ plane — the beam runs +X, the catch rises +Z, the width extrudes +Y — and joined to the
// running body (or placed as a new body when there is none).
type SnapFitDefinition struct {
	Length      func() float64
	Width       func() float64
	Thickness   func() float64
	CatchLength func() float64
	CatchHeight func() float64
}

// SnapFitFeature adds a cantilever snap-fit hook.
//
// Example: a 20×6×2 mm arm with a 3×1.5 mm catch —
//
//	NewPlasticFeatures(part.Features()).AddCantileverSnapFit(
//	    cf(20), cf(6), cf(2), cf(3), cf(1.5)) // cf wraps a constant in func() float64
type SnapFitFeature struct {
	def      *SnapFitDefinition
	featName string
}

// Definition returns the feature's definition.
func (f *SnapFitFeature) Definition() *SnapFitDefinition { return f.def }

// Kind names the feature type.
func (f *SnapFitFeature) Kind() string { return "snap-fit" }

// Recompute builds the hook and joins it to the running body.
func (f *SnapFitFeature) Recompute(in Input) (Output, error) {
	return snapFitBody(in, f.def, featOr(f.featName, "snap-fit"))
}

// snapFitBody builds the cantilever hook as one step-profile prism and combines it with the running
// body. Non-positive dimensions, or a catch longer than the beam, are a clean error (the feature
// goes Sick) rather than a degenerate solid.
func snapFitBody(in Input, def *SnapFitDefinition, feat string) (Output, error) {
	l, w, t := callOrZero(def.Length), callOrZero(def.Width), callOrZero(def.Thickness)
	cl, ch := callOrZero(def.CatchLength), callOrZero(def.CatchHeight)
	if l <= 0 || w <= 0 || t <= 0 {
		return Output{}, fmt.Errorf("%s: length/width/thickness must be > 0 (got %g, %g, %g)", feat, l, w, t)
	}
	if cl <= 0 || ch <= 0 || cl > l {
		return Output{}, fmt.Errorf("%s: catch length/height must be > 0 and catch length ≤ length (got %g, %g; length %g)", feat, cl, ch, l)
	}
	// The cantilever's side profile (XZ): the beam rectangle [0,l]×[0,t] with the catch bump
	// [l-cl,l]×[t,t+ch] on top of the free end, traced CCW.
	profile := []math.Point2{
		{X: 0, Y: 0}, {X: l, Y: 0},
		{X: l, Y: t + ch},
		{X: l - cl, Y: t + ch},
		{X: l - cl, Y: t},
		{X: 0, Y: t},
	}
	hook := buildPrism(profile, sketch.XZPlane(), span{near: 0, far: w}, 0, feat)
	bodies, err := combine(in.Bodies, hook, ops.Join)
	if err != nil {
		return Output{}, err
	}
	return Output{Bodies: bodies}, nil
}

// PlasticFeatures adds the M20-F10 plastic-part features (snap fits; rest pads to follow) into the
// engine.
type PlasticFeatures struct{ engine *PartFeatures }

// NewPlasticFeatures binds the collection to an engine.
func NewPlasticFeatures(engine *PartFeatures) *PlasticFeatures { return &PlasticFeatures{engine} }

// AddCantileverSnapFit adds a cantilever snap-fit hook with the given beam and catch dimensions.
func (c *PlasticFeatures) AddCantileverSnapFit(length, width, thickness, catchLength, catchHeight func() float64) *PartFeature {
	return c.engine.Add(&SnapFitFeature{def: &SnapFitDefinition{
		Length: length, Width: width, Thickness: thickness, CatchLength: catchLength, CatchHeight: catchHeight,
	}})
}

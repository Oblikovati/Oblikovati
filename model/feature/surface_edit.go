// SPDX-License-Identifier: GPL-2.0-only

package feature

import (
	"errors"

	"oblikovati.org/kernel/ops"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
)

// Surface-editing features (M10-F02) act on the most recent body in the running
// state: trim/offset/mid-surface build real planar geometry (kernel/ops); extend
// resolves its inputs then defers the edge-to-target geometry to a later kernel
// phase (Warning, passthrough).

// lastBody returns the most recently added body in the running state, erroring when
// there is none (the edit has no target).
func lastBody(in Input, what string) (*topo.Body, error) {
	if len(in.Bodies) == 0 {
		return nil, errors.New(what + ": no target body in the running state")
	}
	return in.Bodies[len(in.Bodies)-1], nil
}

// replaceLast returns the running bodies with the last one swapped for body.
func replaceLast(running []*topo.Body, body *topo.Body) []*topo.Body {
	out := append([]*topo.Body(nil), running[:len(running)-1]...)
	return append(out, body)
}

// TrimDefinition is the recipe for a surface trim: the cutting plane and which side
// of it to keep.
type TrimDefinition struct {
	CutOrigin    math.Point3
	CutNormal    math.Vector3
	KeepPositive bool
}

// TrimFeature trims the running surface body along a cutting plane (PBI-111).
type TrimFeature struct {
	def      *TrimDefinition
	featName string
}

// Definition returns the trim recipe.
func (t *TrimFeature) Definition() *TrimDefinition { return t.def }

// Kind implements [Feature].
func (t *TrimFeature) Kind() string { return "trim" }

// Recompute clips the running surface body by the cutting plane and keeps one side.
func (t *TrimFeature) Recompute(in Input) (Output, error) {
	target, err := lastBody(in, "trim")
	if err != nil {
		return Output{}, err
	}
	trimmed, err := ops.TrimByPlane(target, t.def.CutOrigin, t.def.CutNormal, t.def.KeepPositive, t.featName)
	if err != nil {
		return Output{}, err
	}
	return Output{Bodies: replaceLast(in.Bodies, trimmed)}, nil
}

// TrimFeatures adds trim features into the engine.
type TrimFeatures struct{ engine *PartFeatures }

// NewTrimFeatures binds the collection to a feature engine.
func NewTrimFeatures(engine *PartFeatures) *TrimFeatures { return &TrimFeatures{engine: engine} }

// AddByPlane trims the running surface body along the cutting plane (origin+normal),
// keeping the positive side when keepPositive.
func (c *TrimFeatures) AddByPlane(origin math.Point3, normal math.Vector3, keepPositive bool) *PartFeature {
	def := &TrimDefinition{CutOrigin: origin, CutNormal: normal, KeepPositive: keepPositive}
	tf := &TrimFeature{def: def, featName: "Trim"}
	pf := c.engine.Add(tf)
	tf.featName = pf.name
	return pf
}

// ExtendDefinition is the recipe for extending a planar surface's boundary edge outward by a
// distance (the edge held by reference key, re-resolved each recompute).
type ExtendDefinition struct {
	EdgeKey  []byte
	Distance func() float64
}

// ExtendFeature extends a planar surface body's boundary edge, growing the face (PBI-111).
// Multi-face and curved-surface extension are the remaining phase-C (NURBS) work.
type ExtendFeature struct {
	def      *ExtendDefinition
	featName string
}

// Definition returns the extend recipe.
func (e *ExtendFeature) Definition() *ExtendDefinition { return e.def }

// Kind implements [Feature].
func (e *ExtendFeature) Kind() string { return "extend" }

// Recompute resolves the boundary edge and extends its face outward; a lost edge → Sick.
func (e *ExtendFeature) Recompute(in Input) (Output, error) {
	body, err := lastBody(in, "extend")
	if err != nil {
		return Output{}, err
	}
	extended, err := ops.ExtendByEdge(body, e.def.EdgeKey, callOrZero(e.def.Distance), e.featName)
	if err != nil {
		return Output{}, err
	}
	return Output{Bodies: replaceLast(in.Bodies, extended)}, nil
}

// ExtendFeatures adds extend features into the engine.
type ExtendFeatures struct{ engine *PartFeatures }

// NewExtendFeatures binds the collection to a feature engine.
func NewExtendFeatures(engine *PartFeatures) *ExtendFeatures { return &ExtendFeatures{engine: engine} }

// Add extends the boundary edge (by reference key) outward by distance.
func (c *ExtendFeatures) Add(edgeKey []byte, distance func() float64) *PartFeature {
	ef := &ExtendFeature{def: &ExtendDefinition{EdgeKey: edgeKey, Distance: distance}, featName: "Extend"}
	pf := c.engine.Add(ef)
	ef.featName = pf.name
	return pf
}

// SurfaceOffsetDefinition is the recipe for a surface offset: the distance along the
// face normal.
type SurfaceOffsetDefinition struct {
	Distance func() float64
}

// SurfaceOffsetFeature offsets the running surface body along its normal (PBI-112).
type SurfaceOffsetFeature struct {
	def      *SurfaceOffsetDefinition
	featName string
}

// Definition returns the offset recipe.
func (o *SurfaceOffsetFeature) Definition() *SurfaceOffsetDefinition { return o.def }

// Kind implements [Feature].
func (o *SurfaceOffsetFeature) Kind() string { return "surface-offset" }

// Recompute offsets the running surface body by the distance along its face normal.
func (o *SurfaceOffsetFeature) Recompute(in Input) (Output, error) {
	target, err := lastBody(in, "face offset")
	if err != nil {
		return Output{}, err
	}
	dist := measure(o.def.Distance)
	if dist == 0 {
		return Output{}, errors.New("face offset: distance is zero")
	}
	offset, err := ops.OffsetSurface(target, dist, o.featName)
	if err != nil {
		return Output{}, err
	}
	return Output{Bodies: replaceLast(in.Bodies, offset)}, nil
}

// SurfaceOffsetFeatures adds offset features into the engine.
type SurfaceOffsetFeatures struct{ engine *PartFeatures }

// NewSurfaceOffsetFeatures binds the collection to a feature engine.
func NewSurfaceOffsetFeatures(engine *PartFeatures) *SurfaceOffsetFeatures {
	return &SurfaceOffsetFeatures{engine: engine}
}

// AddByDistance offsets the running surface body by distance along its normal.
func (c *SurfaceOffsetFeatures) AddByDistance(distance func() float64) *PartFeature {
	of := &SurfaceOffsetFeature{def: &SurfaceOffsetDefinition{Distance: distance}, featName: "Offset"}
	pf := c.engine.Add(of)
	of.featName = pf.name
	return pf
}

// MidSurfaceThickness records one paired-face wall thickness measured during
// mid-surface extraction (consumed by FEA shell meshing, M18).
type MidSurfaceThickness struct {
	Value float64
}

// MidSurfaceThicknesses is the ordered set of per-pair thicknesses of a mid-surface.
type MidSurfaceThicknesses struct {
	items []*MidSurfaceThickness
}

// Count returns the number of recorded thicknesses; Item returns the i-th.
func (ts *MidSurfaceThicknesses) Count() int                      { return len(ts.items) }
func (ts *MidSurfaceThicknesses) Item(i int) *MidSurfaceThickness { return ts.items[i] }

func (ts *MidSurfaceThicknesses) reset() { ts.items = nil }
func (ts *MidSurfaceThicknesses) add(v float64) {
	ts.items = append(ts.items, &MidSurfaceThickness{Value: v})
}

// MidSurfaceDefinition is the recipe for a mid-surface: the maximum wall thickness a
// face pair may have to be treated as a thin wall.
type MidSurfaceDefinition struct {
	MaxThickness float64
}

// MidSurfaceFeature extracts mid-surfaces from the running solid's thin-wall planar
// face pairs, recording each pair's thickness for FEA (PBI-112).
type MidSurfaceFeature struct {
	def         *MidSurfaceDefinition
	thicknesses *MidSurfaceThicknesses
	featName    string
}

// Definition returns the mid-surface recipe.
func (m *MidSurfaceFeature) Definition() *MidSurfaceDefinition { return m.def }

// Thicknesses returns the recorded per-pair wall thicknesses.
func (m *MidSurfaceFeature) Thicknesses() *MidSurfaceThicknesses { return m.thicknesses }

// Kind implements [Feature].
func (m *MidSurfaceFeature) Kind() string { return "mid-surface" }

// Recompute extracts the mid-surface patches and records their thicknesses, replacing
// the running solid with the mid-surface bodies.
func (m *MidSurfaceFeature) Recompute(in Input) (Output, error) {
	target, err := lastBody(in, "mid-surface")
	if err != nil {
		return Output{}, err
	}
	patches, err := ops.MidSurfaces(target, m.def.MaxThickness, m.featName)
	if err != nil {
		return Output{}, err
	}
	m.thicknesses.reset()
	bodies := make([]*topo.Body, len(patches))
	for i, p := range patches {
		bodies[i] = p.Body
		m.thicknesses.add(p.Thickness)
	}
	return Output{Bodies: bodies}, nil
}

// MidSurfaceFeatures adds mid-surface features into the engine.
type MidSurfaceFeatures struct{ engine *PartFeatures }

// NewMidSurfaceFeatures binds the collection to a feature engine.
func NewMidSurfaceFeatures(engine *PartFeatures) *MidSurfaceFeatures {
	return &MidSurfaceFeatures{engine: engine}
}

// AddByThickness extracts mid-surfaces from face pairs no thicker than maxThickness.
func (c *MidSurfaceFeatures) AddByThickness(maxThickness float64) *PartFeature {
	mf := &MidSurfaceFeature{def: &MidSurfaceDefinition{MaxThickness: maxThickness}, thicknesses: &MidSurfaceThicknesses{}, featName: "MidSurface"}
	pf := c.engine.Add(mf)
	mf.featName = pf.name
	return pf
}

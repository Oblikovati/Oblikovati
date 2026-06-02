// SPDX-License-Identifier: GPL-2.0-only

package feature

import "github.com/Oblikovati/oblikovati/kernel/topo"

// BodySource is the consumer-side view of a part's evaluated content that a derived
// feature pulls from — defined here so feature does not import compdef (avoiding a
// cycle; compdef.PartComponentDefinition satisfies it structurally). A change to
// the source bumps ModelGeometryVersion, which a derived feature can watch for
// associative update.
type BodySource interface {
	SurfaceBodies() *topo.SurfaceBodies
	ModelGeometryVersion() string
}

// DerivedPartComponent pulls geometry associatively from another part document
// (PBI-097): each recompute re-reads the source's bodies, so a source edit flows
// through. Scale/Mirror are recorded; applying the geometric transform needs a
// kernel body-transform op (it arrives with assembly occurrence transforms, M11),
// so for now the geometry is pulled as-is.
type DerivedPartComponent struct {
	source BodySource
	Scale  float64
	Mirror bool
}

// Definition returns the derived recipe.
func (d *DerivedPartComponent) Definition() *DerivedPartComponent { return d }

// SourceVersion returns the source's current geometry version (for change tracking).
func (d *DerivedPartComponent) SourceVersion() string { return d.source.ModelGeometryVersion() }

// Kind implements [Feature].
func (d *DerivedPartComponent) Kind() string { return "derived" }

// Recompute appends the source's bodies to the running state.
func (d *DerivedPartComponent) Recompute(in Input) (Output, error) {
	out := append([]*topo.Body(nil), in.Bodies...)
	out = append(out, d.source.SurfaceBodies().All()...)
	return Output{Bodies: out}, nil
}

// NonParametricBaseFeature wraps imported/base bodies (from translation, M17) as a
// feature-tree participant, so downstream parametric features can operate on them
// (PBI-098). The wrapped bodies are frozen (non-parametric).
type NonParametricBaseFeature struct {
	bodies []*topo.Body
}

// Definition returns the wrapped bodies.
func (b *NonParametricBaseFeature) Bodies() []*topo.Body {
	return append([]*topo.Body(nil), b.bodies...)
}

// Kind implements [Feature].
func (b *NonParametricBaseFeature) Kind() string { return "base" }

// Recompute appends the frozen base bodies to the running state.
func (b *NonParametricBaseFeature) Recompute(in Input) (Output, error) {
	return Output{Bodies: append(append([]*topo.Body(nil), in.Bodies...), b.bodies...)}, nil
}

// DerivedComponents and BaseFeatures add derived/imported features into the engine.
type (
	DerivedComponents struct{ engine *PartFeatures }
	BaseFeatures      struct{ engine *PartFeatures }
)

// NewDerivedComponents / NewBaseFeatures bind the collections to an engine.
func NewDerivedComponents(engine *PartFeatures) *DerivedComponents { return &DerivedComponents{engine} }
func NewBaseFeatures(engine *PartFeatures) *BaseFeatures           { return &BaseFeatures{engine} }

// AddDerived adds an associative derived component pulling from source.
func (c *DerivedComponents) AddDerived(source BodySource, scale float64, mirror bool) *PartFeature {
	return c.engine.Add(&DerivedPartComponent{source: source, Scale: scale, Mirror: mirror})
}

// AddBase adds a non-parametric base feature wrapping imported bodies.
func (c *BaseFeatures) AddBase(bodies ...*topo.Body) *PartFeature {
	return c.engine.Add(&NonParametricBaseFeature{bodies: bodies})
}

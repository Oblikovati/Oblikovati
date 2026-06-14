// SPDX-License-Identifier: GPL-2.0-only

package feature

import (
	"fmt"

	"oblikovati.org/kernel/ops"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
)

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
// (PBI-097): each recompute re-reads the source's bodies and applies the derive
// transform, so a source edit flows through. The transform is a general affine — the
// identity pulls the source as-is; a reflection (negative determinant) produces an
// opposite-hand copy, the basis of mirror-into-a-handed-part (#717). Like the derived
// assembly, it carries the source's identity link to detect a stale source across
// sessions (#715) and a break-link freeze.
type DerivedPartComponent struct {
	source    BodySource   // nil after restore until BindSource rebinds it
	transform math.Matrix4 // applied to each pulled body; a reflection (det<0) mirrors it
	link      DeriveSourceLink
	linked    bool
	frozen    []*topo.Body
	outOfDate bool
}

// Definition returns the derived recipe.
func (d *DerivedPartComponent) Definition() *DerivedPartComponent { return d }

// SourceVersion returns the source's current geometry version, or "" when the source is
// not bound (after restore, before [BindSource]).
func (d *DerivedPartComponent) SourceVersion() string {
	if d.source == nil {
		return ""
	}
	return d.source.ModelGeometryVersion()
}

// SourceLink returns the persisted identity of the derive's source document (#715).
func (d *DerivedPartComponent) SourceLink() DeriveSourceLink { return d.link }

// OutOfDate reports whether the source has been edited since this derive was saved.
func (d *DerivedPartComponent) OutOfDate() bool { return d.outOfDate }

// Transform returns the geometry transform the derive applies to its source bodies.
func (d *DerivedPartComponent) Transform() math.Matrix4 { return d.transform }

// Linked reports whether the derive still pulls from its source.
func (d *DerivedPartComponent) Linked() bool { return d.linked }

// BindSource (re)binds the live source after a restore and recomputes staleness: the
// derive is out of date when currentDBRevID differs from the revision in the link.
func (d *DerivedPartComponent) BindSource(source BodySource, currentDBRevID string) {
	d.source = source
	d.outOfDate = d.link.DatabaseRevisionID != "" && currentDBRevID != "" && currentDBRevID != d.link.DatabaseRevisionID
}

// Kind implements [Feature].
func (d *DerivedPartComponent) Kind() string { return "derived" }

// BreakLink freezes the current derived bodies and severs the source link.
func (d *DerivedPartComponent) BreakLink() error {
	bodies, err := d.build()
	if err != nil {
		return err
	}
	d.frozen = bodies
	d.linked = false
	return nil
}

// Recompute appends the derived bodies (or, after a broken link, the frozen bodies) to
// the running state, sharing the associative-or-frozen path with the derived assembly.
func (d *DerivedPartComponent) Recompute(in Input) (Output, error) {
	return recomputeLinked(in, d.linked, d.frozen, d.build)
}

// build transforms each source body by the derive transform. An unbound source — a
// restored derive not yet resolved (or missing) — yields no bodies, so a recompute
// before/without binding is safe.
func (d *DerivedPartComponent) build() ([]*topo.Body, error) {
	if d.source == nil {
		return nil, nil
	}
	var out []*topo.Body
	for i, b := range d.source.SurfaceBodies().All() {
		placed, err := d.placeBody(b, i)
		if err != nil {
			return nil, fmt.Errorf("feature: derived-part transform of body %d: %w", i, err)
		}
		out = append(out, placed)
	}
	return out, nil
}

// placeBody applies the derive transform to one source body. The identity short-circuits
// to the source body unchanged; otherwise [ops.TransformBody] transforms it (flipping
// winding for a reflection) under a distinct lineage so the copy gets independent keys.
func (d *DerivedPartComponent) placeBody(b *topo.Body, index int) (*topo.Body, error) {
	if d.transform.Cells() == math.Identity4().Cells() {
		return b, nil
	}
	return ops.TransformBody(b, d.transform, derivedPartLineage(index))
}

// derivedPartLineage gives each transformed copy a distinct lineage prefix, so the
// derived bodies get reference keys independent of the source's.
func derivedPartLineage(index int) func(topo.Lineage) topo.Lineage {
	prefix := topo.Tok("derivedPart", "body", index)
	return func(l topo.Lineage) topo.Lineage {
		return topo.NewLineage(append([]topo.LineageToken{prefix}, l.Tokens()...)...)
	}
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

// AddDerived adds an associative derived component pulling from source and applying
// transform to its bodies (identity = pull as-is; a reflection mirrors them). link
// records the source document's identity so the derive survives a save and detects a
// stale source on reopen (#715).
func (c *DerivedComponents) AddDerived(source BodySource, transform math.Matrix4, link DeriveSourceLink) *PartFeature {
	return c.engine.Add(&DerivedPartComponent{source: source, transform: transform, link: link, linked: true})
}

// RestoreDerivedPart rebuilds a derived-part component from its persisted recipe — the
// source identity link, the geometry transform, and the linked flag — all UNBOUND. The
// live source is rebound later by [DerivedPartComponent.BindSource] once the part's
// reference graph resolves the source document (#715). Until then it contributes no
// geometry.
func RestoreDerivedPart(link DeriveSourceLink, transform math.Matrix4, linked bool) *DerivedPartComponent {
	return &DerivedPartComponent{transform: transform, link: link, linked: linked}
}

// AddBase adds a non-parametric base feature wrapping imported bodies.
func (c *BaseFeatures) AddBase(bodies ...*topo.Body) *PartFeature {
	return c.engine.Add(&NonParametricBaseFeature{bodies: bodies})
}

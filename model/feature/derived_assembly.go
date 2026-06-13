// SPDX-License-Identifier: GPL-2.0-only

package feature

import (
	"fmt"

	"oblikovati.org/kernel/ops"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
	"oblikovati.org/model/occurrence"
)

// DeriveStyle controls how one source occurrence contributes to a derived assembly's
// base body — the reference API's per-occurrence derive style. The zero value is
// DeriveInclude, so an unstyled occurrence is merged in.
type DeriveStyle int

const (
	// DeriveInclude merges the occurrence's bodies into the derived base.
	DeriveInclude DeriveStyle = iota
	// DeriveExclude omits the occurrence entirely.
	DeriveExclude
	// DeriveSubtract cuts the occurrence's bodies from the merged base.
	DeriveSubtract
)

// String returns the lowercase name of the derive style.
func (s DeriveStyle) String() string {
	switch s {
	case DeriveInclude:
		return "include"
	case DeriveExclude:
		return "exclude"
	case DeriveSubtract:
		return "subtract"
	default:
		return "unknown"
	}
}

// PlacedBody is one source body paired with the world transform of the occurrence that
// places it (in the source assembly's space) and that occurrence, so a derived feature
// can apply per-occurrence [DeriveStyle]. Flattening a source assembly's occurrence
// tree yields these.
//
// Path is the instance-name path from the assembly root to the placing occurrence
// (root first). Because nested sub-occurrences are shared flyweights, Source alone is
// ambiguous when a sub-assembly is placed more than once — the same Source is reached
// through several paths; Path disambiguates which placement this body belongs to (the
// reference API's ComponentOccurrence.OccurrencePath). It is empty for a body placed by
// a top-level occurrence in a non-nested flatten.
type PlacedBody struct {
	Body      *topo.Body
	Transform math.Matrix4
	Source    *occurrence.Occurrence
	Path      occurrence.OccurrencePath
}

// AssemblyBodySource is the consumer-side view of a source assembly that a
// derived-assembly feature pulls from: the placed bodies of its occurrence tree and a
// geometry version to watch for associative update. Defined here so feature does not
// import compdef (compdef.AssemblyComponentDefinition satisfies it structurally,
// avoiding a cycle — compdef imports feature).
type AssemblyBodySource interface {
	PlacedBodies() []PlacedBody
	ModelGeometryVersion() string
}

// DerivedAssemblyComponent derives a source assembly into this part as a base body:
// each recompute pulls the source's placed bodies, transforms each into the part, and
// merges the included ones — cutting the subtracted, skipping the excluded — into one
// multi-lump base. It is the assembly-side counterpart of [DerivedPartComponent]
// (M11-F06): a source edit bumps the source version so a re-derive flows it through,
// and BreakLink freezes the current result and stops pulling.
type DerivedAssemblyComponent struct {
	source AssemblyBodySource
	styles map[*occurrence.Occurrence]DeriveStyle
	linked bool
	frozen []*topo.Body
}

// Definition returns the derived recipe.
func (d *DerivedAssemblyComponent) Definition() *DerivedAssemblyComponent { return d }

// Kind implements [Feature].
func (d *DerivedAssemblyComponent) Kind() string { return "derivedAssembly" }

// SourceVersion returns the source assembly's current geometry version (change tracking).
func (d *DerivedAssemblyComponent) SourceVersion() string { return d.source.ModelGeometryVersion() }

// SetStyle sets the derive style for a source occurrence (default DeriveInclude).
func (d *DerivedAssemblyComponent) SetStyle(o *occurrence.Occurrence, style DeriveStyle) {
	d.styles[o] = style
}

// Linked reports whether the derive still pulls from its source.
func (d *DerivedAssemblyComponent) Linked() bool { return d.linked }

// BreakLink freezes the current derived bodies and severs the source link, so the part
// keeps the derived geometry without further updates (the reference API's break link).
func (d *DerivedAssemblyComponent) BreakLink() error {
	bodies, err := d.derive()
	if err != nil {
		return err
	}
	d.frozen = bodies
	d.linked = false
	return nil
}

// Recompute appends the derived base body (or, after a broken link, the frozen bodies)
// to the running state.
func (d *DerivedAssemblyComponent) Recompute(in Input) (Output, error) {
	return recomputeLinked(in, d.linked, d.frozen, d.derive)
}

// recomputeLinked is the shared associative-or-frozen recompute for the derive-family
// features (derived-assembly, shrinkwrap): while linked it rebuilds from source via
// build and appends the result; once the link is broken it appends the frozen bodies
// captured at BreakLink instead. Keeps the two features' update semantics identical.
func recomputeLinked(in Input, linked bool, frozen []*topo.Body, build func() ([]*topo.Body, error)) (Output, error) {
	out := append([]*topo.Body(nil), in.Bodies...)
	if !linked {
		return Output{Bodies: append(out, frozen...)}, nil
	}
	built, err := build()
	if err != nil {
		return Output{}, err
	}
	return Output{Bodies: append(out, built...)}, nil
}

// derive flattens, transforms, and combines the source's placed bodies per style into
// the derived base bodies (zero bodies when nothing is included, otherwise one).
func (d *DerivedAssemblyComponent) derive() ([]*topo.Body, error) {
	var joins, cuts []*topo.Body
	for i, pb := range d.source.PlacedBodies() {
		style := d.styles[pb.Source]
		if style == DeriveExclude {
			continue
		}
		placed, err := ops.TransformBody(pb.Body, pb.Transform, deriveLineage(i))
		if err != nil {
			return nil, fmt.Errorf("feature: derive-assembly transform of body %d: %w", i, err)
		}
		if style == DeriveSubtract {
			cuts = append(cuts, placed)
		} else {
			joins = append(joins, placed)
		}
	}
	if len(joins) == 0 {
		return nil, nil
	}
	base := topo.MergeBodies(topo.NewLineage(topo.Tok("derivedAssembly", "body", 0)), true, joins...)
	for _, tool := range cuts {
		cut, err := ops.Boolean(ops.Cut, base, tool)
		if err != nil {
			return nil, fmt.Errorf("feature: derive-assembly subtract: %w", err)
		}
		base = cut
	}
	return []*topo.Body{base}, nil
}

// deriveLineage gives each placed copy a distinct lineage prefix, so the same source
// part placed at several occurrences yields independent reference keys.
func deriveLineage(index int) func(topo.Lineage) topo.Lineage {
	prefix := topo.Tok("derivedAssembly", "occ", index)
	return func(l topo.Lineage) topo.Lineage {
		return topo.NewLineage(append([]topo.LineageToken{prefix}, l.Tokens()...)...)
	}
}

// DerivedAssemblyComponents adds derived-assembly features into the engine.
type DerivedAssemblyComponents struct{ engine *PartFeatures }

// NewDerivedAssemblyComponents binds the collection to an engine.
func NewDerivedAssemblyComponents(engine *PartFeatures) *DerivedAssemblyComponents {
	return &DerivedAssemblyComponents{engine}
}

// AddDerived adds an associative derived-assembly component pulling from source.
func (c *DerivedAssemblyComponents) AddDerived(source AssemblyBodySource) *PartFeature {
	d := &DerivedAssemblyComponent{
		source: source,
		styles: map[*occurrence.Occurrence]DeriveStyle{},
		linked: true,
	}
	return c.engine.Add(d)
}

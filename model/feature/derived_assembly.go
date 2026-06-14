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

// DeriveStyleFromName parses a derive style spelling (the inverse of [DeriveStyle.String]),
// reporting whether it is a known style. Used to restore a persisted per-occurrence style.
func DeriveStyleFromName(name string) (DeriveStyle, bool) {
	switch name {
	case "include":
		return DeriveInclude, true
	case "exclude":
		return DeriveExclude, true
	case "subtract":
		return DeriveSubtract, true
	default:
		return DeriveInclude, false
	}
}

// DeriveSourceLink identifies the source assembly document a derived component pulls from,
// captured when the derive is created so the link survives a save (#715). Document is the
// full document name to re-resolve on reopen; InternalName is the source's identity GUID;
// DatabaseRevisionID is the source's recipe revision at derive time — a different revision
// on reopen means the source changed since, i.e. the derive is out of date.
type DeriveSourceLink struct {
	Document           string
	InternalName       string
	DatabaseRevisionID string
}

// DeriveStyleAtPath pairs a non-default derive style with the occurrence path it applies
// to — the persistable, pointer-free form of the per-occurrence styles map. Path is the
// root-first instance-name path of the styled source placement ([PlacedBody.Path]).
type DeriveStyleAtPath struct {
	Path  occurrence.OccurrencePath
	Style DeriveStyle
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

// AssemblyDeriveBinder is the derive-family feature that pulls from an assembly source —
// the derived-assembly and the shrinkwrap. It exposes the persisted source link and
// rebinds the live assembly source on reopen, so the part resolver can rebind both kinds
// uniformly (#715). The derived-PART feature is NOT one of these — it binds a part
// [BodySource], a different interface.
type AssemblyDeriveBinder interface {
	SourceLink() DeriveSourceLink
	BindSource(source AssemblyBodySource, currentDBRevID string)
	// RelinkSource updates the link's source document name to where it actually resolved,
	// so a reference relocated with a moved project tree re-binds and re-saves clean (#750).
	RelinkSource(document string)
}

// DeriveStatus is the drive-state surface common to every derive-family feature
// (derived-assembly, derived-part, shrinkwrap): the persisted source link, whether the
// source is out of date, whether the link is still live, and acknowledging the source to
// re-sync. It is what the public deriveStatus/deriveUpdate surface drives (#751).
type DeriveStatus interface {
	SourceLink() DeriveSourceLink
	OutOfDate() bool
	Linked() bool
	// AcknowledgeSource re-stamps the link's source revision to currentDBRevID and clears
	// the out-of-date flag, recording the source's current state as the new baseline.
	AcknowledgeSource(currentDBRevID string)
}

// Every derive-family feature reports and re-syncs its drive state.
var (
	_ DeriveStatus = (*DerivedAssemblyComponent)(nil)
	_ DeriveStatus = (*DerivedPartComponent)(nil)
	_ DeriveStatus = (*ShrinkwrapComponent)(nil)
)

// DerivedAssemblyComponent derives a source assembly into this part as a base body:
// each recompute pulls the source's placed bodies, transforms each into the part, and
// merges the included ones — cutting the subtracted, skipping the excluded — into one
// multi-lump base. It is the assembly-side counterpart of [DerivedPartComponent]
// (M11-F06): a source edit bumps the source version so a re-derive flows it through,
// and BreakLink freezes the current result and stops pulling.
type DerivedAssemblyComponent struct {
	source AssemblyBodySource // nil after restore until BindSource rebinds it
	styles map[*occurrence.Occurrence]DeriveStyle
	linked bool
	frozen []*topo.Body
	// link is the persisted identity of the source document (#715); zero for a derive
	// built without one (e.g. a pure in-memory test source).
	link DeriveSourceLink
	// pendingStyles holds path-keyed styles parsed from the recipe but not yet rebound to
	// live occurrences — applied in BindSource once the source's PlacedBodies exist.
	pendingStyles []DeriveStyleAtPath
	// outOfDate records, after a rebind, that the source's current recipe revision differs
	// from the one captured in link (the source was edited since this derive was saved).
	outOfDate bool
}

// Definition returns the derived recipe.
func (d *DerivedAssemblyComponent) Definition() *DerivedAssemblyComponent { return d }

// Kind implements [Feature].
func (d *DerivedAssemblyComponent) Kind() string { return "derivedAssembly" }

// SourceVersion returns the source assembly's current geometry version (change tracking),
// or "" when the source is not bound (after restore, before [BindSource]).
func (d *DerivedAssemblyComponent) SourceVersion() string {
	if d.source == nil {
		return ""
	}
	return d.source.ModelGeometryVersion()
}

// SourceLink returns the persisted identity of the derive's source document (#715).
func (d *DerivedAssemblyComponent) SourceLink() DeriveSourceLink { return d.link }

// OutOfDate reports whether the source has been edited since this derive was saved — the
// source resolved on reopen carries a different recipe revision than [DeriveSourceLink]
// captured. Always false for an in-session derive (the link matches the live source).
func (d *DerivedAssemblyComponent) OutOfDate() bool { return d.outOfDate }

// AcknowledgeSource re-stamps the link's source revision and clears out-of-date (#751).
func (d *DerivedAssemblyComponent) AcknowledgeSource(currentDBRevID string) {
	d.link.DatabaseRevisionID = currentDBRevID
	d.outOfDate = false
}

// RelinkSource updates the link's source document name (#750).
func (d *DerivedAssemblyComponent) RelinkSource(document string) { d.link.Document = document }

// BindSource (re)binds the live source assembly after a restore and recomputes staleness:
// the derive is out of date when currentDBRevID (the source's revision now) differs from
// the revision captured in the link. It also rebinds the path-keyed pending styles to the
// now-live occurrences. Both revisions must be non-empty for a meaningful comparison.
func (d *DerivedAssemblyComponent) BindSource(source AssemblyBodySource, currentDBRevID string) {
	d.source = source
	d.outOfDate = d.link.DatabaseRevisionID != "" && currentDBRevID != "" && currentDBRevID != d.link.DatabaseRevisionID
	d.rebindStyles()
}

// rebindStyles re-keys pendingStyles (path → style) onto the live source occurrences by
// matching each placement's path, then clears them. A path no longer present in the source
// is dropped (the placement was removed upstream).
func (d *DerivedAssemblyComponent) rebindStyles() {
	if len(d.pendingStyles) == 0 {
		return
	}
	byPath := make(map[string]DeriveStyle, len(d.pendingStyles))
	for _, e := range d.pendingStyles {
		byPath[e.Path.Key()] = e.Style
	}
	for _, pb := range d.source.PlacedBodies() {
		if style, ok := byPath[pb.Path.Key()]; ok {
			d.styles[pb.Source] = style
		}
	}
	d.pendingStyles = nil
}

// SetStyle sets the derive style for a source occurrence (default DeriveInclude).
func (d *DerivedAssemblyComponent) SetStyle(o *occurrence.Occurrence, style DeriveStyle) {
	d.styles[o] = style
}

// StylesByPath renders the non-default per-occurrence styles in their persistable,
// pointer-free form, keyed by each styled placement's path. Returns nil when every
// occurrence uses the default (include), keeping the recipe minimal.
func (d *DerivedAssemblyComponent) StylesByPath() []DeriveStyleAtPath {
	if d.source == nil || len(d.styles) == 0 {
		return nil
	}
	var out []DeriveStyleAtPath
	seen := map[string]bool{} // an occurrence with several bodies repeats its path; record once
	for _, pb := range d.source.PlacedBodies() {
		style, ok := d.styles[pb.Source]
		if !ok || style == DeriveInclude || seen[pb.Path.Key()] {
			continue
		}
		seen[pb.Path.Key()] = true
		out = append(out, DeriveStyleAtPath{Path: pb.Path, Style: style})
	}
	return out
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
// the derived base bodies (zero bodies when nothing is included, otherwise one). An
// unbound source — a restored derive whose source has not been resolved yet (or is
// missing) — yields no bodies, so a recompute before/without binding is safe.
func (d *DerivedAssemblyComponent) derive() ([]*topo.Body, error) {
	if d.source == nil {
		return nil, nil
	}
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
	return combineDerived(joins, cuts)
}

// combineDerived merges the included bodies into one base and cuts the subtracted tools
// from it, yielding zero bodies when nothing is included or the single merged base.
func combineDerived(joins, cuts []*topo.Body) ([]*topo.Body, error) {
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

// AddDerived adds an associative derived-assembly component pulling from source, recording
// link — the source document's identity — so the derive survives a save and can detect a
// stale source on reopen (#715).
func (c *DerivedAssemblyComponents) AddDerived(source AssemblyBodySource, link DeriveSourceLink) *PartFeature {
	d := &DerivedAssemblyComponent{
		source: source,
		styles: map[*occurrence.Occurrence]DeriveStyle{},
		linked: true,
		link:   link,
	}
	return c.engine.Add(d)
}

// RestoreDerivedAssembly rebuilds a derived-assembly component from its persisted recipe:
// the source identity link, the linked flag, and the path-keyed styles — all UNBOUND. The
// live source is rebound later by [DerivedAssemblyComponent.BindSource] once the part's
// reference graph resolves the source document (#715), at which point styles re-key and
// staleness is computed. Until then the derive contributes no geometry.
func RestoreDerivedAssembly(link DeriveSourceLink, linked bool, styles []DeriveStyleAtPath) *DerivedAssemblyComponent {
	return &DerivedAssemblyComponent{
		styles:        map[*occurrence.Occurrence]DeriveStyle{},
		linked:        linked,
		link:          link,
		pendingStyles: styles,
	}
}

// SPDX-License-Identifier: GPL-2.0-only

package feature

import (
	"fmt"

	"oblikovati.org/api/types"
	"oblikovati.org/kernel/brep"
	"oblikovati.org/kernel/ops"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
)

// ShrinkwrapRemoveStyle selects which source parts a shrinkwrap drops before merging — the reference
// API's shrinkwrap remove style. It ALIASES the canonical api/types definition (ADR-0018: defined once
// in the Apache-2.0 contract, same int values 0/1/2 and wire spellings). Removal makes the result
// lighter by discarding parts a viewer never sees or that are too small to matter.
type ShrinkwrapRemoveStyle = types.ShrinkwrapRemoveStyle

const (
	RemoveNone          = types.RemoveNone          // keeps every part (the full, unsimplified set)
	RemoveSmallParts    = types.RemoveSmallParts    // drops parts whose body volume is below MinPartVolume
	RemoveInternalParts = types.RemoveInternalParts // drops parts fully enclosed by other parts
)

// ShrinkwrapEnvelopeStyle selects how kept parts are replaced by simpler proxy geometry — the reference
// API's envelopes-replace style. It ALIASES the canonical api/types definition (ADR-0018). Envelopes
// erase internal detail (and, by construction, holes) so the result is a lightweight closed solid.
type ShrinkwrapEnvelopeStyle = types.ShrinkwrapEnvelopeStyle

const (
	EnvelopeNone    = types.EnvelopeNone    // keeps each kept part's real geometry
	EnvelopePerPart = types.EnvelopePerPart // replaces each kept part with its axis-aligned bounding box
	EnvelopeWhole   = types.EnvelopeWhole   // replaces the entire kept set with one axis-aligned bounding box
)

// ShrinkwrapDefinition is the recipe for simplifying a source assembly into a
// lightweight base body: which parts to remove and how to envelope the rest. The zero
// value (RemoveNone + EnvelopeNone) reduces to a plain include-everything derive.
type ShrinkwrapDefinition struct {
	RemoveStyle   ShrinkwrapRemoveStyle
	MinPartVolume float64 // threshold for RemoveSmallParts (units³)
	EnvelopeStyle ShrinkwrapEnvelopeStyle
	// PatchHoles fills each kept part's internal voids (cavities) so hollow parts
	// become solid, before removal and enveloping.
	PatchHoles bool
	// MaxHoleDiameter, when > 0, caps through-holes / pockets that open to the surface
	// whose opening spans at most this width — closing them flush while keeping the real
	// outer geometry (#721). Larger holes are left intact. Complements PatchHoles, which
	// only fills disconnected internal voids.
	MaxHoleDiameter float64
}

// BuildShrinkwrap flattens source's occurrence tree, drops parts per the removal
// style, replaces the rest with envelopes per the envelope style, and merges the
// result into a single lightweight solid base body (M11-F06). It returns no body when
// nothing survives removal — the model-side producer of a shrinkwrap substitute
// (#348/#355) and LOD representation (#361).
//
// Example: BuildShrinkwrap(asm, ShrinkwrapDefinition{EnvelopeStyle: EnvelopeWhole})
// yields one box enclosing the whole assembly.
func BuildShrinkwrap(source AssemblyBodySource, def ShrinkwrapDefinition) ([]*topo.Body, error) {
	world, err := worldBodies(source.PlacedBodies())
	if err != nil {
		return nil, err
	}
	if def.PatchHoles {
		world = patchHoles(world)
	}
	if def.MaxHoleDiameter > 0 {
		world = capThroughHoles(world, def.MaxHoleDiameter)
	}
	enveloped, err := applyEnvelope(keepAfterRemoval(world, def), def.EnvelopeStyle)
	if err != nil {
		return nil, err
	}
	return mergeIntoBase(enveloped), nil
}

// patchHoles fills each body's internal voids so hollow parts become solid before the
// rest of the simplification runs.
func patchHoles(world []*topo.Body) []*topo.Body {
	q := ops.DefaultQuality()
	out := make([]*topo.Body, 0, len(world))
	for _, b := range world {
		out = append(out, ops.FillInternalVoids(b, q))
	}
	return out
}

// capThroughHoles caps each body's surface-opening holes no wider than maxDiameter, keeping the
// holed body whenever the cap cannot close it (so a tricky part never fails the whole build).
func capThroughHoles(world []*topo.Body, maxDiameter float64) []*topo.Body {
	out := make([]*topo.Body, 0, len(world))
	for _, b := range world {
		if capped, err := ops.CapHolesByDiameter(b, maxDiameter); err == nil {
			out = append(out, capped)
			continue
		}
		out = append(out, b)
	}
	return out
}

// worldBodies transforms each placed source body into assembly space, giving each a
// distinct lineage prefix so repeated placements of one part keep independent keys
// (reusing the derive-family lineage prefixer).
func worldBodies(placed []PlacedBody) ([]*topo.Body, error) {
	out := make([]*topo.Body, 0, len(placed))
	for i, pb := range placed {
		b, err := ops.TransformBody(pb.Body, pb.Transform, deriveLineage(i))
		if err != nil {
			return nil, fmt.Errorf("feature: shrinkwrap transform of body %d: %w", i, err)
		}
		out = append(out, b)
	}
	return out, nil
}

// keepAfterRemoval returns the world bodies that survive the removal style.
func keepAfterRemoval(world []*topo.Body, def ShrinkwrapDefinition) []*topo.Body {
	switch def.RemoveStyle {
	case RemoveSmallParts:
		return keepLargerThan(world, def.MinPartVolume)
	case RemoveInternalParts:
		return keepVisible(world)
	default:
		return world
	}
}

// keepLargerThan keeps bodies whose volume is at least minVolume.
func keepLargerThan(world []*topo.Body, minVolume float64) []*topo.Body {
	q := ops.DefaultQuality()
	kept := make([]*topo.Body, 0, len(world))
	for _, b := range world {
		if ops.BodyGeometryProperties(b, q).Volume >= minVolume {
			kept = append(kept, b)
		}
	}
	return kept
}

// keepVisible drops internal parts: a body whose every vertex lies inside one of the
// other bodies is fully enclosed (never seen from outside) and is removed.
func keepVisible(world []*topo.Body) []*topo.Body {
	kept := make([]*topo.Body, 0, len(world))
	for i, b := range world {
		if !enclosedByOthers(b, world, i) {
			kept = append(kept, b)
		}
	}
	return kept
}

// enclosedByOthers reports whether every vertex of world[i] lies inside some other
// body, i.e. the part is internal to the assembly. A body with no vertices is never
// treated as enclosed.
func enclosedByOthers(b *topo.Body, world []*topo.Body, i int) bool {
	verts := b.Vertices()
	if len(verts) == 0 {
		return false
	}
	for _, v := range verts {
		if !insideAnyExcept(v.Point(), world, i) {
			return false
		}
	}
	return true
}

// insideAnyExcept reports whether p lies inside any body other than world[skip].
func insideAnyExcept(p math.Point3, world []*topo.Body, skip int) bool {
	for j, other := range world {
		if j != skip && ops.PointInsideBody(other, p) {
			return true
		}
	}
	return false
}

// applyEnvelope replaces kept bodies with bounding-box proxies per style.
func applyEnvelope(kept []*topo.Body, style ShrinkwrapEnvelopeStyle) ([]*topo.Body, error) {
	switch style {
	case EnvelopePerPart:
		return perPartEnvelopes(kept)
	case EnvelopeWhole:
		return wholeEnvelope(kept)
	default:
		return kept, nil
	}
}

// perPartEnvelopes replaces each body with its own axis-aligned bounding box solid.
func perPartEnvelopes(kept []*topo.Body) ([]*topo.Body, error) {
	out := make([]*topo.Body, 0, len(kept))
	for i, b := range kept {
		box, err := envelopeSolid(b.RangeBox(), fmt.Sprintf("shrinkwrapEnvelope%d", i))
		if err != nil {
			return nil, err
		}
		if box != nil {
			out = append(out, box)
		}
	}
	return out, nil
}

// wholeEnvelope replaces the whole kept set with one bounding box enclosing them all.
func wholeEnvelope(kept []*topo.Body) ([]*topo.Body, error) {
	if len(kept) == 0 {
		return nil, nil
	}
	box := kept[0].RangeBox()
	for _, b := range kept[1:] {
		box = box.Union(b.RangeBox())
	}
	solid, err := envelopeSolid(box, "shrinkwrapEnvelope")
	if err != nil || solid == nil {
		return nil, err
	}
	return []*topo.Body{solid}, nil
}

// envelopeSolid builds the axis-aligned box solid for box, or no body for an empty box.
func envelopeSolid(box math.Box, feat string) (*topo.Body, error) {
	if box.IsEmpty() {
		return nil, nil
	}
	b, err := brep.SolidBlock(box.Min, box.Max, feat)
	if err != nil {
		return nil, fmt.Errorf("feature: shrinkwrap envelope %q for box %v: %w", feat, box, err)
	}
	return b, nil
}

// mergeIntoBase combines the proxy/real bodies into one multi-lump solid base body
// (matching the derived-assembly merge), or returns nothing when the set is empty.
func mergeIntoBase(bodies []*topo.Body) []*topo.Body {
	solids := make([]*topo.Body, 0, len(bodies))
	for _, b := range bodies {
		if b != nil {
			solids = append(solids, b)
		}
	}
	if len(solids) == 0 {
		return nil
	}
	lineage := topo.NewLineage(topo.Tok("shrinkwrap", "body", 0))
	return []*topo.Body{topo.MergeBodies(lineage, true, solids...)}
}

// ShrinkwrapComponent derives a source assembly into this part as a simplified,
// lightweight base body (M11-F06): each recompute runs [BuildShrinkwrap] over the
// source, so a source edit re-simplifies; BreakLink freezes the current result. It is
// the simplified-derive flavor of [DerivedAssemblyComponent].
type ShrinkwrapComponent struct {
	source    AssemblyBodySource // nil after restore until BindSource rebinds it
	def       ShrinkwrapDefinition
	linked    bool
	frozen    []*topo.Body
	link      DeriveSourceLink
	outOfDate bool
}

// Definition returns the shrinkwrap recipe holder.
func (s *ShrinkwrapComponent) Definition() *ShrinkwrapComponent { return s }

// Kind implements [Feature].
func (s *ShrinkwrapComponent) Kind() string { return "shrinkwrap" }

// SourceVersion returns the source assembly's geometry version, or "" when the source is
// not bound (after restore, before [BindSource]).
func (s *ShrinkwrapComponent) SourceVersion() string {
	if s.source == nil {
		return ""
	}
	return s.source.ModelGeometryVersion()
}

// SourceLink returns the persisted identity of the shrinkwrap's source document (#715).
func (s *ShrinkwrapComponent) SourceLink() DeriveSourceLink { return s.link }

// OutOfDate reports whether the source assembly has been edited since this shrinkwrap was
// saved — the source resolved on reopen carries a different recipe revision than captured.
func (s *ShrinkwrapComponent) OutOfDate() bool { return s.outOfDate }

// AcknowledgeSource re-stamps the link's source revision and clears out-of-date (#751).
func (s *ShrinkwrapComponent) AcknowledgeSource(currentDBRevID string) {
	s.link.DatabaseRevisionID = currentDBRevID
	s.outOfDate = false
}

// RelinkSource updates the link's source document name (#750).
func (s *ShrinkwrapComponent) RelinkSource(document string) { s.link.Document = document }

// BindSource (re)binds the live source assembly after a restore and recomputes staleness:
// out of date when currentDBRevID differs from the revision captured in the link (#715).
func (s *ShrinkwrapComponent) BindSource(source AssemblyBodySource, currentDBRevID string) {
	s.source = source
	s.outOfDate = s.link.DatabaseRevisionID != "" && currentDBRevID != "" && currentDBRevID != s.link.DatabaseRevisionID
}

// Linked reports whether the shrinkwrap still pulls from its source.
func (s *ShrinkwrapComponent) Linked() bool { return s.linked }

// Options returns the shrinkwrap recipe (removal + envelope styles).
func (s *ShrinkwrapComponent) Options() ShrinkwrapDefinition { return s.def }

// SetOptions replaces the shrinkwrap recipe; the next recompute re-simplifies.
func (s *ShrinkwrapComponent) SetOptions(def ShrinkwrapDefinition) { s.def = def }

// BreakLink freezes the current shrinkwrap result and severs the source link, so the
// part keeps the simplified geometry without further updates.
func (s *ShrinkwrapComponent) BreakLink() error {
	bodies, err := s.build()
	if err != nil {
		return err
	}
	s.frozen = bodies
	s.linked = false
	return nil
}

// build simplifies the bound source; an unbound source (a restored shrinkwrap not yet
// resolved, or missing) yields no bodies, so a recompute before/without binding is safe.
func (s *ShrinkwrapComponent) build() ([]*topo.Body, error) {
	if s.source == nil {
		return nil, nil
	}
	return BuildShrinkwrap(s.source, s.def)
}

// Recompute appends the shrinkwrap base body (or, after a broken link, the frozen
// bodies) to the running state.
func (s *ShrinkwrapComponent) Recompute(in Input) (Output, error) {
	return recomputeLinked(in, s.linked, s.frozen, s.build)
}

// ShrinkwrapComponents adds shrinkwrap features into the engine.
type ShrinkwrapComponents struct{ engine *PartFeatures }

// NewShrinkwrapComponents binds the collection to an engine.
func NewShrinkwrapComponents(engine *PartFeatures) *ShrinkwrapComponents {
	return &ShrinkwrapComponents{engine}
}

// AddShrinkwrap adds an associative shrinkwrap component simplifying source per def,
// recording link — the source assembly document's identity — so the shrinkwrap survives a
// save and detects a stale source on reopen (#715).
func (c *ShrinkwrapComponents) AddShrinkwrap(source AssemblyBodySource, def ShrinkwrapDefinition, link DeriveSourceLink) *PartFeature {
	return c.engine.Add(&ShrinkwrapComponent{source: source, def: def, linked: true, link: link})
}

// RestoreShrinkwrap rebuilds a shrinkwrap component from its persisted recipe — the source
// identity link, the simplification options, and the linked flag — all UNBOUND. The live
// source is rebound later by [ShrinkwrapComponent.BindSource] once the part's reference
// graph resolves the source document (#715); until then it contributes no geometry.
func RestoreShrinkwrap(link DeriveSourceLink, def ShrinkwrapDefinition, linked bool) *ShrinkwrapComponent {
	return &ShrinkwrapComponent{def: def, linked: linked, link: link}
}

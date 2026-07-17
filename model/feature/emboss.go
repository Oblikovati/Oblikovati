// SPDX-License-Identifier: GPL-2.0-only

package feature

import (
	"fmt"

	"oblikovati.org/kernel/diag"
	"oblikovati.org/kernel/ops"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
	"oblikovati.org/model/sketch"
	"oblikovati.org/model/text"
)

// EmbossDefinition is the recipe for an emboss: a closed sketch profile raised from (or engraved
// into) the part by a depth along the sketch-plane normal. Engrave cuts; raise joins.
//
// The profile source is EITHER closed sketch geometry (Sketch + ProfileIndices) OR a sketch
// TEXT entity (Text). When Text is set, the emboss reads the text entity's derived glyph
// profiles on every recompute — so it stores a REFERENCE to the text, not baked outline
// geometry, and editing/moving the text re-shapes the emboss (Inventor's text-emboss model).
type EmbossDefinition struct {
	Sketch         *sketch.Sketch
	ProfileIndices []int
	Text           *sketch.TextBox   // when set, the profile source is this text entity
	Fonts          text.FontResolver // font catalogue for Text (defaults to the embedded faces)
	Depth          func() float64
	Engrave        bool    // cut into the face instead of raising from it
	Taper          float64 // draft angle (radians)
}

// EmbossFeature raises or engraves a closed profile on the part (Inventor's Emboss): the profile
// is extruded a shallow depth along the sketch-plane normal and joined (raise) or cut (engrave).
// (Wrapping the emboss onto a curved face is a refinement; the planar emboss works today.)
type EmbossFeature struct {
	def      *EmbossDefinition
	featName string
	tool     *topo.Body // last raised/engraved prism, exposed so a pattern can replicate it
}

// Definition returns the emboss recipe.
func (f *EmbossFeature) Definition() *EmbossDefinition { return f.def }

// Kind implements [Feature].
func (f *EmbossFeature) Kind() string { return "emboss" }

// Operation and ToolBody let a pattern/mirror replicate this feature with the right boolean —
// an engrave cuts, a raise joins (see [ToolFeature]).
func (f *EmbossFeature) Operation() ops.PartFeatureOperation {
	if f.def.Engrave {
		return ops.Cut
	}
	return ops.Join
}
func (f *EmbossFeature) ToolBody() *topo.Body { return f.tool }

// Recompute extrudes the profile(s) by the depth and joins (raise) or cuts (engrave) the part.
func (f *EmbossFeature) Recompute(in Input) (Output, error) {
	profiles, err := f.resolveProfiles()
	if err != nil {
		return Output{}, err
	}
	d := callOrZero(f.def.Depth)
	if d <= 0 {
		return Output{}, fmt.Errorf("emboss: depth %g must be > 0", d)
	}
	sp, op := orderedSpan(0, d), ops.Join
	if f.def.Engrave {
		sp, op = orderedSpan(0, -d), ops.Cut // cut into the part, below the sketch plane
	}
	f.tool = buildProfilePrisms(profiles, f.def.Sketch.Plane(), sp, f.def.Taper, featOr(f.featName, "emboss"), in.Diag)
	bodies, err := combine(in, f.tool, op)
	if err != nil {
		return Output{}, err
	}
	return Output{Bodies: bodies}, nil
}

// resolveProfiles returns the emboss's source profiles: the referenced text entity's
// derived glyph profiles when Text is set, otherwise the selected closed sketch profiles.
// Reading text geometry here (not at Add time) is what makes the emboss reference the text
// rather than bake its outlines.
func (f *EmbossFeature) resolveProfiles() ([]*sketch.Profile, error) {
	if f.def.Text == nil {
		return resolveClosedProfiles(f.def.Sketch, f.def.ProfileIndices, "emboss")
	}
	profs, err := f.def.Text.TextProfiles(f.fonts())
	if err != nil {
		return nil, fmt.Errorf("emboss: derive text geometry: %w", err)
	}
	if len(profs) == 0 {
		return nil, fmt.Errorf("emboss: text %q produced no glyph geometry", f.def.Text.Text)
	}
	return profs, nil
}

// fonts returns the emboss's font resolver, defaulting to the embedded faces.
func (f *EmbossFeature) fonts() text.FontResolver {
	if f.def.Fonts != nil {
		return f.def.Fonts
	}
	return text.DefaultResolver()
}

// EmbossFeatures adds emboss features into the engine.
type EmbossFeatures struct{ engine *PartFeatures }

// NewEmbossFeatures binds the collection to an engine.
func NewEmbossFeatures(engine *PartFeatures) *EmbossFeatures { return &EmbossFeatures{engine} }

// Add adds an emboss of the sketch's closed profile(s); engrave cuts instead of raising.
func (c *EmbossFeatures) Add(skt *sketch.Sketch, profileIndices []int, depth func() float64, engrave bool, taper float64) *PartFeature {
	def := &EmbossDefinition{Sketch: skt, ProfileIndices: profileIndices, Depth: depth, Engrave: engrave, Taper: taper}
	return c.add(def)
}

// AddText adds an emboss of a sketch TEXT entity: the recipe stores a reference to tb (in
// sketch skt) and derives its glyph profiles on each recompute, so no outline geometry is
// baked into the document and editing the text re-shapes the emboss.
func (c *EmbossFeatures) AddText(skt *sketch.Sketch, tb *sketch.TextBox, depth func() float64, engrave bool, taper float64) *PartFeature {
	def := &EmbossDefinition{Sketch: skt, Text: tb, Depth: depth, Engrave: engrave, Taper: taper}
	return c.add(def)
}

// add registers an emboss feature definition and gives it a unique name. It binds the engine's
// document font resolver (resource-aware) unless the caller already chose one, so text emboss
// resolves a font embedded in the document (ADR-0031), not just the bundled faces.
func (c *EmbossFeatures) add(def *EmbossDefinition) *PartFeature {
	if def.Fonts == nil {
		def.Fonts = c.engine.FontResolver()
	}
	ef := &EmbossFeature{def: def}
	pf := c.engine.Add(ef)
	pf.SetName(c.engine.UniqueName("Emboss"))
	ef.featName = pf.name
	return pf
}

// resolveClosedProfiles returns the closed sketch profiles at the given indices — the shared
// profile resolution for solid-from-region features (extrude, emboss). `what` names the caller
// in errors.
func resolveClosedProfiles(sk *sketch.Sketch, indices []int, what string) ([]*sketch.Profile, error) {
	all := sk.Profiles()
	if len(indices) == 0 {
		return nil, fmt.Errorf("%s: no profile selected", what)
	}
	out := make([]*sketch.Profile, 0, len(indices))
	for _, idx := range indices {
		if idx < 0 || idx >= all.Count() {
			return nil, fmt.Errorf("%s: profile %d not found (sketch has %d)", what, idx, all.Count())
		}
		p := all.Item(idx)
		if !p.IsClosed() {
			return nil, fmt.Errorf("%s: profile is open (cannot form a solid)", what)
		}
		out = append(out, p)
	}
	return out, nil
}

// buildProfilePrisms extrudes each closed profile to a prism over span sp (with taper), merging
// several into one tool body — the shared prism builder for extrude/emboss. rec collects the
// hole-drilling booleans' fallback diagnostics (#1601; nil discards).
func buildProfilePrisms(profiles []*sketch.Profile, plane sketch.Plane, sp span, taper float64, feat string, rec *diag.Recorder) *topo.Body {
	return mergePrisms(profilePrisms(profiles, plane, sp, taper, feat, rec), feat)
}

// profilePrisms extrudes each closed profile to its OWN prism — the per-region tools, unmerged.
// A caller that applies them with a boolean should prefer these to the merged body: see
// combinePrisms for why a merged multi-lump tool is worse than the sum of its lumps.
//
// Abutting profiles (touching cells of one over-split region — a slot with its corner-relief discs)
// are first DISSOLVED into their union outline so they extrude as one clean prism, not several that
// leave coincident interior walls when cut one at a time (#38). Disjoint profiles are untouched.
func profilePrisms(profiles []*sketch.Profile, plane sketch.Plane, sp span, taper float64, feat string, rec *diag.Recorder) []*topo.Body {
	prisms, _ := profilePrismsDissolving(profiles, plane, sp, taper, feat, rec, true)
	return prisms
}

// profilePrismsDissolving is [profilePrisms] with an explicit dissolve switch, returning whether it
// actually fused any abutting group. The switch lets the extrude self-validate: it dissolves first,
// and if that turns a closed solid open (a rare downstream boolean fragility on a merged faceted
// prism) it rebuilds with dissolve OFF, so the dissolve never regresses a working solid (#38).
func profilePrismsDissolving(profiles []*sketch.Profile, plane sketch.Plane, sp span, taper float64, feat string, rec *diag.Recorder, allowDissolve bool) ([]*topo.Body, bool) {
	if !allowDissolve {
		return perProfilePrisms(profiles, plane, sp, taper, feat, rec), false
	}
	groups := abuttingProfileGroups(profiles)
	if len(groups) == len(profiles) {
		// All disjoint: every profile is its own prism. This is the common case (#33's per-region
		// selection), and it keeps a lone circular bore on the analytic-cylinder path.
		return perProfilePrisms(profiles, plane, sp, taper, feat, rec), false
	}
	var prisms []*topo.Body
	dissolved := false
	for _, g := range groups {
		if regions, ok := dissolveGroup(profiles, g); ok {
			dissolved = true
			for _, r := range regions {
				prisms = append(prisms, buildMergedRegionPrism(r, plane, sp, taper, indexedPrismName(feat, len(prisms)), rec))
			}
			continue
		}
		for _, pi := range g {
			prisms = append(prisms, buildPrismWithHoles(profiles[pi], plane, sp, taper, indexedPrismName(feat, len(prisms)), rec))
		}
	}
	return prisms, dissolved
}

// buildMergedRegionPrism extrudes a dissolved abutting group's union outline into a solid prism, then
// drills the group's carried inner loops out of it — a faceted prism (the union of ≥2 abutting cells is
// never a bare circle), so no analytic-cylinder special case applies.
func buildMergedRegionPrism(r mergedRegion, plane sketch.Plane, sp span, taper float64, feat string, rec *diag.Recorder) *topo.Body {
	solid := buildPrism(r.outer, plane, sp, taper, feat)
	if len(r.inners) > 0 {
		return drillProfileHoles(solid, r.inners, plane, sp, taper, feat, rec)
	}
	return solid
}

// indexedPrismName always tags a prism with its running index: the group path (a mix of merged and
// per-region prisms) can't use the count-based [prismName], since a dissolved group changes the prism
// count and its members don't line up with the profile list.
func indexedPrismName(feat string, i int) string {
	return fmt.Sprintf("%s/p%d", feat, i)
}

// perProfilePrisms extrudes one prism per profile, honouring inner loops — the all-disjoint path.
func perProfilePrisms(profiles []*sketch.Profile, plane sketch.Plane, sp span, taper float64, feat string, rec *diag.Recorder) []*topo.Body {
	prisms := make([]*topo.Body, len(profiles))
	for i, p := range profiles {
		prisms[i] = buildPrismWithHoles(p, plane, sp, taper, prismName(feat, i, len(profiles)), rec)
	}
	return prisms
}

// prismName is the per-prism lineage name: the bare feature for a lone prism, else feat/pN.
func prismName(feat string, i, n int) string {
	if n <= 1 {
		return feat
	}
	return fmt.Sprintf("%s/p%d", feat, i)
}

// mergePrisms merges the per-region prisms into one tool body. The regions are distinct cells of
// the same sketch, so they never overlap — a shell merge is exactly their union.
func mergePrisms(prisms []*topo.Body, feat string) *topo.Body {
	if len(prisms) == 1 {
		return prisms[0]
	}
	return topo.MergeBodies(topo.NewLineage(topo.Tok(feat, "merged", 0)), true, prisms...)
}

// buildPrismWithHoles extrudes a profile honoring its inner loops: the outer loop
// becomes a solid prism, then each inner loop (a hole) is extruded over a span that
// overshoots both caps and cut away, yielding a hollow prism (a tube for an annular
// profile). Without this an annular profile extruded as a solid disk.
func buildPrismWithHoles(p *sketch.Profile, plane sketch.Plane, sp span, taper float64, feat string, rec *diag.Recorder) *topo.Body {
	// A full-circle profile with no holes and no taper extrudes to a TRUE cylinder (analytic side
	// face), so thread/chamfer/fillet on it work (#129). Booleans re-facet it on demand (combine →
	// planarized). Other shapes fall through to the faceted prism.
	if taper == 0 && len(p.InnerLoops()) == 0 {
		if c := circleLoop(p.OuterLoop()); c != nil {
			if cyl := buildAnalyticCylinder(c, plane, sp, feat); cyl != nil {
				return cyl
			}
		}
	}
	solid := buildPrism(p.OuterLoop().Polygon(), plane, sp, taper, feat)
	if inner := p.InnerLoops(); len(inner) > 0 {
		return drillProfileHoles(solid, inner, plane, sp, taper, feat, rec)
	}
	return solid
}

// drillProfileHoles cuts each inner loop out of the solid prism, yielding a hollow prism. Each hole
// tool overshoots both caps by the full depth so the cut is clean; a clockwise hole loop is
// normalized to CCW so buildPrism makes a valid outward cut tool (not an inverted one).
func drillProfileHoles(solid *topo.Body, inner []sketch.Loop, plane sketch.Plane, sp span, taper float64, feat string, rec *diag.Recorder) *topo.Body {
	lo, hi := sp.near, sp.far
	if lo > hi {
		lo, hi = hi, lo
	}
	margin := hi - lo
	cut := span{near: lo - margin, far: hi + margin}
	for j, loop := range inner {
		poly := loop.Polygon()
		if outwardSign(poly) < 0 {
			poly = reversedPolygon2D(poly)
		}
		hole := buildPrism(poly, plane, cut, taper, fmt.Sprintf("%s/hole%d", feat, j))
		if res, err := ops.BooleanWithDiagnostics(ops.Cut, solid, hole, rec); err == nil && res != nil {
			solid = res
		}
	}
	return solid
}

// reversedPolygon2D returns the polygon with its vertex order reversed (flipping its
// winding).
func reversedPolygon2D(poly []math.Point2) []math.Point2 {
	out := make([]math.Point2, len(poly))
	for i, p := range poly {
		out[len(poly)-1-i] = p
	}
	return out
}

// SPDX-License-Identifier: GPL-2.0-only

package compdef

import (
	"fmt"

	"oblikovati.org/math"
	"oblikovati.org/model/attr"
	"oblikovati.org/model/bom"
	"oblikovati.org/model/doc"
	"oblikovati.org/model/feature"
	"oblikovati.org/model/occurrence"
	"oblikovati.org/model/param"
	"oblikovati.org/model/sketch"
	"oblikovati.org/yamlcodec"
)

// Assembly persistence (#715): the assembly recipe records its display units and its
// occurrence structure — which component document each occurrence instances, where, and
// in what state. Component documents themselves are NOT embedded; they are resolved
// through the reference graph on reopen (ResolveReferences below), so one part edited on
// disk updates every assembly that places it. A part-only assembly graph stays a tree of
// .obk files referencing each other, exactly as the reference API models it.

// assemblyRecipe is the persisted form of an AssemblyComponentDefinition: display units,
// its first-class parameters (assemblies are parameter holders like parts — M39-F01, #1557),
// and the placed occurrences in placement order. The parameter fields share the part recipe's
// DTOs and codec (param_recipe.go); they are restored before the assembly sketches so a
// dimension expression rebinds to its parameter by name.
type assemblyRecipe struct {
	Units             map[string]string              `yaml:"units,omitempty"`
	Parameters        []parameterRecipe              `yaml:"parameters,omitempty"`
	ParameterGroups   []parameterGroupRecipe         `yaml:"parameterGroups,omitempty"`
	ParameterSettings *parameterSettingsRecipe       `yaml:"parameterSettings,omitempty"`
	DerivedTables     []derivedTableRecipe           `yaml:"derivedParameterTables,omitempty"`
	Occurrences       []occurrenceRecipe             `yaml:"occurrences,omitempty"`
	Properties        []propertyRecipe               `yaml:"properties,omitempty"` // document iProperties (#156)
	Sketches          []sketch.SketchData            `yaml:"sketches,omitempty"`   // assembly-space sketches (#785)
	Features          []assemblyFeatureProgramRecipe `yaml:"features,omitempty"`   // the machining program (#785)
	EndOfFeatures     *int                           `yaml:"endOfFeatures,omitempty"`
	Options           *assemblyOptionsRecipe         `yaml:"options,omitempty"` // assembly editing options (#1981)
}

// assemblyOptionsRecipe persists the assembly editing options (#1981); the transient DeferUpdate flag
// is not stored (a reopened assembly is never mid-defer).
type assemblyOptionsRecipe struct {
	PartFeaturesInitiallyAdaptive            bool   `yaml:"partFeaturesInitiallyAdaptive,omitempty"`
	OnlyActiveComponentIsOpaque              bool   `yaml:"onlyActiveComponentIsOpaque,omitempty"`
	PlaceAndGroundFirstComponentAtOrigin     bool   `yaml:"placeAndGroundFirstComponentAtOrigin"`
	EnableConstraintRedundancyAnalysis       bool   `yaml:"enableConstraintRedundancyAnalysis"`
	DeleteComponentPatternSources            bool   `yaml:"deleteComponentPatternSources,omitempty"`
	SectionAllParts                          bool   `yaml:"sectionAllParts,omitempty"`
	UseLastOccurrenceOrientationForPlacement bool   `yaml:"useLastOccurrenceOrientationForPlacement,omitempty"`
	DefaultLevelOfDetail                     string `yaml:"defaultLevelOfDetail,omitempty"`
	DefaultDesignView                        string `yaml:"defaultDesignView,omitempty"`
}

// assemblyFeatureProgramRecipe is the persisted form of one program entry: the feature's inputs
// plus its name and suppression (the AssemblyFeature wrapper state).
type assemblyFeatureProgramRecipe struct {
	Name       string                      `yaml:"name,omitempty"`
	Suppressed bool                        `yaml:"suppressed,omitempty"`
	Feature    feature.AssemblyFeatureData `yaml:"feature"`
}

// occurrenceRecipe is the persisted form of one placement: the component document it
// instances (full document name), its instance name and transform (16 row-major cells,
// the same form the move feature persists), and its per-instance state flags.
type occurrenceRecipe struct {
	Name       string    `yaml:"name"`
	Component  string    `yaml:"component"`
	Transform  []float64 `yaml:"transform,omitempty"`
	Suppressed bool      `yaml:"suppressed,omitempty"`
	Grounded   bool      `yaml:"grounded,omitempty"`
	Adaptive   bool      `yaml:"adaptive,omitempty"`
	Flexible   bool      `yaml:"flexible,omitempty"` // M12-F06
	// ChildTransforms is a flexible occurrence's per-child independent placement (child name →
	// 16-cell row-major transform), persisting the M12-F06 independent solution per placement.
	ChildTransforms map[string][]float64 `yaml:"childTransforms,omitempty"`
	Substitute      bool                 `yaml:"substitute,omitempty"`
	// BOMStructure is a per-occurrence BOM-structure override (wire spelling); "" ⇒ inherit (#1978).
	BOMStructure string `yaml:"bomStructure,omitempty"`
	// Virtual marks a geometry-free, document-free component persisted inline (no Component to
	// resolve); its part number/description/structure restore the VirtualComponentDefinition (#1979).
	Virtual            bool   `yaml:"virtual,omitempty"`
	VirtualPartNumber  string `yaml:"virtualPartNumber,omitempty"`
	VirtualDescription string `yaml:"virtualDescription,omitempty"`
	VirtualStructure   string `yaml:"virtualStructure,omitempty"`
}

// MarshalRecipe renders the assembly's recipe as YAML bytes (doc.RecipeContent).
func (a *AssemblyComponentDefinition) MarshalRecipe() ([]byte, error) {
	r, err := a.buildRecipe()
	if err != nil {
		return nil, err
	}
	return yamlcodec.Marshal(r)
}

// buildRecipe captures the assembly's parametric state as an [assemblyRecipe] value — the shared
// step behind the YAML save ([MarshalRecipe]) and the fast undo snapshot ([MarshalSnapshot]).
func (a *AssemblyComponentDefinition) buildRecipe() (assemblyRecipe, error) {
	sketches, err := a.sketches.MarshalRecipe()
	if err != nil {
		return assemblyRecipe{}, fmt.Errorf("compdef: marshal assembly sketches: %w", err)
	}
	r := assemblyRecipe{
		Units:             unitsRecipeFor(a.units),
		Parameters:        parametersRecipeOf(a.params),
		ParameterGroups:   parameterGroupsRecipeOf(a.params),
		ParameterSettings: parameterSettingsRecipeOf(a.params),
		DerivedTables:     derivedTablesRecipeOf(a.params),
		Occurrences:       a.occurrencesRecipe(),
		Properties:        propertiesRecipeOf(a.props),
		Sketches:          sketches,
		Features:          a.featuresRecipe(),
	}
	if eof := a.features.EndOfFeaturesPosition(); eof != endOfFeaturesAtEnd {
		r.EndOfFeatures = &eof
	}
	r.Options = optionsRecipeOf(a.options)
	return r, nil
}

// optionsRecipeOf snapshots the assembly editing options for persistence (#1981).
func optionsRecipeOf(o AssemblyOptions) *assemblyOptionsRecipe {
	return &assemblyOptionsRecipe{
		PartFeaturesInitiallyAdaptive:            o.PartFeaturesInitiallyAdaptive,
		OnlyActiveComponentIsOpaque:              o.OnlyActiveComponentIsOpaque,
		PlaceAndGroundFirstComponentAtOrigin:     o.PlaceAndGroundFirstComponentAtOrigin,
		EnableConstraintRedundancyAnalysis:       o.EnableConstraintRedundancyAnalysis,
		DeleteComponentPatternSources:            o.DeleteComponentPatternSources,
		SectionAllParts:                          o.SectionAllParts,
		UseLastOccurrenceOrientationForPlacement: o.UseLastOccurrenceOrientationForPlacement,
		DefaultLevelOfDetail:                     o.DefaultLevelOfDetail,
		DefaultDesignView:                        o.DefaultDesignView,
	}
}

// restoreOptions restores the assembly editing options from a recipe, keeping the defaults when the
// recipe carries none (an older file) (#1981).
func (a *AssemblyComponentDefinition) restoreOptions(rec *assemblyOptionsRecipe) {
	if rec == nil {
		return
	}
	a.options = AssemblyOptions{
		PartFeaturesInitiallyAdaptive:            rec.PartFeaturesInitiallyAdaptive,
		OnlyActiveComponentIsOpaque:              rec.OnlyActiveComponentIsOpaque,
		PlaceAndGroundFirstComponentAtOrigin:     rec.PlaceAndGroundFirstComponentAtOrigin,
		EnableConstraintRedundancyAnalysis:       rec.EnableConstraintRedundancyAnalysis,
		DeleteComponentPatternSources:            rec.DeleteComponentPatternSources,
		SectionAllParts:                          rec.SectionAllParts,
		UseLastOccurrenceOrientationForPlacement: rec.UseLastOccurrenceOrientationForPlacement,
		DefaultLevelOfDetail:                     rec.DefaultLevelOfDetail,
		DefaultDesignView:                        rec.DefaultDesignView,
	}
}

// featuresRecipe captures the machining program in order — each feature whose state is
// self-contained (a sketch-index/scalar/suffix feature). A kind not yet serializable (the box-cut's
// transient tool body, the proxy-cut's occurrence reference) is skipped (#785).
func (a *AssemblyComponentDefinition) featuresRecipe() []assemblyFeatureProgramRecipe {
	sketchIndex := a.sketchIndexFunc()
	var out []assemblyFeatureProgramRecipe
	for i := 0; i < a.features.Count(); i++ {
		af := a.features.Item(i)
		m, ok := af.Definition().(feature.AssemblyFeatureMarshaler)
		if !ok {
			continue
		}
		out = append(out, assemblyFeatureProgramRecipe{
			Name: af.Name(), Suppressed: af.Suppressed(), Feature: m.MarshalAssembly(sketchIndex),
		})
	}
	return out
}

// sketchIndexFunc maps a sketch pointer to its index in the assembly's sketch collection, for a
// feature to persist its sketch reference by index.
func (a *AssemblyComponentDefinition) sketchIndexFunc() func(*sketch.Sketch) int {
	idx := map[*sketch.Sketch]int{}
	for i := 0; i < a.sketches.Count(); i++ {
		idx[a.sketches.Item(i)] = i
	}
	return func(sk *sketch.Sketch) int { return idx[sk] }
}

// sketchSlice returns the assembly's sketches in order, for resolving feature sketch references on
// restore.
func (a *AssemblyComponentDefinition) sketchSlice() []*sketch.Sketch {
	out := make([]*sketch.Sketch, a.sketches.Count())
	for i := range out {
		out[i] = a.sketches.Item(i)
	}
	return out
}

// occurrencesRecipe captures every restorable occurrence — those placed from a component
// document (a non-empty ComponentName). An in-memory placement from a bare definition has
// no document to resolve on reopen, so it is intentionally omitted rather than persisted
// as an unrestorable record.
func (a *AssemblyComponentDefinition) occurrencesRecipe() []occurrenceRecipe {
	var out []occurrenceRecipe
	for _, o := range a.occurrences.All() {
		if v, ok := o.Definition().(*VirtualComponentDefinition); ok {
			out = append(out, virtualOccurrenceRecipe(o, v))
			continue
		}
		if o.ComponentName() == "" {
			continue
		}
		cells := o.Transform().Cells()
		out = append(out, occurrenceRecipe{
			Name:            o.Name(),
			Component:       o.ComponentName(),
			Transform:       cells[:],
			Suppressed:      o.Suppressed(),
			Grounded:        o.Grounded(),
			Adaptive:        o.Adaptive(),
			Flexible:        o.Flexible(),
			ChildTransforms: marshalChildOverrides(o.ChildOverrides()),
			Substitute:      o.IsSubstitute(),
			BOMStructure:    o.BOMStructureOverride(),
		})
	}
	return out
}

// virtualOccurrenceRecipe persists a virtual occurrence inline — its transform/state plus the virtual
// definition's part number/description/structure — so it restores without a backing document (#1979).
func virtualOccurrenceRecipe(o *occurrence.Occurrence, v *VirtualComponentDefinition) occurrenceRecipe {
	cells := o.Transform().Cells()
	return occurrenceRecipe{
		Name:               o.Name(),
		Transform:          cells[:],
		Suppressed:         o.Suppressed(),
		Grounded:           o.Grounded(),
		BOMStructure:       o.BOMStructureOverride(),
		Virtual:            true,
		VirtualPartNumber:  v.PartNumber(),
		VirtualDescription: v.Description(),
		VirtualStructure:   v.BOMStructure().String(),
	}
}

// ApplyRecipe restores the assembly's units and stashes its occurrence records as pending
// (doc.RecipeContent). It does NOT bind occurrences to component definitions: ApplyRecipe
// runs during the store load, before the document joins a workspace, so there is no way to
// open the component documents yet. Binding happens in [ResolveReferences] once the
// workspace registers the assembly (#715).
func (a *AssemblyComponentDefinition) ApplyRecipe(model []byte) error {
	var r assemblyRecipe
	if err := yamlcodec.Unmarshal(model, &r); err != nil {
		return fmt.Errorf("compdef: parse assembly recipe: %w", err)
	}
	return a.applyRecipeStruct(r)
}

// applyRecipeStruct restores the assembly from an already-decoded [assemblyRecipe] — the shared
// tail of [ApplyRecipe] (YAML) and [RestoreSnapshot] (the fast undo codec). It stashes
// occurrences as pending; binding happens in [ResolveReferences] (see ApplyRecipe).
func (a *AssemblyComponentDefinition) applyRecipeStruct(r assemblyRecipe) error {
	if err := applyUnitsTo(&a.units, r.Units); err != nil {
		return err
	}
	applyPropertiesRecipe(a.props, r.Properties)
	// Parameters restore before sketches so a sketch dimension's expression rebinds to its
	// assembly parameter by name (M39-F01, #1557).
	if err := applyParametersTo(a.params, r.ParameterGroups, r.Parameters); err != nil {
		return err
	}
	if err := applyParameterSettingsTo(a.params, r.ParameterSettings); err != nil {
		return err
	}
	if err := applyDerivedTablesTo(a.params, r.DerivedTables); err != nil {
		return err
	}
	// The sketch collection already shares the parameter DAG (ShareParameters, wired in the
	// constructor and resetOccurrences), so each restored sketch binds its dimension
	// expressions against the populated table as it is rebuilt (#1557).
	if err := a.sketches.ApplyRecipe(r.Sketches); err != nil {
		return fmt.Errorf("compdef: restore assembly sketches: %w", err)
	}
	if r.EndOfFeatures != nil {
		a.features.SetEndOfFeatures(*r.EndOfFeatures)
	}
	a.restoreOptions(r.Options)
	a.pending = r.Occurrences
	a.pendingFeatures = r.Features // features bind after occurrences resolve (they snapshot participation)
	return nil
}

// resetOccurrences clears the occurrence structure in place, re-wiring the fresh collection to
// the assembly's event source (the listener NewAssemblyComponentDefinition installs) so
// placements after a restore keep raising occurrence-lifecycle events. The definition pointer
// is preserved so external references to the assembly stay valid.
func (a *AssemblyComponentDefinition) resetOccurrences() {
	a.occurrences = occurrence.NewOccurrences()
	a.occurrences.SetListener(a.events)
	a.props = attr.NewPropertySets() // a restore re-applies the snapshot's properties onto a clean set (#156)
	a.params = param.NewParameters() // a snapshot restore is a full replace; re-add params onto a clean table (#1557)
	a.sketches = sketch.NewSketches()
	a.sketches.ShareParameters(a.params) // re-wire so restored sketches resolve dimension expressions (#1557)
	a.features = NewAssemblyFeatures()   // the program rebuilds from the snapshot (#785)
	a.features.SetBus(a.events.Bus())    // re-wire the recompute event bus the fresh program needs
	a.pendingFeatures = nil
	a.pending = nil
}

// ResolveReferences binds each pending occurrence to its component document, opening the
// component through owner's reference graph (doc.ReferenceResolver). A component that
// cannot be opened becomes a placeholder occurrence whose reference is flagged broken —
// reopening an assembly with a missing component is never fatal (the user repairs the
// reference). owner is the assembly's own document.
func (a *AssemblyComponentDefinition) ResolveReferences(owner *doc.Document) error {
	pending := a.pending
	a.pending = nil
	for _, rec := range pending {
		transform, err := matrixFromRecipe(rec)
		if err != nil {
			return err
		}
		if rec.Virtual {
			a.restoreVirtualOccurrence(rec, transform)
			continue
		}
		def, component := a.resolveComponent(owner, rec.Component)
		occ := a.occurrences.AddByComponentName(rec.Name, def, component, transform)
		occ.SetSuppressed(rec.Suppressed)
		occ.SetGrounded(rec.Grounded)
		occ.SetAdaptive(rec.Adaptive)
		occ.SetFlexible(rec.Flexible)
		occ.SetChildOverrides(parseChildOverrides(rec.ChildTransforms))
		occ.SetSubstitute(rec.Substitute)
		occ.SetBOMStructureOverride(rec.BOMStructure)
	}
	return a.restoreFeatures()
}

// restoreVirtualOccurrence rebuilds a geometry-free virtual occurrence from its inline recipe — no
// component document to resolve (#1979).
func (a *AssemblyComponentDefinition) restoreVirtualOccurrence(rec occurrenceRecipe, transform math.Matrix4) {
	structure, _ := bom.ParseStructure(rec.VirtualStructure)
	def := NewVirtualComponent(rec.Name, rec.VirtualPartNumber, structure)
	def.description = rec.VirtualDescription
	occ := a.occurrences.AddByComponentDefinition(rec.Name, def, transform)
	occ.SetSuppressed(rec.Suppressed)
	occ.SetGrounded(rec.Grounded)
	occ.SetBOMStructureOverride(rec.BOMStructure)
}

// restoreFeatures reconstructs the machining program now that the occurrences are bound (each
// feature snapshots the present occurrences as its participants via AddFeature) and recomputes.
func (a *AssemblyComponentDefinition) restoreFeatures() error {
	pending := a.pendingFeatures
	a.pendingFeatures = nil
	sketches := a.sketchSlice()
	for _, fr := range pending {
		f, err := feature.RestoreAssemblyFeature(fr.Feature, sketches, a.occurrenceByName)
		if err != nil {
			return fmt.Errorf("compdef: restore assembly feature: %w", err)
		}
		// AddFeature owns the attach policy (unique naming, proxy-cut source
		// exclusion — #1612); the persisted name then overrides the default.
		af := a.AddFeature(f)
		af.SetName(fr.Name)
		af.SetSuppressed(fr.Suppressed)
	}
	a.RecomputeFeatures()
	return nil
}

// occurrenceByName resolves a leaf occurrence by its instance name (the proxy-cut source rebind).
func (a *AssemblyComponentDefinition) occurrenceByName(name string) (*occurrence.Occurrence, bool) {
	for _, o := range a.occurrences.All() {
		if o.Name() == name {
			return o, true
		}
	}
	return nil, false
}

// resolveComponent opens the component document named component as a reference of owner,
// returning its placeable definition and the name it actually resolved to — the current
// location, which differs from component when the project tree moved (#750), so the
// occurrence re-binds to the live name and a re-save records it. A target that cannot be
// opened yields a [missingDefinition] placeholder under the original name so the occurrence
// still restores; the broken reference is surfaced through the document's reference status.
func (a *AssemblyComponentDefinition) resolveComponent(owner *doc.Document, component string) (occurrence.Definition, string) {
	child, ok := owner.OpenReference(component)
	if !ok {
		return missingDefinition{}, component
	}
	def, ok := child.Content().(occurrence.Definition)
	if !ok {
		return missingDefinition{}, component
	}
	return def, child.FullDocumentName()
}

// matrixFromRecipe rebuilds an occurrence transform from its persisted cells, erroring on
// a matrix that is not 16 cells (no silent loss of a placement). A record with no cells
// restores at the origin.
func matrixFromRecipe(rec occurrenceRecipe) (math.Matrix4, error) {
	if len(rec.Transform) == 0 {
		return math.Identity4(), nil
	}
	if len(rec.Transform) != 16 {
		return math.Matrix4{}, fmt.Errorf("compdef: occurrence %q transform has %d cells, want 16", rec.Name, len(rec.Transform))
	}
	var cells [16]float64
	copy(cells[:], rec.Transform)
	return math.Matrix4FromCells(cells), nil
}

// missingDefinition is the placeholder a restored occurrence instances when its component
// document cannot be opened — an unresolved reference. It contributes no geometry (an
// empty range box), so the assembly opens and recomputes without the missing part rather
// than failing the whole load.
type missingDefinition struct{}

// RangeBox returns the empty box, satisfying occurrence.Definition.
func (missingDefinition) RangeBox() math.Box { return math.EmptyBox() }

// marshalChildOverrides flattens a flexible occurrence's per-child transforms (M12-F06) to the
// recipe form (child instance name → 16-cell row-major transform).
func marshalChildOverrides(overrides map[string]math.Matrix4) map[string][]float64 {
	if len(overrides) == 0 {
		return nil
	}
	out := make(map[string][]float64, len(overrides))
	for name, m := range overrides {
		cells := m.Cells()
		out[name] = cells[:]
	}
	return out
}

// parseChildOverrides reverses marshalChildOverrides on reopen, skipping malformed entries.
func parseChildOverrides(recorded map[string][]float64) map[string]math.Matrix4 {
	if len(recorded) == 0 {
		return nil
	}
	out := make(map[string]math.Matrix4, len(recorded))
	for name, c := range recorded {
		if len(c) != 16 {
			continue
		}
		var cells [16]float64
		copy(cells[:], c)
		out[name] = math.Matrix4FromCells(cells)
	}
	return out
}

// SPDX-License-Identifier: GPL-2.0-only

package compdef

import (
	"fmt"

	"oblikovati.org/math"
	"oblikovati.org/model/doc"
	"oblikovati.org/model/occurrence"
	"oblikovati.org/persistence/yamlcodec"
)

// Assembly persistence (#715): the assembly recipe records its display units and its
// occurrence structure — which component document each occurrence instances, where, and
// in what state. Component documents themselves are NOT embedded; they are resolved
// through the reference graph on reopen (ResolveReferences below), so one part edited on
// disk updates every assembly that places it. A part-only assembly graph stays a tree of
// .obk files referencing each other, exactly as the reference API models it.

// assemblyRecipe is the persisted form of an AssemblyComponentDefinition: display units
// plus the placed occurrences, in placement order.
type assemblyRecipe struct {
	Units       map[string]string  `yaml:"units,omitempty"`
	Occurrences []occurrenceRecipe `yaml:"occurrences,omitempty"`
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
	Substitute bool      `yaml:"substitute,omitempty"`
}

// MarshalRecipe renders the assembly's recipe as YAML bytes (doc.RecipeContent).
func (a *AssemblyComponentDefinition) MarshalRecipe() ([]byte, error) {
	return yamlcodec.Marshal(assemblyRecipe{
		Units:       unitsRecipeFor(a.units),
		Occurrences: a.occurrencesRecipe(),
	})
}

// occurrencesRecipe captures every restorable occurrence — those placed from a component
// document (a non-empty ComponentName). An in-memory placement from a bare definition has
// no document to resolve on reopen, so it is intentionally omitted rather than persisted
// as an unrestorable record.
func (a *AssemblyComponentDefinition) occurrencesRecipe() []occurrenceRecipe {
	var out []occurrenceRecipe
	for _, o := range a.occurrences.All() {
		if o.ComponentName() == "" {
			continue
		}
		cells := o.Transform().Cells()
		out = append(out, occurrenceRecipe{
			Name:       o.Name(),
			Component:  o.ComponentName(),
			Transform:  cells[:],
			Suppressed: o.Suppressed(),
			Grounded:   o.Grounded(),
			Adaptive:   o.Adaptive(),
			Substitute: o.IsSubstitute(),
		})
	}
	return out
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
	if err := applyUnitsTo(a.units, r.Units); err != nil {
		return err
	}
	a.pending = r.Occurrences
	return nil
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
		def := a.resolveComponent(owner, rec.Component)
		occ := a.occurrences.AddByComponentName(rec.Name, def, rec.Component, transform)
		occ.SetSuppressed(rec.Suppressed)
		occ.SetGrounded(rec.Grounded)
		occ.SetAdaptive(rec.Adaptive)
		occ.SetSubstitute(rec.Substitute)
	}
	return nil
}

// resolveComponent opens the component document named component as a reference of owner,
// returning its placeable definition; a target that cannot be opened (missing/replaced
// file) yields a [missingDefinition] placeholder so the occurrence still restores. The
// owner.OpenReference call records and resolves the edge, so the broken reference is
// surfaced through the document's reference status, not lost.
func (a *AssemblyComponentDefinition) resolveComponent(owner *doc.Document, component string) occurrence.Definition {
	child, ok := owner.OpenReference(component)
	if !ok {
		return missingDefinition{}
	}
	def, ok := child.Content().(occurrence.Definition)
	if !ok {
		return missingDefinition{}
	}
	return def
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

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

// RestoreRecipe replaces the assembly's entire occurrence structure with the snapshot — the
// undo/redo restore path (command.RecipeStore, #763). Unlike a part's RestoreRecipe it cannot
// finish in place: each occurrence must be re-bound to its component document, which needs the
// workspace to open the component (the same reason ApplyRecipe defers binding). So it resets
// the occurrences and re-stashes the snapshot as pending; the owner-aware caller pairs every
// restore with [ResolveReferences] to re-bind (app/undo.go's rebindReferences does this for
// every restore). Resetting in place preserves the definition pointer, so the document Content
// and any held reference to the assembly stay valid, and applying a snapshot to an
// already-populated assembly yields exactly that snapshot rather than a union.
//
// Example:
//
//	before, _ := asm.MarshalRecipe()
//	asm.PlaceComponentFromFile(owner, widget, "widget:1", math.Identity4())
//	asm.RestoreRecipe(before)            // occurrences back to pending
//	asm.ResolveReferences(owner)         // re-bound: the placement is gone
func (a *AssemblyComponentDefinition) RestoreRecipe(model []byte) error {
	a.resetOccurrences()
	return a.ApplyRecipe(model)
}

// resetOccurrences clears the occurrence structure in place, re-wiring the fresh collection to
// the assembly's event source (the listener NewAssemblyComponentDefinition installs) so
// placements after a restore keep raising occurrence-lifecycle events. The definition pointer
// is preserved so external references to the assembly stay valid.
func (a *AssemblyComponentDefinition) resetOccurrences() {
	a.occurrences = occurrence.NewOccurrences()
	a.occurrences.SetListener(a.events)
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
		def, component := a.resolveComponent(owner, rec.Component)
		occ := a.occurrences.AddByComponentName(rec.Name, def, component, transform)
		occ.SetSuppressed(rec.Suppressed)
		occ.SetGrounded(rec.Grounded)
		occ.SetAdaptive(rec.Adaptive)
		occ.SetSubstitute(rec.Substitute)
	}
	return nil
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

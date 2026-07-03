// SPDX-License-Identifier: GPL-2.0-only

package router

import (
	"fmt"

	"oblikovati.org/api/types"
	"oblikovati.org/api/wire"
	"oblikovati.org/app"
	"oblikovati.org/math"
	"oblikovati.org/model/compdef"
	"oblikovati.org/model/occurrence"
)

// The assembly occurrence surface (M11-F01/F02, #728): read the active assembly's
// occurrence tree and place/transform/ground/suppress/replace/remove components.
// Occurrences are addressed by session id (the ids the occurrence push events carry);
// each single-occurrence mutator replies with the affected occurrence's refreshed info,
// and remove replies with the refreshed tree.

// registerAssemblyOccurrenceHandlers wires the assembly.* occurrence methods.
func (r *Router) registerAssemblyOccurrenceHandlers() {
	r.readOnly(wire.MethodAssemblyOccurrences, assemblyQuery(assemblyOccurrences))
	r.mutating(wire.MethodAssemblyPlace, "Place Component", typedAssembly(assemblyPlace))
	r.mutating(wire.MethodAssemblyPlaceByDefinition, "Place Component", typedAssembly(assemblyPlaceByDefinition))
	r.mutating(wire.MethodAssemblyPlaceByDefinitionBatch, "Place Components", typedAssembly(assemblyPlaceByDefinitionBatch))
	r.mutating(wire.MethodAssemblyTransform, "Move Component", typedAssembly(assemblyTransform))
	r.mutating(wire.MethodAssemblyGround, "Ground Component", typedAssembly(assemblyGround))
	r.mutating(wire.MethodAssemblySetFlexible, "Set Flexible", typedAssembly(assemblySetFlexible))
	r.mutating(wire.MethodAssemblySetFlexibleChild, "Set Flexible", typedAssembly(assemblySetFlexibleChild))
	r.mutating(wire.MethodAssemblySuppress, "Suppress Component", typedAssembly(assemblySuppress))
	r.mutating(wire.MethodAssemblyReplace, "Replace Component", typedAssembly(assemblyReplace))
	r.mutating(wire.MethodAssemblyRemove, "Delete Component", typedAssembly(assemblyRemove))
}

// assemblyOccurrences returns the active assembly's occurrence tree.
func assemblyOccurrences(_ *app.Session, asm *compdef.AssemblyComponentDefinition) (wire.OccurrencesResult, error) {
	return occurrenceTreeResult(asm), nil
}

// assemblyPlace places the component held by an open document into the active assembly.
func assemblyPlace(s *app.Session, asm *compdef.AssemblyComponentDefinition, in wire.PlaceOccurrenceArgs) (wire.OccurrenceResult, error) {
	d, err := documentByID(s, in.Document)
	if err != nil {
		return wire.OccurrenceResult{}, fmt.Errorf("%s: %w", wire.MethodAssemblyPlace, err)
	}
	// Place persistently: record the assembly→component reference and the occurrence's
	// document name, so the placement survives a save/reopen (#715) and the occurrence is
	// file-backed (e.g. for mirror-into-part, #717).
	o, err := asm.PlaceComponentFromFile(s.ActiveDocument(), d, in.Name, matrixFromWire(in.Transform))
	if err != nil {
		return wire.OccurrenceResult{}, fmt.Errorf("%s: %w", wire.MethodAssemblyPlace, err)
	}
	return occurrenceReply(o), nil
}

// assemblyPlaceByDefinition places another instance of an existing occurrence's component.
func assemblyPlaceByDefinition(_ *app.Session, asm *compdef.AssemblyComponentDefinition, in wire.PlaceByDefinitionArgs) (wire.OccurrenceResult, error) {
	src, err := occurrenceByID(asm, in.Source, wire.MethodAssemblyPlaceByDefinition)
	if err != nil {
		return wire.OccurrenceResult{}, err
	}
	o := asm.Place(in.Name, src.Definition(), matrixFromWire(in.Transform))
	return occurrenceReply(o), nil
}

// assemblyPlaceByDefinitionBatch places many instances of an existing occurrence's component in one
// call. Each Place is a cheap occurrence append; doing them in a single handler means the assembly's
// geometry version bumps once for the whole batch, so the live host recomputes/re-tessellates once
// instead of per placement — the difference between minutes and seconds for a large assembly.
func assemblyPlaceByDefinitionBatch(_ *app.Session, asm *compdef.AssemblyComponentDefinition, in wire.PlaceByDefinitionBatchArgs) (wire.PlaceByDefinitionBatchResult, error) {
	src, err := occurrenceByID(asm, in.Source, wire.MethodAssemblyPlaceByDefinitionBatch)
	if err != nil {
		return wire.PlaceByDefinitionBatchResult{}, err
	}
	def := src.Definition()
	out := make([]wire.OccurrenceInfo, 0, len(in.Placements))
	for _, p := range in.Placements {
		out = append(out, occurrenceInfo(asm.Place(p.Name, def, matrixFromWire(p.Transform))))
	}
	return wire.PlaceByDefinitionBatchResult{Occurrences: out}, nil
}

// assemblyTransform repositions an occurrence. When the contact solver is on, a move that would
// drive a contact-set member into one of its partners is rejected — the part stops at contact
// (M12-F05); the occurrence keeps its current placement.
func assemblyTransform(_ *app.Session, asm *compdef.AssemblyComponentDefinition, in wire.TransformOccurrenceArgs) (wire.OccurrenceResult, error) {
	o, err := occurrenceByID(asm, in.ID, wire.MethodAssemblyTransform)
	if err != nil {
		return wire.OccurrenceResult{}, err
	}
	target := matrixFromWire(in.Transform)
	if asm.WouldContactBlock(o.ID(), target) {
		return occurrenceReply(o), nil
	}
	o.SetTransform(target)
	return occurrenceReply(o), nil
}

// assemblyGround fixes or releases an occurrence in space.
func assemblyGround(_ *app.Session, asm *compdef.AssemblyComponentDefinition, in wire.GroundOccurrenceArgs) (wire.OccurrenceResult, error) {
	o, err := occurrenceByID(asm, in.ID, wire.MethodAssemblyGround)
	if err != nil {
		return wire.OccurrenceResult{}, err
	}
	o.SetGrounded(in.Grounded)
	return occurrenceReply(o), nil
}

// assemblySetFlexible marks a sub-assembly occurrence flexible (independent per-placement solve)
// or rigid — mutually exclusive with adaptive (M12-F06).
func assemblySetFlexible(_ *app.Session, asm *compdef.AssemblyComponentDefinition, in wire.SetFlexibleOccurrenceArgs) (wire.OccurrenceResult, error) {
	o, err := occurrenceByID(asm, in.ID, wire.MethodAssemblySetFlexible)
	if err != nil {
		return wire.OccurrenceResult{}, err
	}
	o.SetFlexible(in.Flexible)
	return occurrenceReply(o), nil
}

// assemblySetFlexibleChild positions a child component within a flexible sub-assembly
// occurrence independently of the sub-assembly's other placements (M12-F06 independent solve).
func assemblySetFlexibleChild(_ *app.Session, asm *compdef.AssemblyComponentDefinition, in wire.SetFlexibleChildArgs) (wire.OccurrenceResult, error) {
	o, err := occurrenceByID(asm, in.Occurrence, wire.MethodAssemblySetFlexibleChild)
	if err != nil {
		return wire.OccurrenceResult{}, err
	}
	if !o.Flexible() {
		return wire.OccurrenceResult{}, fmt.Errorf("%s: occurrence %d is not flexible", wire.MethodAssemblySetFlexibleChild, in.Occurrence)
	}
	if !subAssemblyHasChild(o, in.Child) {
		return wire.OccurrenceResult{}, fmt.Errorf("%s: flexible occurrence %d has no child named %q", wire.MethodAssemblySetFlexibleChild, in.Occurrence, in.Child)
	}
	o.SetChildTransform(in.Child, matrixFromWire(in.Transform))
	return occurrenceReply(o), nil
}

// subAssemblyHasChild reports whether o's sub-assembly definition has a child instance named
// childName.
func subAssemblyHasChild(o *occurrence.Occurrence, childName string) bool {
	sub, ok := o.Definition().(occurrence.Composite)
	if !ok {
		return false
	}
	for _, c := range sub.Occurrences().All() {
		if c.Name() == childName {
			return true
		}
	}
	return false
}

// assemblySuppress excludes or restores an occurrence from the model (vetoable).
func assemblySuppress(_ *app.Session, asm *compdef.AssemblyComponentDefinition, in wire.SuppressOccurrenceArgs) (wire.OccurrenceResult, error) {
	o, err := occurrenceByID(asm, in.ID, wire.MethodAssemblySuppress)
	if err != nil {
		return wire.OccurrenceResult{}, err
	}
	if err := asm.SetOccurrenceSuppressed(o, in.Suppressed); err != nil {
		return wire.OccurrenceResult{}, fmt.Errorf("%s: %w", wire.MethodAssemblySuppress, err)
	}
	return occurrenceReply(o), nil
}

// assemblyReplace swaps an occurrence's component for another open document's, keeping the
// occurrence's id, name, transform, and state.
func assemblyReplace(s *app.Session, asm *compdef.AssemblyComponentDefinition, in wire.ReplaceOccurrenceArgs) (wire.OccurrenceResult, error) {
	o, err := occurrenceByID(asm, in.ID, wire.MethodAssemblyReplace)
	if err != nil {
		return wire.OccurrenceResult{}, err
	}
	def, err := placeableDefinition(s, in.Document, wire.MethodAssemblyReplace)
	if err != nil {
		return wire.OccurrenceResult{}, err
	}
	asm.Occurrences().Replace(o, def)
	return occurrenceReply(o), nil
}

// assemblyRemove deletes an occurrence and returns the refreshed tree (vetoable).
func assemblyRemove(_ *app.Session, asm *compdef.AssemblyComponentDefinition, in wire.RemoveOccurrenceArgs) (wire.OccurrencesResult, error) {
	o, err := occurrenceByID(asm, in.ID, wire.MethodAssemblyRemove)
	if err != nil {
		return wire.OccurrencesResult{}, err
	}
	if err := asm.DeleteOccurrence(o); err != nil {
		return wire.OccurrencesResult{}, fmt.Errorf("%s: %w", wire.MethodAssemblyRemove, err)
	}
	return occurrenceTreeResult(asm), nil
}

// occurrenceByID resolves an occurrence session id against the assembly, rejecting unknown.
func occurrenceByID(asm *compdef.AssemblyComponentDefinition, id uint64, method string) (*occurrence.Occurrence, error) {
	o, ok := asm.Occurrences().ByID(id)
	if !ok {
		return nil, fmt.Errorf("%s: no occurrence with id %d in the assembly (ids come from assembly.occurrences)", method, id)
	}
	return o, nil
}

// placeableDefinition resolves an open document id to its placeable component definition.
func placeableDefinition(s *app.Session, id uint64, method string) (occurrence.Definition, error) {
	d, err := documentByID(s, id)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", method, err)
	}
	def, ok := d.Content().(occurrence.Definition)
	if !ok {
		return nil, fmt.Errorf("%s: document %d (%s) is not a placeable component", method, id, d.DisplayName())
	}
	return def, nil
}

// occurrenceReply builds the single-occurrence result DTO for o.
func occurrenceReply(o *occurrence.Occurrence) wire.OccurrenceResult {
	return wire.OccurrenceResult{Occurrence: occurrenceInfo(o)}
}

// occurrenceTreeResult renders the assembly's top-level occurrences as the tree result.
func occurrenceTreeResult(asm *compdef.AssemblyComponentDefinition) wire.OccurrencesResult {
	return wire.OccurrencesResult{Occurrences: occurrenceNodes(asm.Occurrences())}
}

// occurrenceNodes renders a collection's occurrences in placement order, recursing into
// each composite's sub-occurrences.
func occurrenceNodes(occs *occurrence.Occurrences) []wire.OccurrenceInfo {
	all := occs.All()
	out := make([]wire.OccurrenceInfo, len(all))
	for i, o := range all {
		out[i] = occurrenceInfo(o)
	}
	return out
}

// occurrenceInfo renders one occurrence (and its nested children) as its wire DTO.
func occurrenceInfo(o *occurrence.Occurrence) wire.OccurrenceInfo {
	info := wire.OccurrenceInfo{
		ID:         o.ID(),
		Name:       o.Name(),
		Transform:  types.Matrix{Cells: o.Transform().Cells()},
		Suppressed: o.Suppressed(),
		Grounded:   o.Grounded(),
		Adaptive:   o.Adaptive(),
		Flexible:   o.Flexible(),
		Substitute: o.IsSubstitute(),
	}
	if subs := o.SubOccurrences(); subs != nil {
		info.Children = occurrenceNodes(subs)
	}
	return info
}

// matrixFromWire converts the contract matrix value to a model transform.
func matrixFromWire(m types.Matrix) math.Matrix4 { return math.Matrix4FromCells(m.Cells) }

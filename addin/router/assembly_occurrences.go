// SPDX-License-Identifier: GPL-2.0-only

package router

import (
	"encoding/json"
	"fmt"

	"oblikovati.org/addin/modelaccess"
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
	r.handlers[wire.MethodAssemblyOccurrences] = assemblyOccurrences
	r.handlers[wire.MethodAssemblyPlace] = assemblyPlace
	r.handlers[wire.MethodAssemblyPlaceByDefinition] = assemblyPlaceByDefinition
	r.handlers[wire.MethodAssemblyTransform] = assemblyTransform
	r.handlers[wire.MethodAssemblyGround] = assemblyGround
	r.handlers[wire.MethodAssemblySetFlexible] = assemblySetFlexible
	r.handlers[wire.MethodAssemblySuppress] = assemblySuppress
	r.handlers[wire.MethodAssemblyReplace] = assemblyReplace
	r.handlers[wire.MethodAssemblyRemove] = assemblyRemove
}

// assemblyOccurrences returns the active assembly's occurrence tree.
func assemblyOccurrences(s *app.Session, _ json.RawMessage) (json.RawMessage, error) {
	asm, err := modelaccess.ActiveAssembly(s)
	if err != nil {
		return nil, err
	}
	return json.Marshal(occurrenceTreeResult(asm))
}

// assemblyPlace places the component held by an open document into the active assembly.
func assemblyPlace(s *app.Session, raw json.RawMessage) (json.RawMessage, error) {
	asm, err := modelaccess.ActiveAssembly(s)
	if err != nil {
		return nil, err
	}
	var in wire.PlaceOccurrenceArgs
	if err := decode(raw, &in); err != nil {
		return nil, err
	}
	d, err := documentByID(s, in.Document)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", wire.MethodAssemblyPlace, err)
	}
	// Place persistently: record the assembly→component reference and the occurrence's
	// document name, so the placement survives a save/reopen (#715) and the occurrence is
	// file-backed (e.g. for mirror-into-part, #717).
	o, err := asm.PlaceComponentFromFile(s.ActiveDocument(), d, in.Name, matrixFromWire(in.Transform))
	if err != nil {
		return nil, fmt.Errorf("%s: %w", wire.MethodAssemblyPlace, err)
	}
	return occurrenceReply(o)
}

// assemblyPlaceByDefinition places another instance of an existing occurrence's component.
func assemblyPlaceByDefinition(s *app.Session, raw json.RawMessage) (json.RawMessage, error) {
	asm, err := modelaccess.ActiveAssembly(s)
	if err != nil {
		return nil, err
	}
	var in wire.PlaceByDefinitionArgs
	if err := decode(raw, &in); err != nil {
		return nil, err
	}
	src, err := occurrenceByID(asm, in.Source, wire.MethodAssemblyPlaceByDefinition)
	if err != nil {
		return nil, err
	}
	o := asm.Place(in.Name, src.Definition(), matrixFromWire(in.Transform))
	return occurrenceReply(o)
}

// assemblyTransform repositions an occurrence.
func assemblyTransform(s *app.Session, raw json.RawMessage) (json.RawMessage, error) {
	var in wire.TransformOccurrenceArgs
	o, err := occurrenceFromArgs(s, raw, &in, &in.ID, wire.MethodAssemblyTransform)
	if err != nil {
		return nil, err
	}
	o.SetTransform(matrixFromWire(in.Transform))
	return occurrenceReply(o)
}

// assemblyGround fixes or releases an occurrence in space.
func assemblyGround(s *app.Session, raw json.RawMessage) (json.RawMessage, error) {
	var in wire.GroundOccurrenceArgs
	o, err := occurrenceFromArgs(s, raw, &in, &in.ID, wire.MethodAssemblyGround)
	if err != nil {
		return nil, err
	}
	o.SetGrounded(in.Grounded)
	return occurrenceReply(o)
}

// assemblySetFlexible marks a sub-assembly occurrence flexible (independent per-placement solve)
// or rigid — mutually exclusive with adaptive (M12-F06).
func assemblySetFlexible(s *app.Session, raw json.RawMessage) (json.RawMessage, error) {
	var in wire.SetFlexibleOccurrenceArgs
	o, err := occurrenceFromArgs(s, raw, &in, &in.ID, wire.MethodAssemblySetFlexible)
	if err != nil {
		return nil, err
	}
	o.SetFlexible(in.Flexible)
	return occurrenceReply(o)
}

// assemblySuppress excludes or restores an occurrence from the model (vetoable).
func assemblySuppress(s *app.Session, raw json.RawMessage) (json.RawMessage, error) {
	asm, err := modelaccess.ActiveAssembly(s)
	if err != nil {
		return nil, err
	}
	var in wire.SuppressOccurrenceArgs
	if err := decode(raw, &in); err != nil {
		return nil, err
	}
	o, err := occurrenceByID(asm, in.ID, wire.MethodAssemblySuppress)
	if err != nil {
		return nil, err
	}
	if err := asm.SetOccurrenceSuppressed(o, in.Suppressed); err != nil {
		return nil, fmt.Errorf("%s: %w", wire.MethodAssemblySuppress, err)
	}
	return occurrenceReply(o)
}

// assemblyReplace swaps an occurrence's component for another open document's, keeping the
// occurrence's id, name, transform, and state.
func assemblyReplace(s *app.Session, raw json.RawMessage) (json.RawMessage, error) {
	asm, err := modelaccess.ActiveAssembly(s)
	if err != nil {
		return nil, err
	}
	var in wire.ReplaceOccurrenceArgs
	if err := decode(raw, &in); err != nil {
		return nil, err
	}
	o, err := occurrenceByID(asm, in.ID, wire.MethodAssemblyReplace)
	if err != nil {
		return nil, err
	}
	def, err := placeableDefinition(s, in.Document, wire.MethodAssemblyReplace)
	if err != nil {
		return nil, err
	}
	asm.Occurrences().Replace(o, def)
	return occurrenceReply(o)
}

// assemblyRemove deletes an occurrence and returns the refreshed tree (vetoable).
func assemblyRemove(s *app.Session, raw json.RawMessage) (json.RawMessage, error) {
	asm, err := modelaccess.ActiveAssembly(s)
	if err != nil {
		return nil, err
	}
	var in wire.RemoveOccurrenceArgs
	if err := decode(raw, &in); err != nil {
		return nil, err
	}
	o, err := occurrenceByID(asm, in.ID, wire.MethodAssemblyRemove)
	if err != nil {
		return nil, err
	}
	if err := asm.DeleteOccurrence(o); err != nil {
		return nil, fmt.Errorf("%s: %w", wire.MethodAssemblyRemove, err)
	}
	return json.Marshal(occurrenceTreeResult(asm))
}

// occurrenceFromArgs decodes an id-bearing request and resolves its occurrence — shared by
// the simple by-id mutators (transform, ground). id points into the decoded struct.
func occurrenceFromArgs(s *app.Session, raw json.RawMessage, in any, id *uint64, method string) (*occurrence.Occurrence, error) {
	asm, err := modelaccess.ActiveAssembly(s)
	if err != nil {
		return nil, err
	}
	if err := decode(raw, in); err != nil {
		return nil, err
	}
	return occurrenceByID(asm, *id, method)
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

// occurrenceReply marshals one occurrence as the single-occurrence result.
func occurrenceReply(o *occurrence.Occurrence) (json.RawMessage, error) {
	return json.Marshal(wire.OccurrenceResult{Occurrence: occurrenceInfo(o)})
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

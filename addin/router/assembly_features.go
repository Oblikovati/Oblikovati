// SPDX-License-Identifier: GPL-2.0-only

package router

import (
	"encoding/json"
	"fmt"

	"oblikovati.org/addin/modelaccess"
	"oblikovati.org/api/types"
	"oblikovati.org/api/wire"
	"oblikovati.org/app"
	"oblikovati.org/kernel/brep"
	"oblikovati.org/kernel/ops"
	"oblikovati.org/math"
	"oblikovati.org/model/compdef"
	"oblikovati.org/model/feature"
	"oblikovati.org/model/occurrence"
)

// The assembly feature program surface (M11-F08, #633/#725): list the machining
// features authored in the active assembly, add a box-tool cut, edit participation and
// suppression, and move the end-of-features rollback marker. Features are addressed by
// their stable id; participants by occurrence session id (the ids the occurrence push
// events carry).

// registerAssemblyFeatureHandlers wires the assemblyFeatures.* and assembly end-of-
// features methods.
func (r *Router) registerAssemblyFeatureHandlers() {
	r.handlers[wire.MethodAssemblyFeaturesList] = assemblyFeaturesList
	r.handlers[wire.MethodAssemblyFeaturesAdd] = assemblyFeaturesAdd
	r.handlers[wire.MethodAssemblyFeaturesAddProxyCut] = assemblyFeaturesAddProxyCut
	r.handlers[wire.MethodAssemblyFeaturesAddHole] = assemblyFeaturesAddHole
	r.handlers[wire.MethodAssemblyFeaturesSetParticipants] = assemblyFeaturesSetParticipants
	r.handlers[wire.MethodAssemblyFeaturesSetParticipantPaths] = assemblyFeaturesSetParticipantPaths
	r.handlers[wire.MethodAssemblyFeaturesSetSuppressed] = assemblyFeaturesSetSuppressed
	r.handlers[wire.MethodAssemblyGetEndOfFeatures] = assemblyGetEndOfFeatures
	r.handlers[wire.MethodAssemblySetEndOfFeatures] = assemblySetEndOfFeatures
}

// assemblyFeaturesList returns the active assembly's feature program and marker state.
func assemblyFeaturesList(s *app.Session, _ json.RawMessage) (json.RawMessage, error) {
	asm, err := modelaccess.ActiveAssembly(s)
	if err != nil {
		return nil, err
	}
	return json.Marshal(assemblyFeaturesResult(asm))
}

// assemblyFeaturesAdd adds a box-tool cut feature and returns its refreshed info.
func assemblyFeaturesAdd(s *app.Session, raw json.RawMessage) (json.RawMessage, error) {
	asm, err := modelaccess.ActiveAssembly(s)
	if err != nil {
		return nil, err
	}
	var in wire.AddAssemblyFeatureArgs
	if err := decode(raw, &in); err != nil {
		return nil, err
	}
	cut, err := assemblyCutFromArgs(in)
	if err != nil {
		return nil, err
	}
	af := asm.AddFeature(cut)
	af.SetName(asm.Features().UniqueName(af.Kind()))
	asm.RecomputeFeatures()
	return json.Marshal(wire.AssemblyFeatureResult{Feature: assemblyFeatureInfo(af)})
}

// assemblyFeaturesAddProxyCut adds a feature whose tool is the proxied geometry of the
// source occurrence, re-resolved each rebuild. The source is excluded from the new
// feature's default participation — a component does not machine itself.
func assemblyFeaturesAddProxyCut(s *app.Session, raw json.RawMessage) (json.RawMessage, error) {
	asm, err := modelaccess.ActiveAssembly(s)
	if err != nil {
		return nil, err
	}
	var in wire.AddProxyCutFeatureArgs
	if err := decode(raw, &in); err != nil {
		return nil, err
	}
	op, err := cutOperation(in.Operation)
	if err != nil {
		return nil, err
	}
	source, ok := asm.Occurrences().ByID(in.Source)
	if !ok {
		return nil, fmt.Errorf("%s: no occurrence with id %d in the assembly", wire.MethodAssemblyFeaturesAddProxyCut, in.Source)
	}
	af := asm.AddFeature(feature.NewAssemblyProxyCutFeature(source, op))
	af.RemoveParticipant(source)
	af.SetName(asm.Features().UniqueName(af.Kind()))
	asm.RecomputeFeatures()
	return json.Marshal(wire.AssemblyFeatureResult{Feature: assemblyFeatureInfo(af)})
}

// assemblyFeaturesAddHole drills a parametric hole through the participants.
func assemblyFeaturesAddHole(s *app.Session, raw json.RawMessage) (json.RawMessage, error) {
	asm, err := modelaccess.ActiveAssembly(s)
	if err != nil {
		return nil, err
	}
	var in wire.AddAssemblyHoleArgs
	if err := decode(raw, &in); err != nil {
		return nil, err
	}
	axis, err := math.NewUnitVector3(in.Axis[0], in.Axis[1], in.Axis[2])
	if err != nil {
		return nil, fmt.Errorf("%s: axis %v is not a direction: %w", wire.MethodAssemblyFeaturesAddHole, in.Axis, err)
	}
	hole, err := feature.NewAssemblyHoleFeature(math.P3(in.Center[0], in.Center[1], in.Center[2]), axis, in.Diameter, in.Depth)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", wire.MethodAssemblyFeaturesAddHole, err)
	}
	af := asm.AddFeature(hole)
	af.SetName(asm.Features().UniqueName(af.Kind()))
	asm.RecomputeFeatures()
	return json.Marshal(wire.AssemblyFeatureResult{Feature: assemblyFeatureInfo(af)})
}

// assemblyFeaturesSetParticipants replaces a feature's participation set.
func assemblyFeaturesSetParticipants(s *app.Session, raw json.RawMessage) (json.RawMessage, error) {
	asm, err := modelaccess.ActiveAssembly(s)
	if err != nil {
		return nil, err
	}
	var in wire.SetAssemblyParticipantsArgs
	if err := decode(raw, &in); err != nil {
		return nil, err
	}
	af, err := assemblyFeatureByID(asm, in.ID, wire.MethodAssemblyFeaturesSetParticipants)
	if err != nil {
		return nil, err
	}
	occs, err := occurrencesByID(asm, in.Participants, wire.MethodAssemblyFeaturesSetParticipants)
	if err != nil {
		return nil, err
	}
	af.SetParticipants(occs)
	asm.RecomputeFeatures()
	return json.Marshal(wire.AssemblyFeatureResult{Feature: assemblyFeatureInfo(af)})
}

// assemblyFeaturesSetParticipantPaths restricts a feature to specific nested occurrence
// paths (or clears the restriction when none are given).
func assemblyFeaturesSetParticipantPaths(s *app.Session, raw json.RawMessage) (json.RawMessage, error) {
	asm, err := modelaccess.ActiveAssembly(s)
	if err != nil {
		return nil, err
	}
	var in wire.SetAssemblyParticipantPathsArgs
	if err := decode(raw, &in); err != nil {
		return nil, err
	}
	af, err := assemblyFeatureByID(asm, in.ID, wire.MethodAssemblyFeaturesSetParticipantPaths)
	if err != nil {
		return nil, err
	}
	paths, err := resolveParticipantPaths(asm, in.Paths, wire.MethodAssemblyFeaturesSetParticipantPaths)
	if err != nil {
		return nil, err
	}
	af.SetParticipantPaths(paths)
	asm.RecomputeFeatures()
	return json.Marshal(wire.AssemblyFeatureResult{Feature: assemblyFeatureInfo(af)})
}

// resolveParticipantPaths validates each instance-name path resolves to an occurrence
// in the assembly, rejecting an unresolvable path (and returning nil to clear when no
// paths are given).
func resolveParticipantPaths(asm *compdef.AssemblyComponentDefinition, paths [][]string, method string) ([]occurrence.OccurrencePath, error) {
	if len(paths) == 0 {
		return nil, nil
	}
	out := make([]occurrence.OccurrencePath, 0, len(paths))
	for _, names := range paths {
		path := occurrence.OccurrencePath(names)
		if _, ok := asm.Occurrences().Resolve(path); !ok {
			return nil, fmt.Errorf("%s: path %v resolves to no occurrence in the assembly", method, names)
		}
		out = append(out, path)
	}
	return out, nil
}

// assemblyFeaturesSetSuppressed suppresses or unsuppresses the named features in batch.
func assemblyFeaturesSetSuppressed(s *app.Session, raw json.RawMessage) (json.RawMessage, error) {
	asm, err := modelaccess.ActiveAssembly(s)
	if err != nil {
		return nil, err
	}
	var in wire.SetAssemblyFeaturesSuppressedArgs
	if err := decode(raw, &in); err != nil {
		return nil, err
	}
	if in.Suppressed {
		asm.Features().SuppressFeatures(in.IDs...)
	} else {
		asm.Features().UnsuppressFeatures(in.IDs...)
	}
	asm.RecomputeFeatures()
	return json.Marshal(assemblyFeaturesResult(asm))
}

// assemblyGetEndOfFeatures returns the active assembly's rollback-marker state.
func assemblyGetEndOfFeatures(s *app.Session, _ json.RawMessage) (json.RawMessage, error) {
	asm, err := modelaccess.ActiveAssembly(s)
	if err != nil {
		return nil, err
	}
	feats := asm.Features()
	return json.Marshal(wire.EndOfFeaturesResult{Position: feats.EndOfFeaturesPosition(), RolledBack: feats.IsRolledBack()})
}

// assemblySetEndOfFeatures moves the rollback marker and returns the refreshed program.
func assemblySetEndOfFeatures(s *app.Session, raw json.RawMessage) (json.RawMessage, error) {
	asm, err := modelaccess.ActiveAssembly(s)
	if err != nil {
		return nil, err
	}
	var in wire.SetEndOfFeaturesArgs
	if err := decode(raw, &in); err != nil {
		return nil, err
	}
	asm.Features().SetEndOfFeatures(in.Position)
	asm.RecomputeFeatures()
	return json.Marshal(assemblyFeaturesResult(asm))
}

// assemblyCutFromArgs builds the assembly-space box-tool cut feature from the request.
func assemblyCutFromArgs(in wire.AddAssemblyFeatureArgs) (feature.Feature, error) {
	op, err := cutOperation(in.Operation)
	if err != nil {
		return nil, err
	}
	min := math.P3(in.ToolMin[0], in.ToolMin[1], in.ToolMin[2])
	max := math.P3(in.ToolMax[0], in.ToolMax[1], in.ToolMax[2])
	tool, err := brep.SolidBlock(min, max, "asmCut")
	if err != nil {
		return nil, fmt.Errorf("%s: tool box %v..%v: %w", wire.MethodAssemblyFeaturesAdd, min, max, err)
	}
	return feature.NewAssemblyCutFeature(tool, op), nil
}

// cutOperation maps a BooleanType wire spelling to the model boolean operation.
func cutOperation(spell string) (ops.PartFeatureOperation, error) {
	bt, ok := types.ParseBooleanType(spell)
	if !ok {
		return 0, fmt.Errorf("%s: unknown operation %q (want difference/union/intersect)", wire.MethodAssemblyFeaturesAdd, spell)
	}
	switch bt {
	case types.BooleanDifference:
		return ops.Cut, nil
	case types.BooleanUnion:
		return ops.Join, nil
	default: // BooleanIntersect
		return ops.Intersect, nil
	}
}

// assemblyFeatureByID resolves a wire feature id against the assembly's program.
func assemblyFeatureByID(asm *compdef.AssemblyComponentDefinition, id uint64, method string) (*compdef.AssemblyFeature, error) {
	af, ok := asm.Features().ByID(id)
	if !ok {
		return nil, fmt.Errorf("%s: no assembly feature with id %d (ids come from assemblyFeatures.list)", method, id)
	}
	return af, nil
}

// occurrencesByID resolves participant occurrence session ids against the assembly,
// rejecting an unknown id.
func occurrencesByID(asm *compdef.AssemblyComponentDefinition, ids []uint64, method string) ([]*occurrence.Occurrence, error) {
	occs := make([]*occurrence.Occurrence, 0, len(ids))
	for _, id := range ids {
		o, ok := asm.Occurrences().ByID(id)
		if !ok {
			return nil, fmt.Errorf("%s: no occurrence with id %d in the assembly", method, id)
		}
		occs = append(occs, o)
	}
	return occs, nil
}

// assemblyFeaturesResult renders the whole program plus the rollback-marker state.
func assemblyFeaturesResult(asm *compdef.AssemblyComponentDefinition) wire.AssemblyFeaturesResult {
	feats := asm.Features()
	out := make([]wire.AssemblyFeatureInfo, feats.Count())
	for i := 0; i < feats.Count(); i++ {
		out[i] = assemblyFeatureInfo(feats.Item(i))
	}
	return wire.AssemblyFeaturesResult{
		Features:      out,
		EndOfFeatures: feats.EndOfFeaturesPosition(),
		RolledBack:    feats.IsRolledBack(),
	}
}

// assemblyFeatureInfo renders one assembly feature as its wire DTO.
func assemblyFeatureInfo(af *compdef.AssemblyFeature) wire.AssemblyFeatureInfo {
	info := wire.AssemblyFeatureInfo{
		ID:         af.ID(),
		Kind:       af.Kind(),
		Name:       af.Name(),
		Suppressed: af.Suppressed(),
	}
	if h := af.Health(); !h.OK() {
		info.Health = h.Reason
	}
	parts := af.Participants()
	info.Participants = make([]uint64, len(parts))
	for i, o := range parts {
		info.Participants[i] = o.ID()
	}
	for _, p := range af.ParticipantPaths() {
		info.ParticipantPaths = append(info.ParticipantPaths, []string(p))
	}
	return info
}

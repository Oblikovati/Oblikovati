// SPDX-License-Identifier: GPL-2.0-only

package router

import (
	"encoding/json"
	"fmt"
	stdmath "math"

	"oblikovati.org/addin/modelaccess"
	"oblikovati.org/api/types"
	"oblikovati.org/api/wire"
	"oblikovati.org/app"
	"oblikovati.org/kernel/brep"
	"oblikovati.org/kernel/ops"
	"oblikovati.org/kernel/topo"
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
	r.readOnly(wire.MethodAssemblyFeaturesList, assemblyFeaturesList)
	r.mutating(wire.MethodAssemblyFeaturesAdd, "Add Assembly Feature", assemblyFeaturesAdd)
	r.mutating(wire.MethodAssemblyFeaturesAddProxyCut, "Add Assembly Feature", assemblyFeaturesAddProxyCut)
	r.mutating(wire.MethodAssemblyFeaturesAddHole, "Hole", assemblyFeaturesAddHole)
	r.mutating(wire.MethodAssemblyFeaturesAddExtrude, "Extrude", assemblyFeaturesAddExtrude)
	r.mutating(wire.MethodAssemblyFeaturesAddRevolve, "Add Assembly Feature", assemblyFeaturesAddRevolve)
	r.mutating(wire.MethodAssemblyFeaturesAddChamfer, "Chamfer", assemblyFeaturesAddChamfer)
	r.mutating(wire.MethodAssemblyFeaturesAddFillet, "Fillet", assemblyFeaturesAddFillet)
	r.mutating(wire.MethodAssemblyFeaturesAddMoveFace, "Move Face", assemblyFeaturesAddMoveFace)
	r.mutating(wire.MethodAssemblyFeaturesAddSweep, "Sweep", assemblyFeaturesAddSweep)
	r.mutating(wire.MethodAssemblyFeaturesEdit, "Edit Assembly Feature", assemblyFeaturesEdit)
	r.mutating(wire.MethodAssemblyFeaturesSetParticipants, "Edit Participants", assemblyFeaturesSetParticipants)
	r.mutating(wire.MethodAssemblyFeaturesSetParticipantPaths, "Edit Participants", assemblyFeaturesSetParticipantPaths)
	r.mutating(wire.MethodAssemblyFeaturesSetSuppressed, "Suppress Assembly Feature", assemblyFeaturesSetSuppressed)
	r.readOnly(wire.MethodAssemblyGetEndOfFeatures, assemblyGetEndOfFeatures)
	r.mutating(wire.MethodAssemblySetEndOfFeatures, "Set End of Features", assemblySetEndOfFeatures)
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
	return json.Marshal(wire.AssemblyFeatureResult{Feature: assemblyFeatureInfo(asm, af)})
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
	return json.Marshal(wire.AssemblyFeatureResult{Feature: assemblyFeatureInfo(asm, af)})
}

// assemblyFeaturesAddExtrude extrudes a closed sketch profile (authored on an assembly
// work plane) into the participants — a profiled pocket or boss.
func assemblyFeaturesAddExtrude(s *app.Session, raw json.RawMessage) (json.RawMessage, error) {
	asm, err := modelaccess.ActiveAssembly(s)
	if err != nil {
		return nil, err
	}
	var in wire.AddAssemblyExtrudeArgs
	if err := decode(raw, &in); err != nil {
		return nil, err
	}
	op, err := cutOperation(in.Operation)
	if err != nil {
		return nil, err
	}
	sk, err := sketchAtIndex(asm, in.SketchIndex)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", wire.MethodAssemblyFeaturesAddExtrude, err)
	}
	if in.Distance <= 0 {
		return nil, fmt.Errorf("%s: distance %g must be positive", wire.MethodAssemblyFeaturesAddExtrude, in.Distance)
	}
	distance := in.Distance
	af := asm.AddFeature(feature.NewAssemblyExtrudeFeature(sk, in.ProfileIndex, op, func() float64 { return distance }))
	af.SetName(asm.Features().UniqueName(af.Kind()))
	asm.RecomputeFeatures()
	return json.Marshal(wire.AssemblyFeatureResult{Feature: assemblyFeatureInfo(asm, af)})
}

// assemblyFeaturesAddRevolve revolves a closed sketch profile (authored on an assembly
// work plane) about the axis line (origin + direction in assembly space) into the
// participants — a turned groove or boss.
func assemblyFeaturesAddRevolve(s *app.Session, raw json.RawMessage) (json.RawMessage, error) {
	asm, err := modelaccess.ActiveAssembly(s)
	if err != nil {
		return nil, err
	}
	var in wire.AddAssemblyRevolveArgs
	if err := decode(raw, &in); err != nil {
		return nil, err
	}
	op, err := cutOperation(in.Operation)
	if err != nil {
		return nil, err
	}
	sk, err := sketchAtIndex(asm, in.SketchIndex)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", wire.MethodAssemblyFeaturesAddRevolve, err)
	}
	axis, err := revolveAxisFromArgs(in)
	if err != nil {
		return nil, err
	}
	angle := in.Angle
	af := asm.AddFeature(feature.NewAssemblyRevolveFeature(sk, in.ProfileIndex, axis, op, func() float64 { return angle }))
	af.SetName(asm.Features().UniqueName(af.Kind()))
	asm.RecomputeFeatures()
	return json.Marshal(wire.AssemblyFeatureResult{Feature: assemblyFeatureInfo(asm, af)})
}

// revolveAxisFromArgs builds the assembly-space revolve axis from the request's origin and
// direction, validating that the direction is non-zero and the angle is in (0,2π].
func revolveAxisFromArgs(in wire.AddAssemblyRevolveArgs) (*feature.WorkAxis, error) {
	dir, err := math.NewUnitVector3(in.Axis[0], in.Axis[1], in.Axis[2])
	if err != nil {
		return nil, fmt.Errorf("%s: axis %v is not a direction: %w", wire.MethodAssemblyFeaturesAddRevolve, in.Axis, err)
	}
	if in.Angle <= 0 || in.Angle > 2*stdmath.Pi+1e-9 {
		return nil, fmt.Errorf("%s: angle %g must be in (0, 2π]", wire.MethodAssemblyFeaturesAddRevolve, in.Angle)
	}
	return feature.NewDatumAxis(math.P3(in.Origin[0], in.Origin[1], in.Origin[2]), dir), nil
}

// assemblyFeaturesAddSweep sweeps an assembly sketch profile along an explicit polyline path
// into the participants — a swept channel or rib.
func assemblyFeaturesAddSweep(s *app.Session, raw json.RawMessage) (json.RawMessage, error) {
	asm, err := modelaccess.ActiveAssembly(s)
	if err != nil {
		return nil, err
	}
	var in wire.AddAssemblySweepArgs
	if err := decode(raw, &in); err != nil {
		return nil, err
	}
	op, err := cutOperation(in.Operation)
	if err != nil {
		return nil, err
	}
	sk, err := sketchAtIndex(asm, in.SketchIndex)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", wire.MethodAssemblyFeaturesAddSweep, err)
	}
	path, err := assemblySweepPath(in.Path, wire.MethodAssemblyFeaturesAddSweep)
	if err != nil {
		return nil, err
	}
	af := asm.AddFeature(feature.NewAssemblySweepFeature(sk, in.ProfileIndex, op, path))
	af.SetName(asm.Features().UniqueName(af.Kind()))
	asm.RecomputeFeatures()
	return json.Marshal(wire.AssemblyFeatureResult{Feature: assemblyFeatureInfo(asm, af)})
}

// assemblySweepPath converts a wire path polyline to assembly-space points, requiring at
// least two.
func assemblySweepPath(points [][3]float64, method string) ([]math.Point3, error) {
	if len(points) < 2 {
		return nil, fmt.Errorf("%s: path needs at least two points, got %d", method, len(points))
	}
	path := make([]math.Point3, len(points))
	for i, p := range points {
		path[i] = math.P3(p[0], p[1], p[2])
	}
	return path, nil
}

// assemblyFeaturesAddChamfer chamfers picked component edges on every participant.
func assemblyFeaturesAddChamfer(s *app.Session, raw json.RawMessage) (json.RawMessage, error) {
	asm, err := modelaccess.ActiveAssembly(s)
	if err != nil {
		return nil, err
	}
	var in wire.AddAssemblyChamferArgs
	if err := decode(raw, &in); err != nil {
		return nil, err
	}
	if in.Distance <= 0 {
		return nil, fmt.Errorf("%s: distance %g must be positive", wire.MethodAssemblyFeaturesAddChamfer, in.Distance)
	}
	suffixes, err := assemblyEdgeSuffixes(asm, in.Edges, wire.MethodAssemblyFeaturesAddChamfer)
	if err != nil {
		return nil, err
	}
	dist := in.Distance
	af := asm.AddFeature(feature.NewAssemblyChamferFeature(suffixes, func() float64 { return dist }))
	af.SetName(asm.Features().UniqueName(af.Kind()))
	asm.RecomputeFeatures()
	return json.Marshal(wire.AssemblyFeatureResult{Feature: assemblyFeatureInfo(asm, af)})
}

// assemblyFeaturesAddFillet rounds picked component edges on every participant.
func assemblyFeaturesAddFillet(s *app.Session, raw json.RawMessage) (json.RawMessage, error) {
	asm, err := modelaccess.ActiveAssembly(s)
	if err != nil {
		return nil, err
	}
	var in wire.AddAssemblyFilletArgs
	if err := decode(raw, &in); err != nil {
		return nil, err
	}
	if in.Radius <= 0 {
		return nil, fmt.Errorf("%s: radius %g must be positive", wire.MethodAssemblyFeaturesAddFillet, in.Radius)
	}
	suffixes, err := assemblyEdgeSuffixes(asm, in.Edges, wire.MethodAssemblyFeaturesAddFillet)
	if err != nil {
		return nil, err
	}
	r := in.Radius
	af := asm.AddFeature(feature.NewAssemblyFilletFeature(suffixes, func() float64 { return r }))
	af.SetName(asm.Features().UniqueName(af.Kind()))
	asm.RecomputeFeatures()
	return json.Marshal(wire.AssemblyFeatureResult{Feature: assemblyFeatureInfo(asm, af)})
}

// assemblyFeaturesAddMoveFace translates picked component faces on every participant.
func assemblyFeaturesAddMoveFace(s *app.Session, raw json.RawMessage) (json.RawMessage, error) {
	asm, err := modelaccess.ActiveAssembly(s)
	if err != nil {
		return nil, err
	}
	var in wire.AddAssemblyMoveFaceArgs
	if err := decode(raw, &in); err != nil {
		return nil, err
	}
	suffixes, err := assemblyFaceSuffixes(asm, in.Faces, wire.MethodAssemblyFeaturesAddMoveFace)
	if err != nil {
		return nil, err
	}
	delta := math.V3(in.Translation[0], in.Translation[1], in.Translation[2])
	af := asm.AddFeature(feature.NewAssemblyMoveFaceFeature(suffixes, delta))
	af.SetName(asm.Features().UniqueName(af.Kind()))
	asm.RecomputeFeatures()
	return json.Marshal(wire.AssemblyFeatureResult{Feature: assemblyFeatureInfo(asm, af)})
}

// assemblyEdgeSuffixes resolves each edge ref to a component-local lineage suffix, after
// validating the edge exists on its occurrence's component body. The suffix is what each
// participant's placed body is matched against at recompute (#735).
func assemblyEdgeSuffixes(asm *compdef.AssemblyComponentDefinition, refs []wire.AssemblyEdgeRef, method string) ([][]byte, error) {
	if len(refs) == 0 {
		return nil, fmt.Errorf("%s: no edges given", method)
	}
	out := make([][]byte, 0, len(refs))
	for _, ref := range refs {
		suffix, err := assemblyRefSuffix(asm, ref.Occurrence, []byte(ref.Edge), method, edgeOnComponent)
		if err != nil {
			return nil, err
		}
		out = append(out, suffix)
	}
	return out, nil
}

// assemblyFaceSuffixes is the face twin of [assemblyEdgeSuffixes].
func assemblyFaceSuffixes(asm *compdef.AssemblyComponentDefinition, refs []wire.AssemblyFaceRef, method string) ([][]byte, error) {
	if len(refs) == 0 {
		return nil, fmt.Errorf("%s: no faces given", method)
	}
	out := make([][]byte, 0, len(refs))
	for _, ref := range refs {
		suffix, err := assemblyRefSuffix(asm, ref.Occurrence, []byte(ref.Face), method, faceOnComponent)
		if err != nil {
			return nil, err
		}
		out = append(out, suffix)
	}
	return out, nil
}

// assemblyRefSuffix validates that key names an entity on the occurrence's component body
// (via present) and returns its component-local lineage suffix.
func assemblyRefSuffix(asm *compdef.AssemblyComponentDefinition, occID uint64, key []byte, method string, present func([]*topo.Body, []byte) bool) ([]byte, error) {
	occ, err := occurrenceByID(asm, occID, method)
	if err != nil {
		return nil, err
	}
	if !present(componentBodies(occ), key) {
		return nil, fmt.Errorf("%s: reference key not found on occurrence %d's component", method, occID)
	}
	return topo.LineageSuffixOf(key), nil
}

// componentBodies returns the evaluated surface bodies of an occurrence's component
// definition, or nil when it is not a body-bearing part.
func componentBodies(occ *occurrence.Occurrence) []*topo.Body {
	def, ok := occ.Definition().(interface {
		SurfaceBodies() *topo.SurfaceBodies
	})
	if !ok {
		return nil
	}
	return def.SurfaceBodies().All()
}

// edgeOnComponent / faceOnComponent report whether any component body carries the key.
func edgeOnComponent(bodies []*topo.Body, key []byte) bool {
	for _, b := range bodies {
		if _, ok := b.FindEdgeByKey(key); ok {
			return true
		}
	}
	return false
}

func faceOnComponent(bodies []*topo.Body, key []byte) bool {
	for _, b := range bodies {
		if _, ok := b.FindFaceByKey(key); ok {
			return true
		}
	}
	return false
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
	return json.Marshal(wire.AssemblyFeatureResult{Feature: assemblyFeatureInfo(asm, af)})
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
	return json.Marshal(wire.AssemblyFeatureResult{Feature: assemblyFeatureInfo(asm, af)})
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
	return json.Marshal(wire.AssemblyFeatureResult{Feature: assemblyFeatureInfo(asm, af)})
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

// assemblyFeaturesEdit sets editable scalars of an assembly feature in place — the
// assembly-context Edit Feature (#725), mirroring the part features.edit. The batch is
// validated before any value is applied, then the feature program recomputes once.
func assemblyFeaturesEdit(s *app.Session, raw json.RawMessage) (json.RawMessage, error) {
	asm, err := modelaccess.ActiveAssembly(s)
	if err != nil {
		return nil, err
	}
	var in wire.EditAssemblyFeatureArgs
	if err := decode(raw, &in); err != nil {
		return nil, err
	}
	af, err := assemblyFeatureByID(asm, in.ID, wire.MethodAssemblyFeaturesEdit)
	if err != nil {
		return nil, err
	}
	ed, ok := af.Definition().(feature.Editable)
	if !ok {
		return nil, fmt.Errorf("%s: feature %d (%s) has no editable scalars", wire.MethodAssemblyFeaturesEdit, af.ID(), af.Kind())
	}
	apply, err := planScalarEdits(asm.Units(), ed, in.Scalars, wire.MethodAssemblyFeaturesEdit)
	if err != nil {
		return nil, err
	}
	apply()
	asm.RecomputeFeatures()
	return json.Marshal(wire.AssemblyFeatureResult{Feature: assemblyFeatureInfo(asm, af)})
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
		out[i] = assemblyFeatureInfo(asm, feats.Item(i))
	}
	return wire.AssemblyFeaturesResult{
		Features:      out,
		EndOfFeatures: feats.EndOfFeaturesPosition(),
		RolledBack:    feats.IsRolledBack(),
	}
}

// assemblyFeatureInfo renders one assembly feature as its wire DTO.
func assemblyFeatureInfo(asm *compdef.AssemblyComponentDefinition, af *compdef.AssemblyFeature) wire.AssemblyFeatureInfo {
	info := wire.AssemblyFeatureInfo{
		ID:         af.ID(),
		Kind:       af.Kind(),
		Name:       af.Name(),
		Suppressed: af.Suppressed(),
		Scalars:    editableScalars(asm.Units(), af.Definition()),
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

// SPDX-License-Identifier: GPL-2.0-only

package router

import (
	"encoding/json"

	"oblikovati.org/addin/modelaccess"
	"oblikovati.org/api/wire"
	"oblikovati.org/app"
	"oblikovati.org/event"
	"oblikovati.org/model/assembly"
	"oblikovati.org/model/compdef"
)

// The assembly representation surface (M12-F04, #361/#367): capture/activate/edit the three
// override-layer families (design-view/positional/level-of-detail) and the model states that
// select one of each. Capture snapshots the active assembly's current state; activate applies
// a representation's overrides (positional activate re-solves). Overrides are addressed by
// occurrence/relationship id; the engine stores them by stable occurrence name.

// registerRepresentationHandlers wires the designReps.*/positionalReps.*/lodReps.*/
// modelStates.* methods.
func (r *Router) registerRepresentationHandlers() {
	r.registerDesignRepHandlers()
	r.registerPositionalRepHandlers()
	r.registerLODRepHandlers()
	r.registerModelStateHandlers()
}

func (r *Router) registerDesignRepHandlers() {
	r.mutating(wire.MethodDesignRepsCapture, "Capture Design View", typedAssembly(designRepsCapture))
	r.readOnly(wire.MethodDesignRepsActivate, typedAssembly(designRepsActivate))
	r.readOnly(wire.MethodDesignRepsList, assemblyQuery(designRepsList))
	r.mutating(wire.MethodDesignRepsDelete, "Delete Design View", typedAssembly(designRepsDelete))
	r.mutating(wire.MethodDesignRepsSetVisibility, "Edit Design View", designRepsSetVisibility)
	r.mutating(wire.MethodDesignRepsSetAppearance, "Edit Design View", designRepsSetAppearance)
	r.mutating(wire.MethodDesignRepsAddSection, "Add Section View", designRepsAddSection)
}

func (r *Router) registerPositionalRepHandlers() {
	r.mutating(wire.MethodPositionalRepsCapture, "Capture Positional Rep", typedAssembly(positionalRepsCapture))
	r.readOnly(wire.MethodPositionalRepsActivate, typedAssembly(positionalRepsActivate))
	r.readOnly(wire.MethodPositionalRepsList, assemblyQuery(positionalRepsList))
	r.mutating(wire.MethodPositionalRepsDelete, "Delete Positional Rep", typedAssembly(positionalRepsDelete))
	r.mutating(wire.MethodPositionalRepsSetOverride, "Edit Positional Rep", positionalRepsSetOverride)
	r.mutating(wire.MethodPositionalRepsSetFlexible, "Edit Positional Rep", positionalRepsSetFlexible)
}

func (r *Router) registerLODRepHandlers() {
	r.mutating(wire.MethodLODRepsCapture, "Capture LOD Rep", typedAssembly(lodRepsCapture))
	r.readOnly(wire.MethodLODRepsActivate, typedAssembly(lodRepsActivate))
	r.readOnly(wire.MethodLODRepsList, assemblyQuery(lodRepsList))
	r.mutating(wire.MethodLODRepsDelete, "Delete LOD Rep", typedAssembly(lodRepsDelete))
	r.mutating(wire.MethodLODRepsSetSuppressed, "Edit LOD Rep", lodRepsSetSuppressed)
}

func (r *Router) registerModelStateHandlers() {
	r.mutating(wire.MethodModelStatesCreate, "Create Model State", typedAssembly(modelStatesCreate))
	r.readOnly(wire.MethodModelStatesActivate, typedAssembly(modelStatesActivate))
	r.readOnly(wire.MethodModelStatesList, assemblyQuery(modelStatesList))
	r.mutating(wire.MethodModelStatesDelete, "Delete Model State", typedAssembly(modelStatesDelete))
}

// --- design-view ---

func designRepsCapture(s *app.Session, asm *compdef.AssemblyComponentDefinition, in wire.CaptureRepArgs) (wire.DesignViewResult, error) {
	d := asm.Representations().CaptureDesignView(in.Name, capturedCamera(s))
	event.Emit(s.Events(), event.After, app.RepresentationCaptured{Kind: "design", Name: d.Name()})
	return wire.DesignViewResult{Representation: designViewInfo(d)}, nil
}

func designRepsActivate(s *app.Session, asm *compdef.AssemblyComponentDefinition, in wire.RepRef) (wire.DesignViewResult, error) {
	d, err := asm.Representations().ActivateDesignView(in.ID)
	if err != nil {
		return wire.DesignViewResult{}, err
	}
	event.Emit(s.Events(), event.After, app.RepresentationActivated{Kind: "design", Name: d.Name()})
	return wire.DesignViewResult{Representation: designViewInfo(d)}, nil
}

func designRepsList(_ *app.Session, asm *compdef.AssemblyComponentDefinition) (wire.DesignViewsResult, error) {
	return designViewsListResult(asm), nil
}

// designViewsListResult renders the active assembly's design-view representations.
func designViewsListResult(asm *compdef.AssemblyComponentDefinition) wire.DesignViewsResult {
	out := make([]wire.DesignViewInfo, 0)
	for _, d := range asm.Representations().AllDesignViews() {
		out = append(out, designViewInfo(d))
	}
	return wire.DesignViewsResult{Representations: out}
}

func designRepsDelete(_ *app.Session, asm *compdef.AssemblyComponentDefinition, in wire.RepRef) (wire.DesignViewsResult, error) {
	asm.Representations().DeleteDesignView(in.ID)
	return designViewsListResult(asm), nil
}

func designRepsSetVisibility(s *app.Session, raw json.RawMessage) (json.RawMessage, error) {
	asm, err := modelaccess.ActiveAssembly(s)
	if err != nil {
		return nil, err
	}
	var in wire.SetVisibilityArgs
	if err := decode(raw, &in); err != nil {
		return nil, err
	}
	occ, err := occurrenceByID(asm, in.Occurrence, wire.MethodDesignRepsSetVisibility)
	if err != nil {
		return nil, err
	}
	if err := asm.Representations().SetVisibility(in.Rep, occ, in.Visible); err != nil {
		return nil, err
	}
	return designViewResult(asm, in.Rep)
}

func designRepsSetAppearance(s *app.Session, raw json.RawMessage) (json.RawMessage, error) {
	asm, err := modelaccess.ActiveAssembly(s)
	if err != nil {
		return nil, err
	}
	var in wire.SetAppearanceArgs
	if err := decode(raw, &in); err != nil {
		return nil, err
	}
	occ, err := occurrenceByID(asm, in.Occurrence, wire.MethodDesignRepsSetAppearance)
	if err != nil {
		return nil, err
	}
	if err := asm.Representations().SetAppearance(in.Rep, occ, in.AppearanceID); err != nil {
		return nil, err
	}
	return designViewResult(asm, in.Rep)
}

func designRepsAddSection(s *app.Session, raw json.RawMessage) (json.RawMessage, error) {
	asm, err := modelaccess.ActiveAssembly(s)
	if err != nil {
		return nil, err
	}
	var in wire.AddSectionArgs
	if err := decode(raw, &in); err != nil {
		return nil, err
	}
	if err := asm.Representations().AddSection(in.Rep, in.Plane); err != nil {
		return nil, err
	}
	return designViewResult(asm, in.Rep)
}

// --- positional ---

func positionalRepsCapture(s *app.Session, asm *compdef.AssemblyComponentDefinition, in wire.CaptureRepArgs) (wire.PositionalResult, error) {
	p := asm.Representations().CapturePositional(in.Name)
	event.Emit(s.Events(), event.After, app.RepresentationCaptured{Kind: "positional", Name: p.Name()})
	return wire.PositionalResult{Representation: positionalInfo(p)}, nil
}

func positionalRepsActivate(s *app.Session, asm *compdef.AssemblyComponentDefinition, in wire.RepRef) (wire.PositionalResult, error) {
	p, err := asm.Representations().ActivatePositional(in.ID)
	if err != nil {
		return wire.PositionalResult{}, err
	}
	event.Emit(s.Events(), event.After, app.RepresentationActivated{Kind: "positional", Name: p.Name()})
	return wire.PositionalResult{Representation: positionalInfo(p)}, nil
}

func positionalRepsList(_ *app.Session, asm *compdef.AssemblyComponentDefinition) (wire.PositionalsResult, error) {
	return positionalsListResult(asm), nil
}

// positionalsListResult renders the active assembly's positional representations.
func positionalsListResult(asm *compdef.AssemblyComponentDefinition) wire.PositionalsResult {
	out := make([]wire.PositionalInfo, 0)
	for _, p := range asm.Representations().AllPositionals() {
		out = append(out, positionalInfo(p))
	}
	return wire.PositionalsResult{Representations: out}
}

func positionalRepsDelete(_ *app.Session, asm *compdef.AssemblyComponentDefinition, in wire.RepRef) (wire.PositionalsResult, error) {
	asm.Representations().DeletePositional(in.ID)
	return positionalsListResult(asm), nil
}

func positionalRepsSetOverride(s *app.Session, raw json.RawMessage) (json.RawMessage, error) {
	asm, err := modelaccess.ActiveAssembly(s)
	if err != nil {
		return nil, err
	}
	var in wire.SetPositionalOverrideArgs
	if err := decode(raw, &in); err != nil {
		return nil, err
	}
	if err := asm.Representations().SetPositionalOverride(in.Rep, in.Relationship, in.IsJoint, in.Value); err != nil {
		return nil, err
	}
	return positionalResult(asm, in.Rep)
}

func positionalRepsSetFlexible(s *app.Session, raw json.RawMessage) (json.RawMessage, error) {
	asm, err := modelaccess.ActiveAssembly(s)
	if err != nil {
		return nil, err
	}
	var in wire.SetFlexibleArgs
	if err := decode(raw, &in); err != nil {
		return nil, err
	}
	occ, err := occurrenceByID(asm, in.Occurrence, wire.MethodPositionalRepsSetFlexible)
	if err != nil {
		return nil, err
	}
	if err := asm.Representations().SetFlexible(in.Rep, occ, in.Flexible); err != nil {
		return nil, err
	}
	return positionalResult(asm, in.Rep)
}

// --- level-of-detail ---

func lodRepsCapture(s *app.Session, asm *compdef.AssemblyComponentDefinition, in wire.CaptureRepArgs) (wire.LODResult, error) {
	l := asm.Representations().CaptureLOD(in.Name)
	event.Emit(s.Events(), event.After, app.RepresentationCaptured{Kind: "lod", Name: l.Name()})
	return wire.LODResult{Representation: lodInfo(l)}, nil
}

func lodRepsActivate(s *app.Session, asm *compdef.AssemblyComponentDefinition, in wire.RepRef) (wire.LODResult, error) {
	l, err := asm.Representations().ActivateLOD(in.ID)
	if err != nil {
		return wire.LODResult{}, err
	}
	event.Emit(s.Events(), event.After, app.RepresentationActivated{Kind: "lod", Name: l.Name()})
	return wire.LODResult{Representation: lodInfo(l)}, nil
}

func lodRepsList(_ *app.Session, asm *compdef.AssemblyComponentDefinition) (wire.LODsResult, error) {
	return lodsListResult(asm), nil
}

// lodsListResult renders the active assembly's level-of-detail representations.
func lodsListResult(asm *compdef.AssemblyComponentDefinition) wire.LODsResult {
	out := make([]wire.LODInfo, 0)
	for _, l := range asm.Representations().AllLODs() {
		out = append(out, lodInfo(l))
	}
	return wire.LODsResult{Representations: out}
}

func lodRepsDelete(_ *app.Session, asm *compdef.AssemblyComponentDefinition, in wire.RepRef) (wire.LODsResult, error) {
	asm.Representations().DeleteLOD(in.ID)
	return lodsListResult(asm), nil
}

func lodRepsSetSuppressed(s *app.Session, raw json.RawMessage) (json.RawMessage, error) {
	asm, err := modelaccess.ActiveAssembly(s)
	if err != nil {
		return nil, err
	}
	var in wire.SetSuppressedArgs
	if err := decode(raw, &in); err != nil {
		return nil, err
	}
	occ, err := occurrenceByID(asm, in.Occurrence, wire.MethodLODRepsSetSuppressed)
	if err != nil {
		return nil, err
	}
	if err := asm.Representations().SetSuppressed(in.Rep, occ, in.Suppressed); err != nil {
		return nil, err
	}
	return lodResult(asm, in.Rep)
}

// --- model states ---

func modelStatesCreate(_ *app.Session, asm *compdef.AssemblyComponentDefinition, in wire.CreateModelStateArgs) (wire.ModelStateResult, error) {
	m := asm.Representations().CreateModelState(in.Name, in.DesignView, in.Positional, in.LevelOfDetail)
	return wire.ModelStateResult{ModelState: modelStateInfo(m)}, nil
}

func modelStatesActivate(s *app.Session, asm *compdef.AssemblyComponentDefinition, in wire.RepRef) (wire.ModelStateResult, error) {
	m, err := asm.Representations().ActivateModelState(in.ID)
	if err != nil {
		return wire.ModelStateResult{}, err
	}
	event.Emit(s.Events(), event.After, app.ModelStateActivated{Name: m.Name()})
	return wire.ModelStateResult{ModelState: modelStateInfo(m)}, nil
}

func modelStatesList(_ *app.Session, asm *compdef.AssemblyComponentDefinition) (wire.ModelStatesResult, error) {
	return modelStatesListResult(asm), nil
}

// modelStatesListResult renders the active assembly's model states.
func modelStatesListResult(asm *compdef.AssemblyComponentDefinition) wire.ModelStatesResult {
	out := make([]wire.ModelStateInfo, 0)
	for _, m := range asm.Representations().AllModelStates() {
		out = append(out, modelStateInfo(m))
	}
	return wire.ModelStatesResult{ModelStates: out}
}

func modelStatesDelete(_ *app.Session, asm *compdef.AssemblyComponentDefinition, in wire.RepRef) (wire.ModelStatesResult, error) {
	asm.Representations().DeleteModelState(in.ID)
	return modelStatesListResult(asm), nil
}

// --- shared helpers ---

// capturedCamera snapshots the session's current camera for a design-view capture.
func capturedCamera(s *app.Session) *assembly.CapturedCamera {
	c := s.Camera()
	return &assembly.CapturedCamera{Eye: c.Eye, Target: c.Target, Up: c.Up, FOV: c.FOV, Orthographic: c.Orthographic}
}

// SPDX-License-Identifier: GPL-2.0-only

package router

import (
	"encoding/json"

	"oblikovati.org/addin/modelaccess"
	"oblikovati.org/api/wire"
	"oblikovati.org/app"
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
	r.handlers[wire.MethodDesignRepsCapture] = designRepsCapture
	r.handlers[wire.MethodDesignRepsActivate] = designRepsActivate
	r.handlers[wire.MethodDesignRepsList] = designRepsList
	r.handlers[wire.MethodDesignRepsDelete] = designRepsDelete
	r.handlers[wire.MethodDesignRepsSetVisibility] = designRepsSetVisibility
	r.handlers[wire.MethodDesignRepsSetAppearance] = designRepsSetAppearance
	r.handlers[wire.MethodDesignRepsAddSection] = designRepsAddSection
}

func (r *Router) registerPositionalRepHandlers() {
	r.handlers[wire.MethodPositionalRepsCapture] = positionalRepsCapture
	r.handlers[wire.MethodPositionalRepsActivate] = positionalRepsActivate
	r.handlers[wire.MethodPositionalRepsList] = positionalRepsList
	r.handlers[wire.MethodPositionalRepsDelete] = positionalRepsDelete
	r.handlers[wire.MethodPositionalRepsSetOverride] = positionalRepsSetOverride
	r.handlers[wire.MethodPositionalRepsSetFlexible] = positionalRepsSetFlexible
}

func (r *Router) registerLODRepHandlers() {
	r.handlers[wire.MethodLODRepsCapture] = lodRepsCapture
	r.handlers[wire.MethodLODRepsActivate] = lodRepsActivate
	r.handlers[wire.MethodLODRepsList] = lodRepsList
	r.handlers[wire.MethodLODRepsDelete] = lodRepsDelete
	r.handlers[wire.MethodLODRepsSetSuppressed] = lodRepsSetSuppressed
}

func (r *Router) registerModelStateHandlers() {
	r.handlers[wire.MethodModelStatesCreate] = modelStatesCreate
	r.handlers[wire.MethodModelStatesActivate] = modelStatesActivate
	r.handlers[wire.MethodModelStatesList] = modelStatesList
	r.handlers[wire.MethodModelStatesDelete] = modelStatesDelete
}

// --- design-view ---

func designRepsCapture(s *app.Session, raw json.RawMessage) (json.RawMessage, error) {
	asm, in, err := repAssemblyAndName(s, raw)
	if err != nil {
		return nil, err
	}
	d := asm.Representations().CaptureDesignView(in.Name, capturedCamera(s))
	return json.Marshal(wire.DesignViewResult{Representation: designViewInfo(d)})
}

func designRepsActivate(s *app.Session, raw json.RawMessage) (json.RawMessage, error) {
	asm, id, err := repAssemblyAndID(s, raw)
	if err != nil {
		return nil, err
	}
	d, err := asm.Representations().ActivateDesignView(id)
	if err != nil {
		return nil, err
	}
	return json.Marshal(wire.DesignViewResult{Representation: designViewInfo(d)})
}

func designRepsList(s *app.Session, _ json.RawMessage) (json.RawMessage, error) {
	asm, err := modelaccess.ActiveAssembly(s)
	if err != nil {
		return nil, err
	}
	out := make([]wire.DesignViewInfo, 0)
	for _, d := range asm.Representations().AllDesignViews() {
		out = append(out, designViewInfo(d))
	}
	return json.Marshal(wire.DesignViewsResult{Representations: out})
}

func designRepsDelete(s *app.Session, raw json.RawMessage) (json.RawMessage, error) {
	asm, id, err := repAssemblyAndID(s, raw)
	if err != nil {
		return nil, err
	}
	asm.Representations().DeleteDesignView(id)
	return designRepsList(s, nil)
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

func positionalRepsCapture(s *app.Session, raw json.RawMessage) (json.RawMessage, error) {
	asm, in, err := repAssemblyAndName(s, raw)
	if err != nil {
		return nil, err
	}
	p := asm.Representations().CapturePositional(in.Name)
	return json.Marshal(wire.PositionalResult{Representation: positionalInfo(p)})
}

func positionalRepsActivate(s *app.Session, raw json.RawMessage) (json.RawMessage, error) {
	asm, id, err := repAssemblyAndID(s, raw)
	if err != nil {
		return nil, err
	}
	p, err := asm.Representations().ActivatePositional(id)
	if err != nil {
		return nil, err
	}
	return json.Marshal(wire.PositionalResult{Representation: positionalInfo(p)})
}

func positionalRepsList(s *app.Session, _ json.RawMessage) (json.RawMessage, error) {
	asm, err := modelaccess.ActiveAssembly(s)
	if err != nil {
		return nil, err
	}
	out := make([]wire.PositionalInfo, 0)
	for _, p := range asm.Representations().AllPositionals() {
		out = append(out, positionalInfo(p))
	}
	return json.Marshal(wire.PositionalsResult{Representations: out})
}

func positionalRepsDelete(s *app.Session, raw json.RawMessage) (json.RawMessage, error) {
	asm, id, err := repAssemblyAndID(s, raw)
	if err != nil {
		return nil, err
	}
	asm.Representations().DeletePositional(id)
	return positionalRepsList(s, nil)
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

func lodRepsCapture(s *app.Session, raw json.RawMessage) (json.RawMessage, error) {
	asm, in, err := repAssemblyAndName(s, raw)
	if err != nil {
		return nil, err
	}
	l := asm.Representations().CaptureLOD(in.Name)
	return json.Marshal(wire.LODResult{Representation: lodInfo(l)})
}

func lodRepsActivate(s *app.Session, raw json.RawMessage) (json.RawMessage, error) {
	asm, id, err := repAssemblyAndID(s, raw)
	if err != nil {
		return nil, err
	}
	l, err := asm.Representations().ActivateLOD(id)
	if err != nil {
		return nil, err
	}
	return json.Marshal(wire.LODResult{Representation: lodInfo(l)})
}

func lodRepsList(s *app.Session, _ json.RawMessage) (json.RawMessage, error) {
	asm, err := modelaccess.ActiveAssembly(s)
	if err != nil {
		return nil, err
	}
	out := make([]wire.LODInfo, 0)
	for _, l := range asm.Representations().AllLODs() {
		out = append(out, lodInfo(l))
	}
	return json.Marshal(wire.LODsResult{Representations: out})
}

func lodRepsDelete(s *app.Session, raw json.RawMessage) (json.RawMessage, error) {
	asm, id, err := repAssemblyAndID(s, raw)
	if err != nil {
		return nil, err
	}
	asm.Representations().DeleteLOD(id)
	return lodRepsList(s, nil)
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

func modelStatesCreate(s *app.Session, raw json.RawMessage) (json.RawMessage, error) {
	asm, err := modelaccess.ActiveAssembly(s)
	if err != nil {
		return nil, err
	}
	var in wire.CreateModelStateArgs
	if err := decode(raw, &in); err != nil {
		return nil, err
	}
	m := asm.Representations().CreateModelState(in.Name, in.DesignView, in.Positional, in.LevelOfDetail)
	return json.Marshal(wire.ModelStateResult{ModelState: modelStateInfo(m)})
}

func modelStatesActivate(s *app.Session, raw json.RawMessage) (json.RawMessage, error) {
	asm, id, err := repAssemblyAndID(s, raw)
	if err != nil {
		return nil, err
	}
	m, err := asm.Representations().ActivateModelState(id)
	if err != nil {
		return nil, err
	}
	return json.Marshal(wire.ModelStateResult{ModelState: modelStateInfo(m)})
}

func modelStatesList(s *app.Session, _ json.RawMessage) (json.RawMessage, error) {
	asm, err := modelaccess.ActiveAssembly(s)
	if err != nil {
		return nil, err
	}
	out := make([]wire.ModelStateInfo, 0)
	for _, m := range asm.Representations().AllModelStates() {
		out = append(out, modelStateInfo(m))
	}
	return json.Marshal(wire.ModelStatesResult{ModelStates: out})
}

func modelStatesDelete(s *app.Session, raw json.RawMessage) (json.RawMessage, error) {
	asm, id, err := repAssemblyAndID(s, raw)
	if err != nil {
		return nil, err
	}
	asm.Representations().DeleteModelState(id)
	return modelStatesList(s, nil)
}

// --- shared helpers ---

// repAssemblyAndName resolves the active assembly and decodes a capture's name argument.
func repAssemblyAndName(s *app.Session, raw json.RawMessage) (*compdef.AssemblyComponentDefinition, wire.CaptureRepArgs, error) {
	var in wire.CaptureRepArgs
	asm, err := modelaccess.ActiveAssembly(s)
	if err != nil {
		return nil, in, err
	}
	err = decode(raw, &in)
	return asm, in, err
}

// repAssemblyAndID resolves the active assembly and decodes a RepRef id argument.
func repAssemblyAndID(s *app.Session, raw json.RawMessage) (*compdef.AssemblyComponentDefinition, uint64, error) {
	var in wire.RepRef
	asm, err := modelaccess.ActiveAssembly(s)
	if err != nil {
		return nil, 0, err
	}
	if err := decode(raw, &in); err != nil {
		return nil, 0, err
	}
	return asm, in.ID, nil
}

// capturedCamera snapshots the session's current camera for a design-view capture.
func capturedCamera(s *app.Session) *assembly.CapturedCamera {
	c := s.Camera()
	return &assembly.CapturedCamera{Eye: c.Eye, Target: c.Target, Up: c.Up, FOV: c.FOV, Orthographic: c.Orthographic}
}

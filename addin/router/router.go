// SPDX-License-Identifier: GPL-2.0-only

// Package router is the host-side API: it dispatches the bridge's JSON method
// contract (commands.*, documents.*, parameters.*, model.*, features.*) to the live
// *app.Session. It is the single place that contract is wired to the model, and is
// pure Go (no cgo, no MCP/HTTP) so it is fully headless-testable. Handle runs on the
// session goroutine (via the Dispatcher), so handlers may touch the model directly.
//
// This is the same contract the future M16 gRPC api/ will serve; keeping it here and
// transport-agnostic lets the bridge retarget onto that layer with minimal churn.
package router

import (
	"encoding/json"
	"fmt"
	"sort"

	"github.com/Oblikovati/api/wire"

	"github.com/Oblikovati/oblikovati/addin/opregistry"
	"github.com/Oblikovati/oblikovati/app"
)

// handlerFunc handles one method: decode args, read/mutate the session, return JSON.
type handlerFunc func(s *app.Session, args json.RawMessage) (json.RawMessage, error)

// Router maps method names to handlers.
type Router struct {
	ops      *opregistry.Registry
	handlers map[string]handlerFunc
}

// New builds a router whose feature operations come from ops.
func New(ops *opregistry.Registry) *Router {
	r := &Router{ops: ops, handlers: map[string]handlerFunc{}}
	r.registerStandardHandlers()
	r.registerMaterialHandlers()
	return r
}

// registerStandardHandlers wires the command/document/parameter/model/sketch/feature/theme
// methods.
func (r *Router) registerStandardHandlers() {
	r.registerCommandHandlers()
	r.handlers[wire.MethodDocumentsList] = listDocuments
	r.handlers[wire.MethodDocumentsCreate] = createDocument
	r.handlers[wire.MethodDocumentsActivate] = activateDocument
	r.handlers[wire.MethodParametersList] = listParameters
	r.handlers[wire.MethodParametersGet] = getParameter
	r.handlers[wire.MethodParametersAdd] = addParameter
	r.handlers[wire.MethodParametersSet] = setParameter
	r.handlers[wire.MethodModelTree] = modelTree
	r.handlers[wire.MethodModelSelection] = modelSelection
	r.registerSketchHandlers()
	r.handlers[wire.MethodFeaturesList] = r.listFeatureKinds
	r.handlers[wire.MethodFeaturesAdd] = r.addFeature
	r.handlers[wire.MethodWorkPlanesList] = listWorkPlanes
	r.handlers[wire.MethodWorkPlanesCreate] = createWorkPlanes
	r.handlers[wire.MethodThemeActive] = themeActive
	r.handlers[wire.MethodThemeList] = themeList
}

// registerSketchHandlers wires the 2D-sketch methods: the spine + enumeration here, and
// the authoring (entity/constraint/dimension/edit/pattern) methods in the companion.
func (r *Router) registerSketchHandlers() {
	r.handlers[wire.MethodSketchCreate] = createSketch
	r.handlers[wire.MethodSketchRectangle] = sketchRectangle
	r.handlers[wire.MethodSketchList] = listSketches
	r.handlers[wire.MethodSketchGet] = getSketch
	r.handlers[wire.MethodSketchEdit] = editSketch
	r.handlers[wire.MethodSketchExitEdit] = exitEditSketch
	r.handlers[wire.MethodSketchSolve] = solveSketch
	r.handlers[wire.MethodSketchDelete] = deleteSketch
	r.handlers[wire.MethodSketchEntities] = enumerateEntities
	r.handlers[wire.MethodSketchConstraints] = enumerateConstraints
	r.handlers[wire.MethodSketchDimensions] = enumerateDimensions
	r.handlers[wire.MethodSketchConstraintStatus] = constraintStatus
	r.handlers[wire.MethodSketchProfiles] = sketchProfiles
	r.registerSketchAuthoringHandlers()
	r.registerSketch3DHandlers()
}

// registerSketch3DHandlers wires the 3D-sketch (Sketch3D) methods: the spine, enumeration,
// and property edits (M22-F01). The 3D authoring methods (addEntity/addConstraint/
// addDimension) are wired by their features (M22 F02+).
func (r *Router) registerSketch3DHandlers() {
	r.handlers[wire.MethodSketch3DCreate] = createSketch3D
	r.handlers[wire.MethodSketch3DList] = listSketches3D
	r.handlers[wire.MethodSketch3DGet] = getSketch3D
	r.handlers[wire.MethodSketch3DEdit] = editSketch3D
	r.handlers[wire.MethodSketch3DExitEdit] = exitEditSketch3D
	r.handlers[wire.MethodSketch3DSolve] = solveSketch3D
	r.handlers[wire.MethodSketch3DDelete] = deleteSketch3D
	r.handlers[wire.MethodSketch3DSetProperty] = setSketch3DProperty
	r.handlers[wire.MethodSketch3DEntities] = enumerateEntities3D
	r.handlers[wire.MethodSketch3DConstraints] = enumerateConstraints3D
	r.handlers[wire.MethodSketch3DDimensions] = enumerateDimensions3D
	r.handlers[wire.MethodSketch3DConstraintStatus] = constraintStatus3D
	r.handlers[wire.MethodSketch3DAddEntity] = addSketch3DEntity
}

// registerSketchAuthoringHandlers wires the sketch mutation methods: property edits,
// entity/constraint/dimension creation, and the edit/pattern operations.
func (r *Router) registerSketchAuthoringHandlers() {
	r.handlers[wire.MethodSketchSetProperty] = setSketchProperty
	r.handlers[wire.MethodSketchAddEntity] = addSketchEntity
	r.handlers[wire.MethodSketchAddConstraint] = addConstraint
	r.handlers[wire.MethodSketchDeleteConstraint] = deleteConstraint
	r.handlers[wire.MethodSketchAddDimension] = addDimension
	r.handlers[wire.MethodSketchDriveDimension] = driveDimension
	r.handlers[wire.MethodSketchTransform] = transformSketch
	r.handlers[wire.MethodSketchAddPattern] = addSketchPattern
	r.handlers[wire.MethodSketchOffset] = offsetSketchEntity
	r.handlers[wire.MethodSketchAddImage] = addSketchImage
	r.handlers[wire.MethodSketchAddFillRegion] = addFillRegion
	r.handlers[wire.MethodSketchAddText] = addText
	r.handlers[wire.MethodSketchAutoDimension] = autoDimensionSketch
	r.handlers[wire.MethodSketchProject] = projectGeometry
}

// registerCommandHandlers wires the command and ribbon methods — the add-in UI surface
// (list/execute/create commands and enumerate the active ribbon, RibbonUI core/07).
func (r *Router) registerCommandHandlers() {
	r.handlers[wire.MethodCommandsList] = listCommands
	r.handlers[wire.MethodCommandsExecute] = executeCommand
	r.handlers[wire.MethodCommandsCreate] = createCommand
	r.handlers[wire.MethodRibbonList] = ribbonList
}

// registerMaterialHandlers wires the appearance/material/assignment/physical-properties
// methods (M19 / ADR-0022).
func (r *Router) registerMaterialHandlers() {
	r.handlers[wire.MethodAppearancesList] = listAppearances
	r.handlers[wire.MethodAppearancesGet] = getAppearance
	r.handlers[wire.MethodAppearancesCreate] = createAppearance
	r.handlers[wire.MethodAppearancesUpdate] = updateAppearance
	r.handlers[wire.MethodMaterialsList] = listMaterials
	r.handlers[wire.MethodMaterialsGet] = getMaterial
	r.handlers[wire.MethodMaterialsCreate] = createMaterial
	r.handlers[wire.MethodMaterialsUpdate] = updateMaterial
	r.handlers[wire.MethodModelAssignMaterial] = assignMaterial
	r.handlers[wire.MethodModelAssignAppearance] = assignAppearance
	r.handlers[wire.MethodModelPhysicalProperties] = physicalProperties
}

// Handle dispatches method with its JSON args (empty args become {}), returning the
// JSON result, or an error for an unknown method or a failed handler.
func (r *Router) Handle(s *app.Session, method string, req []byte) ([]byte, error) {
	h, ok := r.handlers[method]
	if !ok {
		return nil, fmt.Errorf("router: unknown method %q", method)
	}
	args := json.RawMessage(req)
	if len(req) == 0 {
		args = json.RawMessage("{}")
	}
	return h(s, args)
}

// Methods returns the supported method names, sorted — used by self-description.
func (r *Router) Methods() []string {
	out := make([]string, 0, len(r.handlers))
	for m := range r.handlers {
		out = append(out, m)
	}
	sort.Strings(out)
	return out
}

func ok() (json.RawMessage, error) { return json.Marshal(wire.OKResult{OK: true}) }

// decode unmarshals args into v, wrapping the error for a clearer method failure.
func decode(args json.RawMessage, v any) error {
	if err := json.Unmarshal(args, v); err != nil {
		return fmt.Errorf("invalid arguments: %w", err)
	}
	return nil
}

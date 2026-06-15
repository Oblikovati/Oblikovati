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
	"runtime/debug"
	"sort"
	"strings"
	"time"

	"oblikovati.org/addin/opregistry"
	"oblikovati.org/addin/trace"
	"oblikovati.org/api/wire"
	"oblikovati.org/app"
)

// handlerFunc handles one method: decode args, read/mutate the session, return JSON.
type handlerFunc func(s *app.Session, args json.RawMessage) (json.RawMessage, error)

// Router maps method names to handlers.
type Router struct {
	ops      *opregistry.Registry
	handlers map[string]handlerFunc
	trace    *trace.Buffer
}

// New builds a router whose feature operations come from ops.
func New(ops *opregistry.Registry) *Router {
	r := &Router{ops: ops, handlers: map[string]handlerFunc{}, trace: trace.NewBuffer(0)}
	r.registerStandardHandlers()
	r.registerTransactionHandlers()
	r.registerMaterialHandlers()
	r.registerLightingHandlers()
	r.registerGraphicsHandlers()
	r.registerExchangeHandlers()
	r.registerFontHandlers()
	r.registerAddInHandlers()
	r.registerApplicationHandlers()
	r.registerUISurfaceHandlers()
	r.registerOptionHandlers()
	r.registerKeymapHandlers()
	r.registerMessagingHandlers()
	r.registerMiniToolbarHandlers()
	r.registerDialogHandlers()
	r.registerWindowHandlers()
	r.registerUIShellHandlers()
	r.registerTriadHandlers()
	r.registerHelpHandlers()
	r.registerAssemblyDeriveHandlers()
	r.registerAssemblyFeatureHandlers()
	r.registerAssemblyOccurrenceHandlers()
	r.registerAssemblyConstraintHandlers()
	r.registerAssemblyJointHandlers()
	r.registerAssemblyDriveHandlers()
	r.registerAssemblyReplicationHandlers()
	r.registerAssemblyBOMHandlers()
	r.registerDocumentPropertyHandlers()
	r.handlers[wire.MethodLogsTail] = r.logsTail
	r.handlers[wire.MethodScriptRun] = r.scriptsRun
	return r
}

// Trace returns the router's operation-trace buffer, so the host can install its slog handler
// (slog.SetDefault) and have kernel logs land in the same stream the router fills.
func (r *Router) Trace() *trace.Buffer { return r.trace }

// registerStandardHandlers wires the command/document/parameter/model/sketch/feature/theme
// methods.
func (r *Router) registerStandardHandlers() {
	r.registerCommandHandlers()
	r.handlers[wire.MethodDocumentsList] = listDocuments
	r.handlers[wire.MethodDocumentsCreate] = createDocument
	r.handlers[wire.MethodDocumentsActivate] = activateDocument
	r.handlers[wire.MethodDocumentsClose] = closeDocument
	r.handlers[wire.MethodDocumentsCloseAll] = closeAllDocuments
	r.handlers[wire.MethodDocumentsRegisterSubType] = registerDocumentSubType
	r.handlers[wire.MethodDocumentsListSubTypes] = listDocumentSubTypes
	r.registerFileHandlers()
	r.handlers[wire.MethodParametersList] = listParameters
	r.handlers[wire.MethodParametersGet] = getParameter
	r.handlers[wire.MethodParametersAdd] = addParameter
	r.handlers[wire.MethodParametersSet] = setParameter
	r.registerParameterDetailHandlers()
	r.handlers[wire.MethodModelTree] = modelTree
	r.handlers[wire.MethodModelSelection] = modelSelection
	r.handlers[wire.MethodModelReferenceKeys] = referenceKeys
	r.handlers[wire.MethodThreadsTableQuery] = threadsTableQuery
	r.handlers[wire.MethodThreadsResolve] = threadsResolve
	r.handlers[wire.MethodFreeformSetLevel] = freeformSetLevel
	r.handlers[wire.MethodFreeformMoveVertices] = freeformMoveVertices
	r.handlers[wire.MethodFreeformCreaseEdges] = freeformCreaseEdges
	r.registerSketchHandlers()
	r.registerFeatureHandlers()
	r.handlers[wire.MethodWorkPlanesList] = listWorkPlanes
	r.handlers[wire.MethodWorkPlanesCreate] = createWorkPlanes
	r.handlers[wire.MethodWorkPlanesRedefine] = redefineWorkPlane
	r.handlers[wire.MethodWorkPointsCreate] = createWorkPoint

	r.handlers[wire.MethodWorkSurfacesList] = listWorkSurfaces
	r.handlers[wire.MethodWorkSurfacesGet] = getWorkSurface
	r.handlers[wire.MethodWorkSurfacesSetVisible] = setWorkSurfaceVisible
	r.handlers[wire.MethodWorkSurfacesRename] = renameWorkSurface
	r.handlers[wire.MethodThemeActive] = themeActive
	r.handlers[wire.MethodThemeList] = themeList
	r.handlers[wire.MethodViewGetDisplayMode] = getDisplayMode
	r.handlers[wire.MethodViewSetDisplayMode] = setDisplayMode
	r.handlers[wire.MethodViewListDisplayModes] = listDisplayModes
	r.handlers[wire.MethodViewGetCamera] = getCamera
	r.handlers[wire.MethodViewSetCamera] = setCamera
	r.handlers[wire.MethodViewportCapture] = captureViewport
	r.handlers[wire.MethodViewportCaptureWindow] = captureWindow
	r.handlers[wire.MethodViewportSetNormalDebug] = setNormalDebug
	r.handlers[wire.MethodViewportSetMeshColors] = setMeshColors
	r.handlers[wire.MethodInteractionState] = interactionState
	r.handlers[wire.MethodInteractionSetNotice] = interactionSetNotice
	r.handlers[wire.MethodViewsList] = listViews
	r.handlers[wire.MethodViewsAdd] = addView
	r.handlers[wire.MethodViewsActivate] = activateView
	r.handlers[wire.MethodViewsClose] = closeView
	r.handlers[wire.MethodViewsRename] = renameView
	r.handlers[wire.MethodViewsGetLayout] = getLayout
	r.handlers[wire.MethodViewsSetLayout] = setLayout
}

// registerFileHandlers wires the file surface (M03-F07, #608): identity, the
// persisted file-to-file reference records, and reference repair.
func (r *Router) registerFileHandlers() {
	r.handlers[wire.MethodFilesGet] = getFile
	r.handlers[wire.MethodFilesListReferences] = listFileReferences
	r.handlers[wire.MethodFilesReplaceReference] = replaceFileReference
	r.handlers[wire.MethodDocumentsListFileReferences] = listDocumentFileReferences
	r.handlers[wire.MethodDocumentsListAttachments] = listAttachments
	r.handlers[wire.MethodDocumentsAddAttachment] = addAttachment
	r.handlers[wire.MethodDocumentsRemoveAttachment] = removeAttachment
	r.handlers[wire.MethodDocumentsListInterests] = listDocumentInterests
	r.handlers[wire.MethodDocumentsAddInterest] = addDocumentInterest
	r.handlers[wire.MethodDocumentsRemoveInterest] = removeDocumentInterest
	r.handlers[wire.MethodDocumentsHasInterest] = hasDocumentInterest
	r.handlers[wire.MethodDocumentsOpen] = openDocument
	r.handlers[wire.MethodDocumentsSave] = saveDocument
	r.handlers[wire.MethodDocumentsSaveAs] = saveDocumentAs
	r.handlers[wire.MethodDocumentsSaveCopyAs] = saveDocumentCopyAs
	r.handlers[wire.MethodDocumentsBatchSave] = batchSave
}

// registerTransactionHandlers wires the undo/redo control methods — navigate and query
// the active document's transaction-event stream (transaction.undo/redo/state), plus the
// bounded transaction.begin/end/abort that make a batch one undo step or discard it.
func (r *Router) registerTransactionHandlers() {
	r.handlers[wire.MethodTransactionUndo] = undoTransaction
	r.handlers[wire.MethodTransactionRedo] = redoTransaction
	r.handlers[wire.MethodTransactionState] = transactionState
	r.handlers[wire.MethodTransactionBegin] = beginTransaction
	r.handlers[wire.MethodTransactionEnd] = endTransaction
	r.handlers[wire.MethodTransactionAbort] = abortTransaction
}

// registerParameterDetailHandlers wires the member-level parameter surface —
// detail reads, presentation/tolerance/value-list mutations, dependency queries
// and delete (M02-F08, Oblikovati#607).
func (r *Router) registerParameterDetailHandlers() {
	r.handlers[wire.MethodParametersGetDetail] = getParameterDetail
	r.handlers[wire.MethodParametersUpdate] = updateParameter
	r.handlers[wire.MethodParametersSetTolerance] = setParameterTolerance
	r.handlers[wire.MethodParametersSetExpressionList] = setParameterExpressionList
	r.handlers[wire.MethodParametersDelete] = deleteParameter
	r.handlers[wire.MethodParametersDrivenBy] = parameterDrivenBy
	r.handlers[wire.MethodParametersDependents] = parameterDependents
	r.registerParameterGroupHandlers()
	r.registerParameterSettingsHandlers()
	r.registerDerivedTableHandlers()
}

// registerParameterGroupHandlers wires the custom parameter groups (M02-F05,
// Oblikovati#604).
func (r *Router) registerParameterGroupHandlers() {
	r.handlers[wire.MethodParametersGroupsList] = listParameterGroups
	r.handlers[wire.MethodParametersGroupsAdd] = addParameterGroup
	r.handlers[wire.MethodParametersGroupsDelete] = deleteParameterGroup
	r.handlers[wire.MethodParametersGroupsSetDisplayName] = setParameterGroupDisplayName
	r.handlers[wire.MethodParametersGroupsAddMember] = addParameterGroupMember
	r.handlers[wire.MethodParametersGroupsRemoveMember] = removeParameterGroupMember
}

// registerParameterSettingsHandlers wires the document-level parameter
// settings, the tolerance sweep, and the XML exchange (M02-F07, Oblikovati#606).
func (r *Router) registerParameterSettingsHandlers() {
	r.handlers[wire.MethodParametersGetSettings] = getParameterSettings
	r.handlers[wire.MethodParametersSetSettings] = setParameterSettings
	r.handlers[wire.MethodParametersSetAllModelValueType] = sweepParameterModelValues
	r.handlers[wire.MethodParametersExport] = exportParameters
	r.handlers[wire.MethodParametersImport] = importParameters
}

// registerDerivedTableHandlers wires the derived parameter tables (M02-F06,
// Oblikovati#605).
func (r *Router) registerDerivedTableHandlers() {
	r.handlers[wire.MethodParametersDerivedTablesList] = listDerivedTables
	r.handlers[wire.MethodParametersDerivedTablesAdd] = addDerivedTable
	r.handlers[wire.MethodParametersDerivedTablesSetLinked] = setDerivedTableLinked
	r.handlers[wire.MethodParametersDerivedTablesDelete] = deleteDerivedTable
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
	r.handlers[wire.MethodSketchRegionProperties] = sketchRegionProperties
	r.handlers[wire.MethodSketch3DRegionProperties] = sketch3DRegionProperties
	r.handlers[wire.MethodSketchBlockDefinitionCreate] = createBlockDefinition
	r.handlers[wire.MethodSketchBlockDefinitionList] = listBlockDefinitions
	r.handlers[wire.MethodSketchBlockDefinitionDelete] = deleteBlockDefinition
	r.handlers[wire.MethodSketchAddBlockInstance] = addBlockInstance
	r.handlers[wire.MethodSketchListBlockInstances] = listBlockInstances
	r.handlers[wire.MethodSketchSetSplineHandle] = setSplineHandle
	r.handlers[wire.MethodSketch3DSetSplineHandle] = setSplineHandle3D
	r.handlers[wire.MethodSketch3DEditHelix] = sketch3DEditHelix
	r.handlers[wire.MethodSketchSetInferenceOptions] = setInferenceOptions
	r.handlers[wire.MethodSketchGetInferenceOptions] = getInferenceOptions
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
	r.registerSketch3DAuthoringHandlers()
}

// registerSketch3DAuthoringHandlers wires the 3D-sketch mutation/query methods: entity/
// constraint/dimension creation, profiles/paths, edit transform, and include.
func (r *Router) registerSketch3DAuthoringHandlers() {
	r.handlers[wire.MethodSketch3DAddEntity] = addSketch3DEntity
	r.handlers[wire.MethodSketch3DAddConstraint] = addSketch3DConstraint
	r.handlers[wire.MethodSketch3DDeleteConstraint] = deleteSketch3DConstraint
	r.handlers[wire.MethodSketch3DAddDimension] = addSketch3DDimension
	r.handlers[wire.MethodSketch3DDriveDimension] = driveSketch3DDimension
	r.handlers[wire.MethodSketch3DProfiles] = sketch3DProfiles
	r.handlers[wire.MethodSketch3DPaths] = sketch3DPaths
	r.handlers[wire.MethodSketch3DTransform] = transformSketch3D
	r.handlers[wire.MethodSketch3DInclude] = includeSketch3D
	r.handlers[wire.MethodSketch3DIncludeSketch] = includeSketch2DInto3D
	r.handlers[wire.MethodSketch3DAddSurfaceCurve] = addSketch3DSurfaceCurve
}

// registerSketchAuthoringHandlers wires the sketch mutation methods: property edits,
// entity/constraint/dimension creation, and the edit/pattern operations.
func (r *Router) registerSketchAuthoringHandlers() {
	r.handlers[wire.MethodSketchSetProperty] = setSketchProperty
	r.handlers[wire.MethodSketchGetCustomLineType] = getSketchCustomLineType
	r.handlers[wire.MethodSketchSetCustomLineType] = setSketchCustomLineType
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
	r.handlers[wire.MethodSketchEditText] = editText
	r.handlers[wire.MethodSketchGetText] = getText
	r.handlers[wire.MethodSketchAutoDimension] = autoDimensionSketch
	r.handlers[wire.MethodSketchProject] = projectGeometry
}

// registerCommandHandlers wires the command and ribbon methods — the add-in UI surface
// (list/execute/create commands and enumerate the active ribbon, RibbonUI core/07).
func (r *Router) registerCommandHandlers() {
	r.handlers[wire.MethodCommandsList] = listCommands
	r.handlers[wire.MethodCommandsExecute] = executeCommand
	r.handlers[wire.MethodCommandsCreate] = createCommand
	r.handlers[wire.MethodCommandsSetState] = setCommandState
	r.handlers[wire.MethodCommandLineSubmit] = submitCommandLine
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

	// Body topology, queries and facet sets (M07 #293/#629/#630).
	r.handlers[wire.MethodBodyList] = bodyList
	r.handlers[wire.MethodBodyShells] = bodyShells
	r.handlers[wire.MethodBodyWires] = bodyWires
	r.handlers[wire.MethodWireOffsetPlanar] = wireOffsetPlanar
	r.handlers[wire.MethodBodyLocateUsingPoint] = bodyLocateUsingPoint
	r.handlers[wire.MethodBodyFindUsingRay] = bodyFindUsingRay
	r.handlers[wire.MethodBodyIsPointInside] = bodyIsPointInside
	r.handlers[wire.MethodBodyConvexityEdges] = bodyConvexityEdges
	r.handlers[wire.MethodBodyValidate] = bodyValidate
	r.handlers[wire.MethodBodyRangeBox] = bodyRangeBox
	r.handlers[wire.MethodBodyBindTransientKey] = bodyBindTransientKey
	r.handlers[wire.MethodBodyCalculateFacets] = bodyCalculateFacets
	r.handlers[wire.MethodBodyExistingFacets] = bodyExistingFacets
	r.handlers[wire.MethodBodyFacetTolerances] = bodyFacetTolerances
	r.handlers[wire.MethodBodyCalculateStrokes] = bodyCalculateStrokes
	r.handlers[wire.MethodBodyExistingStrokes] = bodyExistingStrokes
	r.handlers[wire.MethodBodyStrokeTolerances] = bodyStrokeTolerances
	r.handlers[wire.MethodFaceCalculateFacets] = faceCalculateFacets
	r.handlers[wire.MethodFaceCalculateStrokes] = faceCalculateStrokes

	// The transient B-rep factory (M07 #628) — session-scoped, no document
	// mutation, so none of these are mutating methods.
	r.handlers[wire.MethodBrepCreatePrimitive] = brepCreatePrimitive
	r.handlers[wire.MethodBrepBoolean] = brepBoolean
	r.handlers[wire.MethodBrepTransform] = brepTransform
	r.handlers[wire.MethodBrepCopy] = brepCopy
	r.handlers[wire.MethodBrepSectionWithPlane] = brepSectionWithPlane
	r.handlers[wire.MethodBrepDeleteFaces] = brepDeleteFaces
	r.handlers[wire.MethodBrepSilhouette] = brepSilhouette
	r.handlers[wire.MethodBrepRuledSurface] = brepRuledSurface
	r.handlers[wire.MethodBrepImprint] = brepImprint
	r.handlers[wire.MethodBrepIdenticalBodies] = brepIdenticalBodies
	r.handlers[wire.MethodBrepCreateFromDefinition] = brepCreateFromDefinition
	r.handlers[wire.MethodBrepDescribe] = brepDescribe
	r.handlers[wire.MethodBrepList] = brepList
	r.handlers[wire.MethodBrepDelete] = brepDelete
}

// registerLightingHandlers wires the lighting-style, light, environment, and shadow methods
// (M16/F03 PBI-155, ADR-0026).
func (r *Router) registerLightingHandlers() {
	r.handlers[wire.MethodLightingGetStyle] = getLightingStyle
	r.handlers[wire.MethodLightingSetStyle] = setLightingStyle
	r.handlers[wire.MethodLightingListStyles] = listLightingStyles
	r.handlers[wire.MethodLightingListLights] = listLights
	r.handlers[wire.MethodLightingAddLight] = addLight
	r.handlers[wire.MethodLightingSetLight] = setLight
	r.handlers[wire.MethodViewGetShadows] = getShadows
	r.handlers[wire.MethodViewSetShadows] = setShadows
	r.handlers[wire.MethodEnvironmentGet] = getEnvironment
	r.handlers[wire.MethodEnvironmentSet] = setEnvironment
	r.handlers[wire.MethodEnvironmentListPresets] = listEnvironmentPresets
	r.handlers[wire.MethodEnvironmentLoadImage] = loadEnvironmentImage
}

// registerGraphicsHandlers wires the client/interaction graphics methods — the add-in
// overlay surface for drawing meshes, heatmaps, lines, markers and labels (M05-F05).
func (r *Router) registerGraphicsHandlers() {
	r.handlers[wire.MethodClientGraphicsSet] = setClientGraphics
	r.handlers[wire.MethodClientGraphicsList] = listClientGraphics
	r.handlers[wire.MethodClientGraphicsDelete] = deleteClientGraphics
	r.handlers[wire.MethodClientGraphicsSetVisible] = setClientGraphicsVisible
	r.handlers[wire.MethodInteractionGraphicsUpdate] = updateInteractionGraphics
	r.handlers[wire.MethodInteractionGraphicsClear] = clearInteractionGraphics
}

// Handle dispatches method with its JSON args (empty args become {}), returning the
// JSON result, or an error for an unknown method or a failed handler.
func (r *Router) Handle(s *app.Session, method string, req []byte) (resp []byte, err error) {
	h, ok := r.handlers[method]
	if !ok {
		return nil, fmt.Errorf("router: unknown method %q", method)
	}
	args := json.RawMessage(req)
	if len(req) == 0 {
		args = json.RawMessage("{}")
	}
	// Recover any handler panic into a detailed error (method + value + stack) so a kernel bug
	// hit by a driver is reported and traced, not fatal. logs.tail is not self-traced (a poll
	// must not flood the very buffer it reads). See Workstream A/B of the diagnostics plan.
	start := time.Now()
	defer func() {
		if rec := recover(); rec != nil {
			stack := string(debug.Stack())
			err = fmt.Errorf("%s: panic: %v", method, rec)
			resp = nil
			r.record(method, time.Since(start), false, "", err.Error(), stack)
		}
	}()
	out, herr := h(s, args)
	if herr != nil {
		herr = methodError(method, herr)
		r.record(method, time.Since(start), false, herr.Error(), "", "")
		return nil, herr
	}
	r.record(method, time.Since(start), true, "", "", "")
	// A document-mutating method that succeeded is a committed edit: emit it as the wire
	// request that produced it, so a collaboration add-in can replicate it (ADR-0004).
	// First cut: only router-path edits; local UI edits are not yet captured.
	if mutatingMethods[method] {
		s.EmitEditCommitted(method, req)
		// The edit may have moved values other documents derive from
		// (M02-F06); recordEdit covers session edits, this covers the wire.
		s.ResyncDerivedFromActiveDocument()
	}
	return out, nil
}

// mutatingMethods is the set of router methods that commit a document mutation worth
// replicating to collaboration peers as an edit.committed event. It is a curated
// allowlist: a method missing here simply is not broadcast (a known first-cut gap,
// failing safe), whereas a read-only method must never appear (it would broadcast noise).
var mutatingMethods = map[string]bool{
	wire.MethodDocumentsCreate:                     true,
	wire.MethodDocumentsImport:                     true,
	wire.MethodParametersAdd:                       true,
	wire.MethodParametersSet:                       true,
	wire.MethodParametersUpdate:                    true,
	wire.MethodParametersSetTolerance:              true,
	wire.MethodParametersSetExpressionList:         true,
	wire.MethodParametersDelete:                    true,
	wire.MethodParametersGroupsAdd:                 true,
	wire.MethodParametersGroupsDelete:              true,
	wire.MethodParametersGroupsSetDisplayName:      true,
	wire.MethodParametersGroupsAddMember:           true,
	wire.MethodParametersGroupsRemoveMember:        true,
	wire.MethodParametersSetSettings:               true,
	wire.MethodParametersSetAllModelValueType:      true,
	wire.MethodParametersImport:                    true,
	wire.MethodParametersDerivedTablesAdd:          true,
	wire.MethodParametersDerivedTablesSetLinked:    true,
	wire.MethodParametersDerivedTablesDelete:       true,
	wire.MethodFeaturesAdd:                         true,
	wire.MethodFeaturesEdit:                        true,
	wire.MethodFeaturesDelete:                      true,
	wire.MethodFeaturesRename:                      true,
	wire.MethodFeaturesSetSuppressed:               true,
	wire.MethodFeaturesReorder:                     true,
	wire.MethodFreeformSetLevel:                    true,
	wire.MethodFreeformMoveVertices:                true,
	wire.MethodFreeformCreaseEdges:                 true,
	wire.MethodWorkPlanesCreate:                    true,
	wire.MethodWorkPlanesRedefine:                  true,
	wire.MethodWorkPointsCreate:                    true,
	wire.MethodWorkSurfacesSetVisible:              true,
	wire.MethodWorkSurfacesRename:                  true,
	wire.MethodAssemblyDeriveCreate:                true,
	wire.MethodAssemblyShrinkwrapCreate:            true,
	wire.MethodAssemblyDeriveBreakLink:             true,
	wire.MethodAssemblyFeaturesAdd:                 true,
	wire.MethodAssemblyFeaturesAddProxyCut:         true,
	wire.MethodAssemblyFeaturesAddHole:             true,
	wire.MethodAssemblyFeaturesAddExtrude:          true,
	wire.MethodAssemblyFeaturesAddSweep:            true,
	wire.MethodAssemblyFeaturesAddChamfer:          true,
	wire.MethodAssemblyFeaturesAddFillet:           true,
	wire.MethodAssemblyFeaturesAddMoveFace:         true,
	wire.MethodAssemblyFeaturesEdit:                true,
	wire.MethodAssemblyFeaturesSetParticipants:     true,
	wire.MethodAssemblyFeaturesSetParticipantPaths: true,
	wire.MethodAssemblyFeaturesSetSuppressed:       true,
	wire.MethodAssemblySetEndOfFeatures:            true,
	wire.MethodModelAssignMaterial:                 true,
	wire.MethodModelAssignAppearance:               true,
	wire.MethodSketchCreate:                        true,
	wire.MethodSketchRectangle:                     true,
	wire.MethodSketchDelete:                        true,
	wire.MethodSketchEdit:                          true,
	wire.MethodSketchExitEdit:                      true,
	wire.MethodSketchSolve:                         true,
	wire.MethodSketchAddEntity:                     true,
	wire.MethodSketchSetSplineHandle:               true,
	wire.MethodSketchBlockDefinitionCreate:         true,
	wire.MethodSketchBlockDefinitionDelete:         true,
	wire.MethodSketchAddBlockInstance:              true,
	wire.MethodSketch3DSetSplineHandle:             true,
	wire.MethodSketch3DEditHelix:                   true,
	wire.MethodSketchSetInferenceOptions:           true,
	wire.MethodSketchAddConstraint:                 true,
	wire.MethodSketchDeleteConstraint:              true,
	wire.MethodSketchAddDimension:                  true,
	wire.MethodSketchDriveDimension:                true,
	wire.MethodSketchSetProperty:                   true,
	wire.MethodSketchSetCustomLineType:             true,
	wire.MethodSketchTransform:                     true,
	wire.MethodSketchAddPattern:                    true,
	wire.MethodSketchOffset:                        true,
	wire.MethodSketchProject:                       true,
	wire.MethodTransactionUndo:                     true,
	wire.MethodTransactionRedo:                     true,
	wire.MethodTransactionEnd:                      true,
	wire.MethodTransactionAbort:                    true,
	wire.MethodFilesReplaceReference:               true,
	wire.MethodDocumentsAddAttachment:              true,
	wire.MethodDocumentsRemoveAttachment:           true,
	wire.MethodDocumentsAddInterest:                true,
	wire.MethodDocumentsRemoveInterest:             true,
	wire.MethodDocumentsOpen:                       true,
}

// record appends an operation entry to the trace, except for logs.tail itself (so polling the
// trace does not append to it).
func (r *Router) record(method string, dur time.Duration, ok bool, errMsg, panicMsg, stack string) {
	if method == wire.MethodLogsTail {
		return
	}
	r.trace.RecordOp(method, dur, ok, errMsg, panicMsg, stack)
}

// methodError prefixes err with the method name when the handler did not already (so every
// failure names the operation that produced it).
func methodError(method string, err error) error {
	if strings.HasPrefix(err.Error(), method) {
		return err
	}
	return fmt.Errorf("%s: %w", method, err)
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

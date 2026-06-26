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

// MethodHandler is a registered wire method. Every handler implements it; a handler that ALSO implements
// [MutatingMethod] is treated as a document edit. Read-only is the absence of that second interface, so
// there is no flag or side table to drift from the handler set.
type MethodHandler interface {
	Handle(s *app.Session, args json.RawMessage) (json.RawMessage, error)
}

// MutatingMethod is the contract a document-editing handler implements so the central seam records one
// undo step (UndoLabel) after it succeeds and broadcasts edit.committed for collaboration replication
// (ADR-0004). Being undoable is therefore a property of the handler's TYPE: a handler that does not
// implement this interface is read-only BY CONSTRUCTION, and one that does cannot exist without declaring
// its UndoLabel — so a mutating method can never silently become non-undoable or non-replicated, the
// one-directional-table drift this replaces (#1426). An empty UndoLabel means "broadcast but record no
// step" (the transaction-control cursor methods and metadata-only edits the recipe does not capture).
type MutatingMethod interface {
	MethodHandler
	UndoLabel() string
}

// readOnlyFunc adapts a query func to [MethodHandler] only — it deliberately does NOT implement
// [MutatingMethod], so the router never records or replicates it.
type readOnlyFunc handlerFunc

func (f readOnlyFunc) Handle(s *app.Session, args json.RawMessage) (json.RawMessage, error) {
	return f(s, args)
}

// mutatingFunc adapts a func plus its undo label to [MutatingMethod], so a document-editing handler
// carries its recording contract in its type.
type mutatingFunc struct {
	fn    handlerFunc
	label string
}

func (m mutatingFunc) Handle(s *app.Session, args json.RawMessage) (json.RawMessage, error) {
	return m.fn(s, args)
}

func (m mutatingFunc) UndoLabel() string { return m.label }

// Compile-time proof that the two adapters sit on the right side of the interface split: a read-only
// handler must NOT satisfy MutatingMethod, a mutating one must.
var (
	_ MethodHandler  = readOnlyFunc(nil)
	_ MutatingMethod = mutatingFunc{}
	_ MethodHandler  = mutatingFunc{}
)

// Router maps method names to handlers.
type Router struct {
	ops      *opregistry.Registry
	handlers map[string]MethodHandler
	trace    *trace.Buffer
}

// readOnly registers a query handler: not a [MutatingMethod], so the router neither records an undo step
// nor replicates it. This is the default registration pattern; a handler that edits the document must use
// [Router.mutating] instead.
func (r *Router) readOnly(method string, fn handlerFunc) {
	r.set(method, readOnlyFunc(fn))
}

// mutating registers a document-editing handler as a [MutatingMethod] under the one pattern every such
// method follows: the router records a single undo step labelled label after it succeeds and broadcasts
// edit.committed so a collaboration add-in replays it (ADR-0004). An empty label means "broadcast but
// record no step" — the transaction-control methods (which move the undo cursor themselves) and
// metadata-only edits the parametric recipe does not capture. Because the label rides in the handler's
// type, a mutating method cannot drift out of the classification (#1426).
func (r *Router) mutating(method, label string, fn handlerFunc) {
	r.set(method, mutatingFunc{fn: fn, label: label})
}

// set records one method's handler, panicking on a duplicate registration (a copy-paste bug that would
// otherwise silently shadow a handler). Both readOnly and mutating route through it.
func (r *Router) set(method string, h MethodHandler) {
	if _, dup := r.handlers[method]; dup {
		panic(fmt.Sprintf("router: duplicate handler registration for %q", method))
	}
	r.handlers[method] = h
}

// New builds a router whose feature operations come from ops.
func New(ops *opregistry.Registry) *Router {
	r := &Router{ops: ops, handlers: map[string]MethodHandler{}, trace: trace.NewBuffer(0)}
	r.registerStandardHandlers()
	r.registerTransactionHandlers()
	r.registerMaterialHandlers()
	r.registerLightingHandlers()
	r.registerGraphicsHandlers()
	r.registerExchangeHandlers()
	r.registerPointCloudHandlers()
	r.registerFontHandlers()
	r.registerAddInHandlers()
	r.registerApplicationHandlers()
	r.registerUISurfaceHandlers()
	r.registerOptionHandlers()
	r.registerColorSchemeHandlers()
	r.registerNamedViewHandlers()
	r.registerStyleHandlers()
	r.registerDisplayHandlers()
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
	r.registerRepresentationHandlers()
	r.registerContactHandlers()
	r.registerAssemblyReplicationHandlers()
	r.registerAssemblyBOMHandlers()
	r.registerDocumentPropertyHandlers()
	r.registerDocumentSettingsHandlers()
	r.registerEndOfPartHandlers()
	r.registerAttributeHandlers()
	r.registerSelectionMutationHandlers()
	r.registerSketchReferenceKeyHandlers()
	r.registerHighlightSetHandlers()
	r.registerDrawingHandlers()
	r.registerDrawingStyleHandlers()
	r.registerDrawingViewHandlers()
	r.registerDrawingAnnotationHandlers()
	r.registerDrawingDimensionHandlers()
	r.registerDrawingSketchHandlers()
	r.registerAnalysisHandlers()
	r.readOnly(wire.MethodLogsTail, r.logsTail)
	r.readOnly(wire.MethodScriptRun, r.scriptsRun)
	return r
}

// Trace returns the router's operation-trace buffer, so the host can install its slog handler
// (slog.SetDefault) and have kernel logs land in the same stream the router fills.
func (r *Router) Trace() *trace.Buffer { return r.trace }

// registerStandardHandlers wires the command/document/parameter/model/sketch/feature/theme
// methods.
func (r *Router) registerStandardHandlers() {
	r.registerCommandHandlers()
	r.readOnly(wire.MethodDocumentsList, listDocuments)
	r.readOnly(wire.MethodDocumentsUpdate, documentsUpdate)
	r.readOnly(wire.MethodDocumentsRebuild, documentsRebuild)
	r.readOnly(wire.MethodDocumentsRequiresUpdate, documentsRequiresUpdate)
	r.mutating(wire.MethodDocumentsCreate, "", createDocument)
	r.readOnly(wire.MethodDocumentsActivate, activateDocument)
	r.readOnly(wire.MethodDocumentsClose, closeDocument)
	r.readOnly(wire.MethodDocumentsCloseAll, closeAllDocuments)
	r.readOnly(wire.MethodDocumentsRegisterSubType, registerDocumentSubType)
	r.readOnly(wire.MethodDocumentsListSubTypes, listDocumentSubTypes)
	r.registerFileHandlers()
	r.readOnly(wire.MethodParametersList, listParameters)
	r.readOnly(wire.MethodParametersGet, getParameter)
	r.mutating(wire.MethodParametersAdd, labelEditParameters, addParameter)
	r.mutating(wire.MethodParametersSet, labelEditParameters, setParameter)
	r.registerParameterDetailHandlers()
	r.readOnly(wire.MethodModelTree, modelTree)
	r.readOnly(wire.MethodModelSelection, modelSelection)
	r.readOnly(wire.MethodModelReferenceKeys, referenceKeys)
	r.readOnly(wire.MethodThreadsTableQuery, threadsTableQuery)
	r.readOnly(wire.MethodThreadsResolve, threadsResolve)
	r.mutating(wire.MethodFreeformSetLevel, "Edit Freeform", freeformSetLevel)
	r.mutating(wire.MethodFreeformMoveVertices, "Edit Freeform", freeformMoveVertices)
	r.mutating(wire.MethodFreeformCreaseEdges, "Crease Edges", freeformCreaseEdges)
	r.registerSketchHandlers()
	r.registerFeatureHandlers()
	r.registerSheetMetalHandlers()
	r.registerFlatPatternHandlers()
	r.readOnly(wire.MethodWorkPlanesList, listWorkPlanes)
	r.mutating(wire.MethodWorkPlanesCreate, "Create Work Plane", createWorkPlanes)
	r.mutating(wire.MethodWorkPlanesRedefine, "Redefine Work Plane", redefineWorkPlane)
	r.mutating(wire.MethodWorkPointsCreate, "Create Work Point", createWorkPoint)

	r.readOnly(wire.MethodWorkSurfacesList, listWorkSurfaces)
	r.readOnly(wire.MethodWorkSurfacesGet, getWorkSurface)
	r.mutating(wire.MethodWorkSurfacesSetVisible, "", setWorkSurfaceVisible)
	r.mutating(wire.MethodWorkSurfacesRename, "Rename Work Surface", renameWorkSurface)
	r.readOnly(wire.MethodThemeActive, themeActive)
	r.readOnly(wire.MethodThemeList, themeList)
	r.readOnly(wire.MethodViewGetDisplayMode, getDisplayMode)
	r.readOnly(wire.MethodViewSetDisplayMode, setDisplayMode)
	r.readOnly(wire.MethodViewListDisplayModes, listDisplayModes)
	r.readOnly(wire.MethodViewGetCamera, getCamera)
	r.readOnly(wire.MethodViewSetCamera, setCamera)
	r.readOnly(wire.MethodViewportCapture, captureViewport)
	r.readOnly(wire.MethodViewportCaptureWindow, captureWindow)
	r.readOnly(wire.MethodViewportSetNormalDebug, setNormalDebug)
	r.readOnly(wire.MethodViewportSetMeshColors, setMeshColors)
	r.readOnly(wire.MethodInteractionState, interactionState)
	r.readOnly(wire.MethodInteractionSetNotice, interactionSetNotice)
	r.readOnly(wire.MethodViewsList, listViews)
	r.readOnly(wire.MethodViewsAdd, addView)
	r.readOnly(wire.MethodViewsActivate, activateView)
	r.readOnly(wire.MethodViewsClose, closeView)
	r.readOnly(wire.MethodViewsRename, renameView)
	r.readOnly(wire.MethodViewsGetLayout, getLayout)
	r.readOnly(wire.MethodViewsSetLayout, setLayout)
}

// registerFileHandlers wires the file surface (M03-F07, #608): identity, the
// persisted file-to-file reference records, and reference repair.
func (r *Router) registerFileHandlers() {
	r.readOnly(wire.MethodFilesGet, getFile)
	r.readOnly(wire.MethodFilesListReferences, listFileReferences)
	r.mutating(wire.MethodFilesReplaceReference, "Replace Reference", replaceFileReference)
	r.readOnly(wire.MethodDocumentsListFileReferences, listDocumentFileReferences)
	r.readOnly(wire.MethodDocumentsListAttachments, listAttachments)
	r.mutating(wire.MethodDocumentsAddAttachment, "", addAttachment)
	r.mutating(wire.MethodDocumentsRemoveAttachment, "", removeAttachment)
	r.readOnly(wire.MethodDocumentsListInterests, listDocumentInterests)
	r.mutating(wire.MethodDocumentsAddInterest, "", addDocumentInterest)
	r.mutating(wire.MethodDocumentsRemoveInterest, "", removeDocumentInterest)
	r.readOnly(wire.MethodDocumentsHasInterest, hasDocumentInterest)

	// Document units of measure + unit/expression service (#146).
	r.readOnly(wire.MethodDocumentsGetUnits, getDocumentUnits)
	r.readOnly(wire.MethodDocumentsSetUnits, setDocumentUnits)
	r.readOnly(wire.MethodUnitsConvert, unitsConvert)
	r.readOnly(wire.MethodUnitsGetStringFromValue, unitsGetStringFromValue)
	r.readOnly(wire.MethodUnitsGetPreciseStringFromValue, unitsGetPreciseStringFromValue)
	r.readOnly(wire.MethodUnitsGetValueFromExpression, unitsGetValueFromExpression)
	r.readOnly(wire.MethodUnitsGetDatabaseUnitsFromExpression, unitsGetDatabaseUnitsFromExpression)
	r.readOnly(wire.MethodUnitsIsExpressionValid, unitsIsExpressionValid)
	r.readOnly(wire.MethodUnitsCompatibleUnits, unitsCompatibleUnits)
	r.readOnly(wire.MethodUnitsGetTypeFromString, unitsGetTypeFromString)
	r.readOnly(wire.MethodUnitsGetStringFromType, unitsGetStringFromType)
	r.readOnly(wire.MethodUnitsGetLocaleCorrectedExpression, unitsGetLocaleCorrectedExpression)
	r.readOnly(wire.MethodUnitsGetDrivingParameters, unitsGetDrivingParameters)

	r.mutating(wire.MethodDocumentsOpen, "", openDocument)
	r.readOnly(wire.MethodDocumentsSave, saveDocument)
	r.readOnly(wire.MethodDocumentsSaveAs, saveDocumentAs)
	r.readOnly(wire.MethodDocumentsSaveCopyAs, saveDocumentCopyAs)
	r.readOnly(wire.MethodDocumentsBatchSave, batchSave)
}

// registerTransactionHandlers wires the undo/redo control methods — navigate and query
// the active document's transaction-event stream (transaction.undo/redo/state), plus the
// bounded transaction.begin/end/abort that make a batch one undo step or discard it.
func (r *Router) registerTransactionHandlers() {
	r.mutating(wire.MethodTransactionUndo, "", undoTransaction)
	r.mutating(wire.MethodTransactionRedo, "", redoTransaction)
	r.readOnly(wire.MethodTransactionState, transactionState)
	r.readOnly(wire.MethodTransactionBegin, beginTransaction)
	r.mutating(wire.MethodTransactionEnd, "", endTransaction)
	r.mutating(wire.MethodTransactionAbort, "", abortTransaction)
	r.readOnly(wire.MethodTransactionHistory, transactionHistory)
	r.mutating(wire.MethodTransactionJumpTo, "", jumpTransaction)
}

// registerParameterDetailHandlers wires the member-level parameter surface —
// detail reads, presentation/tolerance/value-list mutations, dependency queries
// and delete (M02-F08, Oblikovati#607).
func (r *Router) registerParameterDetailHandlers() {
	r.readOnly(wire.MethodParametersGetDetail, getParameterDetail)
	r.mutating(wire.MethodParametersUpdate, labelEditParameters, updateParameter)
	r.mutating(wire.MethodParametersSetTolerance, labelEditParameters, setParameterTolerance)
	r.mutating(wire.MethodParametersSetExpressionList, labelEditParameters, setParameterExpressionList)
	r.mutating(wire.MethodParametersDelete, "Delete Parameter", deleteParameter)
	r.readOnly(wire.MethodParametersDrivenBy, parameterDrivenBy)
	r.readOnly(wire.MethodParametersDependents, parameterDependents)
	r.registerParameterGroupHandlers()
	r.registerParameterSettingsHandlers()
	r.registerDerivedTableHandlers()
}

// registerParameterGroupHandlers wires the custom parameter groups (M02-F05,
// Oblikovati#604).
func (r *Router) registerParameterGroupHandlers() {
	r.readOnly(wire.MethodParametersGroupsList, listParameterGroups)
	r.mutating(wire.MethodParametersGroupsAdd, labelEditParameterGroups, addParameterGroup)
	r.mutating(wire.MethodParametersGroupsDelete, labelEditParameterGroups, deleteParameterGroup)
	r.mutating(wire.MethodParametersGroupsSetDisplayName, labelEditParameterGroups, setParameterGroupDisplayName)
	r.mutating(wire.MethodParametersGroupsAddMember, labelEditParameterGroups, addParameterGroupMember)
	r.mutating(wire.MethodParametersGroupsRemoveMember, labelEditParameterGroups, removeParameterGroupMember)
}

// registerParameterSettingsHandlers wires the document-level parameter
// settings, the tolerance sweep, and the XML exchange (M02-F07, Oblikovati#606).
func (r *Router) registerParameterSettingsHandlers() {
	r.readOnly(wire.MethodParametersGetSettings, getParameterSettings)
	r.mutating(wire.MethodParametersSetSettings, "Edit Parameter Settings", setParameterSettings)
	r.mutating(wire.MethodParametersSetAllModelValueType, labelEditParameters, sweepParameterModelValues)
	r.readOnly(wire.MethodParametersExport, exportParameters)
	r.mutating(wire.MethodParametersImport, "Import Parameters", importParameters)
}

// registerDerivedTableHandlers wires the derived parameter tables (M02-F06,
// Oblikovati#605).
func (r *Router) registerDerivedTableHandlers() {
	r.readOnly(wire.MethodParametersDerivedTablesList, listDerivedTables)
	r.mutating(wire.MethodParametersDerivedTablesAdd, labelEditDerivedParameters, addDerivedTable)
	r.mutating(wire.MethodParametersDerivedTablesSetLinked, labelEditDerivedParameters, setDerivedTableLinked)
	r.mutating(wire.MethodParametersDerivedTablesDelete, labelEditDerivedParameters, deleteDerivedTable)
}

// registerSketchHandlers wires the 2D-sketch methods: the spine + enumeration here, and
// the authoring (entity/constraint/dimension/edit/pattern) methods in the companion.
func (r *Router) registerSketchHandlers() {
	r.mutating(wire.MethodSketchCreate, "Create Sketch", createSketch)
	r.mutating(wire.MethodSketchRectangle, "Add Sketch Geometry", sketchRectangle)
	r.readOnly(wire.MethodSketchList, listSketches)
	r.readOnly(wire.MethodSketchGet, getSketch)
	r.readOnly(wire.MethodSketchDependents, sketchDependents)
	r.mutating(wire.MethodSketchEdit, "", editSketch)
	r.mutating(wire.MethodSketchExitEdit, "", exitEditSketch)
	r.mutating(wire.MethodSketchSolve, "", solveSketch)
	r.mutating(wire.MethodSketchDelete, "Delete Sketch", deleteSketch)
	r.readOnly(wire.MethodSketchEntities, enumerateEntities)
	r.readOnly(wire.MethodSketchConstraints, enumerateConstraints)
	r.readOnly(wire.MethodSketchDimensions, enumerateDimensions)
	r.readOnly(wire.MethodSketchConstraintStatus, constraintStatus)
	r.readOnly(wire.MethodSketchProfiles, sketchProfiles)
	r.readOnly(wire.MethodSketchRegionProperties, sketchRegionProperties)
	r.readOnly(wire.MethodSketch3DRegionProperties, sketch3DRegionProperties)
	r.mutating(wire.MethodSketchBlockDefinitionCreate, "Create Block", createBlockDefinition)
	r.readOnly(wire.MethodSketchBlockDefinitionList, listBlockDefinitions)
	r.mutating(wire.MethodSketchBlockDefinitionDelete, "Delete Block", deleteBlockDefinition)
	r.mutating(wire.MethodSketchAddBlockInstance, "Insert Block", addBlockInstance)
	r.readOnly(wire.MethodSketchListBlockInstances, listBlockInstances)
	r.mutating(wire.MethodSketchSetSplineHandle, "Edit Spline", setSplineHandle)
	r.mutating(wire.MethodSketch3DSetSplineHandle, "Edit Spline", setSplineHandle3D)
	r.mutating(wire.MethodSketch3DEditHelix, "Edit Helix", sketch3DEditHelix)
	r.mutating(wire.MethodSketchSetInferenceOptions, "", setInferenceOptions)
	r.readOnly(wire.MethodSketchGetInferenceOptions, getInferenceOptions)
	r.registerSketchAuthoringHandlers()
	r.registerSketch3DHandlers()
}

// registerSketch3DHandlers wires the 3D-sketch (Sketch3D) methods: the spine, enumeration,
// and property edits (M22-F01). The 3D authoring methods (addEntity/addConstraint/
// addDimension) are wired by their features (M22 F02+).
func (r *Router) registerSketch3DHandlers() {
	r.readOnly(wire.MethodSketch3DCreate, createSketch3D)
	r.readOnly(wire.MethodSketch3DList, listSketches3D)
	r.readOnly(wire.MethodSketch3DGet, getSketch3D)
	r.readOnly(wire.MethodSketch3DEdit, editSketch3D)
	r.readOnly(wire.MethodSketch3DExitEdit, exitEditSketch3D)
	r.readOnly(wire.MethodSketch3DSolve, solveSketch3D)
	r.readOnly(wire.MethodSketch3DDelete, deleteSketch3D)
	r.readOnly(wire.MethodSketch3DSetProperty, setSketch3DProperty)
	r.readOnly(wire.MethodSketch3DEntities, enumerateEntities3D)
	r.readOnly(wire.MethodSketch3DConstraints, enumerateConstraints3D)
	r.readOnly(wire.MethodSketch3DDimensions, enumerateDimensions3D)
	r.readOnly(wire.MethodSketch3DConstraintStatus, constraintStatus3D)
	r.registerSketch3DAuthoringHandlers()
}

// registerSketch3DAuthoringHandlers wires the 3D-sketch mutation/query methods: entity/
// constraint/dimension creation, profiles/paths, edit transform, and include.
func (r *Router) registerSketch3DAuthoringHandlers() {
	r.readOnly(wire.MethodSketch3DAddEntity, addSketch3DEntity)
	r.readOnly(wire.MethodSketch3DAddConstraint, addSketch3DConstraint)
	r.readOnly(wire.MethodSketch3DDeleteConstraint, deleteSketch3DConstraint)
	r.readOnly(wire.MethodSketch3DAddDimension, addSketch3DDimension)
	r.readOnly(wire.MethodSketch3DDriveDimension, driveSketch3DDimension)
	r.readOnly(wire.MethodSketch3DProfiles, sketch3DProfiles)
	r.readOnly(wire.MethodSketch3DPaths, sketch3DPaths)
	r.readOnly(wire.MethodSketch3DTransform, transformSketch3D)
	r.readOnly(wire.MethodSketch3DInclude, includeSketch3D)
	r.readOnly(wire.MethodSketch3DIncludeSketch, includeSketch2DInto3D)
	r.readOnly(wire.MethodSketch3DAddSurfaceCurve, addSketch3DSurfaceCurve)
}

// registerSketchAuthoringHandlers wires the sketch mutation methods: property edits,
// entity/constraint/dimension creation, and the edit/pattern operations.
func (r *Router) registerSketchAuthoringHandlers() {
	r.mutating(wire.MethodSketchSetProperty, "Edit Sketch", setSketchProperty)
	r.readOnly(wire.MethodSketchGetCustomLineType, getSketchCustomLineType)
	r.mutating(wire.MethodSketchSetCustomLineType, "Edit Sketch", setSketchCustomLineType)
	r.mutating(wire.MethodSketchAddEntity, "Add Sketch Geometry", addSketchEntity)
	r.mutating(wire.MethodSketchAddConstraint, "Add Constraint", addConstraint)
	r.mutating(wire.MethodSketchDeleteConstraint, "Delete Constraint", deleteConstraint)
	r.mutating(wire.MethodSketchAddDimension, "Add Dimension", addDimension)
	r.mutating(wire.MethodSketchDriveDimension, "Edit Dimension", driveDimension)
	r.mutating(wire.MethodSketchTransform, "Transform Sketch", transformSketch)
	r.readOnly(wire.MethodSketchCopyTo, sketchCopyTo)
	r.mutating(wire.MethodSketchAddPattern, "Sketch Pattern", addSketchPattern)
	r.mutating(wire.MethodSketchOffset, "Offset Geometry", offsetSketchEntity)
	r.readOnly(wire.MethodSketchAddImage, addSketchImage)
	r.readOnly(wire.MethodSketchAddFillRegion, addFillRegion)
	r.readOnly(wire.MethodSketchAddText, addText)
	r.readOnly(wire.MethodSketchEditText, editText)
	r.readOnly(wire.MethodSketchGetText, getText)
	r.readOnly(wire.MethodSketchAutoDimension, autoDimensionSketch)
	r.mutating(wire.MethodSketchProject, "Project Geometry", projectGeometry)
}

// registerCommandHandlers wires the command and ribbon methods — the add-in UI surface
// (list/execute/create commands and enumerate the active ribbon, RibbonUI core/07).
func (r *Router) registerCommandHandlers() {
	r.readOnly(wire.MethodCommandsList, listCommands)
	r.readOnly(wire.MethodCommandsExecute, executeCommand)
	r.readOnly(wire.MethodCommandsCreate, createCommand)
	r.readOnly(wire.MethodCommandsSetState, setCommandState)
	r.readOnly(wire.MethodCommandLineSubmit, submitCommandLine)
	r.readOnly(wire.MethodRibbonList, ribbonList)
}

// registerMaterialHandlers wires the appearance/material/assignment/physical-properties
// methods (M19 / ADR-0022).
func (r *Router) registerMaterialHandlers() {
	r.readOnly(wire.MethodAppearancesList, listAppearances)
	r.readOnly(wire.MethodAppearancesGet, getAppearance)
	r.readOnly(wire.MethodAppearancesCreate, createAppearance)
	r.readOnly(wire.MethodAppearancesUpdate, updateAppearance)
	r.readOnly(wire.MethodMaterialsList, listMaterials)
	r.readOnly(wire.MethodMaterialsGet, getMaterial)
	r.readOnly(wire.MethodMaterialsCreate, createMaterial)
	r.readOnly(wire.MethodMaterialsUpdate, updateMaterial)
	r.mutating(wire.MethodModelAssignMaterial, "Assign Material", assignMaterial)
	r.mutating(wire.MethodModelAssignAppearance, "Assign Appearance", assignAppearance)
	r.readOnly(wire.MethodModelPhysicalProperties, physicalProperties)

	// Body topology, queries and facet sets (M07 #293/#629/#630).
	r.readOnly(wire.MethodBodyList, bodyList)
	r.readOnly(wire.MethodBodySetVisible, bodySetVisible)
	r.readOnly(wire.MethodBodyRename, bodyRename)
	r.readOnly(wire.MethodBodyDelete, bodyDelete)
	r.readOnly(wire.MethodBodyPhysicalProps, bodyPhysicalProperties)
	r.readOnly(wire.MethodBodyShells, bodyShells)
	r.readOnly(wire.MethodBodyWires, bodyWires)
	r.readOnly(wire.MethodWireOffsetPlanar, wireOffsetPlanar)
	r.readOnly(wire.MethodBodyLocateUsingPoint, bodyLocateUsingPoint)
	r.readOnly(wire.MethodBodyFindUsingRay, bodyFindUsingRay)
	r.readOnly(wire.MethodBodyIsPointInside, bodyIsPointInside)
	r.readOnly(wire.MethodBodyConvexityEdges, bodyConvexityEdges)
	r.readOnly(wire.MethodBodyMinimumDistance, bodyMinimumDistance)
	r.readOnly(wire.MethodBodyValidate, bodyValidate)
	r.readOnly(wire.MethodBodyRangeBox, bodyRangeBox)
	r.readOnly(wire.MethodBodyBindTransientKey, bodyBindTransientKey)
	r.readOnly(wire.MethodBodyCalculateFacets, bodyCalculateFacets)
	r.readOnly(wire.MethodBodyExistingFacets, bodyExistingFacets)
	r.readOnly(wire.MethodBodyFacetTolerances, bodyFacetTolerances)
	r.readOnly(wire.MethodBodyCalculateStrokes, bodyCalculateStrokes)
	r.readOnly(wire.MethodBodyExistingStrokes, bodyExistingStrokes)
	r.readOnly(wire.MethodBodyStrokeTolerances, bodyStrokeTolerances)
	r.readOnly(wire.MethodFaceCalculateFacets, faceCalculateFacets)
	r.readOnly(wire.MethodFaceCalculateStrokes, faceCalculateStrokes)
	r.readOnly(wire.MethodBodyFaceEvaluate, bodyFaceEvaluate)

	// The transient B-rep factory (M07 #628) — session-scoped, no document
	// mutation, so none of these are mutating methods.
	r.readOnly(wire.MethodBrepCreatePrimitive, brepCreatePrimitive)
	r.readOnly(wire.MethodBrepBoolean, brepBoolean)
	r.readOnly(wire.MethodBrepTransform, brepTransform)
	r.readOnly(wire.MethodBrepCopy, brepCopy)
	r.readOnly(wire.MethodBrepSectionWithPlane, brepSectionWithPlane)
	r.readOnly(wire.MethodBrepDeleteFaces, brepDeleteFaces)
	r.readOnly(wire.MethodBrepSilhouette, brepSilhouette)
	r.readOnly(wire.MethodBrepRuledSurface, brepRuledSurface)
	r.readOnly(wire.MethodBrepOffsetFaces, brepOffsetFaces)
	r.readOnly(wire.MethodBrepImprint, brepImprint)
	r.readOnly(wire.MethodBrepIdenticalBodies, brepIdenticalBodies)
	r.readOnly(wire.MethodBrepCreateFromDefinition, brepCreateFromDefinition)
	r.readOnly(wire.MethodBrepDescribe, brepDescribe)
	r.readOnly(wire.MethodBrepList, brepList)
	r.readOnly(wire.MethodBrepDelete, brepDelete)
}

// registerLightingHandlers wires the lighting-style, light, environment, and shadow methods
// (M16/F03 PBI-155, ADR-0026).
func (r *Router) registerLightingHandlers() {
	r.readOnly(wire.MethodLightingGetStyle, getLightingStyle)
	r.readOnly(wire.MethodLightingSetStyle, setLightingStyle)
	r.readOnly(wire.MethodLightingListStyles, listLightingStyles)
	r.readOnly(wire.MethodLightingListLights, listLights)
	r.readOnly(wire.MethodLightingAddLight, addLight)
	r.readOnly(wire.MethodLightingSetLight, setLight)
	r.readOnly(wire.MethodViewGetShadows, getShadows)
	r.readOnly(wire.MethodViewSetShadows, setShadows)
	r.readOnly(wire.MethodEnvironmentGet, getEnvironment)
	r.readOnly(wire.MethodEnvironmentSet, setEnvironment)
	r.readOnly(wire.MethodEnvironmentListPresets, listEnvironmentPresets)
	r.readOnly(wire.MethodEnvironmentLoadImage, loadEnvironmentImage)
}

// registerGraphicsHandlers wires the client/interaction graphics methods — the add-in
// overlay surface for drawing meshes, heatmaps, lines, markers and labels (M05-F05).
func (r *Router) registerGraphicsHandlers() {
	r.readOnly(wire.MethodClientGraphicsSet, setClientGraphics)
	r.readOnly(wire.MethodClientGraphicsList, listClientGraphics)
	r.readOnly(wire.MethodClientGraphicsDelete, deleteClientGraphics)
	r.readOnly(wire.MethodClientGraphicsSetVisible, setClientGraphicsVisible)
	r.readOnly(wire.MethodInteractionGraphicsUpdate, updateInteractionGraphics)
	r.readOnly(wire.MethodInteractionGraphicsClear, clearInteractionGraphics)
	r.registerGraphicsObjectModelHandlers()
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
	// A handler that implements MutatingMethod edits the document. Open the active document's undo stream
	// before it runs, so the delta the central seam records afterwards is measured against the pre-edit
	// state (commitMutation).
	mut, mutates := h.(MutatingMethod)
	if mutates {
		s.EnsureActiveEditBaseline()
	}
	out, herr := h.Handle(s, args)
	if herr != nil {
		herr = methodError(method, herr)
		r.record(method, time.Since(start), false, herr.Error(), "", "")
		return nil, herr
	}
	r.record(method, time.Since(start), true, "", "", "")
	if mutates {
		r.commitMutation(s, method, mut.UndoLabel(), req)
	}
	return out, nil
}

// commitMutation runs the post-success side effects of a document-mutating method — called only when the
// handler implements [MutatingMethod], so undo recording and collaboration replication cannot drift from
// the handler set:
//   - emit edit.committed so a collaboration add-in can replay the wire request (ADR-0004);
//   - resync any values other documents derive from (M02-F06);
//   - record one undo step (the central seam — RecordActiveEdit) when the handler carries a non-empty
//     label, so every API / MCP / Lua mutation is undoable, not just parameter edits.
//
// An empty label is broadcast but records no step: transaction-control methods (undo/redo/end/abort,
// which move the cursor themselves) and metadata-only methods the parametric recipe does not capture.
func (r *Router) commitMutation(s *app.Session, method, label string, req []byte) {
	s.EmitEditCommitted(method, req)
	s.ResyncDerivedFromActiveDocument()
	if label != "" {
		s.RecordActiveEdit(label)
	}
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

// MutatingMethods returns the methods whose handler implements [MutatingMethod] (records an undo step and
// replicates), each mapped to its undo label. It reads the handlers' own type, so it is the live
// classification with no separate list to drift — used by the drift guard test (#1426).
func (r *Router) MutatingMethods() map[string]string {
	out := make(map[string]string, len(r.handlers))
	for m, h := range r.handlers {
		if mut, ok := h.(MutatingMethod); ok {
			out[m] = mut.UndoLabel()
		}
	}
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

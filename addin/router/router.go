// SPDX-License-Identifier: GPL-2.0-only

// (The package comment was promoted to doc.go — #1669, M40 audit D12.)

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
	r.readOnly(wire.MethodDocumentsUpdate, typedPart(documentsUpdate))
	r.readOnly(wire.MethodDocumentsRebuild, typedPart(documentsRebuild))
	r.readOnly(wire.MethodDocumentsRequiresUpdate, partQuery(documentsRequiresUpdate))
	r.mutating(wire.MethodDocumentsCreate, "", typed(createDocument))
	r.readOnly(wire.MethodDocumentsActivate, typed(activateDocument))
	r.readOnly(wire.MethodDocumentsClose, typed(closeDocument))
	r.readOnly(wire.MethodDocumentsCloseAll, typed(closeAllDocuments))
	r.readOnly(wire.MethodDocumentsRegisterSubType, typed(registerDocumentSubType))
	r.readOnly(wire.MethodDocumentsListSubTypes, listDocumentSubTypes)
	r.registerFileHandlers()
	r.readOnly(wire.MethodParametersList, holderQuery(listParameters))
	r.readOnly(wire.MethodParametersGet, typedHolder(getParameter))
	r.mutating(wire.MethodParametersAdd, labelEditParameters, typedHolder(addParameter))
	r.mutating(wire.MethodParametersSet, labelEditParameters, typedHolder(setParameter))
	r.mutating(wire.MethodParametersRename, labelEditParameters, typedHolder(renameParameter))
	r.mutating(wire.MethodParametersConvert, labelEditParameters, typedHolder(convertParameter))
	r.registerParameterDetailHandlers()
	r.readOnly(wire.MethodModelTree, partQuery(modelTree))
	r.readOnly(wire.MethodModelSelection, modelSelection)
	r.readOnly(wire.MethodModelReferenceKeys, partQuery(referenceKeys))
	r.readOnly(wire.MethodThreadsTableQuery, threadsTableQuery)
	r.readOnly(wire.MethodThreadsResolve, threadsResolve)
	r.mutating(wire.MethodFreeformSetLevel, "Edit Freeform", freeformSetLevel)
	r.mutating(wire.MethodFreeformMoveVertices, "Edit Freeform", freeformMoveVertices)
	r.mutating(wire.MethodFreeformCreaseEdges, "Crease Edges", freeformCreaseEdges)
	r.registerSketchHandlers()
	r.registerFeatureHandlers()
	r.registerSheetMetalHandlers()
	r.registerFlatPatternHandlers()
	r.readOnly(wire.MethodWorkPlanesList, ctxQuery(activeWorkHost, listWorkPlanes))
	r.mutating(wire.MethodWorkPlanesCreate, "Create Work Plane", typedCtx(activeWorkHost, createWorkPlanes))
	r.mutating(wire.MethodWorkPlanesRedefine, "Redefine Work Plane", typedCtx(activeWorkHost, redefineWorkPlane))
	r.mutating(wire.MethodWorkPointsCreate, "Create Work Point", typedCtx(activeWorkHost, createWorkPoint))
	r.readOnly(wire.MethodWorkAxesList, ctxQuery(activeWorkHost, listWorkAxes))
	r.mutating(wire.MethodWorkAxesCreate, "Create Work Axis", typedCtx(activeWorkHost, createWorkAxis))

	r.readOnly(wire.MethodWorkSurfacesList, partQuery(listWorkSurfaces))
	r.readOnly(wire.MethodWorkSurfacesGet, typedPart(getWorkSurface))
	r.mutating(wire.MethodWorkSurfacesSetVisible, "", typedPart(setWorkSurfaceVisible))
	r.mutating(wire.MethodWorkSurfacesRename, "Rename Work Surface", typedPart(renameWorkSurface))
	r.readOnly(wire.MethodThemeActive, themeActive)
	r.readOnly(wire.MethodThemeList, themeList)
	r.readOnly(wire.MethodViewGetDisplayMode, getDisplayMode)
	r.readOnly(wire.MethodViewSetDisplayMode, typed(setDisplayMode))
	r.readOnly(wire.MethodViewListDisplayModes, listDisplayModes)
	r.readOnly(wire.MethodViewGetCamera, typed(getCamera))
	r.readOnly(wire.MethodViewSetCamera, typed(setCamera))
	r.readOnly(wire.MethodViewportCapture, typed(captureViewport))
	r.readOnly(wire.MethodViewportCaptureWindow, typed(captureWindow))
	r.readOnly(wire.MethodViewportSetNormalDebug, typed(setNormalDebug))
	r.readOnly(wire.MethodViewportSetMeshColors, typed(setMeshColors))
	r.readOnly(wire.MethodInteractionState, interactionState)
	r.readOnly(wire.MethodInteractionSetNotice, interactionSetNotice)
	r.readOnly(wire.MethodViewsList, typed(listViews))
	r.readOnly(wire.MethodViewsAdd, typed(addView))
	r.readOnly(wire.MethodViewsActivate, typed(activateView))
	r.readOnly(wire.MethodViewsClose, typed(closeView))
	r.readOnly(wire.MethodViewsRename, typed(renameView))
	r.readOnly(wire.MethodViewsGetLayout, typed(getLayout))
	r.readOnly(wire.MethodViewsSetLayout, typed(setLayout))
}

// registerFileHandlers wires the file surface (M03-F07, #608): identity, the
// persisted file-to-file reference records, and reference repair.
func (r *Router) registerFileHandlers() {
	r.readOnly(wire.MethodFilesGet, typed(getFile))
	r.readOnly(wire.MethodFilesListReferences, typed(listFileReferences))
	r.mutating(wire.MethodFilesReplaceReference, "Replace Reference", typed(replaceFileReference))
	r.readOnly(wire.MethodDocumentsListFileReferences, typed(listDocumentFileReferences))
	r.readOnly(wire.MethodDocumentsListAttachments, typed(listAttachments))
	r.mutating(wire.MethodDocumentsAddAttachment, "", typed(addAttachment))
	r.mutating(wire.MethodDocumentsRemoveAttachment, "", typed(removeAttachment))
	r.readOnly(wire.MethodDocumentsListInterests, typed(listDocumentInterests))
	r.mutating(wire.MethodDocumentsAddInterest, "", typed(addDocumentInterest))
	r.mutating(wire.MethodDocumentsRemoveInterest, "", typed(removeDocumentInterest))
	r.readOnly(wire.MethodDocumentsHasInterest, typed(hasDocumentInterest))

	// Document units of measure + unit/expression service (#146).
	r.readOnly(wire.MethodDocumentsGetUnits, ctxQuery(activeUnitsDocument, getDocumentUnits))
	r.mutating(wire.MethodDocumentsSetUnits, "Set Units", typedCtx(activeUnitsDocument, setDocumentUnits))
	r.readOnly(wire.MethodUnitsConvert, typed(unitsConvert))
	r.readOnly(wire.MethodUnitsGetStringFromValue, typedCtx(activeUnitsDocument, unitsGetStringFromValue))
	r.readOnly(wire.MethodUnitsGetPreciseStringFromValue, typedCtx(activeUnitsDocument, unitsGetPreciseStringFromValue))
	r.readOnly(wire.MethodUnitsGetValueFromExpression, typedCtx(activeUnitsDocument, unitsGetValueFromExpression))
	r.readOnly(wire.MethodUnitsGetDatabaseUnitsFromExpression, typedCtx(activeUnitsDocument, unitsGetDatabaseUnitsFromExpression))
	r.readOnly(wire.MethodUnitsIsExpressionValid, typedCtx(activeUnitsDocument, unitsIsExpressionValid))
	r.readOnly(wire.MethodUnitsCompatibleUnits, typedCtx(activeUnitsDocument, unitsCompatibleUnits))
	r.readOnly(wire.MethodUnitsGetTypeFromString, typed(unitsGetTypeFromString))
	r.readOnly(wire.MethodUnitsGetStringFromType, typedCtx(activeUnitsDocument, unitsGetStringFromType))
	r.readOnly(wire.MethodUnitsGetLocaleCorrectedExpression, typed(unitsGetLocaleCorrectedExpression))
	r.readOnly(wire.MethodUnitsGetDrivingParameters, typed(unitsGetDrivingParameters))

	r.mutating(wire.MethodDocumentsOpen, "", typed(openDocument))
	r.readOnly(wire.MethodDocumentsSave, typed(saveDocument))
	r.readOnly(wire.MethodDocumentsSaveAs, typed(saveDocumentAs))
	r.readOnly(wire.MethodDocumentsSaveCopyAs, typed(saveDocumentCopyAs))
	r.readOnly(wire.MethodDocumentsBatchSave, typed(batchSave))
}

// registerTransactionHandlers wires the undo/redo control methods — navigate and query
// the active document's transaction-event stream (transaction.undo/redo/state), plus the
// bounded transaction.begin/end/abort that make a batch one undo step or discard it.
func (r *Router) registerTransactionHandlers() {
	r.mutating(wire.MethodTransactionUndo, "", undoTransaction)
	r.mutating(wire.MethodTransactionRedo, "", redoTransaction)
	r.readOnly(wire.MethodTransactionState, transactionState)
	r.readOnly(wire.MethodTransactionBegin, typed(beginTransaction))
	r.mutating(wire.MethodTransactionEnd, "", endTransaction)
	r.mutating(wire.MethodTransactionAbort, "", abortTransaction)
	r.readOnly(wire.MethodTransactionHistory, typed(transactionHistory))
	r.mutating(wire.MethodTransactionJumpTo, "", typed(jumpTransaction))
}

// registerParameterDetailHandlers wires the member-level parameter surface —
// detail reads, presentation/tolerance/value-list mutations, dependency queries
// and delete (M02-F08, Oblikovati#607).
func (r *Router) registerParameterDetailHandlers() {
	r.readOnly(wire.MethodParametersGetDetail, typed(getParameterDetail))
	r.mutating(wire.MethodParametersUpdate, labelEditParameters, typed(updateParameter))
	r.mutating(wire.MethodParametersSetTolerance, labelEditParameters, typed(setParameterTolerance))
	r.mutating(wire.MethodParametersSetExpressionList, labelEditParameters, typed(setParameterExpressionList))
	r.mutating(wire.MethodParametersDelete, "Delete Parameter", typed(deleteParameter))
	r.readOnly(wire.MethodParametersDrivenBy, typed(parameterDrivenBy))
	r.readOnly(wire.MethodParametersDependents, typed(parameterDependents))
	r.registerParameterGroupHandlers()
	r.registerParameterSettingsHandlers()
	r.registerDerivedTableHandlers()
}

// registerParameterGroupHandlers wires the custom parameter groups (M02-F05,
// Oblikovati#604).
func (r *Router) registerParameterGroupHandlers() {
	r.readOnly(wire.MethodParametersGroupsList, holderQuery(listParameterGroups))
	r.mutating(wire.MethodParametersGroupsAdd, labelEditParameterGroups, typed(addParameterGroup))
	r.mutating(wire.MethodParametersGroupsDelete, labelEditParameterGroups, typed(deleteParameterGroup))
	r.mutating(wire.MethodParametersGroupsSetDisplayName, labelEditParameterGroups, typed(setParameterGroupDisplayName))
	r.mutating(wire.MethodParametersGroupsAddMember, labelEditParameterGroups, typed(addParameterGroupMember))
	r.mutating(wire.MethodParametersGroupsRemoveMember, labelEditParameterGroups, typed(removeParameterGroupMember))
}

// registerParameterSettingsHandlers wires the document-level parameter
// settings, the tolerance sweep, and the XML exchange (M02-F07, Oblikovati#606).
func (r *Router) registerParameterSettingsHandlers() {
	r.readOnly(wire.MethodParametersGetSettings, holderQuery(getParameterSettings))
	r.mutating(wire.MethodParametersSetSettings, "Edit Parameter Settings", typedHolder(setParameterSettings))
	r.mutating(wire.MethodParametersSetAllModelValueType, labelEditParameters, typed(sweepParameterModelValues))
	r.readOnly(wire.MethodParametersExport, holderQuery(exportParameters))
	r.mutating(wire.MethodParametersImport, "Import Parameters", typed(importParameters))
}

// registerDerivedTableHandlers wires the derived parameter tables (M02-F06,
// Oblikovati#605).
func (r *Router) registerDerivedTableHandlers() {
	r.readOnly(wire.MethodParametersDerivedTablesList, holderQuery(listDerivedTables))
	r.mutating(wire.MethodParametersDerivedTablesAdd, labelEditDerivedParameters, typed(addDerivedTable))
	r.mutating(wire.MethodParametersDerivedTablesSetLinked, labelEditDerivedParameters, setDerivedTableLinked)
	r.mutating(wire.MethodParametersDerivedTablesDelete, labelEditDerivedParameters, typed(deleteDerivedTable))
}

// registerSketchHandlers wires the 2D-sketch methods: the spine + enumeration here, and
// the authoring (entity/constraint/dimension/edit/pattern) methods in the companion.
func (r *Router) registerSketchHandlers() {
	r.mutating(wire.MethodSketchCreate, "Create Sketch", typedCtx(activeSketchHost, createSketch))
	r.mutating(wire.MethodSketchRectangle, "Add Sketch Geometry", typedCtx(activeSketchHost, sketchRectangle))
	r.readOnly(wire.MethodSketchList, partQuery(listSketches))
	r.readOnly(wire.MethodSketchGet, typedPart(getSketch))
	r.readOnly(wire.MethodSketchDependents, typedPart(sketchDependents))
	r.mutating(wire.MethodSketchEdit, "", typedPart(editSketch))
	r.mutating(wire.MethodSketchExitEdit, "", typedPart(exitEditSketch))
	r.mutating(wire.MethodSketchSolve, "", typedPart(solveSketch))
	r.mutating(wire.MethodSketchDelete, "Delete Sketch", typedPart(deleteSketch))
	r.readOnly(wire.MethodSketchEntities, typedPart(enumerateEntities))
	r.readOnly(wire.MethodSketchConstraints, typedPart(enumerateConstraints))
	r.readOnly(wire.MethodSketchDimensions, typedPart(enumerateDimensions))
	r.readOnly(wire.MethodSketchConstraintStatus, typedPart(constraintStatus))
	r.readOnly(wire.MethodSketchProfiles, typedPart(sketchProfiles))
	r.readOnly(wire.MethodSketchRegionProperties, typedPart(sketchRegionProperties))
	r.readOnly(wire.MethodSketch3DRegionProperties, typedPart(sketch3DRegionProperties))
	r.mutating(wire.MethodSketchBlockDefinitionCreate, "Create Block", typedPart(createBlockDefinition))
	r.readOnly(wire.MethodSketchBlockDefinitionList, partQuery(listBlockDefinitions))
	r.mutating(wire.MethodSketchBlockDefinitionDelete, "Delete Block", typedPart(deleteBlockDefinition))
	r.mutating(wire.MethodSketchAddBlockInstance, "Insert Block", typedPart(addBlockInstance))
	r.readOnly(wire.MethodSketchListBlockInstances, typedPart(listBlockInstances))
	r.mutating(wire.MethodSketchSetSplineHandle, "Edit Spline", typedPart(setSplineHandle))
	r.mutating(wire.MethodSketch3DSetSplineHandle, "Edit Spline", typedPart(setSplineHandle3D))
	r.mutating(wire.MethodSketch3DEditHelix, "Edit Helix", typedPart(sketch3DEditHelix))
	r.mutating(wire.MethodSketchSetInferenceOptions, "", typed(setInferenceOptions))
	r.readOnly(wire.MethodSketchGetInferenceOptions, getInferenceOptions)
	r.registerSketchAuthoringHandlers()
	r.registerSketch3DHandlers()
}

// registerSketch3DHandlers wires the 3D-sketch (Sketch3D) methods: the spine, enumeration,
// and property edits (M22-F01). The 3D authoring methods (addEntity/addConstraint/
// addDimension) are wired by their features (M22 F02+).
func (r *Router) registerSketch3DHandlers() {
	r.mutating(wire.MethodSketch3DCreate, "Create Sketch", typedPart(createSketch3D))
	r.readOnly(wire.MethodSketch3DList, partQuery(listSketches3D))
	r.readOnly(wire.MethodSketch3DGet, typedPart(getSketch3D))
	r.mutating(wire.MethodSketch3DEdit, "", typedPart(editSketch3D))
	r.mutating(wire.MethodSketch3DExitEdit, "", typedPart(exitEditSketch3D))
	r.mutating(wire.MethodSketch3DSolve, "", typedPart(solveSketch3D))
	r.mutating(wire.MethodSketch3DDelete, "Delete Sketch", typedPart(deleteSketch3D))
	r.mutating(wire.MethodSketch3DSetProperty, "Edit Sketch", typedPart(setSketch3DProperty))
	r.readOnly(wire.MethodSketch3DEntities, typedPart(enumerateEntities3D))
	r.readOnly(wire.MethodSketch3DConstraints, typedPart(enumerateConstraints3D))
	r.readOnly(wire.MethodSketch3DDimensions, typedPart(enumerateDimensions3D))
	r.readOnly(wire.MethodSketch3DConstraintStatus, typedPart(constraintStatus3D))
	r.registerSketch3DAuthoringHandlers()
}

// registerSketch3DAuthoringHandlers wires the 3D-sketch mutation/query methods: entity/
// constraint/dimension creation, profiles/paths, edit transform, and include.
func (r *Router) registerSketch3DAuthoringHandlers() {
	r.mutating(wire.MethodSketch3DAddEntity, "Add Sketch Geometry", addSketch3DEntity)
	r.mutating(wire.MethodSketch3DAddConstraint, "Add Constraint", typedPart(addSketch3DConstraint))
	r.mutating(wire.MethodSketch3DDeleteConstraint, "Delete Constraint", typedPart(deleteSketch3DConstraint))
	r.mutating(wire.MethodSketch3DAddDimension, "Add Dimension", typedPart(addSketch3DDimension))
	r.mutating(wire.MethodSketch3DDriveDimension, "Edit Dimension", typedPart(driveSketch3DDimension))
	r.readOnly(wire.MethodSketch3DProfiles, typedPart(sketch3DProfiles))
	r.readOnly(wire.MethodSketch3DPaths, typedPart(sketch3DPaths))
	r.mutating(wire.MethodSketch3DTransform, "Transform Sketch", typedPart(transformSketch3D))
	r.mutating(wire.MethodSketch3DInclude, "Project Geometry", typedPart(includeSketch3D))
	r.mutating(wire.MethodSketch3DIncludeSketch, "Project Geometry", typedPart(includeSketch2DInto3D))
	r.mutating(wire.MethodSketch3DAddSurfaceCurve, "Add Sketch Geometry", addSketch3DSurfaceCurve)
}

// registerSketchAuthoringHandlers wires the sketch mutation methods: property edits,
// entity/constraint/dimension creation, and the edit/pattern operations.
func (r *Router) registerSketchAuthoringHandlers() {
	r.mutating(wire.MethodSketchSetProperty, "Edit Sketch", typedPart(setSketchProperty))
	r.readOnly(wire.MethodSketchGetCustomLineType, typedPart(getSketchCustomLineType))
	r.mutating(wire.MethodSketchSetCustomLineType, "Edit Sketch", typedPart(setSketchCustomLineType))
	r.mutating(wire.MethodSketchAddEntity, "Add Sketch Geometry", typedPart(addSketchEntity))
	r.mutating(wire.MethodSketchAddConstraint, "Add Constraint", typedPart(addConstraint))
	r.mutating(wire.MethodSketchDeleteConstraint, "Delete Constraint", typedPart(deleteConstraint))
	r.mutating(wire.MethodSketchAddDimension, "Add Dimension", typedPart(addDimension))
	r.mutating(wire.MethodSketchDriveDimension, "Edit Dimension", typedPart(driveDimension))
	r.mutating(wire.MethodSketchTransform, "Transform Sketch", typedPart(transformSketch))
	r.readOnly(wire.MethodSketchCopyTo, typedPart(sketchCopyTo))
	r.mutating(wire.MethodSketchAddPattern, "Sketch Pattern", typedPart(addSketchPattern))
	r.mutating(wire.MethodSketchOffset, "Offset Geometry", typedPart(offsetSketchEntity))
	r.mutating(wire.MethodSketchAddImage, "Insert Image", typedPart(addSketchImage))
	r.mutating(wire.MethodSketchAddFillRegion, "Add Sketch Geometry", typedPart(addFillRegion))
	r.mutating(wire.MethodSketchAddText, "Add Text", typedPart(addText))
	r.mutating(wire.MethodSketchEditText, "Edit Text", typedPart(editText))
	r.readOnly(wire.MethodSketchGetText, typedPart(getText))
	r.readOnly(wire.MethodSketchAutoDimension, typedPart(autoDimensionSketch))
	r.mutating(wire.MethodSketchProject, "Project Geometry", typedPart(projectGeometry))
}

// registerCommandHandlers wires the command and ribbon methods — the add-in UI surface
// (list/execute/create commands and enumerate the active ribbon, RibbonUI core/07).
func (r *Router) registerCommandHandlers() {
	r.readOnly(wire.MethodCommandsList, listCommands)
	r.readOnly(wire.MethodCommandsExecute, typed(executeCommand))
	r.readOnly(wire.MethodCommandsCreate, typed(createCommand))
	r.readOnly(wire.MethodCommandsSetState, typed(setCommandState))
	r.readOnly(wire.MethodCommandLineSubmit, typed(submitCommandLine))
	r.readOnly(wire.MethodRibbonList, ribbonList)
}

// registerMaterialHandlers wires the appearance/material/assignment/physical-properties
// methods (M19 / ADR-0022).
func (r *Router) registerMaterialHandlers() {
	r.readOnly(wire.MethodAppearancesList, listAppearances)
	r.readOnly(wire.MethodAppearancesGet, typed(getAppearance))
	r.readOnly(wire.MethodAppearancesCreate, typed(createAppearance))
	r.readOnly(wire.MethodAppearancesUpdate, typed(updateAppearance))
	r.readOnly(wire.MethodMaterialsList, listMaterials)
	r.readOnly(wire.MethodMaterialsGet, typed(getMaterial))
	r.readOnly(wire.MethodMaterialsCreate, typed(createMaterial))
	r.readOnly(wire.MethodMaterialsUpdate, typed(updateMaterial))
	r.mutating(wire.MethodModelAssignMaterial, "Assign Material", typed(assignMaterial))
	r.mutating(wire.MethodModelAssignAppearance, "Assign Appearance", typed(assignAppearance))
	r.readOnly(wire.MethodModelPhysicalProperties, physicalProperties)

	// Body topology, queries and facet sets (M07 #293/#629/#630).
	r.readOnly(wire.MethodBodyList, partQuery(bodyList))
	r.mutating(wire.MethodBodySetVisible, "", typedPart(bodySetVisible))
	r.mutating(wire.MethodBodyRename, "Rename Body", typedPart(bodyRename))
	r.mutating(wire.MethodBodyDelete, "Delete Body", typedPart(bodyDelete))
	r.readOnly(wire.MethodBodyPhysicalProps, typedPart(bodyPhysicalProperties))
	r.readOnly(wire.MethodBodyShells, typedPart(bodyShells))
	r.readOnly(wire.MethodBodyWires, typedPart(bodyWires))
	r.readOnly(wire.MethodWireOffsetPlanar, typed(wireOffsetPlanar))
	r.readOnly(wire.MethodBodyLocateUsingPoint, typedPart(bodyLocateUsingPoint))
	r.readOnly(wire.MethodBodyFindUsingRay, typedPart(bodyFindUsingRay))
	r.readOnly(wire.MethodBodyIsPointInside, typedPart(bodyIsPointInside))
	r.readOnly(wire.MethodBodyConvexityEdges, typedPart(bodyConvexityEdges))
	r.readOnly(wire.MethodBodyMinimumDistance, typedPart(bodyMinimumDistance))
	r.readOnly(wire.MethodBodyValidate, typedPart(bodyValidate))
	r.readOnly(wire.MethodBodyRangeBox, typedPart(bodyRangeBox))
	r.readOnly(wire.MethodBodyBindTransientKey, typedPart(bodyBindTransientKey))
	r.readOnly(wire.MethodBodyCalculateFacets, typedPart(bodyCalculateFacets))
	r.readOnly(wire.MethodBodyExistingFacets, typedPart(bodyExistingFacets))
	r.readOnly(wire.MethodBodyFacetTolerances, typedPart(bodyFacetTolerances))
	r.readOnly(wire.MethodBodyCalculateStrokes, typedPart(bodyCalculateStrokes))
	r.readOnly(wire.MethodBodyExistingStrokes, typedPart(bodyExistingStrokes))
	r.readOnly(wire.MethodBodyStrokeTolerances, typedPart(bodyStrokeTolerances))
	r.readOnly(wire.MethodFaceCalculateFacets, typedPart(faceCalculateFacets))
	r.readOnly(wire.MethodFaceCalculateStrokes, typedPart(faceCalculateStrokes))
	r.readOnly(wire.MethodBodyFaceEvaluate, typedPart(bodyFaceEvaluate))

	// The transient B-rep factory (M07 #628) — session-scoped, no document
	// mutation, so none of these are mutating methods.
	r.readOnly(wire.MethodBrepCreatePrimitive, typed(brepCreatePrimitive))
	r.readOnly(wire.MethodBrepBoolean, typed(brepBoolean))
	r.readOnly(wire.MethodBrepTransform, typed(brepTransform))
	r.readOnly(wire.MethodBrepCopy, typed(brepCopy))
	r.readOnly(wire.MethodBrepSectionWithPlane, typed(brepSectionWithPlane))
	r.readOnly(wire.MethodBrepDeleteFaces, typed(brepDeleteFaces))
	r.readOnly(wire.MethodBrepSilhouette, typed(brepSilhouette))
	r.readOnly(wire.MethodBrepRuledSurface, typed(brepRuledSurface))
	r.readOnly(wire.MethodBrepOffsetFaces, typed(brepOffsetFaces))
	r.readOnly(wire.MethodBrepImprint, typed(brepImprint))
	r.readOnly(wire.MethodBrepIdenticalBodies, typed(brepIdenticalBodies))
	r.readOnly(wire.MethodBrepCreateFromDefinition, typed(brepCreateFromDefinition))
	r.readOnly(wire.MethodBrepDescribe, typed(brepDescribe))
	r.readOnly(wire.MethodBrepList, brepList)
	r.readOnly(wire.MethodBrepDelete, typed(brepDelete))
}

// registerLightingHandlers wires the lighting-style, light, environment, and shadow methods
// (M16/F03 PBI-155, ADR-0026).
func (r *Router) registerLightingHandlers() {
	r.readOnly(wire.MethodLightingGetStyle, getLightingStyle)
	r.readOnly(wire.MethodLightingSetStyle, typed(setLightingStyle))
	r.readOnly(wire.MethodLightingListStyles, listLightingStyles)
	r.readOnly(wire.MethodLightingListLights, listLights)
	r.readOnly(wire.MethodLightingAddLight, typed(addLight))
	r.readOnly(wire.MethodLightingSetLight, typed(setLight))
	r.readOnly(wire.MethodViewGetShadows, getShadows)
	r.readOnly(wire.MethodViewSetShadows, typed(setShadows))
	r.readOnly(wire.MethodEnvironmentGet, getEnvironment)
	r.readOnly(wire.MethodEnvironmentSet, typed(setEnvironment))
	r.readOnly(wire.MethodEnvironmentListPresets, listEnvironmentPresets)
	r.readOnly(wire.MethodEnvironmentLoadImage, typed(loadEnvironmentImage))
}

// registerGraphicsHandlers wires the client/interaction graphics methods — the add-in
// overlay surface for drawing meshes, heatmaps, lines, markers and labels (M05-F05).
func (r *Router) registerGraphicsHandlers() {
	r.readOnly(wire.MethodClientGraphicsSet, typed(setClientGraphics))
	r.readOnly(wire.MethodClientGraphicsList, listClientGraphics)
	r.readOnly(wire.MethodClientGraphicsDelete, typed(deleteClientGraphics))
	r.readOnly(wire.MethodClientGraphicsSetVisible, typed(setClientGraphicsVisible))
	r.readOnly(wire.MethodInteractionGraphicsUpdate, typed(updateInteractionGraphics))
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

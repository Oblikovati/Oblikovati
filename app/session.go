// SPDX-License-Identifier: GPL-2.0-only

package app

import (
	"errors"
	"fmt"
	"strconv"
	"sync/atomic"

	"oblikovati.org/api/wire"
	"oblikovati.org/app/markingmenu"
	"oblikovati.org/app/options"
	"oblikovati.org/clientgraphics"
	"oblikovati.org/command"
	"oblikovati.org/event"
	"oblikovati.org/math"
	"oblikovati.org/model/bodyapi"
	"oblikovati.org/model/bom"
	"oblikovati.org/model/colorscheme"
	"oblikovati.org/model/compdef"
	"oblikovati.org/model/contentset"
	"oblikovati.org/model/display"
	"oblikovati.org/model/doc"
	"oblikovati.org/model/facetstore"
	"oblikovati.org/model/material"
	"oblikovati.org/model/sketch"
	"oblikovati.org/model/style"
	"oblikovati.org/persistence/dialogmemory"
	"oblikovati.org/persistence/userprefs"
	"oblikovati.org/persistence/viewstate"
	"oblikovati.org/renderer"
	"oblikovati.org/scene"
	"oblikovati.org/theme"
	"oblikovati.org/update"
)

// Session is the running application state and the seam tests drive synthetically.
// It owns the open documents ([doc.Workspace]), the command registry, the selection,
// the active interactive tool, and the event bus. A test (or the ImGui shell) drives
// it through Execute / the input methods (Click, PressKey…) — there is no GPU or
// window involved, so "operating the UI" is fully unit-testable (ADR-0014/0004).
type Session struct {
	workspace                 *doc.Workspace
	store                     doc.Store                // the workspace's persistence backend; nil for in-memory sessions
	sketchInference           *sketch.InferenceOptions // session inference prefs (M06-F10; nil ⇒ defaults)
	facetStore                *facetstore.FacetStore   // tolerance-keyed facet/stroke cache (M07 #293; lazy)
	transientBodies           *bodyapi.TransientBRep   // transient B-rep registry (M07 #628; lazy)
	commands                  *CommandManager
	bindings                  *Bindings              // keyboard shortcut + alias resolver (M05-F17)
	histories                 map[doc.ID]*docHistory // per-document transaction-event streams (undo/redo)
	viewState                 viewstate.Store        // per-user document view/camera persistence (nil ⇒ disabled)
	prefs                     userprefs.Prefs        // global user preferences (ViewCube show/lock/compass/size/…)
	prefsStore                userprefs.Store        // persists prefs to the user config dir (nil ⇒ in-session only)
	bus                       *event.Bus
	selection                 *Selection
	highlightSets             *HighlightSets // named, colored emphasis groups for add-ins (#157)
	tool                      *ToolInstance
	toolTxn                   toolTransaction  // the bounded transaction the active tool holds open (#1750)
	featureEditors            featureEditorSet // full-panel editors, assembled at the composition root (#1617)
	picker                    Picker
	regionPicker              RegionPicker          // resolves a box-select rectangle (nil ⇒ box-select disabled)
	boxSelect                 BoxSelection          // the in-progress rubber-band rectangle, if any
	zoomWindow                BoxSelection          // the in-progress Zoom Window rubber-band, if any (#913 N16)
	zoomWindowArmed           bool                  // the Zoom Window tool is armed: the next left-drag zooms
	constrainedOrbit          bool                  // the Constrained Orbit tool is active: left-drag turntables (#913 N10)
	steeringWheel             bool                  // the SteeringWheels radial nav menu is shown at the cursor (#913 N26)
	entityDrag                sketchDrag            // the in-progress direct drag of sketch entities, if any
	dimensionDrag             dimensionDrag         // the in-progress drag of a sketch dimension's label (#2017)
	placement                 sketchPlacement       // the in-progress drag-to-create press (#2014)
	placementFields           placementFieldState   // in-place dimension input for the shape being placed (#2014)
	formatModes               sketchFormatModes     // Format-panel creation modes (#2015)
	lastCursorSketchPoint     math.Point2           // last cursor position mapped into the sketch plane
	cloudMove                 cloudMoveDrag         // the in-progress interactive drag of a point cloud, if any (#645)
	cvEdit                    cvEditDrag            // the in-progress NURBS control-point drag, if any (M36-F03)
	cageEdit                  cageEditDrag          // the in-progress free-form cage-vertex drag, if any (#2048)
	relaxMode                 bool                  // Relax Mode: drag over/fully-constrained sketch geometry (#791)
	hudEnabled                bool                  // the 2D-sketch dynamic-input HUD is enabled (#790)
	sketchHUD                 sketchHUD             // the dynamic-input HUD's live typing state (#790)
	dimDrag                   dimDragState          // the in-progress drag of a drawing dimension's text/line
	selectOther               selectOther           // the in-progress Select Other cycle, if any
	viewHistory               viewHistory           // recorded views for Previous View (F5)
	selectionPriority         SelectionPriority     // biases no-tool picking (Edge/Face/Part) (#912)
	selectionFilterState      *SelectionFilterState // user-editable no-tool ambient filter + priority order (#1222)
	selectionFilterWindowOpen bool                  // the Selection Filter & Priority window is open (#1222)
	camera                    scene.Camera
	camTween                  cameraTween
	driveAnim                 driveAnimation
	sketchReturnCam           scene.Camera
	activeSketch              *sketch.Sketch
	showSketchConstraints     bool // Show/Hide Constraints: draw a marker per geometric constraint
	activeSketch3D            *sketch.Sketch3D
	pendingDim                *sketch.DimensionConstraint
	overlays                  []renderer.DrawItem
	hiddenBodyKeys            map[string]bool                  // scratch hidden-body set used only when no document is active
	hiddenBodyKeysByDoc       map[doc.ID]map[string]bool       // browser-driven body visibility, scoped per document (#1105)
	graphics                  *clientgraphics.Store            // scratch store used only when no document is active
	graphicsByDoc             map[doc.ID]*clientgraphics.Store // add-in client/interaction graphics, scoped per document (M05-F05)
	addins                    *AddInManager
	clientApps                *ClientApplicationRegistry        // external automation drivers (M05-F01)
	browserPanes              *AddInBrowserPanes                // add-in browser panes (M05-F03)
	dockableWindows           *AddInDockableWindows             // add-in dockable windows (M05-F03)
	taskPanels                *AddInTaskPanels                  // add-in modal task panels (M05-F03)
	appOptions                options.All                       // typed per-user option groups (M05-F11)
	optionsStore              options.Store                     // persists appOptions (nil ⇒ in-session only)
	statusText                string                            // wire-set status-bar message (M05-F09)
	messageCenter             *MessageCenter                    // sectioned errors/warnings tree (M05-F09)
	messageCenterOpen         bool                              // the Messages panel is open
	progress                  *ProgressLedger                   // live progress bars (M05-F09)
	balloonTips               *BalloonTipCenter                 // notification balloons (M05-F09)
	prompts                   *PromptCenter                     // declarative prompts (M05-F09)
	dialogMemoryStore         dialogmemory.Store                // persists suppressions + remembered answers
	miniToolbars              *MiniToolbarRack                  // in-canvas mini-toolbars (M05-F07)
	fileDialogQueue           []FileDialogRequest               // pending add-in file-dialog asks (M05-F08)
	webViews                  map[string]wire.WebDialogSpec     // presented web views (M05-F08)
	webViewOrder              []string                          // web views in creation order
	urlOpener                 URLOpener                         // platform URL opener (head-injected)
	windowFrame               WindowFrameStatus                 // mirrored host-window state (M05-F10)
	triad                     TriadGizmo                        // the move/rotate triad (M05-F13)
	manipulators              *ManipulatorBoard                 // add-in drag handles (M05-F13)
	helpSources               map[string]string                 // add-in help bases by source (M05-F14)
	helpInterceptor           HelpInterceptor                   // before-help veto hook (M05-F14)
	documentSubTypes          map[doc.SubTypeID]DocumentSubType // registered flavors (M05-F15)
	documentSubTypeOrder      []doc.SubTypeID
	addinEnvironments         map[Environment]string                           // registered add-in environments (M05-F16)
	activeAddInEnv            Environment                                      // the entered add-in environment (base when none)
	markingMenus              map[Environment]wire.MarkingMenuView             // radial menus per environment (M05-F12)
	markingMenuStore          markingmenu.Store                                // persists marking-menu customization (nil ⇒ session-only)
	contextMenus              map[string]map[string][]wire.ContextMenuItemSpec // add-in menu injections by kind
	lastCommandID             string                                           // the most recently invoked command, for right-click Repeat (#915 C5)
	classicContextMenu        bool                                             // right-click shows the classic linear menu instead of the radial marking menu (#915 C8)
	objectVisibility          wire.ObjectVisibilityView                        // View ▸ Object-visibility toggles
	cmdInput                  commandInput                                     // command-alias input box state (M05-F17)
	cmdLine                   *CommandLine                                     // Command Window REPL engine (M26)
	commandWindowHidden       bool                                             // Command Window docked panel hidden? (M26; inverted so zero ⇒ visible)
	placementStartedByClick   bool                                             // a geometry tool has its first MOUSE-placed point down (#2033)
	commandFocusWanted        bool                                             // a cancel/ESC asked to refocus the command input (M26; head clears it)
	commandTypeSeed           string                                           // char to seed the command input with when a bare key begins typing (#1751 S2; head clears it)
	grid                      *GridSettings
	themes                    *theme.Library
	themeStore                *theme.Store
	materials                 *material.Library
	materialStore             *material.Store
	recentDocuments           []string                          // recently opened/saved paths, most recent first (M04-F05)
	fileMetadata              map[doc.ID][]FileMetadataValue    // last save's PopulateFileMetadata harvest (M04-F05)
	notice                    string                            // last user-facing notice (e.g. a failed-commit reason)
	visualStyle               renderer.VisualStyle              // how the scene is drawn (View tab's Visual Style)
	lightingStyle             renderer.LightingStyleID          // active lighting preset (View tab's Lighting Style)
	lighting                  renderer.SceneLighting            // the live lighting rig (resolved from the style, then edited)
	colorSchemes              *colorscheme.Registry             // application color schemes — viewport bg + highlight/select palette (M16-F06 #642)
	colorSchemeRev            uint64                            // bumped on scheme/background change; the head re-applies the viewport colors (live preview)
	styles                    *style.Registry                   // document color styles + style-library cascade (M16-F02 #403/#408)
	displayOptions            display.Options                   // app-level display options that parameterize the display modes (M16-F07 #643)
	chamferFlatCorners        bool                              // default three-edge-corner treatment for new chamfers
	tangentChainSelect        bool                              // default: a fillet/chamfer pick selects the whole tangent chain (#1947)
	chamferConcaveOut         bool                              // default concave-edge strategy for new chamfers (true ⇒ outward fill)
	paramsDialogOpen          bool                              // the Manage ▸ Parameters dialog is open
	keymapEditorOpen          bool                              // the Tools ▸ Customize Keyboard panel is open (M05-F17)
	markingMenuEditorOpen     bool                              // the Tools ▸ Customize Marking Menu panel is open (REQ-005)
	lightingPanelOpen         bool                              // the View ▸ Lighting settings panel is open
	namedViewsPanelOpen       bool                              // the View ▸ Named Views panel is open (M16-F03 #404)
	colorStylesPanelOpen      bool                              // the Color Styles panel is open (M16-F02 #403/#408)
	displaySettingsOpen       bool                              // the Display Settings dialog is open (M16-F07 #643)
	unitsSettingsOpen         bool                              // the Document Settings ▸ Units dialog is open (#146)
	historyBrowserOpen        bool                              // the Edit ▸ History Browser window is open
	loadEnvRequested          bool                              // a "Load HDR…" was requested; the head opens the file dialog
	meshImportRequested       bool                              // a "Place Mesh…" was requested; the head opens the file dialog (#700)
	pointCloudRequested       bool                              // an "Import Point Cloud…" was requested; the head opens the file dialog (#645)
	pointCloudRenderDensity   float32                           // session viewport density percent for point clouds (100 = all points)
	pointCloudPointSize       float32                           // session viewport point size for point clouds, in native point pixels
	pointCloudIntensityLow    [4]float32                        // global low-end color for intensity-mode point clouds
	pointCloudIntensityHigh   [4]float32                        // global high-end color for intensity-mode point clouds
	pcIntensityHistograms     map[string]intensityHistogramMemo // ribbon intensity histograms memoized per cloud ResourceID (#645 perf); keyed by string, not *PointCloud, so a deleted cloud's samples are not pinned alive
	fitViewRequested          bool                              // an import added visible geometry; the head fits the camera once (#1645)
	scriptConsoleOpen         bool                              // the Manage ▸ Scripts ▸ Script Console panel is open
	addInCatalogueRequested   bool                              // a Get Started ▸ AddIn Catalogue was requested; the head opens the catalogue window
	preferencesRequested      bool                              // a Get Started ▸ Preferences was requested; the head opens the Preferences window
	capturePath               string                            // a requested viewport PNG capture path; the head writes it after render
	captureWindowPath         string                            // a requested whole-window PNG capture path; the head writes it after the frame composites
	normalDebug               bool                              // viewport normal-debug render (front green / back red); head reads each frame
	meshColors                bool                              // viewport mesh-debug-colors render (each face/triangle a distinct color)
	meshColorsPerTri          bool                              // when meshColors: color per TRIANGLE (else per B-rep face)
	editScope                 editScope                         // while editing a node, hide everything created after it (issue #132)
	asmBodies                 assemblyBodyCache                 // memoized world-space assembly bodies + their occurrences (#769)
	pickIndex                 *assemblyPickIndex                // BVH over placement AABBs for sub-linear ray picking (M34-F5)
	bomPanelOpen              bool                              // the Assemble ▸ Bill of Materials panel is open (#768)
	bomViewKind               bom.ViewKind                      // the BOM panel's selected view (structured / parts-only)
	updateCheckRequested      bool                              // Help ▸ Check for Updates was clicked; the head runs the (network) check
	pendingUpdate             *update.Result                    // last update-check outcome to show in the update window; nil = closed
	txEvents                  []sessionTxEvent                  // append-only transaction log since app open (for bug reports)
	txAudit                   map[doc.ID]*command.SnapshotLog   // per-document delta log the audit's recipes are reconstructed from (#1424)
	bugReport                 bugReportState                    // in-progress Help ▸ Report Bug capture+submit, if any
	bugOutcome                atomic.Pointer[bugResult]         // submit goroutine → frame loop handoff (session never touched off-thread)
	bugSubmitter              bugSubmitter                      // injectable reporting endpoint (DI; lazily defaults to real HTTP)
	addInCat                  addInCatalogue                    // Add-In Catalogue browse/install state (#1164)
}

// Notice returns the last user-facing notice (a failed commit's reason), or "" — shown in
// the status bar so a failed OK is not silent.
func (s *Session) Notice() string { return s.notice }

// SetNotice puts a transient user-facing message in the status bar — used by an add-in to
// surface state the user can't otherwise see (e.g. a collaboration add-in's connection
// status). Cleared on the next user input, like any notice.
func (s *Session) SetNotice(msg string) {
	s.notice = msg
	s.feedNotice(msg) // M26 F03: notices also appear in the Command Window
}

// VisualStyle returns the active scene visual style (default Shaded with Edges).
func (s *Session) VisualStyle() renderer.VisualStyle { return s.visualStyle }

// SetVisualStyle sets how the scene is drawn (the View tab's Visual Style).
func (s *Session) SetVisualStyle(v renderer.VisualStyle) { s.visualStyle = v }

// NewSession creates an empty in-memory session with no persistence store. Its
// documents live only in memory — Save/Open return "no store configured" — which
// is what the unit tests and headless synthetic sessions want. The windowed head
// and the CLI inject a real store via [NewSessionWithStore].
func NewSession() *Session { return newSession(nil) }

// NewSessionWithStore creates a session whose workspace persists through store, so
// File ▸ Open/Save and the CLI fixture commands read and write real .obk packages
// on disk. Pass a persistence.PackageStore from the binary (the app package depends
// only on the doc.Store interface, never on persistence — the DI rule).
func NewSessionWithStore(store doc.Store) *Session { return newSession(store) }

func newSession(store doc.Store) *Session {
	s := &Session{
		store:     store,
		workspace: doc.NewWorkspace(store, contentset.Default()),
		commands:  NewCommandManager(),
		histories: map[doc.ID]*docHistory{}, txAudit: map[doc.ID]*command.SnapshotLog{},
		bus:       event.NewBus(),
		selection: NewSelection(), selectionFilterState: NewSelectionFilterState(),
		camera:         scene.NewCamera(800, 600),
		hiddenBodyKeys: map[string]bool{}, hiddenBodyKeysByDoc: map[doc.ID]map[string]bool{},
		graphics: clientgraphics.NewStore(), graphicsByDoc: map[doc.ID]*clientgraphics.Store{},
		featureEditors: defaultFeatureEditors(),
		addins:         NewAddInManager(),
		clientApps:     NewClientApplicationRegistry(),
		appOptions:     options.Defaults(),
		visualStyle:    renderer.ShadedWithEdges,
		// Three Point is the out-of-the-box rig for every visual style: a studio
		// key/fill/back setup reads far better than the legacy single headlight now
		// that the whole rig lights every shaded mode (ADR-0026 §8).
		lightingStyle:           renderer.LightingThreePoint,
		lighting:                renderer.SceneLightingFor(renderer.LightingThreePoint),
		chamferFlatCorners:      true, // match Inventor's default flat three-edge-corner blend
		tangentChainSelect:      true, // match Inventor's tangent propagation: a pick grabs the tangent chain
		hudEnabled:              true, // dynamic-input HUD on by default, like Inventor (#790)
		chamferConcaveOut:       true, // concave edges fill the inside corner by default (outward)
		pointCloudRenderDensity: 100,
		pointCloudPointSize:     1,
		pointCloudIntensityLow:  [4]float32{1, 0, 0, 1},
		pointCloudIntensityHigh: [4]float32{1, 1, 0, 1},
	}
	s.initSurfaceCenters()
	s.seedVisualState()
	s.graphics.SetBodyResolver(s.resolveOverlayMesh) // scratch store (no active document)
	s.messageCenter.sink = s.routeMessage            // M26 F03: mirror message-center entries to the command line
	s.initShellSurfaces()
	s.wireDocumentWatchers()
	return s
}

// initSurfaceCenters wires the session's UI surface managers (add-in panes,
// message/progress/prompt centers, mini-toolbars, manipulators) — split from
// newSession to keep the composition root readable.
func (s *Session) initSurfaceCenters() {
	s.browserPanes = NewAddInBrowserPanes()
	s.dockableWindows = NewAddInDockableWindows()
	s.taskPanels = newAddInTaskPanels()
	s.messageCenter = NewMessageCenter()
	s.progress = NewProgressLedger()
	s.balloonTips = NewBalloonTipCenter()
	s.prompts = NewPromptCenter()
	s.miniToolbars = NewMiniToolbarRack()
	s.manipulators = NewManipulatorBoard()
}

// wireDocumentWatchers starts the session's background document and transaction watchers. Split out
// of newSession (like seedVisualState) to keep it short — one place for the watch wiring.
func (s *Session) wireDocumentWatchers() {
	s.watchDocumentCloses()
	s.watchDocumentSwitches() // #1105: drop the prior document's selection when a different one is activated
	s.watchDocumentInterests()
	s.watchNewDocumentProjection()      // a new document opens in the configured projection (#camera-ortho)
	s.watchDocumentIdentityCollisions() // open-time identity-GUID clash → reassign + notify
	s.watchTransactions()               // append-only transaction log for bug reports
	s.watchDrawingExport()              // Drawing tab Export DXF: write the sheet when its file dialog is answered
}

// seedVisualState seeds the M16 visualization registries (color schemes, color styles, display
// options/settings) and the default IBL environment. Split out of newSession to keep it short
// (one place for the visual defaults). The embedded Sky map is the default environment for
// every visual style — IBL plus the sky as the viewport background (ADR-0026 §8).
func (s *Session) seedVisualState() {
	s.lighting.Environment = renderer.DefaultEnvironment()
	s.colorSchemes = colorscheme.NewRegistry()
	s.styles = style.NewRegistry()
	s.displayOptions = display.DefaultOptions()
}

// initShellSurfaces seeds the M05 add-in UI-shell state: web views, the default
// radial marking menus, context-menu injections, and the object-visibility
// toggles (everything toggleable starts shown).
func (s *Session) initShellSurfaces() {
	s.webViews = map[string]wire.WebDialogSpec{}
	s.markingMenus = defaultMarkingMenus()
	s.contextMenus = map[string]map[string][]wire.ContextMenuItemSpec{}
	s.objectVisibility = wire.ObjectVisibilityView{
		WorkPlanes: true, WorkAxes: true, WorkPoints: true, Sketches: true,
	}
	s.helpSources = map[string]string{}
	s.documentSubTypes = map[doc.SubTypeID]DocumentSubType{}
	s.registerBuiltInSubTypes()
	s.addinEnvironments = map[Environment]string{}
}

// AddIns returns the add-in registry (ApplicationAddIns).
func (s *Session) AddIns() *AddInManager { return s.addins }

// ClientApps returns the external-client registry (ClientApplications).
func (s *Session) ClientApps() *ClientApplicationRegistry { return s.clientApps }

// Camera returns the active view's camera sized to the current viewport. Camera state is
// per-view — a document owns a Views collection and each view owns a camera — so switching
// documents or views shows that view's saved camera rather than resetting it. The
// transient viewport pixel size is carried on the session's cached frame; with no active
// document the cached frame is returned as a fallback.
func (s *Session) Camera() scene.Camera {
	if v := s.ActiveView(); v != nil {
		c := s.camera // carry the transient viewport pixel size (Width/Height)
		c.Eye, c.Target, c.Up, c.FOV = v.Eye, v.Target, v.Up, v.FOV
		c.Orthographic = orthoForView(v)
		return c
	}
	return s.camera
}

// SetCamera applies c to the active view (the per-view source of truth), caches the frame
// (for the no-document fallback and the pixel size), marks the view framed, and keeps the
// picker's view in sync so a click hit-tests against the current camera.
func (s *Session) SetCamera(c scene.Camera) {
	s.camera = c
	if v := s.ActiveView(); v != nil {
		v.Eye, v.Target, v.Up, v.FOV = c.Eye, c.Target, c.Up, c.FOV
		v.Framed = true
	}
	if ca, ok := s.picker.(interface{ SetCamera(scene.Camera) }); ok {
		ca.SetCamera(c)
	}
}

// ActiveView returns the active document's active view, or nil when no document is active.
func (s *Session) ActiveView() *doc.View {
	d := s.ActiveDocument()
	if d == nil {
		return nil
	}
	return d.Views().Active()
}

// EnterSketch activates a sketch for editing (the Sketch environment); ExitSketch
// leaves it. ActiveSketch returns the sketch being edited, or nil. Entering swings the
// camera to face the sketch plane head-on (remembering the prior view); exiting swings
// it back — the head ticks these transitions (TickCameraAnimation).
func (s *Session) EnterSketch(sk *sketch.Sketch) {
	s.activeSketch = sk
	// The contextual environment switched: add-ins re-aim their UI (M05-F12).
	event.Emit(s.bus, event.After, EnvironmentChanged{Environment: SketchEnvironment})
	s.emitSketchEdit(sk.Seq(), sk.Name(), true) // SketchEvents surface (#148)
	sk.Edit()
	s.beginEditScope(sk.Seq()) // hide features/datums created after this sketch while editing it
	s.sketchReturnCam = s.camera
	s.animateCameraTo(s.camera.Facing(
		sk.Plane().Origin(), sk.Plane().Normal().AsVector(), sk.Plane().YAxis().AsVector(),
	),
		sketchViewTweenSeconds)
}

func (s *Session) ExitSketch() {
	if s.activeSketch != nil {
		s.emitSketchEdit(s.activeSketch.Seq(), s.activeSketch.Name(), false) // SketchEvents surface (#148)
		s.activeSketch.ExitEdit()
		s.activeSketch = nil
		event.Emit(s.bus, event.After, EnvironmentChanged{Environment: BaseEnvironment})
		s.pendingDim = nil // no dangling edit box after leaving the sketch
		s.endEditScope()   // restore the rolled-back features (caller recomputes)
		s.animateCameraTo(s.sketchReturnCam, sketchViewTweenSeconds)
	}
}
func (s *Session) ActiveSketch() *sketch.Sketch { return s.activeSketch }

// Workspace/Commands/Events/Selection expose the session's subsystems.
func (s *Session) Workspace() *doc.Workspace { return s.workspace }
func (s *Session) Commands() *CommandManager { return s.commands }
func (s *Session) Events() *event.Bus        { return s.bus }
func (s *Session) Selection() *Selection     { return s.selection }

// PointCloudRenderDensity returns the viewport render density for attached scan points, as a
// percentage where 100 renders every eligible point and 0 renders none.
func (s *Session) PointCloudRenderDensity() float32 { return s.pointCloudRenderDensity }

// SetPointCloudRenderDensity sets the viewport render density for attached scan points. Values
// outside 0..100 are clamped because UI drags and future API callers share this setter.
func (s *Session) SetPointCloudRenderDensity(percent float32) {
	if percent < 0 {
		percent = 0
	}
	if percent > 100 {
		percent = 100
	}
	s.pointCloudRenderDensity = percent
}

// PointCloudPointSize returns the native point size used for attached scan points.
func (s *Session) PointCloudPointSize() float32 { return s.pointCloudPointSize }

// SetPointCloudPointSize sets the native point size used for attached scan points. Values outside
// 1..10 are clamped because UI drags and future API callers share this setter.
func (s *Session) SetPointCloudPointSize(size float32) {
	if size < 1 {
		size = 1
	}
	if size > 10 {
		size = 10
	}
	s.pointCloudPointSize = size
}

// PointCloudIntensityRamp returns the session-wide low/high colors used for intensity-mode point
// clouds. The ramp is viewport state, not persisted point-cloud metadata.
func (s *Session) PointCloudIntensityRamp() (low, high [4]float32) {
	return s.pointCloudIntensityLow, s.pointCloudIntensityHigh
}

// SetPointCloudIntensityRamp sets the session-wide intensity ramp. Alpha is forced opaque because
// point-cloud display modes select color, not point visibility.
func (s *Session) SetPointCloudIntensityRamp(low, high [4]float32) {
	s.pointCloudIntensityLow = normalizedPointCloudRampColor(low)
	s.pointCloudIntensityHigh = normalizedPointCloudRampColor(high)
}

func normalizedPointCloudRampColor(c [4]float32) [4]float32 {
	return [4]float32{clamp01(c[0]), clamp01(c[1]), clamp01(c[2]), 1}
}

// SelectionFilterState returns the user-editable no-tool ambient selection filter and priority
// order (the Selection Filter & Priority window, #1222). Mutating it re-pushes the priority
// order into the picker.
func (s *Session) SelectionFilterState() *SelectionFilterState { return s.selectionFilterState }

// ActiveDocument returns the workspace's active document, or nil.
func (s *Session) ActiveDocument() *doc.Document { return s.workspace.ActiveDocument() }

// Execute runs the command with the given id, firing CommandStarted (Before) and
// CommandEnded (After) and refusing a disabled command.
func (s *Session) Execute(id string) error {
	c, ok := s.commands.ByID(id)
	if !ok {
		return fmt.Errorf("app: no command %q", id)
	}
	if !c.IsEnabled(s) {
		return fmt.Errorf("app: command %q is disabled", id)
	}
	s.lastCommandID = id // the right-click Repeat target (#915 C5)
	event.Emit(s.bus, event.Before, CommandStarted{ID: id})
	err := c.run(s)
	event.Emit(s.bus, event.After, CommandEnded{ID: id, Failed: err != nil})
	return err
}

// LastCommandID returns the most recently invoked command's id and whether one exists — the
// target of the right-click "Repeat <command>" entry (#915 C5).
func (s *Session) LastCommandID() (string, bool) {
	return s.lastCommandID, s.lastCommandID != ""
}

// RepeatLastCommand re-invokes the most recently run command, if any and still enabled; with no
// prior command it is a no-op. This backs the marking menu's idle "Repeat" entry (#915 C5).
func (s *Session) RepeatLastCommand() error {
	id, ok := s.LastCommandID()
	if !ok {
		return nil
	}
	return s.Execute(id)
}

// ClassicContextMenu reports whether the viewport right-click shows the classic linear menu
// instead of the radial marking menu (#915 C8).
func (s *Session) ClassicContextMenu() bool { return s.classicContextMenu }

// SetClassicContextMenu chooses the right-click menu style (true = classic linear, false =
// radial marking menu).
func (s *Session) SetClassicContextMenu(classic bool) {
	s.classicContextMenu = classic
	if err := s.saveMarkingMenuCustomization(); err != nil {
		s.SetNotice("marking menu: " + err.Error())
	}
}

// ToggleContextMenuStyle flips between the radial marking menu and the classic linear menu.
func (s *Session) ToggleContextMenuStyle() {
	s.classicContextMenu = !s.classicContextMenu
	if err := s.saveMarkingMenuCustomization(); err != nil {
		s.SetNotice("marking menu: " + err.Error())
	}
}

// Invoke is the alias-driven entry (Inventor command alias): typing a command alias
// runs the matching command.
func (s *Session) Invoke(alias string) error {
	c, ok := s.commands.ByAlias(alias)
	if !ok {
		return fmt.Errorf("app: no command for alias %q", alias)
	}
	return s.Execute(c.id)
}

// ErrNoActiveDoc is returned by the save verbs when there is no active document to
// act on (an empty session).
var ErrNoActiveDoc = errors.New("app: no active document to save")

// ErrNeedsPath is returned by [Session.SaveActiveDocument] when the active document
// has never been saved — its name is not yet an .obk path (a freshly created
// "PartN"). File ▸ Save catches this and falls back to Save As to prompt for a
// destination.
var ErrNeedsPath = errors.New("app: document has no file path yet; use Save As")

// OpenDocument loads the package at path into the workspace and makes it the active,
// visible document. It is the core of File ▸ Open and the CLI open command.
//
//	d, err := session.OpenDocument("/models/bracket.obk")
func (s *Session) OpenDocument(path string) (*doc.Document, error) {
	d, err := s.workspace.Open(path, true)
	if err != nil {
		return nil, err
	}
	if part, ok := d.Content().(*compdef.PartComponentDefinition); ok {
		s.relinkPointCloudProvenance(part) // restore datum-cloud provenance links (#645)
	}
	s.documentHistory(d) // open the event stream now so the first edit's before-snapshot is the open state
	s.loadViewState(d)   // restore this user's saved camera/view layout (kept outside the .obk)
	s.rememberRecentDocument(path)
	return d, nil
}

// OpenComponentForPlacement loads a component document into memory so the active assembly can
// instance it, WITHOUT making it visible or active — placing a part must never switch the tab
// away from the assembly (Inventor's invisible Documents.Open). The document is reachable but
// shows no tab; the user opens it in a tab later via Edit/Open on the occurrence
// ([OpenOccurrenceDocument]).
func (s *Session) OpenComponentForPlacement(path string) (*doc.Document, error) {
	d, err := s.workspace.OpenWithOptions(path, doc.OpenOptions{Visible: false, Background: true})
	if err != nil {
		return nil, err
	}
	s.documentHistory(d) // track its edit stream so an Edit-in-tab later has a clean before-state
	return d, nil
}

// NewPart creates a realized part document with a unique "PartN" name and makes it active
// (the workspace activates a newly added document), so the viewport and ribbon switch to the
// part environment. It backs the New Part command on the ZeroDoc ribbon and the File menu.
//
//	d, err := session.NewPart()
func (s *Session) NewPart() (*doc.Document, error) {
	ev := FileNew{DocumentType: doc.Part}
	if out := event.Emit(s.bus, event.Before, ev); out.Vetoed() {
		s.notice = out.Reason
		return nil, &doc.VetoError{Operation: "new part", Reason: out.Reason}
	}
	d, err := compdef.AddPart(s.workspace, s.uniqueDocumentName("Part"), true)
	if err != nil {
		return nil, err
	}
	s.documentHistory(d) // open the event stream now so the first edit is undoable to the empty part
	event.Emit(s.bus, event.After, ev)
	return d, nil
}

// NewAssembly creates a realized assembly document with a unique "AssemblyN" name and makes it
// active (the workspace activates a newly added document), so the viewport and ribbon switch to
// the assembly environment. It backs the New Assembly command on the ZeroDoc ribbon (#762) — the
// counterpart of [Session.NewPart].
//
//	d, err := session.NewAssembly()
func (s *Session) NewAssembly() (*doc.Document, error) {
	ev := FileNew{DocumentType: doc.Assembly}
	if out := event.Emit(s.bus, event.Before, ev); out.Vetoed() {
		s.notice = out.Reason
		return nil, &doc.VetoError{Operation: "new assembly", Reason: out.Reason}
	}
	d, err := compdef.AddAssembly(s.workspace, s.uniqueDocumentName("Assembly"), true)
	if err != nil {
		return nil, err
	}
	s.documentHistory(d) // open the event stream now so the first placement is undoable to the empty assembly
	event.Emit(s.bus, event.After, ev)
	return d, nil
}

// uniqueDocumentName returns the first "<prefix>N" name not currently open, so New Part / New
// Assembly never clash with an open document.
func (s *Session) uniqueDocumentName(prefix string) string {
	taken := map[string]bool{}
	for _, d := range s.workspace.Documents() {
		taken[d.DisplayName()] = true
	}
	for n := 1; ; n++ {
		if name := prefix + strconv.Itoa(n); !taken[name] {
			return name
		}
	}
}

// SaveActiveDocument writes the active document back to its existing .obk path. A
// document that was never saved has no path yet — we detect this by the absence of
// the [doc.PackageExtension] suffix, since new documents are minted with bare names
// like "Part1" — and return [ErrNeedsPath] so the UI can prompt via Save As.
func (s *Session) SaveActiveDocument() error {
	return s.SaveDocument(s.workspace.ActiveDocument())
}

// SaveActiveDocumentAs writes the active document to path, which becomes its new
// identity. It is the core of File ▸ Save As and the CLI save-as command.
func (s *Session) SaveActiveDocumentAs(path string) error {
	return s.SaveDocumentAs(s.workspace.ActiveDocument(), path)
}

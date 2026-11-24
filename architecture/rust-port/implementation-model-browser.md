# Native model browser

The Model panel reads the active document owner on each GUI snapshot. It shows
the document name, origin geometry, parameter values, unconsumed sketches, and
the authored feature program. A sketch with exactly one consumer appears below
that feature. A sketch with several consumers remains at top level, as in Go.
The model supplies consumers in program order; the browser
and canonical sketch query use that same relation. Names do not identify rows.

Each sketch row carries its stable document-sketch ID and the canonical
`sketch.get` inspection: current DOF, health, plane, entity count, and edit state.
An inspection failure stays on its row. It does not remove the sketch or stop
other rows from appearing. Labels containing `##` remain literal visible names.
Invalid duplicate IDs, missing consumer IDs, and NUL text are refused before
widget submission.

A double-click sends the document and sketch IDs to the existing owner edit
operation. The same operation remains available through script and add-in owner
wrappers. The owner checks document focus, sketch continuity, plane resolution,
and transaction state. Entry does not author a feature or add an undo step.
The selected row represents the actual current sketch edit. A single click does
not create a separate model selection in GUI state.

The GUI frame test drives a double-click and checks the exact action IDs. Host
tests cover duplicate names, invalid and inactive document IDs, edit-state
refresh, single and multiple feature consumers, program order, and unchanged history.
The native authoring test reopens its rectangle through browser pointer events
and compares the model pixels before and after entry. It requires visible
committed sketch pixels before that comparison.

Linux live acceptance also used XTest mouse and keyboard input with the real
MCPBridge loaded. It created a Part, parameter, and constrained rectangle;
double-clicked the browser sketch; and finished the edit. Read-only bridge
queries confirmed the reopened sketch was editing, healthy, and at zero DOF.
MCP window captures showed the browser and rectangle. Parameter edits and
undo/redo then passed. The application, bridge listener, and Xvfb process were
closed after the run.

This is an initial browser port. Origin rows are informational. Whole-model
selection, viewport selection synchronization, context menus, feature health
badges, visibility controls, timeline interleaving, and complete Go browser
behavior remain to be ported. The general Extrude command and cube UI journey
also remain incomplete. Issue #3314 tracks feature diagnostics in the browser.

The pinned browser selection tests derive their deselection click from the
current viewport bounds and verify that its view ray misses the solid. The
[native receipt](receipts/2026-11-24-browser-viewport-deselection-3896/README.md)
covers clearing a nested sketch, clearing a Find result, and toggling an
origin plane with another browser docked on the right.

The published nested-sketch and grouped Find cases use the saved
`workspace-layout-native.opd` Part and read its actual browser rows.
Both cases also pass on the pinned fixture. See the
[native fixture receipt](receipts/2026-11-24-model-browser-named-part-3896/README.md).

The representation and model-state tests retain the named blank metric
Part on the published source. They locate the actual document captions
and check the empty workspace within the current viewport bounds. See
the [empty Part receipt](receipts/2026-11-24-model-browser-empty-viewport-3896/README.md).

Six additional pinned cases passed with the same viewport check, covering
edit scope, model-state duplication, view copying, camera autosave,
annotation scale, and saved camera restoration. See the
[native runs](receipts/2026-11-24-empty-browser-state-view-3896/README.md).

The origin and extrusion hierarchy cases use the saved Part on the
published source. They read the feature and sketch labels from the
loaded document. The [native receipt](receipts/2026-11-24-model-browser-hierarchy-fixture-3896/README.md)
also covers X11 deselection with the Appearance browser docked.

The [Favorites and search runs](receipts/2026-11-24-model-browser-favorites-dock-3896/README.md)
exercise the dock icons through native pointer input. The Favorites case
also checks that switching panes leaves the solid in the same view.

The [origin navigation and Favorites viewport runs](receipts/2026-11-24-browser-origin-and-favorites-fixture-3896/README.md)
keep the named block active on the published source and preserve keyboard
highlighting and the Part view.

The viewport click helper works in empty Parts as well as solid Parts.
The [pinned origin visibility runs](receipts/2026-11-24-browser-origin-visibility-3896/README.md)
check plane toggles and view switches while a view is locked.

Nine [published view cases](receipts/2026-11-24-browser-view-fixture-3896/README.md)
open the saved blank metric Part and exercise its `BrowserView` copy through
the Model Browser. The cases cover annotation scale, camera restoration,
publication, and origin visibility with locked or bulk visibility commands.

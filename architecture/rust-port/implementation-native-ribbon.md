# Native document ribbon

Side-docked tabs reserve enough width for their full labels. The command
groups start beside that measured tab column, within the existing ribbon
bounds.

Ribbon Customization selects the clicked command in either list on a
secondary click. The selected row can then be transferred with the
list's transfer control. The [native customization runs](receipts/2026-11-24-native-customization-transactions-3856/README.md)
also verify applying and cancelling drafts across ribbon targets.
The [native list and selector runs](receipts/2026-11-24-ribbon-customization-lists-3856/README.md)
cover target order, selection, selector dismissal, and tab retention.
The [customizable Zoom commands](receipts/2026-11-24-ribbon-customization-zoom-3856/README.md)
use the same viewport navigation handler as the navigation bar.
The source list includes 2-D Drawing as a configurable entry. The ascending
source-list scenario also expects an unlabeled row whose command is unidentified.
The Large and Text headings align with their checkboxes across font widths.
Home and End select the first or last staged row when the target list has focus.
The [native edit and import scenarios](receipts/2026-11-24-ribbon-customization-edits-3856/README.md)
cover reordering, repeated separators, committing with OK, and keeping a
draft across target and dialog tab switches. Invalid XML closes the picker,
shows the error in Customize, and preserves the draft.
If the file is missing, acknowledging the warning returns focus to the
selected filename. Cancelling the picker leaves Customize open with Apply
disabled. [Native missing-file scenario](receipts/2026-11-26-ribbon-missing-import-3856/README.md)
The [native selection scenarios](receipts/2026-11-24-ribbon-customization-selection-3856/README.md)
cover source selection across target and category changes, separator-only
drafts, and per-target display options.
Secondary-click selection transfers keyboard focus to the clicked
command list. Ctrl+A then selects rows in that list.
[Native focus checks](receipts/2026-11-26-ribbon-secondary-focus-3856/README.md)
Removing a staged range preserves the next surviving command's
selection, so repeated Remove clicks continue in order.
[Native removal checks](receipts/2026-11-26-ribbon-removal-next-3856/README.md)
Escape from the open category list discards the staged edit. The pinned
and published scenarios continue into an unrelated Measure sequence.
[Source defect and native results](scenario-defects/ribbon-category-escape-measure.md)
The [native command presentation scenarios](receipts/2026-11-24-ribbon-customization-presentation-3856/README.md)
check small and large captions, large icon-only buttons, and a divider
between two small command icons.
The [native keyboard transfer scenario](receipts/2026-11-24-ribbon-customization-keyboard-transfer-3856/README.md)
checks Enter in the source list and Tab followed by Space on Add.
The [native list navigation runs](receipts/2026-11-24-ribbon-customization-list-navigation-3856/README.md)
cover paging staged rows, selecting all visible source rows, and keeping a
staged ribbon edit while viewing Keyboard customization.
Page Up and Page Down in the available-command list first select the
visible edge. Pressing the key again from that edge advances the page.
[Native source paging checks](receipts/2026-11-26-ribbon-visible-page-3856/README.md)
Reopening Customize clears the Keyboard search and restores both filters
to All. The selected customization tab is retained.
[Keyboard initial-state checks](receipts/2026-11-26-keyboard-customization-initial-3856/README.md)
Copy to Clipboard exports the full Keyboard draft in the command list's
alphabetical order, even with a search active.
[Native clipboard checks](receipts/2026-11-26-keyboard-copy-3856/README.md)
The [native entry-point runs](receipts/2026-11-24-ribbon-customization-entry-points-3856/README.md)
cover Tools Options and the View and Sketch title menus. They also check
that opening and closing Customize preserves a selected solid face.
The [native file chooser runs](receipts/2026-11-24-ribbon-customization-draft-files-3856/README.md)
check that canceling Import or Export preserves pending edits. Export
writes the applied ribbon settings and leaves the draft open in Customize.
The [native import runs](receipts/2026-11-24-ribbon-customization-import-drafts-3856/README.md)
check that a file replaces pending edits on Tools and Manage while
preserving the selected target. Cancel restores the applied settings.
The grouped Plane menu opens the native Offset Plane dialog from its
Offset from Plane entry. The entry uses the same availability check and
action as the main Plane button. The menu presents the construction choices
with icons and dividers between the offset, reference, and orientation groups.
Ungrouping shows Offset from Plane as its own button. When Work Features
has collapsed to fit the ribbon, regrouping keeps the panel compact
until a ribbon width change triggers a new measurement.
Shell's Button Size menu switches the main-panel command between
Standard and Large. Its setting follows the ribbon profile, including
XML import and export. Global Text Off still hides its caption.
Shell can also be added to Quick Access from its main-panel menu.
The shortcut uses the same availability check and action as the
ribbon control, and Quick Access owns its removal.
The Marking Menu tab stages the Classic Context Menu preference.
Apply saves it with any changed menu assignments in a single write,
and Cancel discards the draft.
With Sketch1 in edit, Customize shows the configured 2D Sketch marking
menu positions and saved presentation settings. Closing keeps the sketch
in edit with the same geometry.
[Native initial-state runs](receipts/2026-11-26-marking-menu-initial-3856/README.md)
The radial menu can show icons and captions, icons only, or captions
only. Full Menu includes applicable canvas command rows. Short Menu
includes the configured overflow rows. Radial Menu Only shows the
configured radial buttons. Classic continues to show its full linear menu.
Both command lists resize with the floating Customize window without
losing staged edits. Reopening restores its default size and keeps its
position within the visible workspace.

The Rust shell uses a fixed ribbon band across the window instead of the temporary
floating command palette. The File menu and quick-access controls remain available.
The model browser remains below the band.

The source references are Go `app/ribbon_key.go`, `app/ribbon.go`,
`app/commands_standard.go`, `app/commands_sketch.go`, and
`head/ui/chrome_ribbon.go` at `aa8297d1`. The Rust implementation uses the canonical
API ribbon keys. The desktop owner supplies the actual document type with each GUI
snapshot; the GUI does not infer that type from a document name or sketch geometry.

No open document selects ZeroDoc and its Get Started / Launch commands. A Part
provides Create & Modify / Sketch, Manage / Parameters, and View / Navigate. Sketch
editing adds Sketch / Create and Exit. Assembly snapshots select the Assembly key
and the available shared Manage and View controls; Drawing, Presentation, and
unresolved document types retain their own keys and available View controls.

The contextual Sketch tab is selected on sketch entry. Sketch exit selects the
base tab. Manual tab choices persist between environment transitions, including a
switch between documents of the same kind. The selection state belongs to one GUI
owner, rather than the Go renderer's process-global previous environment.

All command buttons call the existing typed host actions. Parameters and document
path dialogs retain their existing owner operations. The temporary palette is
removed; it is not a second command route.

Issue #3798 tracks this work. Verification includes canonical document-key tests,
context transition and manual-selection tests, actual ImGui pointer and keyboard
input, high-DPI full-band input capture, native document/assembly snapshots, and the
native rectangle edit/save/reopen journey. The browser camera/pixel regression
compares the model region below the contextual ribbon, whose contents must change.

The supported panels use [measured collapse and flyouts](implementation-native-ribbon-overflow.md)
when the window is narrow. That implementation records its own verification scope.

Full Go parity remains open: icon assets, the complete command catalogue,
ribbon customization, add-in command registration, document types' own unported
tools, and a content-area viewport layout. The Extrude UI and complete
cube journey also remain open.

## Sketch row context menu

The model browser provides the Go-style right-click Edit Sketch entry (#3799).
It sends the same typed owner action as a double-click. Document and sketch IDs
scope the popup; labels do not establish identity. Availability is checked on
each frame, including transaction or document changes while a popup is open.
Dismissing a menu sends no action. Visibility, rename, selection synchronization,
and the remaining Go browser menu entries remain separate port work.

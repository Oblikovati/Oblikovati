# Native document ribbon

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

Full Go parity remains open: icon assets, the complete command catalogue, panel
collapse/flyouts, ribbon customization, add-in command registration, document types'
own unported tools, and a content-area viewport layout. The Extrude UI and complete
cube journey also remain open.

## Sketch row context menu

The model browser provides the Go-style right-click Edit Sketch entry (#3799).
It sends the same typed owner action as a double-click. Document and sketch IDs
scope the popup; labels do not establish identity. Availability is checked on
each frame, including transaction or document changes while a popup is open.
Dismissing a menu sends no action. Visibility, rename, selection synchronization,
and the remaining Go browser menu entries remain separate port work.

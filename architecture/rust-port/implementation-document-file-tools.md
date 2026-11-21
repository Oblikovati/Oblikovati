# Document file tools

The File menu uses the current document owner for New Part, Open, Save, Save As,
and Close. Open and Save As display an owned path draft. Accepting a draft sends
the canonical API request through the typed document client. The GUI performs no
file reads or writes. An unsaved Save request opens the same path dialog through
the owner's typed `SavePathRequired` notice, including a request received before
the first GUI frame.

The path field receives keyboard focus when the dialog appears. Enter submits
a nonempty path; Escape and Cancel close the draft without a file operation.
Changing documents clears a document-bound Save As draft. Open remains available
without an active document. Save As and Close keep the selected document ID and
are disabled during an open transaction. NUL text is refused before widget
submission, and Unicode paths remain intact in the canonical request.

Close sends `force: false`. Unsaved changes cause a local error and preserve the
document. Failed Open and an unsupported saved-package version also preserve
the current document and its history. File operations use the existing save
binding, atomic package publication, document restore, and close guards.

The GUI frame tests exercise menu choices, keyboard submit, mouse and keyboard
cancellation, empty paths, document switches, and canonical request fields. They
drain pending input, layout, and navigation transitions before typing. Host
tests use the named memory-file adapter to check save/close/reopen and refusal
paths. The native authoring test creates and edits the parameter-driven
rectangle, cancels a Save, saves it, closes its document, and reopens it through
menu and text input. It verifies the restored dimensions and zero DOF. A second
native test covers the Save notice before GUI initialization.

Linux live acceptance uses OS mouse and keyboard input with the real MCPBridge
loaded. It creates the Part and rectangle, changes `size` from 10 mm to 20 mm,
uses Undo and Redo, saves a package, closes its document, and reopens it through
the File menu. It repeats Open in a fresh desktop. Read-only MCP queries confirm a healthy zero-DOF sketch,
`size = 20 mm`, and both dimension expressions still equal to `size`. MCP PNG
captures show the dialog and restored rectangle. Window-close protocol requests
retire the saved desktop and add-in normally. Each isolated Xvfb server closes
after its driver exits.

The initial OS driver used Xlib symbol-name lookup for literal punctuation.
Xlib returned `NoSymbol` for `/`, `-`, and `.`, so the entered name lost those
characters. The driver now uses character keysyms for single characters and
rejects `NoSymbol`. The two malformed test files were removed, and the corrected
run published the exact selected path. No native text-input fallback was added.

These are path-entry dialogs. Directory browsing, recent-file menus, complete
ribbon parity, and the general Extrude UI remain open. The production profile
prerequisite is tracked in Martensite #390.

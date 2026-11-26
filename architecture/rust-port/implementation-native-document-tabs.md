# Native document tabs

Issue #3800 ports the document strip from Go
`head/ui/chrome_doctabs.go` at `aa8297d1`. The strip is below the ribbon and to
the right of the model browser. It reads each document's name, visibility, dirty
state and active identity from the file owner.

Only visible documents receive tabs. Session IDs identify the tabs; duplicate
names and names containing `##` remain literal labels. Dirty documents show an
asterisk. External activation selects the owner's tab once. Passive frames do
not send activation commands. A refused command restores the owner's selection.

The GUI sends the canonical `ActivateDocumentArgs` through the existing typed
`documents.activate` client. The desktop owner refuses hidden or missing targets,
active sketch edits and transactions in either document before activation. The
existing scene, browser, ribbon and parameter snapshots then follow the active
owner. Script and add-in wrappers forward the same host operation.

A tab's close button sends the existing `CloseDocumentArgs` with `force = false`.
The file owner's save guard decides whether close can proceed. Refusal retains
the document; the tab strip does not discard history or save implicitly. Save and
Save As remain available through the existing File tools.

The focused evidence covers real ImGui pointer input, duplicate names, literal
labels, external activation, hidden/disabled tabs, guarded close requests, owner
activation, edit and transaction refusal, and the unsupported owner reply. The
native regression suite also checks camera isolation, model pixels, resize,
keyboard commands and browser editing. The full local gate and real MCPBridge
mouse journey are required before publication.

The native Home preview and the pinned tab scenarios use the same pixel check
below the rendered thumbnail caption. The 2× placement test gives the Part tab
room in a saved workspace with the asset panels closed. See the
[native tab preview checks](receipts/2026-11-24-document-tab-previews-4078/README.md).

The [current saved-tab cases](receipts/2026-11-25-document-tabs-saved-parts-3900/README.md)
use disposable copies of PartOne and PartTwo. Inactive Properties reads
the actual file name, location and size, and both tab close controls
preserve the saved bytes.

Native Ctrl+Tab and Ctrl+Shift+Tab preserve sketch editing through Home
and wrap between Home and the last numbered window. The
[Wayland and X11 runs](receipts/2026-11-26-document-tabs-wayland-keys-3900/README.md)
verify the shortcut in both directions.

Native tab reordering keeps the active drawing visible and activates an
inactive Part when it is dragged, while keeping the drawing open.
[Current Document Tabs runs](receipts/2026-11-26-document-tabs-current-wayland-3900/README.md)
include the Wayland reorder trace and full reports for both backends.

Dragging a drawing out of the strip opens a native window. The main
window activates another attached document when one is available and
shows Home when the detached drawing was the only attached document.
The [native Wayland detachment](receipts/2026-11-26-document-tabs-wayland-detach-3900/README.md)
keeps the drawing toplevel mapped while the main window switches between Parts.
Right-click opens the detached tab menu, and Escape dismisses it. Clicking
the detached sheet updates the shared ribbon and browser, and selecting a
Part restores its controls. [Native drawing context and menu](receipts/2026-11-26-document-tabs-wayland-drawing-context-3900/README.md)
Detached drawing and sheet preview tests take the same ImGui context
lease as the main frame owner. [Parallel test results](receipts/2026-11-26-gui-context-lease-4078/README.md)

Remaining Go parity includes per-document camera restoration, document-close
confirmation choices, docking and a dedicated content-area viewport layout.
These tabs do not complete the Extrude or cube journey.

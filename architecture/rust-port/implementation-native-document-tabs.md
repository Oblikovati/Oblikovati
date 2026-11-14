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

Remaining Go parity includes per-document camera restoration, document-close
confirmation choices, docking and a dedicated content-area viewport layout.
These tabs do not complete the Extrude or cube journey.

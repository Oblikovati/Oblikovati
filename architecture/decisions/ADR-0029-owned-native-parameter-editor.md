# ADR-0029: Owned native parameter editor

Date: 2026-10-14. Status: Proposed for the Rust target design.
Tracking: [#3563](https://github.com/Oblikovati/Oblikovati/issues/3563),
[#3630](https://github.com/Oblikovati/Oblikovati/issues/3630).
Boundary: the existing internal document owner and native GUI frame loop.

## Problem and deletion gate

The Go Parameters window supports model and user rows, numeric formulas,
text and Boolean values, creation, row commands, value lists, tolerances,
groups, and derived tables. The native Rust command shell does not expose
these controls. The source is Go revision
`aa8297d1b0ac33c024f19e54821536d76e8a447a`, `head/ui/parameters_window.go`,
`head/ui/parameters_row.go`, and their child editors.

Replace the empty parameter GUI seam with one panel on the existing Dear ImGui
context and Martensite submission path. Do not introduce a separate renderer,
GUI document model, or string-keyed mutation path. Complete all baseline panel
controls before accepting the editor port. The full application port remains
open until its independent milestone proofs pass.

## Proposed decision

The document owner publishes an immutable parameter view only while the panel
is open. Each row contains its document identity, model parameter identity,
and the canonical Apache API `ParameterDetail` presentation contract. Model
identity binds the widgets even after a rename, search, or document switch.
The GPL GUI packet is internal and does not extend the public host API.
The view calls the shared typed detail projector directly. The public JSON
contract omits zero-valued fields, including negative zero. That encoding is
not an internal model view. Row comparisons and matching tolerance responses
keep float bits so a sign-only change requests a new frame (#3614).

Widgets emit owned actions. The document owner validates the active document
and parameter identity before it enters the existing session command path.
That path owns undo, recompute, identity reservation, exposed properties, and
observer notification. Text and Boolean edits use their corresponding model
operations. The existing numeric-only wire Set operation keeps its contract.
No add-in imports the GPL GUI packet or sends raw JSON.

Text edits retain their authored value until the owner returns a result with
the same target, request identity, and attempted value. A failed edit remains
visible and is not retried automatically. Successful edits reseed from the
committed model view. Creation fields clear only after the matching successful
result. Document changes and deleted rows retire their old widget state. Dialog
controls also scope their Dear ImGui identities by document and parameter;
clearing Rust buffers alone does not retire an active C widget's input state.
Creation, value-list, and tolerance editors share one checked request lifecycle.
Clean cells may be reseeded to keep the retained-cell budget bounded; dirty,
pending, and failed cells are protected. Strings are checked before C widgets
receive them. A local edit failure does not terminate the native window.

Metadata changes and owner results request a new logical GUI frame. They do
not replay command actions or require a fixed number of warm-up frames.
Copied draw packets remain valid after the GUI context advances.

## Evidence and remaining acceptance

Focused tests prove numeric, text, Boolean, rename, comment, Key, Export,
creation, Copy to User Parameter, Delete, value lists, undo, stable identities,
stale requests, and dependency refusal. List edits preserve authored order and
custom-value policy. Clearing a list keeps its current value. Parameter ID zero
is a valid editor target. Real widgets prove Unicode creation, multiline list
input, and text commit on focus loss. Failed list selections stay visible after
the combo closes, and pending selections cannot submit another edit. Real menu items prove that Copy remains available
for read-only rows and Delete is disabled for them. Software Vulkan and AMD
native tests prove panel opening, metadata publication, and local result
rendering through the shared native GUI path.

Tolerance tests cover working-unit deviations, all four model-value choices,
angle and unitless parameters, undo, exact double-precision input, signed zero,
and active-widget isolation across document and parameter changes. The JSON
zero-omission test remains unchanged; only the internal GUI view avoids that
serialization boundary.

Group membership controls now create or reuse a keyed group, preserve other
memberships, and remove all memberships without deleting parameters or groups.
Existing group display names and client IDs remain unchanged. Tests prove undo,
Unicode input, read-only row menus, and active-input isolation. The separate
group management journey remains part of the wider UI port.

The linked-table section now renders canonical table identities, source names,
linked names, and health. Delete uses the existing typed document host and
refuses stale document or table identities. Undo restores the same table and
parameter identities. Deletion invalidates surviving readers, as the Go and
API paths specify; failed values carry a visible warning and retain their
last-good text. Feature-owned tables refuse direct deletion.

The source-document link picker projects each other open part or assembly with
healthy numeric user parameters. It retains stable document and parameter
identities until submission, then resolves those identities again before it
calls the existing typed add-table client method. Empty selections create the
same empty table that the Go client creates. Tolerance text uses the document formatter and display units without changing
stored working-unit deviations. Invalid metadata retires stale edit targets
and appears as a local view issue; corrected metadata restores the panel.
Repeated identical issues do not force idle redraws. The child editors need foreground interaction and screenshot proof.
The full local `make gate` passed with all 13 crates above 80% line coverage
and 0.92% duplication. The whole editor still needs a foreground
edit/save/reopen test and screenshot inspection through the MCP bridge.
These focused checks do not prove completion of the editor or R1V/R4.

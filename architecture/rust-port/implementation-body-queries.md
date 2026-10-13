# Current native body enumeration — partial R7

Tracking: [port epic #3563](https://github.com/Oblikovati/Oblikovati/issues/3563).
Source behavior: Go `addin/router/bodies.go`, `bodyListResult` and `bodyInfo`.

The desktop now serves the existing API-owned `body.list` contract. The file host
injects one `PartTopologyAccess` capability for enumeration and assignment
verification. A file-only host refuses enumeration without this native owner.
No DTO, method name, or geometry implementation is copied into automation.

Enumeration borrows current named results in feature replay order. Each returned
key is checked against document identity, history ownership, generation, and the
exact retained body. Failed and suppressed features have no current body to list.
Indices and default `SolidN` names describe list order; they never define identity.
Queries can prepare a missing current result, but do not edit authored history or
consume a pending window update. Rendering and queries share result storage.

Face, edge, vertex, shell, and wire counts use unique native B-rep entities. They
do not count repeated oriented uses or read tessellation. Stored body names and
legacy color-style names are exposed with each body. Names come from the live
document snapshot; legacy style names remain in the retained package. Invalid
metadata is an error. The material ID uses the current body override or part
assignment. This field reports authored intent, not resolved material properties.

Tests cover plain and holed native prisms, exact counts, stable keys after parameter
edits, failed-feature removal, query-before-render, assignment through a listed
key, undo/redo, saved display metadata after reopening, and missing owner errors.
The full gate includes the existing per-face OCCT corpus and degenerate inputs.
Kernel recognizers, tolerance constants, fallback sites, and type assertions have
no net increase. No geometry algorithm changes in this increment.

This capability covers the retained sketch-prism results. General feature
families, face enumeration, legacy-key migration, lost-reference health, and full
appearance rendering remain open. Foreground
MCPBridge acceptance and full R0–R9 acceptance remain open; this is not a full port.

## Body rename, 2026-10-13

The file host now serves the existing API-owned `body.rename` request and
`BodyInfoResult`. It resolves the supplied index against the current native body
list before making an edit. The name is stored under the returned persistent key,
not under the index. An empty name removes the override and returns the current
`SolidN` default. Whitespace and Unicode are preserved, as in Go. No new API DTO,
method string, or geometry path is introduced.

Body names are now owned document snapshot metadata. They participate in command
history, undo/redo, transaction abort, dirty state, load, and atomic save. The
`bodyNames` package section keeps its existing schema. Unchanged names and legacy
empty/null entries remain intact; renamed entries do not alter body color styles,
model content, or the model-only recipe digest. Malformed name entries are refused
before a document is registered. Read-back uses current snapshot names, so undo
cannot expose the retained package's obsolete name.

Source contract: Go `addin/router/bodies.go` (`bodyRename`),
`addin/router/body_rename_test.go`, and `model/doc/document.go` (`SetBodyName`).
Tests use the typed client and the real native topology owner, with a named fake
filesystem. They cover full reply metadata, clearing, undo/redo, explicit abort,
recompute, failed-feature index changes, save/reopen, rejected indices, missing
owners, and save failure followed by retry. The native body key remains stable;
no coordinate matching or index-based identity is used.

This ports the rename API for current body results. It does not add a body browser
UI, unsupported feature families, or legacy-key migration.
General profile construction and the full port remain open.

Unchanged renames follow the existing Rust session policy in
[the history implementation](implementation-history.md#automatic-commands-and-unchanged-groups--partial-r2):
they add no undo step and do not make a clean snapshot dirty. Go's `SetBodyName`
marks the document dirty even when the name does not change, while its snapshot
history adds no step. This dirty-flag difference is explicit; exact parity for
that Go side effect is not claimed. The Rust route keeps the shared command and
snapshot policy rather than adding a body-specific dirty flag.

Validation on local Linux: all 13 body-owner tests, six model snapshot tests,
and four body-name codec tests pass. Go's four `TestBodyRename*` tests pass at
`aa8297d1b0ac33c024f19e54821536d76e8a447a`; an isolated test overlay confirms
that an unchanged Go rename sets dirty without adding an undo entry. The
matching Rust test verifies its documented snapshot policy and preserved redo.

The final `make gate` passed for the complete discovered 13-crate workspace,
including native display tests on Xvfb/lavapipe, build, lint, tests, coverage,
documentation, SPDX, and duplication checks. The lowest per-crate line coverage
is 91.84%; duplication is 0.71%. The first run reached documentation checking
and rejected an extra blank line; that was fixed before the successful run.
No push or PR is part of this increment. Full R0–R9 acceptance remains open.

## Body visibility, 2026-10-13

The existing `body.setVisible` route now resolves its index against the current
native owner and changes document-local view state. `body.list` and the command
reply report the same flag. As in Go `app/render.go`, hidden-body keys belong to
the open document session: they survive activation changes, recompute, and
modelling undo, but are released on close. They are not saved in the package and
do not create undo entries or change dirty state. Explicit transaction abort
does not undo a visibility edit. Hidden bodies remain available to exact queries
and whole-part mass analysis.

A separate checked display revision invalidates the scene after a visibility
change while the modelling generation stays unchanged. Repeated requests for the
same flag do not trigger a new scene. The viewport filters bodies before mesh
creation through one presentation pipeline; hidden bodies do not produce mesh
failures. Feature health still reports all evaluated features. A failed visibility
lookup is an error, not a successful empty scene. Default presentation callers
use the same pipeline with all bodies visible.

Owner tests cover hide/show replies, complete scene replacement, stable modelling
generation, no dirty changes, invalid indices, activation, recompute and index
changes after a failed feature, undo, save/close/reopen with the same persistent
key, and mass queries. Session tests cover idempotence and display-revision
exhaustion without mutation. Viewport tests prove that hidden bodies are not
meshed and that a visibility error reaches the caller.

A real native-window test on Xvfb/lavapipe opens a stored part through the typed
API, verifies body pixels, hides the body, verifies no body pixels, then shows it
and compares the entire captured frame byte-for-byte with the original. The
capture is also checked against the presented X11 window pixels. This is local
native display evidence, not foreground MCPBridge or full R0–R9 acceptance.

The Go body-visibility route and document-scope tests pass at baseline
`aa8297d1b0ac33c024f19e54821536d76e8a447a`. The Rust body-owner suite passes
18 tests, including the rename regression tests. The final full local gate
covers all 13 crates, with native Linux display tests, formatting, lint, build,
coverage, documentation, SPDX, and duplication checks. No modelling algorithm,
released kernel pin, or stored document schema changes in this increment.
No push or PR was made.

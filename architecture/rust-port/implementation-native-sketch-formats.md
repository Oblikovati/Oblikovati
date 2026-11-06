# Native sketch curve formats

Issue: #3845. Target: released Martensite 0.10.0.

## Required behavior

Port Go #2015's convert-or-arm rule for the model's supported curve roles.
Construction applies to curves; Centerline applies only to lines. Convert each
applicable selected entity by its current role in one history step. With no
applicable selection, arm the matching creation mode without dirtying the file.
Construction reads the effective role, including an independent centerline role.

Use the API-owned format-mode value. Preserve source construction members when a
creation mode is off. Previews and commits must have identical roles. Decline
construction Point placement explicitly because the model currently supports
construction curves, not construction points. Center-point markers and driven
dimensions are separate pending capabilities; these controls do not claim them.

## Verification plan

Model tests cover authored order, mixed flags, applicability, invalid IDs,
atomic refusal, unchanged numerical geometry and retained verification. Session
tests cover owned picks, veto, filtered membership, history and stale scopes.
Owner tests cover armed modes, tool switching, preview/commit parity, independent
centerline axes and unsupported Point admission. GUI tests use pointer events at
both scales and check pressed states and transaction guards.

Native X11 acceptance must create and convert curves, inspect dash patterns,
cancel previews, test undo/redo and save/fresh-process reopen through OS input.
MCPBridge supplies read-only queries and screenshots. Inspect actual PNGs, close
owned desktops, run the full gate and compile checks, then publish one batch.

## Implementation and regression evidence

The model classifies each owned selection once, in authored order, before any
metadata changes. The session prepares the candidate and skips history adoption
when no metadata changes. This keeps empty and inapplicable selections from
advancing generation or retiring a tool. Changed conversions use one history
step without recomputing coordinates, and reuse Delete's checked selection
restoration after a verified veto rollback.

The native owner uses the current checked selection for convert-or-arm. Undo
invalidates old picks. A native test exposed an attempt to convert those stale
stored picks even though the public selection was empty. The failing regression
is retained in `/tmp/oblikovati-sketch-formats-undo-red.log`; the owner now arms
from the live empty selection without rebinding those old handles. Explicit
session commands continue to refuse stale selections.

Construction and centerline remain independent. A centerline is effectively
construction even when its explicit construction flag is off. Changing that
explicit flag does not remove the axis. The existing save schema already holds
both lists; no package migration or wire DTO was added.

Placement members now carry the independent axis role. Existing builders default
it to false. Preview and commit use the same recipe or resolved private result.
The native sampler retains both roles, and preview and committed rendering share
one role-to-line-pattern rule. Axis pattern has priority over construction,
including selected geometry. The pressed-mode button uses the owned scalar C++
adapter; no vector class crosses the Rust/C++ ABI.

Creation modes are transient owner state using the API's `SketchFormatModes`.
Only Construction and Centerline controls are ported here. There is no new wire
setter or claim that the remaining format-mode API is implemented. A construction
Point request is refused before retiring an existing placement tool. Intrinsic
construction members, such as center-rectangle diagonals, remain when a mode is
off. No kernel algorithm, tolerance constant or geometry recognizer was added.

## Final qualification

The final `make gate` passed all 13 crates and all 385 head tests on Xvfb.
Per-crate coverage is 91.94% to 99.60% (model 98.87%, session 98.59%, head 92.74%,
GUI 98.25%). Duplicated lines are 1.03% (930 of 90,676 lines). Formatting, Clippy,
Markdown, SPDX and workflow checks passed. The final gate log is
`/tmp/oblikovati-sketch-formats-final-gate.log`.

Windows x64 GNU and macOS ARM64 host/GUI Clippy checks passed, including the
scalar pressed-button adapter and pure placement-refresh tests. These compile
checks do not claim native runtime qualification on those platforms.

The real X11 driver is `/tmp/oblikovati-x11-native-sketch-formats.py`, with
captures and read-only model receipts in
`/tmp/oblikovati-x11-native-sketch-formats/`. It creates an axis, switches its
private preview role, converts selected roles, retains highlight patterns,
creates and cancels a construction-circle preview, and tests undo/redo. The
final sketch is healthy with DOF 7; this test deliberately leaves the curves
free. Save and a fresh process retain exact geometry and reference keys. The
saved role lists contain only the line as a centerline and only the circle as
explicit construction. Removing the axis role after reopen makes that line
normal; Undo restores the original role and keys.

The actual preview, selected-format and reopened PNGs were inspected. Both owned
desktops closed normally. The same worktree and bounded Kache storage were
reused; no new worktree or target tree was created. Unmerged branches were
retained. The comment/coherence and hard-coded-value review completed before
push; new ribbon coordinates use logical pixels and framebuffer scaling.

# Native sketch entity deletion

Tracking: [#3841](https://github.com/Oblikovati/Oblikovati/issues/3841).
Go reference: `app/sketch_delete.go`, including the Delete-key guard in #1232.

The Sketch ribbon Modify panel and Delete key call one owner operation. They
require an active sketch, live owned picks, no active tool and no caller
transaction. Empty or non-editor contexts do nothing. Text entry keeps Delete;
a held key does not repeat the action. The owner checks context again before
calling the existing session deletion command. Native add-in and script wrappers
forward the same operation. Scene synchronization follows adoption.

The document session owns selection. Its new owned-selection entry point
delegates to `delete_selected_sketch_picks`; it does not add a second permanent
selection store or geometry dependency closure. One candidate command deletes
selected entities, bound relations and unowned Model sources. Surviving sources
and user parameters retain their existing ownership. Successful adoption clears
picks and creates one history step. Undo restores model state, not transient
selection. No new public wire API is introduced.

A commit veto can restore the exact recipe with a new generation. The owner
retains selected identities after that refusal by checking the original pick
scope and exact sketch/parameter recipes before refreshing their generation
stamps. Existing membership is restored before its filter, so a narrowed filter
does not prune retained picks. Unrelated edits or aborts still invalidate old
picks; the refusal path never silently rebinds them.

Session tests cover mixed categories, source cleanup, one-step history, veto,
empty/editor/tool guards, stale picks and filter retention. Owner tests cover
shared-line geometry and relation cleanup, transaction refusal, tool/no-editor
guards and a named observer veto. Real ImGui tests cover the ribbon at both
display scales, keyboard dispatch, text ownership and held-key suppression.
The full local gate passes all 13 crates and 370 head tests. Line coverage is
91.96% to 99.60%; duplicated lines are 1.01%. Windows x64 GNU and macOS ARM64
host/GUI Clippy pass. These compile checks do not claim native desktop coverage
on those platforms.

Native X11 mouse/keyboard acceptance deletes two picked lines with Delete and
restores both through one Undo. Ribbon deletion removes a parameter-driven
circle, its diameter dimension and unowned Model source while preserving an
independent point and the user `size` parameter. Tool-active and empty-selection
Delete are inert. Undo/redo restore the same identities and dependency records.
Save/fresh-process reopen retains exact surviving geometry, no deleted circle
or dimension, the user source, healthy DOF two and empty transient selection.
MCPBridge reads and captures verify these results. Deletion, undo and reopened
captures were inspected. Both acceptance desktops closed normally.

This PBI connects visible entity picking to deletion. Standalone dimension-label
and constraint-marker picking remain separate UI work. The existing mixed
command is reused without adding unreachable UI branches for those categories.
The general Extrude, cube and named-face journey remains incomplete.

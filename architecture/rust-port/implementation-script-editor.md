# Script editor buffer and edit history — partial R7

Tracking: [port epic #3563](https://github.com/Oblikovati/Oblikovati/issues/3563).
This ports the source/selection/edit-history portion of Go `script/console/editor`,
`textbuf` and `history` at the fixed legacy revision, under ADR-0009/0011. It is the
source core for the [console controller](implementation-script-console.md), not a
native panel or a claim that the full editor surface is complete.

`ScriptEditor` owns Unicode-character lines, the anchored selection, the vertical
goal column and reversible source changes. The buffer always has at least one
line. Initial load normalizes CRLF like the Go buffer; insertion preserves its
input text. Columns count characters rather than UTF-8 bytes. Renderer line reads
name invalid indexes instead of exposing unchecked slices. External caret targets
clamp to document bounds before they can enter an edit.

One replacement path handles typing, paste, selected-text deletion, Backspace and
Delete. Multiline replacement joins the retained prefix and suffix around inserted
lines, and records exactly the removed and inserted source. Undo and redo replay
that same replacement operation with inverse text; their caret lands after the
restored or inserted span, matching Go. A fresh edit clears redo and loading a new
source resets selection and history. No model transaction is created by source
editing; Run still sends the source through the existing script host boundary.

History follows the Go coalescing rules: contiguous single nonblank typing joins
an insertion run; spaces, tabs and newlines start boundaries. Contiguous single
character backward or forward deletion joins its previous removal. Replacements
that both remove and insert do not coalesce. A multiline paste is one operation;
the exact legacy rules determine whether a later typed character can join it.
No second mutation or history engine exists in a native widget.

Newline carries the ordered selection start line's leading spaces/tabs. Horizontal
arrows wrap across lines and collapse non-extended selections to their near/far
edge. Shift movement retains the anchor. Vertical movement keeps its original
goal column through shorter lines, with Go's special top/start and bottom/end
behavior at document bounds. Smart Home toggles indentation/column zero; line End,
document start/end and Select All use the same caret path.

Tests cover Unicode columns, CRLF load, reversed multiline selections, paste,
line joins, auto-indent, empty/boundary no-ops, word-boundary typing coalescing,
backward/forward delete coalescing, undo/redo, redo invalidation, source reset,
clamped targets, arrow selection collapse, sticky vertical columns and smart Home.
Separate-position edits and replacements prove unrelated history entries remain
separate. The pinned Go editor/textbuf/history suites are the behavior references.

Run `cargo test -p oblikovati-script --test editor` for these checks and `make gate`
for full local Linux acceptance. Bracket/call analysis, completion/help and native input/drawing integration
remain open. Signed mouse-coordinate translation belongs to the native input
adapter; core positions use nonnegative character indexes. No foreground UI or
rendered-editor acceptance is claimed. Windows and macOS remain deferred.

[Word navigation and block commands](implementation-script-editor-blocks.md) now
extend this core with ASCII Lua identifier selection and reversible indent/comment
operations. Native panel and remaining analysis/search operations are still open.

[Literal search and Find-bar state](implementation-script-editor-search.md) now
provide ordered hits, generation-checked selection, wrapping navigation and
whole-source Replace All through this same history path.

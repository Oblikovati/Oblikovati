# Editor word navigation and block commands — partial R7

Tracking: [port epic #3563](https://github.com/Oblikovati/Oblikovati/issues/3563).
This extends the [source/history core](implementation-script-editor.md), porting
Go `textbuf/class.go`, word movement in `cursor.go`, and `editor/lines.go`.
It adds no native input or drawing ownership and no model/API operation.

Word commands classify ASCII letters, digits and underscore as Lua identifier
characters; space/tab are inline blanks; other characters form punctuation runs.
Columns still count Unicode characters. Left skips preceding blanks and one run,
wrapping to the previous line's end from column zero. Right skips the current run
and following blanks; when that run reaches line end, it immediately crosses to
the next line. The regression tests retain that Go behavior rather than stopping
one extra time at end-of-line. Shift movement retains its anchor. Double-click
selection selects only an identifier under a clamped hit and collapses off-word hits.

Indent, Outdent and Toggle Line Comment operate on the ordered selected line span,
or the caret line. A selection ending at column zero excludes its final line.
Indent adds four spaces. Outdent removes up to four leading spaces or one tab.
Comment toggling removes prefixes only when all nonblank lines are already
commented; mixed blocks are commented, including nesting existing comments.
New prefixes use the block's minimum indentation and leave blank lines unchanged.
Removal accepts bare `--` and strips at most one following space.

Each command uses the existing one replacement/history path and reselects the
resulting block, so repeated shortcuts use the same lines. Undo and redo retain
one step per block command. A genuinely empty span/replacement does not create
history, matching Go; an unchanged nonempty replacement retains the Go history
behavior. No duplicate block mutation or undo engine exists in a widget.

Tests cover identifier/punctuation/blank boundaries, ASCII-vs-Unicode word rules,
line wrapping, extended selection, double-click off-word collapse, shared comment
indentation, mixed/bare prefixes, blank and Unicode-space lines, reversed selected
line spans, exclusion at final column zero, partial-space/tab outdent, retained
block selection and undo/redo. The existing editor suite also passes.

Run `cargo test -p oblikovati-script --test editor --test editor_blocks` and the
full local Linux `make gate`. Go editor/textbuf/history tests remain the behavior
oracle. Native completion/help, signed native mouse input
and native panel bindings remain open. Windows and macOS remain deferred.

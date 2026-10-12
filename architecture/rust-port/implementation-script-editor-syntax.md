# Lua syntax highlighting and visible-line context — partial R7

Tracking: [port epic #3563](https://github.com/Oblikovati/Oblikovati/issues/3563).
This ports Go `script/console/lualex` and the source-state traversal used by the
head's `code_editor_render.go`. The Rust lexer supplies token kinds and character
spans; it does not draw colors or establish native panel acceptance.

`tokenize_line` returns ordered, nonempty half-open Unicode character spans.
Kinds are identifier, keyword, builtin, string, number, comment and operator.
ASCII spaces and tabs have no token. Other punctuation produces one token per
character. Quoted strings honor escaped characters and stop at the line boundary.
Numbers retain Go's liberal decimal/hexadecimal, fraction and exponent scan rules:
coloring is not a proof that a literal is valid Lua. The isolated worker parser
remains responsible for compile diagnostics.

Long strings and comments carry their exact equals-sign bracket level across
lines. Only a matching closer ends the construct. Empty continuation lines keep
the state without producing an empty token. Source after a closer is scanned
normally and can open another construct. A typed state enum distinguishes plain
source, long strings and long comments, avoiding contradictory flag combinations.
Arbitrary incoming levels are bounded by line traversal rather than used as an
allocation size.

Completion and highlighting now share the same internal Lua 5.1 keyword/builtin
vocabulary and ASCII identifier predicate. This removes the temporary completion-
local table. The completion worker-global test still checks the complete builtin
set, while syntax tests assert the classification of every completion entry.
Lua 5.4-only and sandbox-unavailable names remain plain identifiers, matching the
Rust runtime rather than the legacy highlighter's broader vocabulary.

`ScriptEditor::highlight_lines(first, last)` highlights a half-open visible line
range. It reads prior source lines to establish long-bracket context, as the Go
renderer does. Every request reads current source, so edits, undo and redo change
following-line highlighting without a stale cache. Queries leave caret, selection
and edit history unchanged. Invalid bounds return an error naming the range and
valid extent. Empty ranges and the mandatory empty source line remain valid.

Seventeen syntax tests cover the Go lexer examples, escapes and incomplete source,
long-bracket continuation/closing, post-closer tokens, all shared vocabulary,
Unicode columns, numeric scan boundaries, arbitrary carried levels, viewport
ranges and source edits affecting later lines. Generated bracket levels and
character pairs check progress and ordered in-bounds spans. Go lexer/completion
oracle tests and the Rust editor/completion regression suites pass.
Run `cargo test -p oblikovati-script --test syntax` and full Linux `make gate`.
The native syntax palette/drawing, popup controls and generated method-help
catalog remain open. Windows and macOS remain deferred.

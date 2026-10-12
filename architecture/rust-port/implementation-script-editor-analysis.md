# Editor bracket and call analysis — partial R7

Tracking: [port epic #3563](https://github.com/Oblikovati/Oblikovati/issues/3563).
This ports Go `script/console/editor/brackets.go` and `calls.go` to the Rust
[source editor](implementation-script-editor.md). It supplies queries for future
bracket highlights, signature help and hover documentation; no native widget or
completion catalog is added here.

`matching_bracket` first checks the caret, then the preceding character. It returns
the nearby bracket and its partner in that order. Parentheses, square brackets and
braces scan in either direction with same-kind nesting, across empty and nonempty
lines. A missing partner proceeds to the other candidate before returning no match.
The positions use Unicode character columns, like the other editor commands.

`enclosing_call` scans left on the current line for the nearest unmatched parenthesis
or brace. It reads an immediately preceding ASCII identifier chain separated by
dots. Anonymous groups, whitespace before the opener and prior-line calls return
no name. `method_chain_at` reads the hovered identifier and the dotted segments to
its left. Hovering an intermediate segment returns only that prefix. As in Go, a
position just past the word can return that word. Out-of-range lines and columns
return no result rather than clamping to a different word. Negative native mouse
coordinates still require validation in the future signed-coordinate UI adapter.

These are deliberately lexical editor queries, not Lua parsing or semantic lookup.
Go counts bracket characters in strings and comments, ignores other bracket kinds
when matching a pair, and uses one combined nesting depth for call parentheses and
braces. Those behaviors remain explicit compatibility limits. The existing ASCII
identifier predicate is shared with word navigation instead of being redeclared.
Queries borrow current source, change neither selection nor history, and need no
cached spans that can become stale after edits.

Nine Rust tests cover forward/backward nesting for every bracket kind, caret
priority and fallback, document boundaries, empty lines, Unicode positions,
lexical string/comment and mixed-kind behavior, table and nested calls, closed and
anonymous groups, intermediate hover segments, invalid hits, and edit/undo/redo/
reload behavior. The corresponding Go bracket and call tests pass as the oracle.
Run `cargo test -p oblikovati-script --test editor_analysis` and full Linux
`make gate`. Completion/help catalogs and native editor bindings remain open;
Windows and macOS remain deferred.

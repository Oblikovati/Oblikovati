# Script completion engine and checked acceptance — partial R7

Tracking: [port epic #3563](https://github.com/Oblikovati/Oblikovati/issues/3563).
This ports Go `script/console/complete` and the prefix replacement from
`head/ui/code_editor_complete.go` onto the Rust editor. The engine accepts an
injected snapshot of the host's available canonical method names. It does not
claim that every API method is implemented. Rebuild it when that snapshot changes.

An ordered namespace tree deduplicates names. Completion reads the ASCII dotted
identifier chain to the left of the caret, with Unicode character positions for
replacement spans. Empty trailing segments enumerate a namespace. A concat
operator or leading dot stops traversal. Non-host roots and unknown namespaces
return no candidates. Namespace traversal is case-sensitive, while the final
prefix matches ASCII case-insensitively. Suggestions have a stable text order,
keyword/builtin/module/method kind and full API reference detail. A callable node
with children remains a module, matching Go's presentation rule.

Bare suggestions use Lua 5.1 keywords and the globals the isolated Rust worker
actually exposes. The Go highlighter/completer's Lua 5.4 vocabulary is not copied:
`goto`, `rawlen`, `utf8`, unavailable loaders, I/O and debug globals must not be
advertised. A worker test enumerates its actual global table and compares the
entire sorted set with completion builtins. This includes Lua 5.1's coroutine
table, which the base library installs even though the explicit standard-library
flags only select table, string and math. The test will detect future sandbox
changes that require updated suggestions.

Editor completion results bind their private candidates and replacement prefix
to the source generation and caret. Acceptance checks those values and the index
before writing. Foreign, edited, undone or reloaded results are refused. Caret
movement also refuses acceptance at a different position. Errors identify the
invalid span or index and expected state. Valid acceptance replaces only the
prefix through the existing editor journal, preserving the source suffix and
providing one undo/redo step. Like Go, a different selection anchor with the same
caret does not widen the prefix replacement.

Nine tests cover namespace enumeration, nested namespaces, kinds, details, case
rules, duplicates, unknown paths, concat, empty queries, Unicode columns, clamped
query positions, prefix acceptance, undo/redo, invalid indices, source/caret
invalidation and the full worker builtin set. The Go completion tests pass too.
Run `cargo test -p oblikovati-script --test editor_completion` and Linux
`make gate` for the full workspace checks.

The native popup still requires input routing, visibility, selection navigation,
drawing and integration with the shared Linux window. Method help still requires
a contract-generated documentation catalog and the bracket/call query bindings.
No native UI acceptance is claimed. Windows and macOS remain deferred.

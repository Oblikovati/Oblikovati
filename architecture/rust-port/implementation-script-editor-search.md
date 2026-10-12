# Editor literal search and Find-bar state — partial R7

Tracking: [port epic #3563](https://github.com/Oblikovati/Oblikovati/issues/3563).
This ports Go `editor/search.go` and the head's `code_editor_find.go` state rules
onto the [source editor core](implementation-script-editor.md). It does not draw
a native Find bar or complete the remaining editor analysis/completion surfaces.

Find reports literal, case-sensitive, non-overlapping occurrences in line and
Unicode-character order. Empty queries return no hits. Find does not cross line
breaks. Replace All deliberately uses whole-source text, matching the Go command,
so its query can include line breaks even when Find returns no highlighted hits.
It records the whole replacement through the existing edit/history path as one
undo step and returns its count. Empty/no-match queries leave history untouched;
a matched unchanged replacement retains Go's journal behavior.

Selecting a match sets its anchored half-open span and vertical goal column.
Replacing that selection uses ordinary editor insertion and history. Match
coordinates are readable but cannot be forged through public fields. Each match
also carries a weak source-generation token. Edits, undo/redo and source reload
invalidate old tokens; foreign-editor tokens are refused too. Invalid selection
attempts name the span and leave current caret/selection unchanged. The identity
is ephemeral editor state, not persistent model naming or geometric identity.

`EditorSearch` owns the query, ordered hits and current index. Changing a query or
opening/refreshing Find selects the first hit. Next/Previous wrap in both directions
and report a one-based ordinal with total hits. Replace All refreshes the matches,
including replacements that still contain the query. Source changes suppress
stale highlights and ordinals immediately; the next navigation refreshes and
selects the current first hit instead of applying stale coordinates. This makes
invalidation explicit rather than retaining the Go widget's cached spans after
unrelated source edits.

Tests cover adjacent/overlapping query inputs, literal case behavior, Unicode
columns, line-oriented Find versus cross-line Replace All, selected replacement,
one-step undo/redo, no-match history/generation preservation, live foreign-editor
matches, stale tokens after edit/undo/reload, query refresh, Next/Previous wrapping,
post-edit highlight invalidation and replacement text containing the query.

Run `cargo test -p oblikovati-script --test editor_search` and full local Linux
`make gate`. Go editor search tests remain the behavior oracle. Bracket/call
analysis, completion/help and native editor/Find-bar bindings remain open.
Windows and macOS remain deferred; no foreground panel acceptance is claimed.

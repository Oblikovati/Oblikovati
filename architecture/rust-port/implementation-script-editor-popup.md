# Completion popup controls and tooltip priority — partial R7

Tracking: [port epic #3563](https://github.com/Oblikovati/Oblikovati/issues/3563).
This ports the state/key rules from Go `head/ui/code_editor_complete.go` and the
layout rules from `code_editor_complete_render.go`. It builds on the checked
[completion engine](implementation-script-editor-completion.md) and generated
[method help](implementation-script-editor-help.md). Native input mapping,
font metrics, colored rectangles/text and foreground panel acceptance remain open.

`CompletionPopup` owns an injected method engine, current completion result,
selected index and visibility state. Refresh opens suggestions for a nonempty
prefix; an explicit trigger also opens an empty prefix, as for Ctrl+Space or a
typed dot in the Go view. Empty candidate sets stay hidden. A refresh retains a
valid selection index and resets an out-of-range index to zero. Replacing the
host method engine discards old candidates and selection so removed capabilities
cannot remain in the popup.

Source-generation and caret checks protect drawing and key handling as well as
acceptance. Edits, undo/redo, reload, a moved caret or another editor hide old
suggestions. Hidden/unrelated keys return false for normal editor routing.
Visible Up/Down clamp selection at the first/last item; Escape dismisses without
editing source. Enter/Tab accept only the selected prefix through the checked
editor replacement and close the popup. Undo/redo remain ordinary editor history.
A direct acceptance attempt on stale/hidden state returns a named error and hides
the popup. Focus loss can call `dismiss` without altering source.

The borrowed frame view reports the full immutable candidate slice, selection,
first visible index, row count, width in character cells and source prefix anchor.
At most eight rows are visible; the window follows selection without wrapping.
Width is the longest candidate's Unicode character count plus four cells for the
kind glyph and padding. Native drawing maps the source anchor to one line below
the caret. Kind glyphs remain k/b/m/f for keyword/builtin/module/method.
Signature and hover help return no tooltip while current suggestions are visible,
then use the generated catalog once the popup is dismissed or becomes stale.

Nine tests cover triggers, missing candidates, clamped navigation, selection
retention/reset, host-engine replacement, Enter/Tab acceptance and undo/redo,
Escape/normal-key routing, stale and foreign editor state, row scrolling, Unicode
width, glyphs and signature/hover suppression. Completion and help regression
suites pass. Go head source is the behavior reference; no native popup execution
is claimed. Run `cargo test -p oblikovati-script --test editor_popup` and full
Linux `make gate`. Windows and macOS remain deferred.

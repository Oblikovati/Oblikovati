# Extrude shortcut (#3931)

The desktop keymap registers `Create.Extrude` with `E` as its default chord. The
pinned extrusion scenario requires one key press. The Go reference instead uses
an Enter-terminated `E` alias and a `Shift+E` chord. The Rust binding follows
the scenario and does not claim `Shift+E`, because an existing user keymap may
already assign it to another command.

The frame loop routes the resolved action only when Extrude is available in the
active Part. An empty Part opens plane selection; a Part with a sketch opens the
extrusion dialog. GUI text capture takes priority, and an unavailable action
leaves sketch alias input available. Focus loss clears held keys. Keymap
customization and collision checks apply to the same action as for the other
shortcuts. On restore, a saved assignment keeps its chord and a conflicting new
default is disabled. This also admits saved single-letter chords from older
versions, while new edits retain the existing chord validation rules.

The configured Part marking menu routes `Create.Extrude` through the same
plane-selection or dialog path as the ribbon (#4049). The native pointer
regressions cover an empty part and a part with a finished sketch.

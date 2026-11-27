# Tab between Extrude values

The published Properties scenario opens an asymmetric rectangular
extrusion and edits Distance A to 8 mm. Tab focuses Distance B, where
typing 6 mm changes the preview to span -6 mm through 8 mm. Shift+Tab
returns focus to Distance A. Properties remains open and the document
has no committed feature or undo entry.

The native test checks the focused editor in the completed frame and
the preview geometry after editing. All seven steps passed on each
desktop using the existing Properties keyboard navigation.
[Wayland output](wayland.xml), [X11 output](x11.xml),
[published input checksums](published-inputs.sha256).

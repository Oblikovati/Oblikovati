# Open Extrude with the selected default

The published Extrude examples save an 8 mm preset, commit a 12 mm
extrusion, reopen Properties, and select the saved preset. Each example
chooses a different default, cancels the current draft, and starts
another Extrude command.

| Default preference | Distance A in the new draft |
| --- | --- |
| No Preset | 10 mm |
| Last Used | 12 mm |
| Use Current Preset | 8 mm |

The test checks the menu's selected state, the new preview request,
the displayed preset selection, and that the 12 mm extrusion stays as
the single feature and undo entry.

Both Linux desktops passed all 30 Extrude steps. The three Revolve
examples remain unbound at profile setup.
[Wayland output](wayland.xml), [X11 output](x11.xml),
[published input checksums](published-inputs.sha256).

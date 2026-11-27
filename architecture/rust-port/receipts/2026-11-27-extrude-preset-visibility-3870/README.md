# Hide Presets in Extrude

The pinned Extrude scenario starts with Hide Presets checked. Clearing
it shows the No Preset selector, add button, and gear between the
breadcrumb and Input Geometry. The gear opens Preset Settings. Save
Current, Rename Current, Delete Current, and Use Current Preset are
disabled while no preset is selected. Recently Used and No Preset are
checked. Dismissing the menu keeps the draft open, and checking Hide
Presets again removes the row.

Extrude presets retain the numeric distance and direction in the
desktop session. The native reuse test saves a named preset, closes the
draft, changes the distance in a new draft, and restores the saved value
by selecting the preset.

The named-preset test also exercises Save Current, Rename Current,
Use Current Preset, and Delete Current. The Last Used test accepts
an extrusion and checks that the next draft starts with its distance.

All eight pinned steps and both native regressions passed on each
desktop. [Wayland scenario](wayland.xml), [Wayland native tests](wayland-reuse.txt),
[X11 scenario](x11.xml), [X11 native tests](x11-reuse.txt).

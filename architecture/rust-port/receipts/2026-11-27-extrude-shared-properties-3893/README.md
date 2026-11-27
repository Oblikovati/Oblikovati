# Shared Extrude Properties controls

The published preview case edits the 20 mm extrusion in
`extrusion-rect-20mm.opd` to a draft depth of 8 mm. The preview button
appears beside the feature and sketch breadcrumb. Hide Preview
reveals the committed 20 mm solid. Show Preview restores the 8 mm
draft. The profile, plane, input value, and enabled OK remain visible.
The committed feature retains its 20 mm distance throughout.

The published Preset Settings case opens `revolve-profile.opd` in an
isolated application profile. It selects the rectangle in Extrude and
checks the complete sort and new-command default choices. Dismissing
the menu leaves Properties and the draft open.
[Published input checksums](published-inputs.sha256).

Both published cases passed on Wayland and X11 with 11 steps per
desktop. [Wayland preview](wayland-preview.xml), [Wayland presets](wayland-presets.xml),
[X11 preview](x11-preview.xml), [X11 presets](x11-presets.xml).

The published reopen-and-cancel case passed ten steps per desktop. The
pinned Hide Presets case passed eight steps per desktop at snapshot
`b576445b`. [Wayland edit](wayland-edit.xml), [Wayland pinned](wayland-pinned.xml),
[X11 edit](x11-edit.xml), [X11 pinned](x11-pinned.xml).

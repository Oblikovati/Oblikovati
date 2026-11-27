# Duplicate document appearances

The Appearance Editor footer offers Duplicate and Duplicate as Generic
for an applied asset. Both commands add and select a separately named
document appearance. Duplicate keeps the source type and recipe.
Duplicate as Generic presents Generic controls and retains the common
finish, Tint, bump, relief, and diffuse scattering used in the preview.

Both published scenarios passed all 14 steps on Wayland and X11. The
converted Matte Ceramic and Wavy bump preview matched the source pixels.
A native save check created both kinds of copy, renamed the Generic copy,
and decoded the saved Part to verify three independent assets. Generic
creation and the Ceramic custom-image cancel scenario passed 23
regression steps on each backend. All ten document package asset tests
passed.

The build used base `0c3569667`, the [implementation and tests](test-input.sha256),
and the existing local GUI filesystem test-module visibility edit.
Wayland used nested Weston and X11 used Openbox. Both used Xvfb and
Vulkan Lavapipe. The runs used a copy of the [published scenario and
fixture inputs](published-inputs.sha256). The shared source appended six
specifications during verification. All earlier scenarios, notes, and
the fixture matched the run snapshot. Strict provenance remains open
under #3978.

- Duplicate on [Wayland](copy-wayland.json) and [X11](copy-x11.json)
- Duplicate as Generic on [Wayland](generic-wayland.json) and [X11](generic-x11.json), with a [Wayland screenshot](generic-copy-wayland.png)
- Generic creation on [Wayland](create-wayland.json) and [X11](create-x11.json)
- Ceramic image cancellation on [Wayland](cancel-wayland.json) and [X11](cancel-x11.json)
- Native saved copies on [Wayland](save-wayland.txt) and [X11](save-x11.txt)

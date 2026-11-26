# ViewCube HUD in Part and Assembly

All 27 amended Viewport Orientation scenarios passed on Wayland and X11,
with 291 steps on each backend. The new case used the published solid
Part and a blank Assembly. Samples on both sides of the idle ViewCube
matched the viewport canvas while the Navigation Bar and lower-left XYZ
triad stayed visible. Hovering in Front view revealed Home, four cardinal
triangles, two open roll arcs, and the menu chevron.

In the oblique view, the test read the submitted Front, Right, and Top
glyphs and checked that their stems followed the corresponding cube
edges. It also checked framebuffer contrast in Dark and Light. The
original lettering failed the Light check on both backends. After
the weight and size adjustment, both passed. The Wayland captures were
visually checked in both workspaces and themes.

The reports used the product and scenario changes in `6ac7b8e28` and
existing local GUI filesystem edits. The separate relocation of the
profile isolation test in that commit followed the full native runs.
Both backends used Xvfb and Vulkan Lavapipe, with nested Weston for
Wayland and Openbox for X11. Published input hashes matched before and
after the runs. Issue #3978 tracks strict build and runtime provenance.

- [Wayland report](wayland.json) and [step arguments](wayland-arguments.json)
- [X11 report](x11.json) and [step arguments](x11-arguments.json)
- [Published input hashes](inputs.sha256)

# Ribbon Appearance on Wayland and X11

All 18 current scenarios and 201 steps passed on each backend. The run
covered all appearance modes, panel titles, open menus, the reset
confirmation, and ribbon changes while editing a sketch. It used the
published Part and planar sketch fixtures.

The head library runner used `OBK_BDD_AREA=application`,
`OBK_BDD_FEATURE=ribbon-appearance` and the shared scenario directory as
`OBK_BDD_ROOT`. Wayland used nested Weston 13 and X11 used Openbox.
Both ran in Xvfb with Vulkan Lavapipe. The test build included existing
local GUI filesystem edits. Issue #3978 tracks strict provenance.

- [Wayland report](wayland.json) and [step arguments](wayland-arguments.json)
- [X11 report](x11.json) and [step arguments](x11-arguments.json)
- [Published input hashes](inputs.sha256)

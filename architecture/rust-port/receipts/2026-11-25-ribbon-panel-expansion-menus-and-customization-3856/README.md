# Native ribbon expansion menus and customization

Fourteen current cases passed 171 steps. They verified the Home and
Sketch tab transitions, Options layout reset cancellation, expanded
command and title menus, and the placement of custom commands.

The Linux X11 build included the scenario bindings in `808af2e75` and
existing local GUI filesystem, opening and saving edits. The runs used
Xvfb, Openbox and Vulkan Lavapipe. The [input hashes](current-inputs.sha256)
matched the isolated source and fixtures. Issue #3978 tracks strict build
and runtime provenance.

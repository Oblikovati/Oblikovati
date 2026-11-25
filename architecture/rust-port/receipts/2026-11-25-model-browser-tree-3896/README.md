# Native Model Browser tree and origin controls

Nine current cases passed 102 steps on the published Part. The cases verified
branch expansion, extrusion selection and its context menu, the additional
pane, origin plane selection and visibility, work-feature filtering, and
the Part root menu.

The Linux X11 build included the scenario bindings in `6d0f4212f` and
existing local GUI filesystem, opening and saving edits. Runs used Xvfb,
Openbox and Vulkan Lavapipe. The [input hashes](current-inputs.sha256)
matched the isolated source and fixtures. Issue #3978 tracks strict build
and runtime provenance.

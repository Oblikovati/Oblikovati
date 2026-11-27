# Native Ribbon Customization tab retention in Sketch1

Both current cases passed 24 steps. They opened the published Part, edited
its saved Sketch1, switched or reopened Customize, and verified the
remembered tab and unchanged sketch geometry.

The build used `b1373d818`, the sketch fixture and tab bindings in this
commit, a separate draft dialog-row diagnostic, and existing local GUI
filesystem, opening and saving edits. The native runs used Linux X11,
Xvfb, Openbox and Vulkan Lavapipe. The [input hashes](current-inputs.sha256)
matched the isolated documents and preference seed. Issue #3978 tracks
strict build and runtime provenance.

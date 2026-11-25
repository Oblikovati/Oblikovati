# Native ribbon expansion controls and command placement

Sixteen current cases passed 172 steps. They checked keyboard expansion,
main and expanded command placement, pinning, Tools Options, and
restoring controls after tab and display changes.

The Linux X11 build included the scenario bindings in `808af2e75` and
existing local GUI filesystem, opening and saving edits. The runs used
Xvfb, Openbox and Vulkan Lavapipe. The [input hashes](current-inputs.sha256)
matched the isolated source and fixtures. Issue #3978 tracks strict build
and runtime provenance.

# Native temporary ribbon expansions and viewport interaction

Fourteen current cases passed 149 steps. They verified temporary panels,
Escape and title dismissal, restoring the full ribbon, viewport face
selection, and interaction with browser features while panels are pinned
or unpinned.

The Linux X11 build included the scenario bindings in `808af2e75` and
existing local GUI filesystem, opening and saving edits. The runs used
Xvfb, Openbox and Vulkan Lavapipe. The [input hashes](current-inputs.sha256)
matched the isolated source and fixtures. Issue #3978 tracks strict build
and runtime provenance.

# Native Ribbon Panel Expansion sketch menus and pinning

Eight published cases passed 87 steps. They verified opening Sketch Create
and Constrain, interacting with the Constrain context menu, and closing or
switching expanded panels in the Sketch and 3D Model ribbons.

The Linux X11 build included the head scenario bindings in `808af2e75`
and existing local GUI filesystem, opening and saving edits. Runs used
Xvfb, Openbox and Vulkan Lavapipe. The [input hashes](current-inputs.sha256)
matched the isolated source and fixtures. Issue #3978 tracks strict build
and runtime provenance.

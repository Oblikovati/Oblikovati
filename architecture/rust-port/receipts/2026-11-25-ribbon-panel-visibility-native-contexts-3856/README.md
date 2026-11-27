# Native ribbon panel context changes

Seven published cases passed 86 steps. The tests used the saved Part tabs,
the visible solid, the Home workspace and the active Sketch1 fixture to
check visibility settings, keyboard focus, pointer handling outside the
selector, and Sketch panel hide and restore.

The Linux X11 build used the test changes in `7068db362` and existing
local GUI filesystem, opening and saving edits. The runs used Xvfb,
Openbox and Vulkan Lavapipe. The [input hashes](current-inputs.sha256)
matched the isolated source and fixtures. Issue #3978 tracks strict
build and runtime provenance.

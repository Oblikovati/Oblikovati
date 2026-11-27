# Document Opening with the full shared fixture tree

All 27 published Document Opening scenarios and 315 steps passed on Wayland.
Each scenario ran with its own headless Weston compositor, Vulkan Lavapipe,
and frame-loop input. The full shared fixture directory remained available.
The runner reported no scenario source changes during the individual runs.

The earlier single-compositor run lost its surface after Weston crashed.
Using a fresh compositor for each scenario allowed the remaining cases to
reach the application. The only failing scenario in that isolated batch was
the thumbnail context menu. Its corrected rerun supplies the passing result
in the combined report. [Context menu follow-up](../2026-11-26-file-picker-context-menu-3889/README.md)

The base build was `a8f03ae2d` with existing local GUI filesystem and test
edits. Some later cases used rebuilt executables containing unrelated ribbon
work. The thumbnail context-menu binding was corrected in `30a22b041`.
Strict build and fixture provenance remains tracked under #3978.

[Combined report](cucumber.json) · [Feature hash](feature.sha256)

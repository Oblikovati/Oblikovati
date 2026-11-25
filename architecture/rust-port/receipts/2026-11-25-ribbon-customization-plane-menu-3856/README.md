# Native grouped Plane menu

All eleven steps passed. The Plane arrow opened a vertical menu with
Plane, Offset from Plane, and Three Points. The assertions checked
icons adjacent to the choice labels and two horizontal dividers in
the rendered menu. Escape dismissed it without opening a plane
properties panel or adding a feature. [Native Cucumber report](case-028/cucumber.json)

The Offset from Plane choice still opened its face-selection dialog
and canceled without adding a feature. The separate X11 group and
ungroup check and main Plane action also passed.

The build used `a2c380cc1` with this commit's changes and existing local
GUI filesystem, opening, and saving edits. The isolated inputs
matched the published source. The test ran on Linux X11 with Xvfb,
Openbox, and Vulkan Lavapipe.
[Input hashes](../2026-11-25-ribbon-customization-current-3856/current-inputs.sha256)

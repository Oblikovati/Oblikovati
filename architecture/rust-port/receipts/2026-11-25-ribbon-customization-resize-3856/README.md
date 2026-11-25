# Native Customize resize and reopen

The [baseline](before-case-107/cucumber.json) measured the outer window
growing while Available Commands stayed at 230 by 200 pixels.
The [resize case](case-107/cucumber.json) subsequently passed fifteen
steps. Both lists grew to show more rows. Shrinking preserved the
staged command, its unchecked display options, and the pending marker.
Cancel closed Customize without adding a ribbon command.

The [reopen case](case-108/cucumber.json) passed eight steps. Closing
the resized window and reopening it from Tools restored the default
window and list dimensions. The test clicks the Close label drawn by
Customize itself. The window movement regression also passed all ten
steps, and the open-selector Close regression passed all twelve.

The [input hashes](../2026-11-25-ribbon-customization-current-3856/current-inputs.sha256)
matched the isolated source and profile. The Linux X11 builds used
changes from this commit over `c76e810e4` and `572388501`, the pinned
fixture fallback, and existing local GUI filesystem, opening, and saving
edits. They ran on Xvfb with Openbox and Vulkan Lavapipe. Issue #3978
tracks strict build and runtime provenance. GUI and Head test Clippy
passed with warnings denied.

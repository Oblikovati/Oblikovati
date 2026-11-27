# Full Panel Visibility run after the selector fixes

The [published native run](cucumber.json) at `e05f0d6c7` passed 41 of 55
scenarios and 501 of 515 executed steps. The previous
[full run](../2026-11-24-panel-visibility-inactive-tabs-full-3856/README.md)
passed 35 scenarios. The six additional passes cover direct and right-docked
selector dismissal, picking a face, restoring a hidden Modify title,
dismissing that title, and selector help.

Fourteen scenarios stopped at an undefined step. All matched steps passed.
The [feature-wide binding inventory](../2026-11-24-selector-help-3856/published-bindings.json)
lists 101 unresolved steps, including later steps in cases that stopped early.

The run used the [49-file published snapshot](../2026-11-24-ribbon-panel-visibility-3856/snapshot.json)
at `d53b12c1115aa35f38876e51976f7a28b3d43a43` with X11, Xvfb, Openbox, and
Vulkan Lavapipe. The Head build included existing local file-browser edits.
See [#3856](https://github.com/Oblikovati/Oblikovati/issues/3856).

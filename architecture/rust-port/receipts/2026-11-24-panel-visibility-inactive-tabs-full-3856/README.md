# Full Panel Visibility run after the inactive-tab fix

The [published native run](cucumber.json) at `b5e1b9899` passed 35 of 55
scenarios and 460 of 480 executed steps. The previous
[published run](../2026-11-24-panel-visibility-handoffs-full-3856/README.md)
passed 30 scenarios. The five additional passes cover independent tab
visibility, hiding and restoring a Sketch panel, reopening the sketch,
reflow around an open expansion, and Show Panels from an inactive tab.

Eighteen scenarios stopped at an undefined step. The direct selector and
right-docked selector cases still failed on the second click. Focused runs
after this build passed those cases and verified
[face selection](../2026-11-24-selector-face-3856/README.md) and
[selector help](../2026-11-24-selector-help-3856/README.md).

The run used X11, Xvfb, Openbox, and Vulkan Lavapipe. All 55 cases came from
the [49-file published snapshot](../2026-11-24-ribbon-panel-visibility-3856/snapshot.json)
at `d53b12c1115aa35f38876e51976f7a28b3d43a43`.

See [#3856](https://github.com/Oblikovati/Oblikovati/issues/3856).

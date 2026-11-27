# Full Ribbon Panel Visibility run after Tools handoffs

The [published native run](cucumber.json) at `8deaf75f2` passed 30 of 55
scenarios and 411 of 436 executed steps. Twenty-three cases stopped at an
undefined step. Two cases failed when a second click on the selector
button reopened the checklist. The later
[selector and Modify title fix](../2026-11-24-modify-hidden-title-3856/README.md)
records focused passing runs for that behavior.

The run used X11, Xvfb, Openbox and Vulkan Lavapipe. It used the
[49-file published snapshot](../2026-11-24-ribbon-panel-visibility-3856/snapshot.json)
at `d53b12c1115aa35f38876e51976f7a28b3d43a43` and selected the
full feature with no scenario filter.

See [#3856](https://github.com/Oblikovati/Oblikovati/issues/3856).

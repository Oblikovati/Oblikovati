# Integrated Ribbon Display run

The [full published feature](cucumber.json) passed 48 of 50 scenarios
and 512 of 514 executed steps at `dd55e3d84`. Document Settings and the
Sketch command label round trip both passed in this run.

The two remaining cases are **Continue a modeling command after switching
ribbon tabs** and **Minimize and restore the ribbon during a modeling
preview**. Both stopped at an undefined fixture step. Their expected Shell
previews depend on the kernel behavior recorded in
[Martensite #554](https://github.com/Oblikovati/Martensite/issues/554).

The [35-file published snapshot](../2026-11-24-ribbon-display-full-3856/snapshot.json)
was `9241d437e7cad18412942240d25f9e58c0fde911`. The run used X11, Xvfb,
Openbox and Vulkan Lavapipe. It selected the full `ribbon-display`
feature with no scenario filter.

See [#3856](https://github.com/Oblikovati/Oblikovati/issues/3856).

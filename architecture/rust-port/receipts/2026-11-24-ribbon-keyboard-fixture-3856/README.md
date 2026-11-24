# Ribbon Keyboard Navigation with the published Part fixture

The [published snapshot](snapshot.json) names
`fixtures/cad/workspace-layout-native.opd` for camera rotation, face picking,
and browser navigation. The [initial binding inventory](before/published-bindings.json)
found three uses of undefined fixture steps. The steps now open that Part
through the native Open dialog. The [new inventory](after/published-bindings.json)
reports zero unresolved steps.

The [first full published run](before/published/cucumber.json) passed 17 of
18 scenarios, including camera rotation and face picking. Browser navigation
selected the intended sketch, but the test expected the outline for a hidden
sketch. The saved profile was already visible and displayed the theme's
selection color. The updated check passed all 13 steps on the
[published Part](after/published-browser/cucumber.json) and on the
[pinned hidden profile](after/pinned-browser/cucumber.json).

The native runs used application base `277dce9f7` with the local fixture
bindings, X11, Xvfb, Openbox, and Vulkan Lavapipe. The full pinned run is
recorded in [its own receipt](../2026-11-24-ribbon-keyboard-pinned-3856/README.md).
See [#3856](https://github.com/Oblikovati/Oblikovati/issues/3856).

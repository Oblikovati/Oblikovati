# Current native Ribbon Panel Visibility

Forty-three dependency-ready cases passed 508 steps on Linux X11 with
Xvfb, Openbox and Vulkan Lavapipe. The 54 published declarations
expanded to 55 executions. Both examples in the outline reached Show Panels
from an inactive tab and failed on the missing Primitives row.

Nine executions require entries absent from the 3D Model or Manage panel
menus, including Primitives, Simulation, 3D Print, and Update. The
[Manage run](case-52/cucumber.json) and [side-docked 3D Model run](case-53/cucumber.json)
record the missing inventory in the native UI.

The [Shell panel visibility run](case-41/cucumber.json) and
[cancel run](case-42/cucumber.json) opened Shell properties on the native
block and reached the missing preview before face selection. The
selected-face path also has a separate
[pinned kernel refusal](../2026-11-25-ribbon-docking-current-3860/README.md)
and [v0.14.0 tolerance reproduction](../2026-11-25-martensite-v014-shell-check-4106/README.md).

The [custom panel case](case-43/cucumber.json) expects User Commands to be
absent from the direct selector. The native selector includes the row, as
documented by the application, and the
[native customization regression](custom-selector-regression.txt)
passed hide and restore from both visibility controls.

Individual passing reports are in the
[fixture](../2026-11-25-ribbon-panel-visibility-fixture-3856/README.md),
[native contexts](../2026-11-25-ribbon-panel-visibility-native-contexts-3856/README.md),
[Sketch and Tools](../2026-11-25-ribbon-panel-visibility-sketch-tools-3856/README.md),
and [menus and docking](../2026-11-25-ribbon-panel-visibility-menus-3856/README.md)
receipts. The final passing checks covered the [floating ribbon](case-08/cucumber.json)
and [retained Modify title strip](case-38/cucumber.json).

The runs used the test changes identified in the individual receipts and
the scenario bindings in this commit. Builds included existing local GUI
filesystem, opening and saving edits. The [input hashes](current-inputs.sha256)
matched the isolated source and fixtures. See the
[combined report](cucumber.json) and [scenario inventory](scenario-inventory.json).
Issue #3978 tracks strict build and runtime provenance.

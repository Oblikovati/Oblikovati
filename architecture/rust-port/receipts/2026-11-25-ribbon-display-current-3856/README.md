# Current native Ribbon Display

The 48 dependency-ready scenarios passed 556 steps on the published feature.
The final checks verified Home and Part transitions, contextual menu dismissal,
and Normal and Text Off appearance on the actual native Part and Sketch1.

Two scenarios still require a valid Shell preview on the published solid.
Their Cucumber reports retain the undefined setup or preview bindings. The
[pinned native probe](../2026-11-25-ribbon-docking-current-3860/README.md)
reproduced the tolerance refusal on the application solid. The
[tagged kernel probe](../2026-11-25-martensite-v014-shell-check-4106/README.md)
also reproduced tolerance widening on the 20 mm cube using Martensite v0.14.0.

The earlier [mode controls](../2026-11-25-ribbon-display-mode-controls/README.md),
[drawing context](../2026-11-25-ribbon-display-drawing-context-3856/README.md),
[sketch and tab cycling](../2026-11-25-ribbon-display-sketch-and-tab-cycling-3856/README.md),
[wheel and width checks](../2026-11-25-ribbon-display-wheel-and-width-3856/README.md),
[overflow and settings](../2026-11-25-ribbon-display-overflow-and-settings-3856/README.md),
[sketch and workspace checks](../2026-11-25-ribbon-display-sketch-and-workspace-3856/README.md),
and [final context checks](../2026-11-25-ribbon-display-final-context-3856/README.md)
keep their individual reports.

The matching builds used Linux X11 with Xvfb, Openbox, and Vulkan Lavapipe and
included existing local GUI filesystem, opening and saving edits. The published
[input hashes](current-inputs.sha256) matched the isolated copies. See the
[combined report](cucumber.json) and [scenario inventory](scenario-inventory.json).
Issue #3978 tracks strict build and runtime provenance.

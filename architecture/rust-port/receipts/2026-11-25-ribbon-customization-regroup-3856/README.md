# Native Plane grouping and ribbon reflow

The first case passed eleven steps. Ungrouping opened a compact
Work Features panel with separate Offset from Plane and Three Points
buttons. Regrouping restored the Plane drop-down in the compact
panel while the published Part stayed active.
[Ungroup and regroup](case-026/cucumber.json)

The second case passed eight steps. The setup regrouped Plane in a
1,240-pixel-wide window and dismissed the open flyout. A native
title-bar double-click maximized the window, expanded Work Features,
and showed more command labels. Plane retained its arrow and the
Part stayed active. [Title-bar reflow](case-027/cucumber.json)

The earlier run returned Work Features to the full ribbon as soon as
the choices were regrouped. The X11 group and ungroup regression
passed with the revised measurement behavior.
[Earlier run](before-regroup/cucumber.json) ·
[Earlier frame](before-regroup/work-features-expanded-after-regroup.png)

The Linux X11 builds used `64aece048` and this commit's changes with
the existing local GUI filesystem, opening, and saving edits.
Both cases ran on Xvfb with Openbox and Vulkan Lavapipe.
[Input hashes](../2026-11-25-ribbon-customization-current-3856/current-inputs.sha256)

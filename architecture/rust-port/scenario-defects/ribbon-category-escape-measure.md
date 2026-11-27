# Measure steps follow the ribbon category Escape scenario

**Discard a pending edit from an open command category list** starts with
an empty Part Tools target. It stages 2D Sketches, focuses Macros in the
open selector, and presses Escape. Reopening Customize shows All Commands,
the original empty target, and 2D Sketches available again. The scenario
then closes Customize.

The next step asks the user to click Done in the Measure panel, followed
by steps that remove a customized Measure command. The scenario never
opens Measure or stages the command. The functional notes describe the
Escape sequence through the empty target on reopening. They specify no
Measure interaction for this sequence.

| Source | Native result |
| --- | --- |
| Pinned `b576445b`, line 733 | [15 steps passed on Wayland before the Measure step](../receipts/2026-11-26-ribbon-category-escape-source-3856/pinned-wayland/cucumber.json) |
| Published source, line 842 | [17 steps passed on Wayland](../receipts/2026-11-26-ribbon-category-escape-source-3856/published-wayland/cucumber.json) and [17 on X11](../receipts/2026-11-26-ribbon-category-escape-source-3856/published-x11/cucumber.json) before the Measure step |

All three runs stopped at the first Measure step, which has no binding.
The build used `9e58ae120` content with existing local filesystem and
marking-menu edits. Wayland used headless Weston. X11 used Xvfb and Openbox.
Both used Vulkan Lavapipe. [Pinned feature hash](../receipts/2026-11-26-ribbon-category-escape-source-3856/pinned-input.sha256)
and [published input hashes](../receipts/2026-11-26-ribbon-category-escape-source-3856/published-inputs.sha256)
identify the source.

End the Escape scenario at the close assertion. Move the Measure steps
to a scenario that configures Measure and opens its panel. The source
folder's `BLOCKERS.md` records the correction for the scenario author.
The full declaration remains blocked. [#3856](https://github.com/Oblikovati/Oblikovati/issues/3856)

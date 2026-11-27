# Native Shell button size

Three cases passed 37 steps. The first used the Modify command's
Button Size menu to change from a small icon beside its label to a
large icon above its label. Chamfer moved to make room. Standard
restored the original presentation. [Change button size](case-025/cucumber.json)

The second case switched to Tools and back and confirmed that the
Large check mark and icon size remained. The third switched to
Text Off, located the same Shell image in the ribbon, and reopened
its context menu without a caption. It restored Standard and Normal
appearance. [Tab retention](case-050/cucumber.json) ·
[Text Off retention](case-109/cucumber.json)

The settings tests verify that older XML defaults to Standard and
that a saved Large value survives reload. Reset restores Standard.
A failed save preserves the last applied value.

The Linux X11 builds used `f9805725c`, the changes in this commit,
and the existing local GUI filesystem, opening, and saving edits.
The native runs used Xvfb, Openbox, and Vulkan Lavapipe.
[Input hashes](../2026-11-25-ribbon-customization-current-3856/current-inputs.sha256)

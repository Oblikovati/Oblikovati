# Native Customize placement and Part selection

The [window movement case](case-105/cucumber.json) passed ten steps.
It dragged Customize by the title bar and confirmed that reopening
from Tools restored the moved position with the same Part active.
The [face selection case](case-113/cucumber.json) passed thirteen steps.
It selected one face on the named solid fixture and verified the visible
Part and the one-face indicator after closing Customize.

Both cases kept the published Part active through setup and used
the [hashed inputs](../2026-11-25-ribbon-customization-current-3856/current-inputs.sha256).
The Linux X11 build used `d85418ec7` with the scenario binding changes in
this commit and existing local GUI filesystem, opening, and saving
edits. The tests ran on Xvfb with Openbox and Vulkan Lavapipe.
Head test Clippy passed with warnings denied.

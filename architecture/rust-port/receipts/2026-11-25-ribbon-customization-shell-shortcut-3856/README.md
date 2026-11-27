# Native Shell shortcut

The published case passed fifteen steps. Adding Shell left the
Modify command visible and drew its icon in Quick Access. The ribbon
menu disabled Add while the shortcut existed. The icon's own menu
removed it, and the ribbon menu then enabled Add again.
[Add and remove Shell](case-030/cucumber.json)

The same native build passed the pinned Extrude duplicate-shortcut
case and the published Shell Button Size case. GUI and Head test
Clippy passed with warnings denied.

The Linux X11 build used `fbcb0854d`, this commit's changes, and
existing local GUI filesystem, opening, and saving edits. The
case ran with Xvfb, Openbox, and Vulkan Lavapipe.
[Input hashes](../2026-11-25-ribbon-customization-current-3856/current-inputs.sha256)

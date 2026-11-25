# Native Customize command sort order

The [published case](case-106/cucumber.json) passed all twelve steps.
The test read the available list's own draw list, checked descending
order through All Commands and Sketch Tab, and verified that closing
and reopening Customize restored ascending order and the empty Part
Tools target. The View category command-transfer regression also
passed all sixteen steps.

The [input hashes](../2026-11-25-ribbon-customization-current-3856/current-inputs.sha256)
matched the isolated source and profile. The Linux X11 build used the
changes in this commit and existing local GUI filesystem, opening,
and saving edits. Runs used Xvfb with Openbox and Vulkan Lavapipe.
Head test Clippy passed with warnings denied.

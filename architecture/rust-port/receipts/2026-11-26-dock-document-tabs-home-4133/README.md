# Restore the Model dock tab after leaving Home

The native test opens PartOne and the solid block from the Docking System
fixtures. It selects Log for PartOne and Model for the solid Part. It
opens Home and switches from there to each document in turn. The checks
verify both remembered selections, the shared dock ID, the solid Part's
Model Browser, and PartOne's empty viewport.

On the baseline build at `6a2b00a86`, the test reached the solid Part after
leaving Home and found its Model tab inactive on both Wayland and X11.
The fix records the outgoing selection as Home opens and retries restoration
on the next frame if Model has yet to reappear.

The build used the test and dock changes with existing local filesystem
edits. Wayland used headless Weston. X11 used Xvfb and Openbox. Both used
Vulkan Lavapipe. [Test results](test-results.txt) and [input hashes](inputs.sha256)

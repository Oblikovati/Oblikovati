# Side dock minimum widths during resize

The native X11 test widens the right dock at 1800 × 900 using its splitter,
then resizes a workspace with a visible Part to 940 × 680. The previous
layout put the left viewport boundary at x=140, including the splitter.
The new layout gives Model 190 px and Appearance 225 px and leaves a
521 px viewport. The test verifies a visible solid, at least 55% viewport
width at 780 × 580, and the return to the exact preferred viewport bounds
at 1800 × 900. [Before](before.png) · [After](after.png)

All four GUI workspace layout tests passed. The native docking module
passed seven of nine tests. Switching back to the original Part with
Parameters docked and activating Favorites from the View menu failed on
the base revision as well as with the change.

The published Appearance Browser resize scenario passed 12 of 13 steps
with both builds. The final splitter drag back did not recover the
original bounds in either build. The result also reproduced with the
change under Openbox. [Base scenario](appearance-baseline.json) ·
[Updated scenario](appearance-current.json) ·
[Scenario and fixture hashes](appearance-inputs.sha256)

The Linux builds used base revision `91f53ed08` with the native regression
test and the pending local filesystem and Ribbon Customization edits.

# Application closing diagnostic

This is a diagnostic, not a verified acceptance receipt. The live scenario source
was unversioned and is not in the pinned snapshot. Its feature hash before and
after the run was `d87eaf041fc93701e0de5c911dd3b4c0622386d184f05954a8409400e412c0d0`.
The test ran application commit `2dee60eef` with local unbounded build inputs.

All three scenarios and 14 steps passed under Xvfb and Vulkan Lavapipe. The
close request used the X11 window-manager protocol; the Cancel choice used an
XTEST mouse click. Fixture setup used frame-loop input. The source marks clean
and no-document exit as reference capture gaps. Build provenance and receipt
reuse remain open in #3978.

The dirty prompt capture was inspected. The run log, source copy, Cucumber and
JUnit output, source observation, and artifact hashes are retained here.

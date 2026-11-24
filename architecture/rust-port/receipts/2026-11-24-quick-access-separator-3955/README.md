# Quick Access separator

The separator scenario passed all eight steps with the supplied blank
Metric Part. The context menu displayed the complete Add Separator label.
Selecting it added a visible divider next to Extrude and kept the same
Part active.

The [Cucumber report](separator-padding/cucumber.json) and
[final frame](separator-padding/final.png) came from base revision
`4b2ee0a0c15dc8dc54e0cd316552de97e5c4d217` with the menu padding fix and
existing local GUI filesystem, opening, and saving edits. The run used
Linux x86-64, Xvfb, Openbox, and Vulkan Lavapipe. The
[input hashes](current-inputs.sha256) matched the shared source and its
isolated copy after the run. Strict build and runtime provenance remains
open under #3978.

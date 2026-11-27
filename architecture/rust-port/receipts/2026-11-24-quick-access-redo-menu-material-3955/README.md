# Quick Access Redo, menu, and Material Browser

The three scenarios passed individually, with 30 steps. Each used the
supplied blank Metric Part.

| Scenario | Steps | Evidence |
| --- | ---: | --- |
| Show Redo disabled when no redo action is available | 7 | [Cucumber](redo-next/cucumber.json) |
| Inspect and dismisses quick access command choices | 12 | [Cucumber](choices-next/cucumber.json) |
| Open and close the Material Browser from quick access | 11 | [Cucumber](material-next/cucumber.json) |

The native runs used application revision `bf162e5f52092e5780bc35a6a4a5bf69ab028adc`
with existing local GUI filesystem, opening, and saving edits. The
environment was Linux x86-64, Xvfb, Openbox, and Vulkan Lavapipe. The
[input hashes](current-inputs.sha256) matched the shared source and
isolated copy after the runs. Strict build and runtime provenance remains
open under #3978.

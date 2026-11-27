# Quick Access customization with the supplied Part

Three current scenarios passed individually, with 44 steps. Each opened
the supplied blank Metric Part and kept that document active.

| Scenario | Steps | Evidence |
| --- | ---: | --- |
| Add and remove a quick access command | 15 | [Cucumber](add-remove-first/cucumber.json) |
| Move the quick access toolbar below and above the ribbon | 14 | [Cucumber](move-row-first/cucumber.json) |
| Add and remove a ribbon command through its context menu | 15 | [Cucumber](ribbon-toggle-first/cucumber.json) |

The native runs used base revision `ec41dbd6fb37e7450d8843ed2fb9ba68b65bb4f5`
with the fixture bindings and existing local GUI filesystem, opening, and
saving edits. The environment was Linux x86-64, Xvfb, Openbox, and Vulkan
Lavapipe. The [input hashes](current-inputs.sha256) matched the shared
source and isolated copy after the runs. Strict build and runtime
provenance remains open under #3978.

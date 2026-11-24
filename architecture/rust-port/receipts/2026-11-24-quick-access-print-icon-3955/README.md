# Native Quick Access Print icon

The compact printer icon appeared when Print was selected. The tests found
the actual icon in the toolbar draw list, both above and below the ribbon.
The selected setting also stayed associated with Part documents when
switching to the supplied Assembly and back. The three scenarios passed
individually with 43 steps.

| Scenario | Steps | Evidence |
| --- | ---: | --- |
| Add and remove a quick access command | 15 | [Cucumber](print-icon-toggle/cucumber.json) |
| Move the quick access toolbar below and above the ribbon | 14 | [Cucumber](print-icon-placement/cucumber.json) |
| Keep a standard shortcut selection when switching document contexts | 14 | [Cucumber](print-icon-context/cucumber.json) |

The native runs used base revision `bf162e5f52092e5780bc35a6a4a5bf69ab028adc`
with the Print icon and direct image checks, plus existing local GUI
filesystem, opening, and saving edits. The environment was Linux x86-64,
Xvfb, Openbox, and Vulkan Lavapipe. The [input hashes](current-inputs.sha256)
matched the shared source and isolated copy after the runs. Strict build
and runtime provenance remains open under #3978.

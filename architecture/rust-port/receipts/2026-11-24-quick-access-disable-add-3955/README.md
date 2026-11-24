# Disable the add option for an existing shortcut

The native scenario passed all 11 steps with the supplied blank Metric
Part active. Adding Extrude disabled the ribbon menu's Add to Quick Access
Toolbar option. Removing the shortcut enabled the option again.

The [Cucumber report](disable-add-native/cucumber.json) and
[final frame](disable-add-native/final.png) came from base revision
`ece7b065e231d71a95fb18488d1a06263af7b04d` with the fixture binding change and
existing local GUI filesystem, opening, and saving edits. The environment
was Linux x86-64, Xvfb, Openbox, and Vulkan Lavapipe. The
[input hashes](current-inputs.sha256) matched the shared source and its
isolated copy after the run. Strict build and runtime provenance remains
open under #3978.

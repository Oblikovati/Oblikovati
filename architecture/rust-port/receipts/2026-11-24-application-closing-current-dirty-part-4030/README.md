# Cancel application close with an unsaved Part

The current scenario passed all nine steps. The native application opened a
disposable copy of the supplied Part and created an unsaved sketch on the
XY datum plane. Closing the application displayed the save-changes prompt.
Cancel dismissed it and kept the Part and application open.

The [Cucumber report](cancel-dirty-xy/cucumber.json) and
[final frame](cancel-dirty-xy/final.png) came from base revision
`d12f20a28f84e54b8558910d00fa56aedc4876fa` with the updated sketch step
binding and existing local GUI filesystem, opening, and saving edits.
The environment was Linux x86-64, Xvfb, Openbox, and Vulkan Lavapipe.
The [input hashes](current-inputs.sha256) matched the shared source and
isolated copy after the run. Strict build and runtime provenance remains
open under #3978.

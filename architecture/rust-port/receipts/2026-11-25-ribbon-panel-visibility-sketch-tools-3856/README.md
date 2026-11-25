# Native Ribbon Panel Visibility in Sketch and Tools

Two current cases passed 22 steps. The Sketch1 case finished and reopened
the saved sketch using its actual browser name and checked its identity
and geometry. The Tools case hid and restored Tutorials from Show Panels,
checked both checklist states, and confirmed that the Part stayed active.

The test build was based on `626b53a8b` and included the fixture and
scenario bindings under review and existing local GUI filesystem, opening
and saving edits. The Linux X11 runs used Xvfb, Openbox and Vulkan
Lavapipe. The [input hashes](current-inputs.sha256) matched the isolated
source and fixtures. Issue #3978 tracks strict build and runtime provenance.

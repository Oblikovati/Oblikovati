# Inference scope integration

The native X11 scenario `Keep a pinned constraint expansion across sketch edit`
passed 13 of 13 steps on 2026-11-19. The feature SHA-256 was
`645576c3b62e89cdb6c668d5579123e6880a33f9e4a8020f922f1b08e0502ca2`.

The tested tree combined `rust-work` at `3a3844c6b` with the scope branch at
`93a02387e`, the merge resolution, and the scenario bindings in the merge
commit. The API dependency was pinned to `051956aa29a83643e84a955531633cabac352373`.
The test used Xvfb and the Vulkan lavapipe driver.

This is a focused scenario result, not a full-feature run or verified build
provenance. The current-command, selected-geometry, and invalid-scope model
tests also passed. See #4107 and #3978.

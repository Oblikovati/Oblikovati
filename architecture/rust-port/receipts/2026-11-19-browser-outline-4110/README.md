# Browser selection outline

The model-browser selection and context-menu scenarios passed all 9 and 10
steps on 2026-11-19. The feature SHA-256 was
`603382ec4009d73a74e960c5981d255c4ca9ac5489b94fcf57193c7b09458b2e`.
The run used the issue worktree based on `b7d69d4ca`, Xvfb, and Vulkan
lavapipe. The native X11 mouse test checked the browser selection state, blue
edge, and toolbar before and after an empty-space click. The two images show
those states.

The former whole-frame color test counted 36 blue pixels in the axis triad
after selection cleared. The new color test excludes UI window rectangles and
has a regression that detects real blue pixels outside those rectangles.
These focused results are not a full-feature run or verified build provenance.
See issue #4110 and issue #3978.

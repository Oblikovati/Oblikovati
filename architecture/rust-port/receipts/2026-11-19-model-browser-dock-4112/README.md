# Model browser docking

The resize, float and redock, and opposite-side scenarios passed with 9, 9,
and 8 steps on 2026-11-19. The feature SHA-256 was
`603382ec4009d73a74e960c5981d255c4ca9ac5489b94fcf57193c7b09458b2e`.
The focused runs used this worktree based on `b7d69d4ca`, Xvfb, and Vulkan
lavapipe. Two X11 pointer tests also passed for resizing and docking on both
edges. Each dock check required a nonzero dock ID and viewport adjacency.

These focused results do not constitute a full-feature run or verified build
provenance. See issue #4112 and issue #3978.

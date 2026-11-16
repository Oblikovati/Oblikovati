<!-- SPDX-License-Identifier: GPL-2.0-only -->

# Precise Input pin diagnostic (#4017)

The pinned `move-21` scenario passed seven native frame-loop steps at scenario
snapshot `e76cb749ed43c73a91c24854057da7c890041f51`. The same scenario failed
before the change at the checked-state assertion. The final screenshot was
inspected. A separate native frame-loop test dragged the title bar while pinned
and after unpinning; the first drag kept the window position and the second
moved it. The control stays interactive while pinned.

The Go reference did not establish equivalent Precise Input pin behavior.
These results do not establish OS-level input, persistence across application
restarts, or bounded build provenance (#3978). They were obtained from an
uncommitted test tree on Linux with Xvfb and Vulkan Lavapipe. The full Move
feature run is not represented by these focused results.

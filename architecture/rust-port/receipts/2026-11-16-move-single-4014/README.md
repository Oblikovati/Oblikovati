<!-- SPDX-License-Identifier: GPL-2.0-only -->

# Move single-selection diagnostic (#4014)

The pinned `move-10` scenario passed seven native frame-loop steps on snapshot
`e76cb749ed43c73a91c24854057da7c890041f51`. The test toggles single-selection
optimization through the Move panel, selects the ellipse, checks the Base Point
stage and prompt, and cancels. It checks the retained sketch recipe, history,
and visible ellipse. The final capture was reviewed.

The run used an uncommitted test tree on Linux with Xvfb and Vulkan Lavapipe.
It does not establish OS-level input or bounded build provenance (#3978). The
last full Move report predates this change.

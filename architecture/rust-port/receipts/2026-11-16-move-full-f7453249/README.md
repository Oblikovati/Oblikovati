<!-- SPDX-License-Identifier: GPL-2.0-only -->

# Move feature diagnostic

The pinned Move feature ran against application commit
`f7453249d` and scenario snapshot
`e76cb749ed43c73a91c24854057da7c890041f51` on Linux, Xvfb and Vulkan
Lavapipe. Nine of 27 scenarios passed; 18 failed or stopped at an undefined
step. The normal runner stops each scenario at its first failure, so the
report does not enumerate every remaining gap.

The runner used frame-loop injected input. The results are diagnostic: OS-level
input and bounded build provenance are not established (#3978). The in-progress
Auto Fade and Move recovery work was not in this tested application commit.

<!-- SPDX-License-Identifier: GPL-2.0-only -->

# Move Copy diagnostic (#4014)

The pinned `move-08` scenario passed all eight steps in a native frame-loop
run at scenario snapshot `e76cb749ed43c73a91c24854057da7c890041f51`. It
reopened Move, enabled Copy, selected an ellipse and an endpoint, then checked
that both original and displaced ellipses remained after Done. The checks
included the sketch identities, original center and visible outlines. The final
capture was inspected.

The test used an uncommitted test tree on Linux with Xvfb and Vulkan Lavapipe.
It does not establish OS-level input or bounded build provenance (#3978). The
previous full Move run passed 4 of 27 scenarios; it predates this wording
change and is not a result for the current tree.

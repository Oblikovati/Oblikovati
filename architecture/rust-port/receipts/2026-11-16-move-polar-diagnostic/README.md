<!-- SPDX-License-Identifier: GPL-2.0-only -->

# Polar Move diagnostic

On scenario snapshot `e76cb749ed43c73a91c24854057da7c890041f51`, the temporary
`move-15` binding passed eight steps and stopped at the visible-position step.
The test had assumed that “lower” meant below the original ellipse; the source
does not establish that reference, so that failure does not prove a product
defect. The scenario folder requests the missing preview setup and reference.
The temporary binding was not registered as an acceptance test.

A separate native test entered 20 mm at zero degrees with the retained polar
mode, checked a 2 cm sketch-X displacement and the visible ellipse, then undid
the move and checked the original sketch recipe and outline. The diagnostic
capture shows the accepted position before undo and was reviewed. These runs
used frame-loop input on Linux, Xvfb and Vulkan Lavapipe. They do not establish
OS-level input, bounded build provenance (#3978), or verified scenario acceptance.

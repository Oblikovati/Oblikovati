<!-- SPDX-License-Identifier: GPL-2.0-only -->

# Precise Move diagnostic (#4014)

The pinned `move-06` scenario passed all ten steps with the native frame-loop
runner. The test checks that the ellipse center moves 1 cm in sketch X and
zero in Y, as well as its visible position and retention after Done. The
scenario snapshot is `e76cb749ed43c73a91c24854057da7c890041f51`. The image
was inspected and shows the ellipse after the Move panel closed.

This is a diagnostic run from an uncommitted test tree. It does not establish
OS-level input or bounded build provenance (#3978), and does not verify the
complete Move feature.

The subsequent full feature diagnostic used the same test source and scenario
pin. It passed 2 of 27 scenarios and reported 25 failures or undefined steps.
The report and log are in `full/`. A failed step stops its scenario, so the
report does not enumerate every remaining gap.

<!-- SPDX-License-Identifier: GPL-2.0-only -->

# Move history diagnostics (#4014)

The pinned `move-07` and `move-09` scenarios each passed eight steps using the
native frame-loop runner. The first checks the precise 10 mm move, the original
sketch recipe after undo, the displaced center after redo, and the visible
outline. The second checks the copied and original ellipses through undo and
redo. The final captures have the same hash, as both end at the original
ellipse after a second undo. The capture was inspected.

The scenario snapshot is `e76cb749ed43c73a91c24854057da7c890041f51`. These
runs used an uncommitted test tree. They do not establish OS-level input or
bounded build provenance (#3978), and do not verify the complete Move feature.

<!-- SPDX-License-Identifier: GPL-2.0-only -->

# Precise Input invalid-coordinate diagnostic (#3916)

Application commit: `db013eeb04e277e7d3d24ba55d8899ab816a8081`.
Scenario snapshot: `e76cb749ed43c73a91c24854057da7c890041f51`.

`precise-input-03` passed all seven steps in a native Xvfb window with the
Lavapipe Vulkan driver. The strengthened test compares the complete sketch and
its undo/redo history before and after the rejected input. It checks each
reopened coordinate field for a finite number and checks that the invalid `a`
is absent. The final capture shows the reopened toolbar and numeric fields.
The pinned source checkout passed verification after the run.

As a negative control, an earlier draft of the final assertion re-entered `a`
after reopening the toolbar. The last step failed with `expected the invalid
value to be cleared`; the injected input was then removed. The associated
`negative-control.json` is a failed diagnostic, not a scenario pass.

These runs use frame-loop input, and their host and build inputs are unbounded
(#3978). They are diagnostic evidence, not reusable acceptance receipts. The
full workspace gate and operating-system input checks remain open.

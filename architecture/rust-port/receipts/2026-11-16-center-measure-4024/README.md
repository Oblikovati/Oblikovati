<!-- SPDX-License-Identifier: GPL-2.0-only -->

# Centered rectangle measurement diagnostic

The pinned `rectangle-center-10` diagnostic first stopped at an unbound
Measure step; `red-gap-report.json` retains that run, and later failures are
tainted. After binding the current wording, the focused native frame-loop run
passed. It selected two curves, checked 8.000 mm and 4.000 mm lengths,
90.00 degrees and 0.000 mm minimum distance, then confirmed that closing
Measure preserved the sketch and its history. The report is `gap-report.json`.
The scenario snapshot was `e76cb749ed43c73a91c24854057da7c890041f51`, checked
before and after. This is diagnostic evidence, not verified acceptance (#3978).

A separate standalone run used XTest, Xvfb, Vulkan Lavapipe and an isolated
YZ startup sketch. It created a centered rectangle by entering `8`, Tab, `4`,
Enter, exited placement, and opened Measure with `M`. Mouse clicks on adjacent
sides produced `result.png`, which shows both curve selections and the expected
values. `closed.png` shows the sketch after Done. MCPBridge listed the same
four side IDs, two construction line IDs and projected origin before and after
closing Measure. The GUI and Xvfb stopped after the run.

The standalone build used worktree base `997d5043d` with uncommitted test
bindings. Its GUI SHA-256 was
`86e0f1cd3deec0642c47d050c80324f3de7711ce10d18c234dcc1ba4655d8794`;
MCPBridge SHA-256 was
`2942bdba7a756d20ea17f24fb18e9eb41f495e20b1d259caf9d28d77e237b20b`.
The scripts and screen coordinates are specific to this run. This OS-input
observation is supporting evidence, not a pinned acceptance receipt.

The full pinned feature diagnostic passed 9 of 15 scenarios; six stopped at
unbound steps. `full-gap-report.json` retains that run. The full gate was not
run.

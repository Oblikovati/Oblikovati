<!-- SPDX-License-Identifier: GPL-2.0-only -->

# Startup sketch OS-input check

On 2026-11-16, XTest sent mouse and keyboard events to the standalone GUI
under Xvfb. MCPBridge captured its window. The initial binary used the
startup-sketch worktree based on `1817cb72e`; it had uncommitted changes.
`xy.png` shows Sketch1 in Front after choosing x-y. `yz.png` shows Sketch1
in Right after choosing y-z through the metric template chooser.

`xz_retry.png` exposed a failure: x-z opened in Bottom view. After the camera
fix, the standalone GUI was rebuilt and started with the same isolated
preferences. `xz_fixed.png` shows its startup sketch in Top orientation.
The control label is rotated in that view; this capture does not establish
that its text orientation is correct. The rebuilt GUI SHA-256 was
`86e0f1cd3deec0642c47d050c80324f3de7711ce10d18c234dcc1ba4655d8794`.
MCPBridge SHA-256 was
`2942bdba7a756d20ea17f24fb18e9eb41f495e20b1d259caf9d28d77e237b20b`.
The preference was restored to `none`, and the GUI and Xvfb were stopped.

`gap-report.json` is a filtered diagnostic run of `new-sketch-09` against
snapshot `e76cb749ed43c73a91c24854057da7c890041f51`. It used frame-loop
input and is not an acceptance receipt. The snapshot verification passed
before and after the run. Native build provenance remains open in #3978.

The full feature diagnostic reported nine passing scenarios and one unbound
scenario (`new-sketch-06`). It ran before the final direct placement assertion
was added to the focused test. `full-gap-report.json` retains that report.

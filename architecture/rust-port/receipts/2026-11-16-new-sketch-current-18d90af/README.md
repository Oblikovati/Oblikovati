<!-- SPDX-License-Identifier: GPL-2.0-only -->

# New Sketch diagnostic

On 2026-11-16 the full `new-sketch` feature ran at application commit
`18d90af0d` against scenario snapshot `e76cb749ed43c73a91c24854057da7c890041f51`.
The snapshot verification passed before and after the run. Nine scenarios
passed; `new-sketch-06` had an unbound step. Its accumulated sketch fixture
is requested in the source folder's `BLOCKERS.md`.

The command used `OBK_BDD_FEATURE=new-sketch`, `OBK_BDD_DIAGNOSTIC=1`,
`VK_ICD_FILENAMES=/usr/share/vulkan/icd.d/lvp_icd.json`, and
`xvfb-run -a -s '-screen 0 1600x1200x24' cargo test --locked -p
oblikovati-head --lib sketch_scenarios_use_the_native_window -- --nocapture`.
It used the main worktree target and frame-loop input. The unrelated modified
`README.md` was present. Xvfb stopped after the test.

This is a diagnostic, not an acceptance receipt: the runner's build provenance
is not bounded (#3978), and input was not sent through the operating system.

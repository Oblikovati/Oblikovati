# Circle midpoint fixture diagnostic

Issue #3996. The run used application base commit
`a55445b6fdef447bd493969491ef5c2d75737034` plus the staged test-source patch
with SHA-256 `2f0c582d6aa7b402ea151fde210e75472cbb66a8937c5fd9b547bebada04865d`.
The Martensite lockfile revision was `d051ff699982faba4ec2df3ee9a47e14f38dedbd`.
The pinned scenario snapshot was `e76cb749ed43c73a91c24854057da7c890041f51`;
all 1,292 files were verified before and after the runs. The circle feature
SHA-256 was `a18dcad10755d0ceb8b49b1d907b3800b1c6847c8b3548eb5d2535487b2be445`
before and after. The ellipse feature SHA-256 was
`af53af88f7a32b441d76154fd75aeb36e68d4e673c47d21ecefa5be14cef5a6f`.

Runs used Linux x86-64, Xvfb at 1600 × 900, Lavapipe via
`/usr/share/vulkan/icd.d/lvp_icd.json`, and `XMODIFIERS=@im=none`. Each native
run used the pinned checkout, a unique report directory, and the same dedicated
worktree target. Its command was:

```sh
xvfb-run -a -s '-screen 0 1600x900x24' env \
  XMODIFIERS=@im=none VK_ICD_FILENAMES=/usr/share/vulkan/icd.d/lvp_icd.json \
  OBK_BDD_ROOT=/tmp/rust-midpoint-3996-JM1wi6AQ/scenarios \
  OBK_BDD_REPORTS=/tmp/rust-midpoint-3996-JM1wi6AQ/report-SCENARIO \
  OBK_BDD_FEATURE=FEATURE OBK_BDD_SCENARIO=SCENARIO \
  CARGO_TARGET_DIR=/tmp/rust-midpoint-3996-JM1wi6AQ/target \
  cargo test --locked -p oblikovati-head --lib \
    sketch_scenarios_use_the_native_window -- --nocapture
```

- Circle 11 passed 1 scenario and all 12 steps. The final Undo assertion checks
  the midpoint constraint, line and constraint identities, endpoint positions,
  retained circle and center, and a rendered line witness. The final PNG is
  `circle-11/circle-center-point-11-midpoint-after-undo.png`.
- Circle 10 reached the prior blocker: 4 steps passed, then the HUD showed
  `0.491 mm` where the scenario expects `0.000 mm`. Later steps did not run.
- Ellipse 12 passed 1 scenario and all 7 steps using the shared fixture.
- Ellipse 20 created the shared fixture, then stopped at an undefined step:
  `the user open Circle dropdown and choose Ellipse`. Later steps did not run.
- The negative unit test confirmed a plain line without a midpoint relation is
  rejected. `cargo fmt --all --check` and `cargo clippy --locked -p
  oblikovati-head --all-targets -- -D warnings` passed.

These are frame-loop input diagnostics, not OS-level input acceptance. Build
provenance is not bounded, so they are not verified receipts (#3978).

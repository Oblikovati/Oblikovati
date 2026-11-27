# Docking System full local run

All 11 scenarios and 108 steps passed on application commit
`5ccb98c7bda6778405a69d163a12e1b2db03c76b`. The corrected dock-tab menu
scenario used the native `.opd` fixture. The restart scenario launched the
desktop twice. The captured layouts before and after restart have identical
pixels. I inspected the restored layout and docking preview.

`inputs.sha256` records the live feature, support files, and fixture hashes.
The hashes matched before and after the run. `scenarios.bundle` contains the
local scenario snapshot used for the run. The snapshot excludes the live
blocker notes. `snapshot.json` identifies the source commit and files.

The desktop binary was built with
`cargo build --locked -p oblikovati-head --bin oblikovati`. The runner used
Linux Xvfb at 1440 × 900 with Openbox and Vulkan Lavapipe. The runner selected
the full `docking-system` feature with no scenario filter.

```sh
cargo test --locked -p oblikovati-head --lib \
  application_scenarios_use_the_native_window -- --nocapture
```

The dock-tab menu and restart driver used X11 mouse and keyboard input.
Other steps also used scripted window events and rendered-frame checks.
Cucumber JSON and JUnit record the result. The application build and runtime
environment were unsealed, so this is a local pass and does not qualify as
verified ledger evidence under issue #3978.

<!-- SPDX-License-Identifier: GPL-2.0-only -->

# New Sketch view preference diagnostic

On 2026-11-16, `new-sketch-10 Compare automatic normal view on creation and
edit` passed its six steps in diagnostic mode on application base `5be6fb1e3`
with uncommitted bindings now in `4c74204d9` (tracked tree
`a3296b32b9eb0c0316e7647959c983116a23adc4`). The scenario source snapshot was
`e76cb749ed43c73a91c24854057da7c890041f51`, verified before and after the
run. The feature SHA-256 was
`e4011485e2cfe78197b73450bea0d26ba2b0dbc54f2307d3b27364ffe9d9a20d`.

The fixture created six Part documents and one finished sketch in Part6. It
selected Home view, applied the disabled Part preference through Application
Options, and created Sketch2 on YZ. The test compared the actual camera with
the saved oblique camera. It then finished Sketch2, applied the enabled
preference, selected Edit Sketch on that same browser row, checked the Right
ViewCube label and camera direction, and checked the active sketch identity.

The Linux Xvfb run used Vulkan Lavapipe and frame-loop pointer and keyboard
input. It saved no screenshot. Strict head Clippy, formatting, and docs lint
passed. This is a diagnostic, not a verified BDD receipt: the full journey has
not been checked with OS input and build provenance is unbounded (#3978). The
API route and Go compatibility remain open (#4010). The test and Xvfb exited.

```sh
OBK_BDD_ROOT=/tmp/obk-view-bdd-scenarios \
OBK_BDD_REPORTS=/tmp/obk-view-bdd-report4 OBK_BDD_FEATURE=new-sketch \
OBK_BDD_SCENARIO=new-sketch-10 OBK_BDD_DIAGNOSTIC=1 \
CARGO_TARGET_DIR=/tmp/obk-view-bdd-target CARGO_PROFILE_TEST_DEBUG=0 \
CARGO_INCREMENTAL=0 VK_ICD_FILENAMES=/usr/share/vulkan/icd.d/lvp_icd.json \
XMODIFIERS=@im=none xvfb-run -a -s '-screen 0 1920x1200x24 -noreset' \
  cargo test --offline --locked -p oblikovati-head --lib \
  sketch_scenarios_use_the_native_window -- --nocapture
```

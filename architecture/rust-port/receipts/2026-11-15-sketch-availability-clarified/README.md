<!-- SPDX-License-Identifier: GPL-2.0-only -->

# Clarified sketch ribbon availability diagnostic

On 2026-11-15 the full pinned feature ran on Linux x86-64 using Xvfb and Mesa
software Vulkan. The native harness injects input through the GUI frame loop.
One of three scenarios passed, with 21 of 23 executed steps passing. Both
outside-edit scenarios stopped at Line being disabled. Later availability
checks did not run. The edit-sketch scenario passed its prompt, tooltip,
Finish and workspace-restoration checks. Issue #3956 remains open.

The application base was `76347ad3b` with the changed step file SHA-256
`626c4492d2cece12029e17b887d9f7189db873441d1c4c1a3b9047a260f115d8`.
The source snapshot `05317bbb2d0809a532b29b7ff6f726a5722b865f` was verified
before and after the run. The feature SHA-256 was
`59959ff37cf45000a2fc634c665c70a0b784ae721f6fe8db1dd7d4399d659a0d`.
Build provenance remains open (#3978); this is not acceptance evidence.

```sh
OBK_BDD_ROOT=/tmp/obk-sketch-avail-3956-scenarios \
OBK_BDD_REPORTS=/tmp/obk-sketch-avail-3956-report \
OBK_BDD_FEATURE=ribbon-command-availability \
VK_ICD_FILENAMES=/usr/share/vulkan/icd.d/lvp_icd.json XMODIFIERS=@im=none \
xvfb-run -a -s '-screen 0 1920x1080x24 -noreset' \
cargo test --locked -p oblikovati-head --lib \
  application_scenarios_use_the_native_window -- --nocapture
```

The run used a dedicated Cargo target with debug information and incremental
compilation disabled. The command failed because of the two scenario failures.
The JSON, JUnit, source report and captures are retained here.

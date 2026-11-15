# Viewport zoom checks

On 2026-11-15, the pinned zoom feature ran at application commit
`581e868dc3a1a65750a273e2071e2f041b0f41c9`: seven scenarios and 74 steps
passed. The source checkout remained clean, and the live feature hash matched
the pinned file. The Zoom All capture was inspected. Five scenarios have
individual receipts. The two Zoom Selected scenarios await a face-specific
framing oracle: the current size assertion measures the solid extent, not the
selected face. See issue #3940.

The native harness injects input into the frame loop; it does not establish
OS-level keyboard and mouse delivery. The full Go baseline remains open in
issue #3861.

```sh
OBK_BDD_ROOT=/tmp/obk-zoom-3881-snapshot \
OBK_BDD_REPORTS=/tmp/obk-zoom-3881-acceptance \
OBK_BDD_FEATURE=viewport-zoom \
CARGO_TARGET_DIR=/home/vmiguel/git/oblikovati-workspace/oblikovati-rust/target \
XMODIFIERS=@im=none VK_ICD_FILENAMES=/usr/share/vulkan/icd.d/lvp_icd.json \
xvfb-run -a cargo test --locked -p oblikovati-head --lib \
  application_scenarios_use_the_native_window -- --nocapture
```

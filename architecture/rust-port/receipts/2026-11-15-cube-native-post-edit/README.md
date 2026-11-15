# Native cube integration check

Application revision: `fbc3068bfd8a24c9aa09563d8afadf39c0afee4e`.
On Linux with Xvfb and Vulkan Lavapipe, the following test passed:

```sh
XMODIFIERS=@im=none VK_ICD_FILENAMES=/usr/share/vulkan/icd.d/lvp_icd.json \
xvfb-run -a -s '-screen 0 1920x1200x24' \
  cargo test --locked -p oblikovati-head --lib \
  native_new_part_button_creates_and_selects_an_unsaved_document -- --nocapture
```

The test drives native frame-loop events through a parameter-driven sketch,
extrusion, parameter edit, history, save/reopen, per-face checks and reference
resolution. It is not an OS-input run or a pinned BDD scenario. One pre-reentry
pixel assertion can accept an unrelated gray pixel; #3969 tracks its repair.
Use the separate [live cube receipt](../2026-11-15-live-parameter-cube/README.md)
for OS-input and MCPBridge observations. The solve precision certificate remains
open (#3928).

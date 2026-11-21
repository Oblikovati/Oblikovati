# Native capture display size

On clean application commit `687d7dea0`, the native composition test failed
under default `xvfb-run`: its 1280 by 1024 screen cannot contain the complete
1440 by 900 window. X11 `GetImage` returned `BadMatch` while reading the native
drawable. The unchanged test passed on a 1920 by 1080 screen, and passed again
through `tools/with-xvfb.sh`. The test compares the complete native drawable;
no capture assertion or render path changed.

The wrapper's two fake-process tests check the screen configuration, argument
and status forwarding, a missing command, and a missing `xvfb-run`. Strict
build-tool Clippy and Markdown lint passed. `make gate-xvfb` launched the
gate and propagated its failure: the API pin is not yet released (#3965).
Compressed logs retain the test results. This is focused evidence, not a
passing full gate. The shared setup and remaining gate are tracked in #3973.

```sh
CARGO_TARGET_DIR=/tmp/obk-native-composition-target \
CARGO_PROFILE_DEV_DEBUG=0 CARGO_PROFILE_TEST_DEBUG=0 CARGO_INCREMENTAL=0 \
VK_ICD_FILENAMES=/usr/share/vulkan/icd.d/lvp_icd.json \
./tools/with-xvfb.sh cargo test --locked -p oblikovati-head --lib \
  native_owned_imgui_packet_composes_with_the_document_after_context_destruction \
  -- --nocapture
```

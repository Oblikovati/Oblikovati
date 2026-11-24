# First click dispatches Save

The [isolated gate failure](before.log) captured the button input but
dispatched no Save action. The test used a fixed point and supplied no
active document. It now supplies an active Part and locates the enabled
Save icon in the rendered toolbar.

The [rerun](after.log) captured the press and release, without any pointer
motion event, and dispatched Save exactly once.

```sh
tools/with-xvfb.sh env VK_ICD_FILENAMES=/usr/share/vulkan/icd.d/lvp_icd.json \
  LP_NUM_THREADS=4 cargo test --locked -p oblikovati-head --lib \
  first_click_without_motion_is_captured_and_dispatched_once -- --nocapture
```

See [#3860](https://github.com/Oblikovati/Oblikovati/issues/3860).

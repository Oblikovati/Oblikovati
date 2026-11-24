# Native GUI packet composition

The [isolated gate failure](before.log) lost all 23,955 baseline solid
pixels when the GUI packet was applied. The test supplied an empty document
state to the GUI, which drew the Home workspace over the active Part.

The test now builds the packet from the native owner's active Part state.
The [rerun](after.log) passed the solid pixel, scene generation, GUI
bounds, context lifetime, and renderer retirement checks.

```sh
tools/with-xvfb.sh env VK_ICD_FILENAMES=/usr/share/vulkan/icd.d/lvp_icd.json \
  LP_NUM_THREADS=4 cargo test --locked -p oblikovati-head --lib \
  native_owned_imgui_packet_composes_with_the_document_after_context_destruction -- --nocapture
```

See [#3860](https://github.com/Oblikovati/Oblikovati/issues/3860).

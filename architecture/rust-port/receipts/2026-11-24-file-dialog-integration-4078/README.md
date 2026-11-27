# File dialog integration

The gate reported three missing File menu actions. An isolated run reproduced
the missing Save dispatch in the [before log](before.log). The tests clicked
fixed coordinates from an earlier layout. They now find File and each menu
row in the rendered frame before sending pointer input.

All four File dialog integration tests passed on Linux X11 under Xvfb after
the change. The assertions still check Open and Save As path dispatch,
Save and Close behavior, and cancellation. See the [after log](after.log).

The four tests also pass with four Cargo test threads. A test-local guard
holds Dear ImGui's active context for the lifetime of each engine. A run
without the guard failed when tests tried to activate overlapping contexts.
See the [parallel before](parallel-before.log) and [after](parallel-after.log)
logs.

```sh
tools/with-xvfb.sh env VK_ICD_FILENAMES=/usr/share/vulkan/icd.d/lvp_icd.json \
  LP_NUM_THREADS=4 cargo test --locked -p oblikovati-gui --test file_dialog \
  -- --test-threads=1
```

See [#4078](https://github.com/Oblikovati/Oblikovati/issues/4078).

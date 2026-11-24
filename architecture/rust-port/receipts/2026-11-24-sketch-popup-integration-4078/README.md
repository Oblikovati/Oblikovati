# Sketch canvas dismissal integration

The [gate reproduction](before.log) dispatched no dismissal token. The
[viewport probe](viewport-probe.log) showed that a click inside the reported
viewport still could not open the canvas menu with the original fixture.
The fixture declared sketch authoring but had no active document.

The test now supplies an active Part and derives its pointer positions
from the viewport bounds. It confirms that the popup opens, then checks
the token emitted by Escape and by clicking outside it. Both paths passed.
See the [after log](after.log).

```sh
tools/with-xvfb.sh env VK_ICD_FILENAMES=/usr/share/vulkan/icd.d/lvp_icd.json \
  LP_NUM_THREADS=4 cargo test --locked -p oblikovati-gui \
  --test sketch_context_dismiss -- --nocapture
```

See [#4078](https://github.com/Oblikovati/Oblikovati/issues/4078).

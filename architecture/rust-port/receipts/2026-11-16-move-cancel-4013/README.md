<!-- SPDX-License-Identifier: GPL-2.0-only -->

# Move cancellation native run

On 2026-11-16, the pinned `move-11` scenario passed seven steps on the native
window using application commit `beda936ec22fcb3e389039ef83d4eabd22ce79e4`.
The scenario checkout was clean at
`e76cb749ed43c73a91c24854057da7c890041f51` before and after the run. The Move
feature SHA-256 was
`42fe188a2145d98a47e57b310df3e4ba7cd80f071d54cd1df9ada567c2b94f1b`.

The binding creates an editable ellipse, reopens Move with single-selection
optimization, selects the ellipse and its center, and cancels before an
endpoint. It checks the endpoint prompt, closed Move state, Ready status,
visible ellipse, exact retained sketch recipe, and undo/redo labels. The
capture shows the retained ellipse after Escape. The run used Xvfb at
1920 × 1200 and Vulkan Lavapipe, with frame-loop injected input.

Before the commit, the full Move feature ran with the same test sources. It
reported one passing scenario and 26 failing or undefined scenarios; the full
report and log are under `full/`. Some existing bindings could not reach Move
when the Modify panel was collapsed (#4014). Other steps are unbound. The
feature is not complete. The focused run is not a verified acceptance receipt:
OS-level input and native build provenance (#3978) are not established.

```sh
OBK_BDD_FEATURE=move OBK_BDD_SCENARIO=move-11 \
  cargo test --locked -p oblikovati-head --lib \
  sketch_scenarios_use_the_native_window -- --nocapture
```

# Native detached drawing acceptance

The desktop opens the drawing as a system-managed window with its own sheet,
bottom tab, and tab context menu. Openbox identified two distinct top-level
windows. The capture shows the drawing title bar and tab above the main Part
workspace. The drawing uses the same Martensite runtime as the main window.

The three detachment scenarios passed with Openbox and real X11 drag, mouse,
and keyboard events. A click on the drawing sheet changed the context of the
main ribbon and browser. A click on the main Part tab restored its context
and left the drawing open. The native close regression restored the same
drawing to the main strip. A second regression verified that hiding the main
strip kept the native drawing and its bottom tab visible.

The desktop passed a Windows GNU cross-compile check. The live checks in
this receipt ran on Linux Xvfb with Vulkan Lavapipe. The other Document Tabs
scenarios used generated native documents and a mix of X11 and scripted
input in the desktop window.

The feature and support hashes matched before and after the full run.
All 39 scenarios and 343 steps passed on `ea2a6f9bb`.
The scenario source was unpinned and the build and runtime were unsealed.
Strict verification remains blocked under #3978.

```sh
tools/with-xvfb.sh cargo test --locked -p oblikovati-head \
  --lib application_scenarios_use_the_native_window -- --nocapture
```

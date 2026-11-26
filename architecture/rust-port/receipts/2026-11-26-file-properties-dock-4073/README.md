# Dock File Properties from Open

The Wayland reproduction dragged File Properties over the Material Browser
dock target. Its dock ID stayed zero while the target's dock ID was four.
The Open modal blocked the workspace receiving the drag.

Open now retains its draft and hides the picker while the independent File
Properties window is visible. The inspector keeps reading the chosen file's
metadata and thumbnail while Open is suspended. Closing File Properties
returns to the picker with its previous location, size, filename, and preview.

On Wayland and X11, the native test docked File Properties beside Material
Browser. Cancel revealed Material Browser in the first frame without File
Properties. The picker restored the selected filename and preview. Reopening
Properties selected its saved dock tab, and another Cancel restored the same
picker state. The active Part was preserved. Floating title-bar close and
sequential Escape checks also passed on both backends.

The published File Properties inspection scenario passed all 15 steps on
each backend against the shared fixture tree. All 28 focused GUI tests
passed. [Wayland report](wayland.xml) · [X11 report](x11.xml) · [Published inputs](current-inputs.sha256)

The build used `d82ad8888f5b5c89b89403a327ecac468b2b8612`, this change, and
existing local GUI filesystem and test edits. Wayland used headless Weston
with Pixman and frame-loop input. X11 used Xvfb with Openbox and native drag
and close input. Both used Vulkan Lavapipe.

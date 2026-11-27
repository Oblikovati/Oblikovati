# Keyboard focus wraps in the direct panel selector

The [native test](native.log) opens the 3D Model selector, focuses the
first row with Down, and checks the focus IDs after Up and Down. It
reaches the last row again through forward navigation, compares all
seven checkbox states, and confirms that the selector stays open.

The code at `bc808fb13` already wrapped focus. The test ran on Linux X11
with Xvfb, Openbox, and Vulkan Lavapipe. The published benchmark names
3D Print as the last row, which remains a gap in the current panel catalog.

See [#3856](https://github.com/Oblikovati/Oblikovati/issues/3856).

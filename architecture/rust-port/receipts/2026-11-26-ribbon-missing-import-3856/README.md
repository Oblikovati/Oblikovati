# Handle a missing customization import file

The native scenario opens a nonexistent XML path from the Customize import
picker. It checks the missing-file warning, acknowledges it, and verifies
that the picker reselects the full filename. Cancelling the picker leaves
Customize open without a pending change and with Apply disabled.

| Published scenario | Wayland | X11 |
| --- | --- | --- |
| Handle a missing customization import file | [11 steps passed](wayland/cucumber.json) | [11 steps passed](x11/cucumber.json) |

The existing behavior passed without a product-code change. The build
used `9e58ae120` content with existing local filesystem and marking-menu
edits. Wayland used headless Weston. X11 used Xvfb and Openbox. Both used
Vulkan Lavapipe. [Published input hashes](inputs.sha256)

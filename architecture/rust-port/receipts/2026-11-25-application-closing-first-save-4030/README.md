# Cancel the first Save As during application close

All eight steps passed on Wayland and X11. Escape dismissed Save As,
aborted application exit, and left the same metric Part open with its
unsaved XY sketch and no file path.
[Wayland report](wayland.json) · [X11 report](x11.json)

The [initial X11 run](x11-before.json) reached Save As and stopped at
the published Escape step, which had no binding. The revised X11 test
sends Alt+F4 and Escape through Openbox and XTest. The Wayland test
injects the compositor close request and the physical Escape event
through the native application frame loop on headless Weston.

Both runs used Vulkan Lavapipe and a build from `e2e23f78a` with the
test change in this commit and existing local GUI filesystem and Ribbon
Customization edits. The [feature hash](current-inputs.sha256) matched
before and after the runs. Strict provenance remains open under #3978.

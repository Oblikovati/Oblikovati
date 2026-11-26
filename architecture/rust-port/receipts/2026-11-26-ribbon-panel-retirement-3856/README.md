# Removing an applied User Commands panel

The ordinary Apply and removal scenario passed all 15 steps on Wayland and
X11. Applying the staged circle and separator showed the panel. Applying
their removal left Customize open with an empty target and hid the panel.
The separator remained reusable after scrolling the source list back up.

The Shell panel-retention scenario passed all 25 steps on both backends.
It verified that the empty User Commands panel stayed visible until
Customize closed and that Shell and the solid remained visible. The Shell
assertions here cover panel behavior around the visible solid. The certified
cutaway required by the separate preview scenarios is tracked in Martensite #554.

These runs used `357e5179a73372b69d454a98951aec2cdadd3bd2` with the existing
local GUI filesystem test-module visibility edit. Wayland used nested Weston
13, X11 used Openbox, and both used Xvfb and Vulkan Lavapipe. The [input hashes](current-inputs.sha256)
matched the shared source before and after the runs. Strict build and runtime
verification remains open under #3978.

- Apply and removal on [Wayland](retirement-wayland.json) and [X11](retirement-x11.json)
- Shell panel retention on [Wayland](retain-shell-wayland.json) and [X11](retain-shell-x11.json)

# Dock tab bar menu on Wayland and X11

The published scenario reached the Model dock on Wayland but stopped at
the plus button when its driver tried to connect to X11. The menu, item,
tab, and close clicks now use the Wayland window frame loop or XTest on
X11. The checks use rendered controls and confirm that Log opens in
Model's dock. Closing Log removes its tab and leaves Model visible in the
same dock.

The published scenario passed all nine steps with the current Part under
[Wayland](wayland/cucumber.json) and [X11](x11/cucumber.json). The bundled Part
also passed the [standalone regression](standalone.txt) on both backends.
[Log open](wayland/log-selected.png) · [Model after closing Log](wayland/log-closed.png)

The build used `03fd7abfa` with the menu driver change and existing local
filesystem and marking-menu edits. Wayland used headless Weston, X11 used
Xvfb and Openbox, and both used Vulkan Lavapipe.
[Published input hashes](inputs.sha256) · [Bundled input hash](bundled-input.sha256)

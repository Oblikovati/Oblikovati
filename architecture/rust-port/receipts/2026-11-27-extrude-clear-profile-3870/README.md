# Clear and repick an Extrude profile

The native regression starts with the rectangular preview in the
bundled rectangle-and-circle fixture. Clear removes the preview and
disables OK. Clicking the separate circle creates a new preview. A
second Clear waits for a fresh viewport pick, then a rectangle click
restores its preview. The 10 mm distance, document generation, and
empty undo history stay the same.

The published multi-profile scenarios still require simultaneous
selection, preview, and authoring for distinct regions. This
regression covers clearing and reselecting one profile.

[Wayland output](wayland-native.txt), [X11 output](x11-native.txt).

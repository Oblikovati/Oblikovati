# Restore a docked Projects window from New document

The Wayland reproduction seeded a Projects dock alongside Material Browser,
then restarted the native application. Opening Projects from New document
restored the docked tab, but the chooser modal kept its Done control out of
reach.

When Projects has a valid saved dock, New document now keeps its chooser
state while Projects is shown independently. Closing Projects restores
the chooser. The published floating workflow opens Projects above the chooser.

The native regression checks that Projects restores its dock after a restart,
accepts Done and Escape, and reveals Material Browser on the first frame
without Projects. Both closures restore the selected metric template and
the chooser's previous bounds. The active Part remains selected.

The published scenario passed all 14 steps on both backends.
[Wayland report](wayland.xml) [X11 report](x11.xml)

All 62 focused GUI tests passed. The five related native tests passed on
X11. On Wayland, four passed in the batch and the restart regression
passed in a fresh compositor after a Vulkan surface-loss error during
the batch restart.

The build used `088ed781a` with this change and the existing local GUI
filesystem edits. Wayland used headless Weston, X11 used Xvfb and Openbox,
and both rendered with Vulkan Lavapipe. The X11 restart regression uses
XTest for Done and Escape. The separately injected live-dock case scripts
Cancel in the next frame to check when the sibling tab becomes visible.
[Published input hashes](current-inputs.sha256)

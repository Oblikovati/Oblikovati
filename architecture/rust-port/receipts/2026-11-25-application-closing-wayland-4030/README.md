# Cancel application close with an unsaved Part

All nine steps of the current published scenario passed on Wayland and X11.
The test opened a disposable copy of the published extrusion Part, created
an unsaved XY sketch, selected the application's Close control, and verified
the Yes, No, and Cancel prompt. Cancel dismissed the prompt and left the
Part and the application open.
[Wayland report](wayland.json) · [Wayland prompt](wayland-prompt.png) · [X11 report](x11.json)

The Wayland run connected to headless Weston with no X11 display. It
delivered the two button clicks through the frame loop. The X11 run used
native XTest clicks under Xvfb. Both used Vulkan Lavapipe and the
`4e46aba91` build with the Wayland host and closing scenario changes,
alongside existing local filesystem and Ribbon Customization edits.
The shared source is unversioned. Strict provenance remains open under #3978.
[Published input hashes](current-inputs.sha256)

# Save and discard a modified Part when closing

Both published scenarios passed all nine steps on Wayland and X11.
Save updated the existing Part and the new XY sketch was present after
reopening. Discard preserved the original bytes and the sketch was absent
after reopening. The tests staged separate copies of the published Part.
[Save on Wayland](save-wayland.json) · [Save on X11](save-x11.json)
· [Discard on Wayland](discard-wayland.json) · [Discard on X11](discard-x11.json)

The [initial Wayland run](save-wayland-before.json) failed when the Alt+F4
step tried to connect to X11. The revised test injects the
close-request event the Wayland compositor would deliver, then clicks the
visible prompt through the native frame loop. The Discard scenario uses
the rendered application Close control. X11 uses Openbox and XTest to
deliver Alt+F4 and the clicks.

The builds used `d41d10843` with the test changes in this commit and
existing local GUI filesystem and Ribbon Customization edits. Both used
Vulkan Lavapipe. Wayland used headless Weston with no X11 display.
The [input hashes](current-inputs.sha256) matched before and after the
runs. The scenario mount is unversioned. Strict provenance remains open
under #3978.

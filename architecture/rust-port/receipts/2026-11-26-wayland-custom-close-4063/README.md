# Route the application bar Close control through the unsaved changes flow

The published case creates an unsaved XY sketch in a writable copy of
the native Part. The application bar Close control opens a save prompt
naming that Part. Cancelling the prompt leaves the same Part active
with its sketch count and dirty flag intact and the file bytes unchanged.

Wayland uses native pointer events delivered through nested Weston 13
on private Xvfb with Openbox. The test clicks the rendered Close and
Cancel controls at their compositor-relative positions. The protocol
trace records the delivered pointer events. The X11 run uses Xvfb
with Openbox and tint2. Both render with Vulkan Lavapipe.

At baseline `dac95b38c`, the Wayland steps sent scripted clicks into the
application frame loop. The verification build includes the native
pointer bindings and the existing local GUI filesystem edits.
[Input hashes](inputs.sha256)

All seven published steps passed on Wayland and X11. The native Wayland
clean-close and desktop-shortcut regression cases also passed.
[Wayland report](wayland.json) · [X11 report](x11.json) · [Native trace](test-results.txt)
